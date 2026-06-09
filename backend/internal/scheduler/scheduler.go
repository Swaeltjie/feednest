package scheduler

import (
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/feednest/backend/internal/fetcher"
	"github.com/feednest/backend/internal/readability"
	"github.com/feednest/backend/internal/store"
)

// maxItemsPerFeed caps how many items a single feed can trigger processing for
// in one fetch. gofeed will happily parse thousands of <item> entries out of a
// (up to 10MB) feed body, and each NEW item triggers a synchronous
// readability.Extract — a multi-strategy outbound HTTP fetch that can take tens
// of seconds. Without a cap, a single feed (which an attacker only needs to get
// a user to subscribe to) can make the server perform thousands of outbound
// fetches and occupy a worker slot for hours, starving every other feed and
// generating large amounts of outbound traffic (an amplification/SSRF-to-public
// vector via the article URLs).
const maxItemsPerFeed = 200

// capItems keeps at most maxItemsPerFeed items, preferring the newest by
// PublishedAt so the most relevant articles survive truncation. Items without a
// PublishedAt sort last (treated as oldest) so dated items are never dropped in
// favour of undated ones.
func capItems(items []fetcher.FeedItem) []fetcher.FeedItem {
	if len(items) <= maxItemsPerFeed {
		return items
	}
	sort.SliceStable(items, func(i, j int) bool {
		ti, tj := items[i].PublishedAt, items[j].PublishedAt
		switch {
		case ti == nil && tj == nil:
			return false
		case ti == nil:
			return false
		case tj == nil:
			return true
		default:
			return ti.After(*tj)
		}
	})
	return items[:maxItemsPerFeed]
}

type Scheduler struct {
	store       *store.Queries
	interval    time.Duration
	stop        chan struct{}
	stopOnce    sync.Once
	fetchSem    chan struct{} // limits concurrent immediate fetches
	backfilling atomic.Bool   // ensures only one thumbnail backfill runs at a time
	rescoring   atomic.Bool   // ensures only one engagement/score rescore runs at a time
}

func New(store *store.Queries, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    store,
		interval: interval,
		stop:     make(chan struct{}),
		fetchSem: make(chan struct{}, 5),
	}
}

func (s *Scheduler) Start() {
	go func() {
		go s.backfillThumbnails() // run concurrently, don't block first fetch
		go s.rescore()            // populate engagement + article scores
		s.fetchAll()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.fetchAll()
				// Self-heal any articles still missing a thumbnail (e.g. a
				// transient extraction failure at ingest time).
				go s.backfillThumbnails()
				// Recompute feed engagement and article scores so the
				// "smart" sort reflects recent reading behaviour.
				go s.rescore()
			case <-s.stop:
				return
			}
		}
	}()
	log.Printf("Feed scheduler started (interval: %v)", s.interval)
}

// backfillThumbnails fetches thumbnails for existing articles that are missing
// them. It is guarded so overlapping invocations (startup + periodic) don't
// duplicate work.
func (s *Scheduler) backfillThumbnails() {
	if !s.backfilling.CompareAndSwap(false, true) {
		return
	}
	defer s.backfilling.Store(false)

	articles, err := s.store.GetArticlesMissingThumbnails(500)
	if err != nil {
		log.Printf("scheduler: failed to get articles missing thumbnails: %v", err)
		return
	}
	if len(articles) == 0 {
		return
	}

	log.Printf("scheduler: backfilling thumbnails for %d articles", len(articles))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	var filled atomic.Int64

	for _, a := range articles {
		wg.Add(1)
		sem <- struct{}{}

		go func(id int64, articleURL string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scheduler: recovered from panic backfilling thumbnail for article %d: %v", id, r)
				}
			}()

			result, err := readability.Extract(articleURL)
			if err != nil || result.ThumbnailURL == "" {
				return
			}

			if err := s.store.UpdateArticleThumbnail(id, result.ThumbnailURL); err != nil {
				log.Printf("scheduler: failed to update thumbnail for article %d: %v", id, err)
				return
			}
			filled.Add(1)
		}(a.ID, a.URL)
	}

	wg.Wait()
	log.Printf("scheduler: backfilled %d/%d thumbnails", filled.Load(), len(articles))
}

// rescore recomputes per-feed engagement scores from recent reading behaviour
// and then re-scores recent articles for the "smart" sort. It is guarded so
// overlapping invocations (startup + periodic) don't duplicate work. Engagement
// is updated first because article scoring reads feeds.engagement_score.
func (s *Scheduler) rescore() {
	if !s.rescoring.CompareAndSwap(false, true) {
		return
	}
	defer s.rescoring.Store(false)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("scheduler: recovered from panic during rescore: %v", r)
		}
	}()

	if err := s.store.UpdateFeedEngagementScores(); err != nil {
		log.Printf("scheduler: failed to update feed engagement scores: %v", err)
		return
	}

	scored, err := s.store.ScoreRecentArticles()
	if err != nil {
		log.Printf("scheduler: failed to score recent articles: %v", err)
		return
	}
	log.Printf("scheduler: rescored %d recent articles", scored)
}

func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *Scheduler) FetchFeedNow(feedID int64, feedURL string, userID int64) {
	// Acquire the concurrency slot BEFORE spawning, non-blocking. The old code
	// blocked on the semaphore inside the goroutine, so the semaphore bounded
	// concurrency but not the number of spawned/blocked goroutines — an
	// authenticated user spamming /api/feeds or /retry could pile up unbounded
	// goroutines. Overflow is dropped; the periodic scheduler reconciles it.
	select {
	case s.fetchSem <- struct{}{}:
	default:
		log.Printf("scheduler: immediate fetch of feed %d dropped, fetch queue full", feedID)
		return
	}
	go func() {
		defer func() { <-s.fetchSem }()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("scheduler: recovered from panic during immediate fetch of feed %d (%s): %v", feedID, feedURL, r)
			}
		}()
		result, err := fetcher.FetchFeed(feedURL)
		if err != nil {
			log.Printf("scheduler: immediate fetch failed for %s: %v", feedURL, err)
			s.store.SetFeedError(feedID, err.Error())
			return
		}

		if result.Title != "" {
			update := &store.FeedMetadataUpdate{}
			// Only set title/siteURL if the feed doesn't have one yet
			// (preserves user-set titles from PUT /api/feeds/{id})
			feed, feedErr := s.store.GetFeed(feedID, userID)
			if feedErr == nil && feed.Title == "" {
				update.Title = &result.Title
				update.SiteURL = &result.SiteURL
			}
			if result.IconURL != "" {
				update.IconURL = &result.IconURL
			}
			if err := s.store.UpdateFeedMetadata(feedID, update); err != nil {
				log.Printf("scheduler: failed to update metadata for feed %d: %v", feedID, err)
			}
		}

		// Cap items so a single (possibly hostile) feed cannot trigger
		// thousands of outbound readability fetches in one go.
		result.Items = capItems(result.Items)

		for _, item := range result.Items {
			// Skip readability extraction for articles that already exist so
			// repeated /retry calls don't re-extract the entire feed.
			if s.store.ArticleExistsByGUID(feedID, item.GUID) {
				continue
			}

			thumbnailURL := item.ThumbnailURL
			contentRaw := item.ContentRaw

			// Sanitize blocked content from RSS raw content
			if readability.IsBlockedContent(contentRaw) {
				contentRaw = ""
			}

			var contentClean string
			if item.URL != "" {
				if extracted, err := readability.Extract(item.URL); err == nil {
					contentClean = extracted.Content
					if thumbnailURL == "" {
						thumbnailURL = extracted.ThumbnailURL
					}
				}
			}

			if _, err := s.store.CreateArticleAndApplyRules(
				userID, feedID, item.GUID, item.Title, item.URL, item.Author,
				contentRaw, contentClean, thumbnailURL,
				item.PublishedAt, item.WordCount, item.ReadingTime,
			); err != nil {
				log.Printf("scheduler: failed to create article %q: %v", item.GUID, err)
			}
		}

		if err := s.store.ClearFeedError(feedID); err != nil {
			log.Printf("scheduler: failed to clear error for feed %d: %v", feedID, err)
		}
		if err := s.store.UpdateFeedLastFetched(feedID); err != nil {
			log.Printf("scheduler: failed to update last_fetched for feed %d: %v", feedID, err)
		}
		log.Printf("scheduler: immediate fetch of %s (%d items)", feedURL, len(result.Items))
	}()
}

func (s *Scheduler) fetchAll() {
	feeds, err := s.store.GetFeedsDueForFetch()
	if err != nil {
		log.Printf("scheduler: failed to get feeds: %v", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	log.Printf("scheduler: fetching %d feeds", len(feeds))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, feed := range feeds {
		wg.Add(1)
		sem <- struct{}{}

		go func(feedID, userID int64, feedURL, feedTitle string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scheduler: recovered from panic fetching feed %d (%s): %v", feedID, feedURL, r)
				}
			}()

			result, err := fetcher.FetchFeed(feedURL)
			if err != nil {
				log.Printf("scheduler: failed to fetch %s: %v", feedURL, err)
				s.store.SetFeedError(feedID, err.Error())
				return
			}

			if result.Title != "" {
				update := &store.FeedMetadataUpdate{}
				if feedTitle == "" {
					update.Title = &result.Title
					update.SiteURL = &result.SiteURL
				}
				if result.IconURL != "" {
					update.IconURL = &result.IconURL
				}
				if err := s.store.UpdateFeedMetadata(feedID, update); err != nil {
					log.Printf("scheduler: failed to update metadata for feed %d: %v", feedID, err)
				}
			}

			// Cap items so a single (possibly hostile) feed cannot trigger
			// thousands of outbound readability fetches in one go.
			result.Items = capItems(result.Items)

			newItems := 0
			for _, item := range result.Items {
				// Skip readability extraction for articles that already exist
				if s.store.ArticleExistsByGUID(feedID, item.GUID) {
					continue
				}

				thumbnailURL := item.ThumbnailURL
				contentRaw := item.ContentRaw

				// Sanitize blocked content from RSS raw content
				if readability.IsBlockedContent(contentRaw) {
					contentRaw = ""
				}

				var contentClean string
				if item.URL != "" {
					// Fetch the article page once for BOTH content and a
					// thumbnail (og:image/twitter:image). Feeds like BBC and
					// Hacker News often ship no image in the RSS item, so the
					// page-level thumbnail is the only source.
					if extracted, err := readability.Extract(item.URL); err == nil {
						contentClean = extracted.Content
						if thumbnailURL == "" {
							thumbnailURL = extracted.ThumbnailURL
						}
					}
					// Last resort: an <img> embedded in the RSS content itself.
					if thumbnailURL == "" {
						thumbnailURL = readability.ExtractThumbnailFromHTML(item.ContentRaw)
					}
				}

				created, err := s.store.CreateArticleAndApplyRules(
					userID, feedID, item.GUID, item.Title, item.URL, item.Author,
					contentRaw, contentClean, thumbnailURL,
					item.PublishedAt, item.WordCount, item.ReadingTime,
				)
				if err != nil {
					log.Printf("scheduler: failed to create article %q: %v", item.GUID, err)
					continue
				}
				if created {
					newItems++
				}
			}

			if err := s.store.ClearFeedError(feedID); err != nil {
				log.Printf("scheduler: failed to clear error for feed %d: %v", feedID, err)
			}
			if err := s.store.UpdateFeedLastFetched(feedID); err != nil {
				log.Printf("scheduler: failed to update last_fetched for feed %d: %v", feedID, err)
			}
			log.Printf("scheduler: fetched %s (%d new / %d total items)", feedURL, newItems, len(result.Items))
		}(feed.ID, feed.UserID, feed.URL, feed.Title)
	}

	wg.Wait()
}

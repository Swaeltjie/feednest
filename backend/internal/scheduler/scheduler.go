package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/feednest/backend/internal/fetcher"
	"github.com/feednest/backend/internal/readability"
	"github.com/feednest/backend/internal/store"
)

type Scheduler struct {
	store    *store.Queries
	interval time.Duration
	stop     chan struct{}
}

func New(store *store.Queries, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    store,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go func() {
		s.fetchAll()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.fetchAll()
			case <-s.stop:
				return
			}
		}
	}()
	log.Printf("Feed scheduler started (interval: %v)", s.interval)
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) FetchFeedNow(feedID int64, feedURL string) {
	go func() {
		result, err := fetcher.FetchFeed(feedURL)
		if err != nil {
			log.Printf("scheduler: immediate fetch failed for %s: %v", feedURL, err)
			return
		}

		if result.Title != "" {
			s.store.UpdateFeedMetadata(feedID, &store.FeedMetadataUpdate{
				Title:   &result.Title,
				SiteURL: &result.SiteURL,
				IconURL: &result.IconURL,
			})
		}

		for _, item := range result.Items {
			thumbnailURL := item.ThumbnailURL
			contentRaw := item.ContentRaw

			// Sanitize blocked content from RSS raw content
			if readability.IsBlockedContent(contentRaw) {
				contentRaw = ""
			}

			var contentClean string
			if item.URL != "" {
				if clean, err := readability.ExtractContent(item.URL); err == nil {
					contentClean = clean
				}
				if thumbnailURL == "" {
					thumbnailURL = readability.ExtractThumbnailFromHTML(item.ContentRaw)
				}
			}

			s.store.CreateArticle(
				feedID, item.GUID, item.Title, item.URL, item.Author,
				contentRaw, contentClean, thumbnailURL,
				item.PublishedAt, item.WordCount, item.ReadingTime,
			)
		}

		s.store.UpdateFeedLastFetched(feedID)
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

		go func(feedID int64, feedURL, feedTitle string) {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := fetcher.FetchFeed(feedURL)
			if err != nil {
				log.Printf("scheduler: failed to fetch %s: %v", feedURL, err)
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
				s.store.UpdateFeedMetadata(feedID, update)
			}

			for _, item := range result.Items {
				thumbnailURL := item.ThumbnailURL
				contentRaw := item.ContentRaw

				// Sanitize blocked content from RSS raw content
				if readability.IsBlockedContent(contentRaw) {
					contentRaw = ""
				}

				var contentClean string
				if item.URL != "" {
					if clean, err := readability.ExtractContent(item.URL); err == nil {
						contentClean = clean
					}
					if thumbnailURL == "" {
						thumbnailURL = readability.ExtractThumbnailFromHTML(item.ContentRaw)
					}
				}

				s.store.CreateArticle(
					feedID, item.GUID, item.Title, item.URL, item.Author,
					contentRaw, contentClean, thumbnailURL,
					item.PublishedAt, item.WordCount, item.ReadingTime,
				)
			}

			s.store.UpdateFeedLastFetched(feedID)
			log.Printf("scheduler: fetched %s (%d items)", feedURL, len(result.Items))
		}(feed.ID, feed.URL, feed.Title)
	}

	wg.Wait()
}

# UX Enhancements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add infinite scroll, feed autodiscovery, feed error indicators, and reader customization to FeedNest.

**Architecture:** All four features are independent and can be implemented sequentially. Infinite scroll modifies the existing articles store and page template. Autodiscovery adds a new backend endpoint and extends the Add Feed modal. Error indicators add a DB column and sidebar UI. Reader customization adds a popover component and CSS variables backed by the settings API.

**Tech Stack:** Go (Chi, SQLite), SvelteKit 5 (Svelte 5 runes), TailwindCSS 4, TypeScript

---

## Task 1: Infinite Scroll — Articles Store

**Files:**
- Modify: `frontend/src/lib/stores/articles.ts`

**Step 1: Add `loadMore` and `hasMore` to the articles store**

Add a `loadingMore` flag to the store state, a `loadMore` method that appends results, and expose `hasMore`:

```typescript
// In the writable state type, add:
loadingMore: boolean;

// Initial state becomes:
{ articles: [], total: 0, loading: false, loadingMore: false }

// Add after the existing `load` method:
async loadMore(filters: ArticleFilters = {}) {
    const thisLoad = ++loadId;
    update((s) => {
        if (s.articles.length >= s.total) return s;
        return { ...s, loadingMore: true };
    });
    const params = new URLSearchParams();
    if (filters.status) params.set('status', filters.status);
    params.set('sort', filters.sort || 'smart');
    if (filters.feed) params.set('feed', String(filters.feed));
    if (filters.category) params.set('category', String(filters.category));
    if (filters.tag) params.set('tag', filters.tag);
    if (filters.search) params.set('search', filters.search);

    let currentPage = 1;
    update((s) => {
        currentPage = Math.floor(s.articles.length / 30) + 1;
        return s;
    });
    params.set('page', String(currentPage));

    try {
        const data = await api.get<ArticlesResponse>(`/api/articles?${params}`);
        if (thisLoad !== loadId) return;
        update((s) => ({
            articles: [...s.articles, ...(data.articles || [])],
            total: data.total,
            loading: false,
            loadingMore: false,
        }));
    } catch {
        if (thisLoad !== loadId) return;
        update((s) => ({ ...s, loadingMore: false }));
    }
},
```

**Step 2: Run frontend check**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 3: Commit**

```bash
git add frontend/src/lib/stores/articles.ts
git commit -m "feat: add loadMore method to articles store for infinite scroll"
```

---

## Task 2: Infinite Scroll — Page Template

**Files:**
- Modify: `frontend/src/routes/+page.svelte`

**Step 1: Add IntersectionObserver and sentinel element**

Add a `sentinelEl` binding and set up the observer. In the `<script>` block, add:

```typescript
let sentinelEl: HTMLElement | undefined = $state();

$effect(() => {
    if (!sentinelEl) return;
    const observer = new IntersectionObserver(
        (entries) => {
            if (entries[0].isIntersecting && !$articles.loading && !$articles.loadingMore && $articles.articles.length < $articles.total) {
                articles.loadMore(currentFilters);
            }
        },
        { rootMargin: '300px' }
    );
    observer.observe(sentinelEl);
    return () => observer.disconnect();
});
```

**Step 2: Add sentinel div and loading spinner in the template**

Replace the existing "Showing X of Y articles" block (around lines 619-623) with:

```svelte
{#if $articles.articles.length < $articles.total}
    <div bind:this={sentinelEl} class="flex items-center justify-center py-6">
        {#if $articles.loadingMore}
            <div class="w-5 h-5 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
        {/if}
    </div>
{:else if $articles.articles.length > 0}
    <div class="text-center py-4 text-sm text-[var(--color-text-tertiary)]">
        Showing all {$articles.total} articles
    </div>
{/if}
```

**Step 3: Update keyboard nav `G` to trigger loadMore**

In the keyboard shortcuts, update the `G` handler to also trigger loadMore when jumping to end:

```typescript
'G': (e) => {
    const articleList = $articles.articles;
    if (articleList.length > 0) {
        selectedIndex = articleList.length - 1;
        scrollSelectedIntoView();
        if (articleList.length < $articles.total) {
            articles.loadMore(currentFilters);
        }
    }
},
```

**Step 4: Run frontend check**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 5: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: add infinite scroll with IntersectionObserver sentinel"
```

---

## Task 3: Feed Autodiscovery — Backend Endpoint

**Files:**
- Create: `backend/internal/api/handlers/discover.go`
- Modify: `backend/internal/api/router.go:85` (add route)

**Step 1: Create the discover handler**

Create `backend/internal/api/handlers/discover.go`:

```go
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
	"github.com/mmcdole/gofeed"
)

type DiscoverHandler struct{}

func NewDiscoverHandler() *DiscoverHandler {
	return &DiscoverHandler{}
}

type DiscoveredFeed struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type DiscoverRequest struct {
	URL string `json:"url"`
}

var linkRe = regexp.MustCompile(`(?i)<link[^>]+>`)
var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
var typeRe = regexp.MustCompile(`(?i)type\s*=\s*["']([^"']+)["']`)
var titleRe = regexp.MustCompile(`(?i)title\s*=\s*["']([^"']+)["']`)
var relRe = regexp.MustCompile(`(?i)rel\s*=\s*["']([^"']+)["']`)

func (h *DiscoverHandler) Discover(w http.ResponseWriter, r *http.Request) {
	var req DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	if err := urlutil.IsSafeURL(req.URL); err != nil {
		http.Error(w, `{"error":"unsafe URL"}`, http.StatusBadRequest)
		return
	}

	// First, try parsing as a feed directly
	client := urlutil.SafeHTTPClient(15 * time.Second)
	httpReq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		http.Error(w, `{"error":"invalid URL"}`, http.StatusBadRequest)
		return
	}
	httpReq.Header.Set("User-Agent", "FeedNest/1.0")
	httpReq.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, text/html, */*")

	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch URL"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		http.Error(w, `{"error":"failed to read response"}`, http.StatusBadGateway)
		return
	}

	// Try parsing as feed
	fp := gofeed.NewParser()
	if feed, err := fp.ParseString(string(body)); err == nil && feed != nil {
		feedType := "rss"
		if feed.FeedType == "atom" {
			feedType = "atom"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"feeds": []DiscoveredFeed{{URL: req.URL, Title: feed.Title, Type: feedType}},
		})
		return
	}

	// Not a feed — parse HTML for <link rel="alternate"> tags
	html := string(body)
	var feeds []DiscoveredFeed

	links := linkRe.FindAllString(html, -1)
	for _, link := range links {
		relMatch := relRe.FindStringSubmatch(link)
		if len(relMatch) < 2 || !strings.Contains(strings.ToLower(relMatch[1]), "alternate") {
			continue
		}

		typeMatch := typeRe.FindStringSubmatch(link)
		if len(typeMatch) < 2 {
			continue
		}
		feedType := strings.ToLower(typeMatch[1])
		if !strings.Contains(feedType, "rss") && !strings.Contains(feedType, "atom") && !strings.Contains(feedType, "xml") {
			continue
		}

		hrefMatch := hrefRe.FindStringSubmatch(link)
		if len(hrefMatch) < 2 {
			continue
		}
		feedURL := hrefMatch[1]

		// Resolve relative URLs
		if !strings.HasPrefix(feedURL, "http") {
			base, err := url.Parse(req.URL)
			if err != nil {
				continue
			}
			ref, err := url.Parse(feedURL)
			if err != nil {
				continue
			}
			feedURL = base.ResolveReference(ref).String()
		}

		title := ""
		titleMatch := titleRe.FindStringSubmatch(link)
		if len(titleMatch) >= 2 {
			title = titleMatch[1]
		}

		shortType := "rss"
		if strings.Contains(feedType, "atom") {
			shortType = "atom"
		}

		feeds = append(feeds, DiscoveredFeed{URL: feedURL, Title: title, Type: shortType})
	}

	if len(feeds) == 0 {
		http.Error(w, `{"error":"no feeds found at this URL"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"feeds": feeds})
}
```

**Step 2: Register the route in `router.go`**

In `backend/internal/api/router.go`, inside the protected route group, after the feeds routes (after line 85), add:

```go
discoverH := handlers.NewDiscoverHandler()
r.Post("/api/feeds/discover", discoverH.Discover)
```

**Step 3: Build and test**

Run: `cd backend && go build ./...`
Expected: clean build

**Step 4: Commit**

```bash
git add backend/internal/api/handlers/discover.go backend/internal/api/router.go
git commit -m "feat: add feed autodiscovery endpoint POST /api/feeds/discover"
```

---

## Task 4: Feed Autodiscovery — Frontend Modal

**Files:**
- Modify: `frontend/src/routes/+page.svelte` (Add Feed modal section, around lines 673-753)

**Step 1: Add discover state variables**

In the `<script>` block, add near the other add-feed state variables:

```typescript
let discoveredFeeds = $state<{ url: string; title: string; type: string }[]>([]);
let selectedDiscovered = $state<Set<string>>(new Set());
let discovering = $state(false);
```

**Step 2: Add discover function**

```typescript
async function handleDiscover() {
    if (!feedUrl.trim()) return;
    discovering = true;
    addFeedError = '';
    discoveredFeeds = [];
    selectedDiscovered = new Set();
    try {
        const data = await api.post<{ feeds: { url: string; title: string; type: string }[] }>(
            '/api/feeds/discover',
            { url: feedUrl.trim() }
        );
        if (data.feeds.length === 1) {
            // Single feed — add directly
            feedUrl = data.feeds[0].url;
            await handleAddFeed();
        } else {
            discoveredFeeds = data.feeds;
            selectedDiscovered = new Set(data.feeds.map((f) => f.url));
        }
    } catch (err) {
        addFeedError = err instanceof Error ? err.message : 'No feeds found at this URL';
    } finally {
        discovering = false;
    }
}

async function handleAddDiscovered() {
    addingFeed = true;
    addFeedError = '';
    try {
        for (const feedInfo of discoveredFeeds.filter((f) => selectedDiscovered.has(f.url))) {
            const body: Record<string, unknown> = { url: feedInfo.url };
            if (feedCategoryId) body.category_id = feedCategoryId;
            if (newCategoryName.trim()) body.new_category = newCategoryName.trim();
            await api.post('/api/feeds', body);
        }
        await feeds.load();
        await categories.load();
        showAddFeedModal = false;
        discoveredFeeds = [];
        await articles.load(currentFilters);
    } catch (err) {
        addFeedError = err instanceof Error ? err.message : 'Failed to add feeds';
    } finally {
        addingFeed = false;
    }
}
```

**Step 3: Update `openAddFeed` to reset discover state**

Add to the existing `openAddFeed` function:

```typescript
discoveredFeeds = [];
selectedDiscovered = new Set();
```

**Step 4: Update the modal template**

Replace the current "Add Feed" button in the modal with a two-button layout, and add a discovered feeds list. The button section (currently around the `handleAddFeed` onclick) becomes:

After the category input section and before the button row, add the discovered feeds picker:

```svelte
{#if discoveredFeeds.length > 1}
    <div>
        <label class="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
            Found {discoveredFeeds.length} feeds
        </label>
        <div class="space-y-2 max-h-40 overflow-y-auto rounded-xl border border-[var(--color-border)] p-2">
            {#each discoveredFeeds as feed}
                <label class="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-[var(--color-elevated)] cursor-pointer text-sm">
                    <input
                        type="checkbox"
                        checked={selectedDiscovered.has(feed.url)}
                        onchange={() => {
                            const next = new Set(selectedDiscovered);
                            if (next.has(feed.url)) next.delete(feed.url);
                            else next.add(feed.url);
                            selectedDiscovered = next;
                        }}
                        class="rounded accent-[var(--color-accent)]"
                    />
                    <span class="flex-1 truncate text-[var(--color-text-primary)]">
                        {feed.title || feed.url}
                    </span>
                    <span class="text-xs text-[var(--color-text-tertiary)] uppercase">{feed.type}</span>
                </label>
            {/each}
        </div>
    </div>
{/if}
```

Update the button row: if `discoveredFeeds.length > 1`, show "Add Selected" calling `handleAddDiscovered`. Otherwise show two buttons: "Discover" (calls `handleDiscover`) and "Add" (calls `handleAddFeed` directly):

```svelte
<div class="flex justify-end gap-3 pt-2">
    <button
        onclick={() => (showAddFeedModal = false)}
        class="px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)]
            hover:bg-[var(--color-elevated)] rounded-xl transition-colors"
    >
        Cancel
    </button>
    {#if discoveredFeeds.length > 1}
        <button
            onclick={handleAddDiscovered}
            disabled={addingFeed || selectedDiscovered.size === 0}
            class="px-5 py-2 text-sm font-medium text-white rounded-xl accent-gradient
                hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity
                shadow-lg shadow-blue-500/25"
        >
            {addingFeed ? 'Adding...' : `Add ${selectedDiscovered.size} Feed${selectedDiscovered.size !== 1 ? 's' : ''}`}
        </button>
    {:else}
        <button
            onclick={handleDiscover}
            disabled={discovering || !feedUrl.trim()}
            class="px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)]
                border border-[var(--color-border)] rounded-xl
                hover:bg-[var(--color-elevated)] disabled:opacity-50 transition-colors"
        >
            {discovering ? 'Finding...' : 'Discover'}
        </button>
        <button
            onclick={handleAddFeed}
            disabled={addingFeed || !feedUrl.trim()}
            class="px-5 py-2 text-sm font-medium text-white rounded-xl accent-gradient
                hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity
                shadow-lg shadow-blue-500/25"
        >
            {addingFeed ? 'Adding...' : 'Add Feed'}
        </button>
    {/if}
</div>
```

**Step 5: Run frontend check**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 6: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: add feed autodiscovery UI with multi-feed picker"
```

---

## Task 5: Feed Error Indicators — Backend

**Files:**
- Modify: `backend/internal/store/migrations.go` (add column)
- Modify: `backend/internal/models/feed.go` (add field)
- Modify: `backend/internal/store/feed_store.go` (add methods, update queries)
- Modify: `backend/internal/scheduler/scheduler.go` (store errors)
- Modify: `backend/internal/api/handlers/feeds.go` (add retry endpoint)
- Modify: `backend/internal/api/router.go` (add route)

**Step 1: Add migration for `last_error` column**

In `backend/internal/store/migrations.go`, add after the existing schema indexes (before the closing backtick):

```sql
-- Feed error tracking (added via ALTER for existing DBs)
```

And add a separate migration call after `runMigrations`. Add a new function:

```go
func runAlterMigrations(db *sql.DB) {
	// Add last_error column if it doesn't exist
	db.Exec("ALTER TABLE feeds ADD COLUMN last_error TEXT")
}
```

Call this from `NewQueries` (or wherever `runMigrations` is called). Check `backend/internal/store/db.go` for the call site.

**Step 2: Update the Feed model**

In `backend/internal/models/feed.go`, add to the `Feed` struct:

```go
LastError *string `json:"last_error"`
```

**Step 3: Update feed store queries**

In `backend/internal/store/feed_store.go`:

- Update `ListFeeds` query to include `f.last_error` in the SELECT and Scan.
- Update `GetFeed` query to include `f.last_error` in the SELECT and Scan.
- Update `GetFeedsDueForFetch` query to include `last_error` in the SELECT and Scan.
- Add new methods:

```go
func (q *Queries) SetFeedError(id int64, errMsg string) error {
	_, err := q.db.Exec("UPDATE feeds SET last_error = ? WHERE id = ?", errMsg, id)
	return err
}

func (q *Queries) ClearFeedError(id int64) error {
	_, err := q.db.Exec("UPDATE feeds SET last_error = NULL WHERE id = ?", id)
	return err
}
```

**Step 4: Update scheduler to store/clear errors**

In `backend/internal/scheduler/scheduler.go`:

In `fetchAll`, after `fetcher.FetchFeed` fails (line ~123), add:
```go
s.store.SetFeedError(feedID, err.Error())
return
```

After a successful fetch (before `UpdateFeedLastFetched`), add:
```go
s.store.ClearFeedError(feedID)
```

Same pattern in `FetchFeedNow`: store error on failure, clear on success.

**Step 5: Add retry endpoint**

In `backend/internal/api/handlers/feeds.go`, add a `Retry` method:

```go
func (h *FeedHandler) Retry(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	feed, err := h.store.GetFeed(id, userID)
	if err != nil {
		http.Error(w, `{"error":"feed not found"}`, http.StatusNotFound)
		return
	}

	if h.scheduler != nil {
		h.scheduler.FetchFeedNow(feed.ID, feed.URL)
	}

	w.WriteHeader(http.StatusAccepted)
}
```

**Step 6: Register route**

In `backend/internal/api/router.go`, after the feeds DELETE route (line 85), add:

```go
r.Post("/api/feeds/{id}/retry", feedsH.Retry)
```

**Step 7: Build and test**

Run: `cd backend && go build ./... && go test ./...`
Expected: clean build, all tests pass

**Step 8: Commit**

```bash
git add backend/internal/store/migrations.go backend/internal/models/feed.go \
    backend/internal/store/feed_store.go backend/internal/scheduler/scheduler.go \
    backend/internal/api/handlers/feeds.go backend/internal/api/router.go
git commit -m "feat: add feed error tracking with last_error column and retry endpoint"
```

---

## Task 6: Feed Error Indicators — Frontend

**Files:**
- Modify: `frontend/src/lib/stores/feeds.ts` (add `last_error` to interface)
- Modify: `frontend/src/lib/components/Sidebar.svelte` (show error dot + tooltip)

**Step 1: Update Feed interface**

In `frontend/src/lib/stores/feeds.ts`, add to the `Feed` interface:

```typescript
last_error: string | null;
```

Add a `retry` method to the feeds store:

```typescript
async retry(id: number) {
    await api.post(`/api/feeds/${id}/retry`);
},
```

**Step 2: Update Sidebar to show error indicators**

In `frontend/src/lib/components/Sidebar.svelte`, in both the categorized and uncategorized feed `<button>` elements, after the feed title `<span>`, add an error indicator:

```svelte
{#if feed.last_error}
    <span
        class="w-2 h-2 rounded-full bg-orange-500 flex-shrink-0"
        title={feed.last_error}
    ></span>
{/if}
```

Add a "Retry" option to the context menu (alongside "Remove Feed"):

```svelte
{#if contextMenu?.feed.last_error}
    <button
        onclick={() => { if (contextMenu) { feeds.retry(contextMenu.feed.id); closeContextMenu(); } }}
        class="flex items-center gap-2.5 w-full px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] transition-colors"
    >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Retry Fetch
    </button>
{/if}
```

**Step 3: Run frontend check**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 4: Commit**

```bash
git add frontend/src/lib/stores/feeds.ts frontend/src/lib/components/Sidebar.svelte
git commit -m "feat: show feed error indicators in sidebar with retry action"
```

---

## Task 7: Reader Customization — Settings Store

**Files:**
- Modify: `frontend/src/lib/stores/settings.ts`
- Modify: `backend/internal/api/handlers/settings.go` (add allowed keys)

**Step 1: Add reader settings keys to backend allowlist**

In `backend/internal/api/handlers/settings.go`, update `allowedSettingKeys`:

```go
var allowedSettingKeys = map[string]bool{
	"theme": true, "view_mode": true, "default_sort": true,
	"articles_per_page": true, "auto_mark_read": true,
	"refresh_interval": true, "language": true,
	"font_size": true, "compact_mode": true,
	"reader_font_size": true, "reader_font_family": true,
	"reader_line_height": true, "reader_content_width": true,
}
```

**Step 2: Extend the settings store with reader preferences**

Rewrite `frontend/src/lib/stores/settings.ts` to include reader settings and API persistence:

```typescript
import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export type Theme = 'light' | 'dark' | 'system';
export type ReaderFontSize = 'small' | 'medium' | 'large' | 'xl';
export type ReaderFontFamily = 'sans' | 'serif' | 'mono';
export type ReaderLineHeight = 'compact' | 'comfortable' | 'spacious';
export type ReaderContentWidth = 'narrow' | 'medium' | 'wide';

interface SettingsState {
	theme: Theme;
	readerFontSize: ReaderFontSize;
	readerFontFamily: ReaderFontFamily;
	readerLineHeight: ReaderLineHeight;
	readerContentWidth: ReaderContentWidth;
}

const READER_FONT_SIZE_MAP: Record<ReaderFontSize, string> = {
	small: '15px', medium: '17px', large: '19px', xl: '21px',
};
const READER_FONT_FAMILY_MAP: Record<ReaderFontFamily, string> = {
	sans: 'ui-sans-serif, system-ui, sans-serif',
	serif: 'ui-serif, Georgia, Cambria, serif',
	mono: 'ui-monospace, SFMono-Regular, monospace',
};
const READER_LINE_HEIGHT_MAP: Record<ReaderLineHeight, string> = {
	compact: '1.5', comfortable: '1.75', spacious: '2.0',
};
const READER_CONTENT_WIDTH_MAP: Record<ReaderContentWidth, string> = {
	narrow: '580px', medium: '680px', wide: '820px',
};

export { READER_FONT_SIZE_MAP, READER_FONT_FAMILY_MAP, READER_LINE_HEIGHT_MAP, READER_CONTENT_WIDTH_MAP };
```

Keep the existing theme logic. Add methods `setReaderFontSize`, `setReaderFontFamily`, `setReaderLineHeight`, `setReaderContentWidth` that update the store and call `api.put('/api/settings', { reader_font_size: value })`. Add a `loadFromAPI` method that fetches settings on app init.

Load defaults from localStorage as a fast path, persist to API for cross-device sync.

**Step 3: Build backend, run frontend check**

Run: `cd backend && go build ./...`
Run: `cd frontend && npm run check`
Expected: both clean

**Step 4: Commit**

```bash
git add frontend/src/lib/stores/settings.ts backend/internal/api/handlers/settings.go
git commit -m "feat: add reader customization settings to store and backend allowlist"
```

---

## Task 8: Reader Customization — Popover Component

**Files:**
- Create: `frontend/src/lib/components/ReaderSettings.svelte`
- Modify: `frontend/src/lib/components/ArticleReader.svelte`
- Modify: `frontend/src/routes/article/[id]/+page.svelte`

**Step 1: Create ReaderSettings popover component**

Create `frontend/src/lib/components/ReaderSettings.svelte`:

A popover with four rows of segmented buttons for font size, font family, line height, and content width. Import from the settings store. Each button calls the corresponding setter. Use Svelte 5 runes syntax. The component receives an `open` bindable prop and positions itself relative to the trigger button.

The popover should have:
- Title "Reader Settings" at the top
- Four labeled rows with segmented button groups
- Close on click outside (via `<svelte:window onclick>` pattern)
- Smooth fade-in animation

**Step 2: Add "Aa" button and popover to ArticleReader**

In `frontend/src/lib/components/ArticleReader.svelte`:

- Import `ReaderSettings` and the CSS maps from settings store
- Add a `readerSettingsOpen` state variable
- Add an "Aa" button in the header toolbar (between the star and external link buttons)
- Render `<ReaderSettings bind:open={readerSettingsOpen} />` next to the button
- Apply reader settings as inline CSS variables on the article content `<div>`:

```svelte
style="font-size: {READER_FONT_SIZE_MAP[$settings.readerFontSize]};
       font-family: {READER_FONT_FAMILY_MAP[$settings.readerFontFamily]};
       line-height: {READER_LINE_HEIGHT_MAP[$settings.readerLineHeight]};
       max-width: {READER_CONTENT_WIDTH_MAP[$settings.readerContentWidth]};"
```

**Step 3: Apply same settings to article/[id]/+page.svelte**

Import the same maps and apply inline styles to the article content div.

**Step 4: Run frontend check**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 5: Commit**

```bash
git add frontend/src/lib/components/ReaderSettings.svelte \
    frontend/src/lib/components/ArticleReader.svelte \
    frontend/src/routes/article/[id]/+page.svelte
git commit -m "feat: add reader customization popover with font, spacing, and width controls"
```

---

## Task 9: Final Integration Test

**Step 1: Run full backend tests**

Run: `cd backend && go test ./...`
Expected: all pass

**Step 2: Run full frontend checks**

Run: `cd frontend && npm run check`
Expected: 0 errors

**Step 3: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: cleanup after UX enhancements"
```

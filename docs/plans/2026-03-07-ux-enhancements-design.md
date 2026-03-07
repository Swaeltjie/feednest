# FeedNest UX Enhancements Design

Date: 2026-03-07

## Features (in priority order)

### 1. Infinite Scroll

The article list watches scroll position. When the user scrolls within 300px of the bottom and more articles exist (`total > articles.length`), the next page loads automatically and appends. A small spinner shows at the bottom during loading.

**Changes:**

- **articles store** - Add `loadMore(filters)` that increments page and appends results instead of replacing. Track `hasMore` derived from `total > articles.length`.
- **+page.svelte** - Add an `IntersectionObserver` on a sentinel element at the bottom of the article list. When it intersects, call `loadMore`. Disable when loading or no more pages.
- **Keyboard nav** - `j`/`k` and `G` automatically trigger load-more if the selected index approaches the end.

### 2. Feed Autodiscovery

When the user submits a URL in the Add Feed modal, the backend first tries parsing it as a feed. If that fails, it fetches the HTML and extracts all `<link rel="alternate">` tags with RSS/Atom types. Returns the discovered feed URLs.

**Changes:**

- **Backend** - New endpoint `POST /api/feeds/discover` accepting `{ url: string }`. Returns `{ feeds: [{ url, title, type }] }`. Uses the existing `SafeHTTPClient` and `IsSafeURL` checks.
- **Frontend Add Feed modal** - After the user types a URL and clicks "Add", call discover first. If one feed found, add it directly. If multiple, show a list with checkboxes so the user can select which to subscribe to. If the URL is already a valid feed, skip discovery.

### 3. Feed Error Indicators

The backend tracks the last fetch error per feed. The sidebar shows a small orange/red dot on feeds that have a fetch error. Hovering shows a tooltip with the error message.

**Changes:**

- **Backend** - Add `last_error` column to `feeds` table. On fetch failure in the scheduler, store the error message. On successful fetch, clear it. Expose `last_error` in the feed list API response.
- **Feed model** - Add `LastError *string` field.
- **Sidebar** - Show a warning dot next to feeds with `last_error`. Tooltip on hover shows the error text. Click the dot to retry (calls `POST /api/feeds/{id}/retry`).
- **Backend** - New endpoint `POST /api/feeds/{id}/retry` that triggers an immediate fetch for that feed.

### 4. Reader Customization

A popover in the reader header with four settings: font size (small/medium/large/xl), font family (serif/sans-serif/monospace), line height (compact/comfortable/spacious), content width (narrow/medium/wide). Persisted per-user via the existing settings API.

**Changes:**

- **Reader toolbar popover** - A small "Aa" button in the ArticleReader header that opens a popover with the 4 controls as segmented buttons.
- **CSS variables** - Map each setting to CSS custom properties applied to the article content div. Defaults: 17px, sans-serif, 1.75, 680px.
- **Persistence** - Save to the existing `/api/settings` endpoint as `reader_font_size`, `reader_font_family`, `reader_line_height`, `reader_content_width`. Load on app init from the settings store.
- **Apply to both** `ArticleReader.svelte` and `article/[id]/+page.svelte`.

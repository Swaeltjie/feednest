# FeedNest v2 Frontend Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform the FeedNest frontend from a functional but flat UI into a visually stunning app with glassmorphism, adaptive dark mode layers, hero cards, dense article list with snippets, and rich animations.

**Architecture:** All changes are CSS/Svelte template rewrites of existing components. One backend change adds a `snippet` field to article list responses. No new dependencies needed — all animations are CSS keyframes/transitions, glassmorphism uses `backdrop-filter: blur()`.

**Tech Stack:** SvelteKit 5 (runes), Tailwind CSS v4, Go backend

**Design doc:** `docs/plans/2026-03-07-feednest-v2-redesign-design.md`

---

### Task 1: Backend — Add snippet field to article list API

**Context:** The article list API currently returns empty strings for `content_raw` and `content_clean` to keep responses small. We need a `snippet` field — plain text, ~160 chars, extracted from `content_clean` — so the frontend can show preview text in article rows.

**Files:**
- Modify: `backend/internal/models/article.go`
- Modify: `backend/internal/store/article_store.go`
- Test: `backend/internal/store/article_store_test.go`

**Step 1: Add Snippet field to Article model**

In `backend/internal/models/article.go`, add a `Snippet` field to the `Article` struct after the `ContentClean` field:

```go
ContentClean string     `json:"content_clean,omitempty"`
Snippet      string     `json:"snippet,omitempty"`
```

**Step 2: Update ListArticles SQL query to compute snippet**

In `backend/internal/store/article_store.go`, the `ListArticles` function currently selects `'', ''` for content_raw and content_clean (around line 97). Change the SELECT to also include a snippet computed from `content_clean`:

Replace:
```go
query := fmt.Sprintf(`
    SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', '',
        a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
```

With:
```go
query := fmt.Sprintf(`
    SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', '',
        COALESCE(SUBSTR(REPLACE(REPLACE(REPLACE(a.content_clean, CHAR(10), ' '), CHAR(13), ' '), '  ', ' '), 1, 200), ''),
        a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
```

Note: This uses SQLite's SUBSTR to truncate to 200 chars. The HTML stripping happens at fetch time (readability already extracts clean text), but `content_clean` may still have some HTML tags. We'll strip those in a Go helper.

Actually, simpler approach — compute the snippet in Go after the query. Keep the SQL as-is (returning empty content), and instead add a helper function:

In `backend/internal/store/article_store.go`, add at the top of the file (after imports):

```go
import (
    "regexp"
    // ... existing imports
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func makeSnippet(html string, maxLen int) string {
    text := htmlTagRe.ReplaceAllString(html, "")
    text = strings.Join(strings.Fields(text), " ")
    if len(text) > maxLen {
        text = text[:maxLen]
        if i := strings.LastIndex(text, " "); i > maxLen-40 {
            text = text[:i]
        }
        text += "…"
    }
    return text
}
```

Then in `ListArticles`, change the query to select `content_clean` instead of empty string:

Replace:
```go
SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', '',
```

With:
```go
SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', a.content_clean,
```

And after the Scan, compute the snippet and clear content_clean:

```go
for rows.Next() {
    var a models.Article
    if err := rows.Scan(&a.ID, &a.FeedID, &a.GUID, &a.Title, &a.URL, &a.Author, &a.ContentRaw, &a.ContentClean,
        &a.ThumbnailURL, &a.PublishedAt, &a.FetchedAt, &a.WordCount, &a.ReadingTime,
        &a.IsRead, &a.IsStarred, &a.ReadAt, &a.Score, &a.FeedTitle, &a.FeedIconURL); err != nil {
        return nil, 0, err
    }
    a.Snippet = makeSnippet(a.ContentClean, 160)
    a.ContentClean = ""
    articles = append(articles, a)
}
```

**Step 3: Write a test for makeSnippet**

In `backend/internal/store/article_store_test.go`, add:

```go
func TestMakeSnippet(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        maxLen int
        want   string
    }{
        {"strips HTML", "<p>Hello <b>world</b></p>", 100, "Hello world"},
        {"truncates long text", strings.Repeat("word ", 50), 30, "word word word word word…"},
        {"empty input", "", 100, ""},
        {"collapses whitespace", "hello   \n  world", 100, "hello world"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := makeSnippet(tt.input, tt.maxLen)
            if tt.maxLen < 100 && len(got) > tt.maxLen+5 {
                t.Errorf("snippet too long: got %d chars, max %d", len(got), tt.maxLen)
            }
            if tt.want != "" && got != tt.want {
                // For truncation tests, just check it starts right and ends with …
                if !strings.HasPrefix(got, "word word") {
                    t.Errorf("makeSnippet(%q, %d) = %q, want prefix match", tt.input, tt.maxLen, got)
                }
            }
        })
    }
}
```

**Step 4: Run tests**

```bash
cd backend && go test ./internal/store/ -v -run TestMakeSnippet
```

**Step 5: Build and verify**

```bash
cd backend && go build ./cmd/feednest/ && go test ./...
```

**Step 6: Commit**

```bash
git add backend/internal/models/article.go backend/internal/store/article_store.go backend/internal/store/article_store_test.go
git commit -m "feat: add snippet field to article list API"
```

---

### Task 2: CSS foundation — Design tokens and animations

**Context:** Before touching any components, we need to establish the CSS design system: color tokens for adaptive dark layers, glassmorphism utility classes, animation keyframes, and the shimmer skeleton. This goes in `app.css` and is used by all subsequent tasks.

**Files:**
- Modify: `frontend/src/app.css`

**Step 1: Replace app.css with the full design system**

Replace the entire contents of `frontend/src/app.css` with:

```css
@import 'tailwindcss';
@plugin '@tailwindcss/typography';

/* ─── Adaptive dark mode layers ─── */
:root {
  --color-base: #f8f9fa;
  --color-surface: #f0f2f5;
  --color-card: #ffffff;
  --color-elevated: #ffffff;
  --color-border: rgba(0, 0, 0, 0.08);
  --color-border-hover: rgba(0, 0, 0, 0.15);
  --color-accent: #2563eb;
  --color-accent-end: #7c3aed;
  --color-accent-glow: rgba(37, 99, 235, 0.12);
  --color-text-primary: #111827;
  --color-text-secondary: #6b7280;
  --color-text-tertiary: #9ca3af;
}

.dark {
  --color-base: #0d1117;
  --color-surface: #161b22;
  --color-card: #1c2128;
  --color-elevated: #22272e;
  --color-border: rgba(255, 255, 255, 0.06);
  --color-border-hover: rgba(255, 255, 255, 0.12);
  --color-accent: #3b82f6;
  --color-accent-end: #8b5cf6;
  --color-accent-glow: rgba(59, 130, 246, 0.15);
  --color-text-primary: #e6edf3;
  --color-text-secondary: #8b949e;
  --color-text-tertiary: #484f58;
}

/* ─── Glassmorphism ─── */
.glass {
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid var(--color-border);
}
.dark .glass {
  background: rgba(22, 27, 34, 0.75);
}

.glass-card {
  background: var(--color-card);
  border: 1px solid var(--color-border);
  transition: all 0.2s ease;
}
.glass-card:hover {
  border-color: var(--color-border-hover);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}
.dark .glass-card:hover {
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

/* ─── Gradient accent ─── */
.accent-gradient {
  background: linear-gradient(135deg, var(--color-accent), var(--color-accent-end));
}
.accent-gradient-text {
  background: linear-gradient(135deg, var(--color-accent), var(--color-accent-end));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.accent-underline {
  position: relative;
}
.accent-underline::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(135deg, var(--color-accent), var(--color-accent-end));
  border-radius: 1px;
}

/* ─── Glow effect for active sidebar items ─── */
.glow-active {
  background: var(--color-accent-glow);
  box-shadow: 0 0 16px var(--color-accent-glow);
}

/* ─── Shimmer skeleton loading ─── */
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.skeleton {
  background: linear-gradient(
    90deg,
    var(--color-border) 25%,
    var(--color-border-hover) 50%,
    var(--color-border) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: 8px;
}

/* ─── Staggered fade-in ─── */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.fade-in-up {
  animation: fadeInUp 0.3s ease-out both;
}

/* ─── Star bounce ─── */
@keyframes starBounce {
  0% { transform: scale(1); }
  50% { transform: scale(1.3); }
  100% { transform: scale(1); }
}
.star-bounce {
  animation: starBounce 0.2s ease;
}

/* ─── Slide transitions ─── */
@keyframes slideInRight {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
@keyframes slideOutRight {
  from { transform: translateX(0); opacity: 1; }
  to { transform: translateX(100%); opacity: 0; }
}

/* ─── Auth page animated gradient ─── */
@keyframes gradientShift {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}
.auth-bg {
  background: linear-gradient(-45deg, #0d1117, #1e1b4b, #172554, #0c4a6e, #0d1117);
  background-size: 400% 400%;
  animation: gradientShift 15s ease infinite;
}

/* ─── Unread left accent border ─── */
.unread-accent {
  border-left: 3px solid transparent;
  border-image: linear-gradient(to bottom, var(--color-accent), var(--color-accent-end)) 1;
}

/* ─── Hero card gradient overlay ─── */
.hero-overlay {
  background: linear-gradient(to top, rgba(0,0,0,0.85) 0%, rgba(0,0,0,0.4) 40%, transparent 100%);
}

/* ─── Global transition defaults ─── */
* {
  scroll-behavior: smooth;
}

/* ─── Custom scrollbar ─── */
::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: var(--color-border-hover);
  border-radius: 3px;
}
::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-tertiary);
}
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/app.css
git commit -m "feat: add CSS design system with adaptive dark tokens, glassmorphism, animations"
```

---

### Task 3: Update Article type in frontend store

**Context:** The backend now returns a `snippet` field. Update the frontend `Article` interface to include it.

**Files:**
- Modify: `frontend/src/lib/stores/articles.ts`

**Step 1: Add snippet to Article interface**

In `frontend/src/lib/stores/articles.ts`, add `snippet` to the `Article` interface after `content_raw`:

```typescript
export interface Article {
	id: number;
	feed_id: number;
	title: string;
	url: string;
	author: string;
	content_clean: string;
	content_raw: string;
	snippet: string;
	thumbnail_url: string;
	// ... rest unchanged
}
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/lib/stores/articles.ts
git commit -m "feat: add snippet field to Article interface"
```

---

### Task 4: Redesign ArticleCard as HeroCard

**Context:** Replace the current flat ArticleCard with a visually stunning hero card. Full-bleed image with gradient overlay, bold white title over the image, frosted glass metadata bar. Used for the top 2-3 featured articles.

**Files:**
- Modify: `frontend/src/lib/components/ArticleCard.svelte`

**Step 1: Rewrite ArticleCard.svelte**

Replace the entire contents of `frontend/src/lib/components/ArticleCard.svelte`:

```svelte
<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';

	let {
		article,
		selected = false,
		index = 0,
	}: { article: Article; selected?: boolean; index?: number } = $props();

	function handleStar(e: Event) {
		e.preventDefault();
		e.stopPropagation();
		articles.toggleStar(article.id, !article.is_starred);
	}

	let starAnimating = $state(false);
	function handleStarWithBounce(e: Event) {
		handleStar(e);
		starAnimating = true;
		setTimeout(() => (starAnimating = false), 200);
	}
</script>

<a
	href="/article/{article.id}"
	class="group relative block rounded-2xl overflow-hidden glass-card fade-in-up"
	style="animation-delay: {index * 60}ms; min-height: 280px;"
	class:ring-2={selected}
	class:ring-blue-500={selected}
>
	<!-- Background image or gradient fallback -->
	{#if article.thumbnail_url}
		<img
			src={article.thumbnail_url}
			alt=""
			class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
			loading="lazy"
		/>
	{:else}
		<div class="absolute inset-0 accent-gradient opacity-20"></div>
	{/if}

	<!-- Gradient overlay -->
	<div class="absolute inset-0 hero-overlay"></div>

	<!-- Content positioned at bottom -->
	<div class="relative h-full flex flex-col justify-end p-5">
		<!-- Frosted glass metadata strip -->
		<div
			class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs text-white/80 mb-3 self-start"
			style="background: rgba(255,255,255,0.1); backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);"
		>
			{#if article.feed_icon_url}
				<img src={article.feed_icon_url} alt="" class="w-3.5 h-3.5 rounded-full" />
			{/if}
			<span>{article.feed_title}</span>
			<span class="opacity-50">·</span>
			<span>{timeAgo(article.published_at)}</span>
			{#if article.reading_time > 0}
				<span class="opacity-50">·</span>
				<span>{article.reading_time} min</span>
			{/if}
		</div>

		<!-- Title -->
		<h3
			class="text-xl font-bold text-white leading-snug line-clamp-2 drop-shadow-lg"
		>
			{article.title}
		</h3>

		{#if article.snippet}
			<p class="text-sm text-white/60 line-clamp-1 mt-1.5">
				{article.snippet}
			</p>
		{/if}
	</div>

	<!-- Star button (top-right) -->
	<button
		onclick={handleStarWithBounce}
		class="absolute top-3 right-3 p-2 rounded-full transition-all {article.is_starred
			? 'text-yellow-400 bg-yellow-400/20'
			: 'text-white/50 hover:text-white bg-black/20 hover:bg-black/40'} {starAnimating ? 'star-bounce' : ''}"
	>
		<svg class="w-5 h-5" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
			<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
		</svg>
	</button>

	<!-- Unread indicator -->
	{#if !article.is_read}
		<div class="absolute top-3 left-3 w-2.5 h-2.5 rounded-full accent-gradient shadow-lg shadow-blue-500/30"></div>
	{/if}
</a>
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/ArticleCard.svelte
git commit -m "feat: redesign ArticleCard as hero card with glass overlay"
```

---

### Task 5: Redesign ArticleList as rich dense rows

**Context:** Replace the current minimal list rows with visually rich rows: thumbnail, title, snippet, source with favicon, unread accent border, hover effects with frosted glass.

**Files:**
- Modify: `frontend/src/lib/components/ArticleList.svelte`

**Step 1: Rewrite ArticleList.svelte**

Replace the entire contents of `frontend/src/lib/components/ArticleList.svelte`:

```svelte
<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';

	let {
		article,
		selected = false,
		index = 0,
	}: { article: Article; selected?: boolean; index?: number } = $props();

	let starAnimating = $state(false);

	function handleStar(e: Event) {
		e.preventDefault();
		e.stopPropagation();
		articles.toggleStar(article.id, !article.is_starred);
		starAnimating = true;
		setTimeout(() => (starAnimating = false), 200);
	}
</script>

<a
	href="/article/{article.id}"
	class="group flex items-start gap-4 px-4 py-3.5 transition-all duration-200 fade-in-up
		border-b border-[var(--color-border)]
		hover:bg-[var(--color-elevated)] hover:shadow-md
		{article.is_read ? 'opacity-60 hover:opacity-90' : ''}
		{!article.is_read ? 'unread-accent' : 'border-l-3 border-l-transparent'}
		{selected ? 'bg-[var(--color-accent-glow)] ring-1 ring-inset ring-[var(--color-accent)]/30' : ''}"
	style="animation-delay: {index * 30}ms;"
>
	<!-- Thumbnail -->
	{#if article.thumbnail_url}
		<div class="flex-shrink-0 w-16 h-16 rounded-lg overflow-hidden bg-[var(--color-border)]">
			<img
				src={article.thumbnail_url}
				alt=""
				class="w-full h-full object-cover group-hover:scale-110 transition-transform duration-300"
				loading="lazy"
			/>
		</div>
	{:else}
		<div class="flex-shrink-0 w-16 h-16 rounded-lg accent-gradient opacity-15 flex items-center justify-center">
			<span class="text-2xl font-bold text-[var(--color-text-primary)] opacity-40">
				{article.feed_title?.charAt(0)?.toUpperCase() || '?'}
			</span>
		</div>
	{/if}

	<!-- Content -->
	<div class="flex-1 min-w-0">
		<h3 class="text-sm font-semibold text-[var(--color-text-primary)] leading-snug line-clamp-1 group-hover:text-[var(--color-accent)] transition-colors">
			{article.title}
		</h3>

		{#if article.snippet}
			<p class="text-xs text-[var(--color-text-secondary)] line-clamp-2 mt-0.5 leading-relaxed">
				{article.snippet}
			</p>
		{/if}

		<div class="flex items-center gap-2 mt-1.5 text-xs text-[var(--color-text-tertiary)]">
			{#if article.feed_icon_url}
				<img src={article.feed_icon_url} alt="" class="w-3.5 h-3.5 rounded-full" />
			{/if}
			<span class="font-medium text-[var(--color-text-secondary)]">{article.feed_title}</span>
			<span class="opacity-40">·</span>
			<span>{timeAgo(article.published_at)}</span>
			{#if article.reading_time > 0}
				<span class="opacity-40">·</span>
				<span>{article.reading_time} min</span>
			{/if}
		</div>
	</div>

	<!-- Star -->
	<button
		onclick={handleStar}
		class="flex-shrink-0 p-1.5 rounded-lg transition-all
			{article.is_starred
				? 'text-yellow-500 hover:text-yellow-400'
				: 'text-[var(--color-text-tertiary)] hover:text-yellow-500 opacity-0 group-hover:opacity-100'}
			{starAnimating ? 'star-bounce' : ''}"
	>
		<svg class="w-4 h-4" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
			<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
		</svg>
	</button>
</a>
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/ArticleList.svelte
git commit -m "feat: redesign ArticleList with thumbnails, snippets, and rich hover"
```

---

### Task 6: Create SkeletonLoader component

**Context:** Replace the text "Loading articles..." with beautiful shimmer skeleton screens. Featured row gets 2 large shimmer rectangles, list rows get thumbnail + text shimmer lines.

**Files:**
- Create: `frontend/src/lib/components/SkeletonLoader.svelte`

**Step 1: Create the component**

```svelte
<script lang="ts">
	let { mode = 'hybrid' }: { mode?: 'cards' | 'list' | 'hybrid' } = $props();
</script>

<div class="p-4 space-y-4">
	{#if mode === 'hybrid' || mode === 'cards'}
		<!-- Featured hero skeletons -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each [0, 1] as i}
				<div class="rounded-2xl overflow-hidden" style="animation-delay: {i * 100}ms;">
					<div class="skeleton h-64 w-full"></div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- List row skeletons -->
	<div class="space-y-1">
		{#each Array(6) as _, i}
			<div
				class="flex items-start gap-4 px-4 py-3.5 fade-in-up"
				style="animation-delay: {(i + 2) * 60}ms;"
			>
				<div class="skeleton w-16 h-16 rounded-lg flex-shrink-0"></div>
				<div class="flex-1 space-y-2">
					<div class="skeleton h-4 w-3/4"></div>
					<div class="skeleton h-3 w-full"></div>
					<div class="skeleton h-3 w-1/3"></div>
				</div>
			</div>
		{/each}
	</div>
</div>
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/SkeletonLoader.svelte
git commit -m "feat: add shimmer skeleton loading component"
```

---

### Task 7: Redesign the main dashboard page

**Context:** The big one. Rewrite `+page.svelte` to use the new design system: frosted glass toolbar with gradient active tabs, featured hero row (top 2-3 articles as hero cards), remaining articles as dense list, skeleton loading, adaptive dark backgrounds, staggered fade-in animations.

**Files:**
- Modify: `frontend/src/routes/+page.svelte`

**Step 1: Rewrite the dashboard**

Replace the entire contents of `frontend/src/routes/+page.svelte`. This is a complete rewrite. Key changes:

- Background uses `var(--color-surface)` instead of `bg-gray-50 dark:bg-gray-900`
- Toolbar uses `glass` class for frosted glass effect
- Filter tabs use gradient underline instead of background toggle
- Featured row: first 3 articles with thumbnails displayed as hero cards in a grid
- Remaining articles displayed as dense list rows
- Skeleton loading replaces the spinner
- All items get staggered `fade-in-up` animation
- Add feed modal uses glass styling
- View mode affects whether the split hero/list layout is used or all-cards/all-list

```svelte
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ArticleCard from '$lib/components/ArticleCard.svelte';
	import ArticleList from '$lib/components/ArticleList.svelte';
	import SkeletonLoader from '$lib/components/SkeletonLoader.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { articles, type ArticleFilters } from '$lib/stores/articles';
	import { feeds, categories } from '$lib/stores/feeds';
	import { setupKeyboardShortcuts } from '$lib/utils/keyboard';

	type ViewMode = 'hybrid' | 'cards' | 'list';
	type FilterTab = 'all' | 'unread' | 'starred';
	type SortOption = 'smart' | 'newest' | 'oldest';
	type SidebarView = 'all' | 'starred' | 'feed' | 'category';

	let sidebarCollapsed = $state(false);
	let mobileMenuOpen = $state(false);
	let viewMode = $state<ViewMode>(
		(typeof localStorage !== 'undefined' && (localStorage.getItem('feednest_view') as ViewMode)) ||
			'hybrid'
	);
	let filterTab = $state<FilterTab>('all');
	let sortOption = $state<SortOption>('smart');
	let sidebarView = $state<SidebarView>('all');
	let activeFeedId = $state<number | null>(null);
	let activeCategoryId = $state<number | null>(null);
	let showAddFeedModal = $state(false);
	let feedUrl = $state('');
	let feedCategoryId = $state<number | undefined>(undefined);
	let addingFeed = $state(false);
	let addFeedError = $state('');
	let initialized = $state(false);
	let selectedIndex = $state(-1);
	let cleanupKeyboard: (() => void) | undefined;

	const FEATURED_COUNT = 3;

	let currentFilters = $derived<ArticleFilters>({
		status: filterTab === 'unread' ? 'unread' : filterTab === 'starred' ? 'starred' : undefined,
		sort: sortOption,
		feed: sidebarView === 'feed' && activeFeedId ? activeFeedId : undefined,
		category: sidebarView === 'category' && activeCategoryId ? activeCategoryId : undefined,
	});

	let featuredArticles = $derived(
		viewMode === 'hybrid'
			? $articles.articles.filter((a) => a.thumbnail_url).slice(0, FEATURED_COUNT)
			: []
	);

	let featuredIds = $derived(new Set(featuredArticles.map((a) => a.id)));

	let listArticles = $derived(
		viewMode === 'hybrid'
			? $articles.articles.filter((a) => !featuredIds.has(a.id))
			: $articles.articles
	);

	const filterTabs: { value: FilterTab; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'unread', label: 'Unread' },
		{ value: 'starred', label: 'Starred' },
	];

	function setViewMode(mode: ViewMode) {
		viewMode = mode;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('feednest_view', mode);
		}
	}

	function selectAll() {
		sidebarView = 'all';
		activeFeedId = null;
		activeCategoryId = null;
		filterTab = 'all';
		mobileMenuOpen = false;
	}

	function selectStarred() {
		sidebarView = 'starred';
		activeFeedId = null;
		activeCategoryId = null;
		filterTab = 'starred';
		mobileMenuOpen = false;
	}

	function selectFeed(id: number) {
		sidebarView = 'feed';
		activeFeedId = id;
		activeCategoryId = null;
		filterTab = 'all';
		mobileMenuOpen = false;
	}

	function selectCategory(id: number) {
		sidebarView = 'category';
		activeFeedId = null;
		activeCategoryId = id;
		filterTab = 'all';
		mobileMenuOpen = false;
	}

	function openAddFeed() {
		showAddFeedModal = true;
		feedUrl = '';
		feedCategoryId = undefined;
		addFeedError = '';
		mobileMenuOpen = false;
	}

	async function handleAddFeed() {
		if (!feedUrl.trim()) return;
		addingFeed = true;
		addFeedError = '';
		try {
			await feeds.add(feedUrl.trim(), feedCategoryId);
			showAddFeedModal = false;
			articles.load(currentFilters);
		} catch (err) {
			addFeedError = err instanceof Error ? err.message : 'Failed to add feed';
		} finally {
			addingFeed = false;
		}
	}

	onMount(async () => {
		try {
			await Promise.all([feeds.load(), categories.load()]);
			await articles.load(currentFilters);
		} finally {
			initialized = true;
		}

		cleanupKeyboard = setupKeyboardShortcuts({
			j: () => {
				const articleList = $articles.articles;
				if (articleList.length > 0) {
					selectedIndex = Math.min(selectedIndex + 1, articleList.length - 1);
				}
			},
			k: () => {
				if (selectedIndex > 0) {
					selectedIndex = selectedIndex - 1;
				}
			},
			enter: () => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					goto(`/article/${articleList[selectedIndex].id}`);
				}
			},
			s: () => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					const a = articleList[selectedIndex];
					articles.toggleStar(a.id, !a.is_starred);
				}
			},
			m: () => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					const a = articleList[selectedIndex];
					articles.toggleRead(a.id, !a.is_read);
				}
			},
			d: () => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					articles.dismiss(articleList[selectedIndex].id);
					if (selectedIndex >= articleList.length - 1) {
						selectedIndex = Math.max(0, articleList.length - 2);
					}
				}
			},
			v: () => {
				const modes: ViewMode[] = ['hybrid', 'cards', 'list'];
				const current = modes.indexOf(viewMode);
				setViewMode(modes[(current + 1) % modes.length]);
			},
			'/': () => {
				const searchInput = document.querySelector<HTMLInputElement>(
					'input[type="search"], input[placeholder*="earch"]'
				);
				if (searchInput) {
					searchInput.focus();
				}
			},
		});
	});

	onDestroy(() => {
		cleanupKeyboard?.();
	});

	$effect(() => {
		if (!initialized) return;
		const filters = currentFilters;
		articles.load(filters);
	});

	let pageTitle = $derived(() => {
		if (sidebarView === 'starred') return 'Starred';
		if (sidebarView === 'feed' && activeFeedId) {
			const feed = $feeds.find((f) => f.id === activeFeedId);
			return feed?.title || 'Feed';
		}
		if (sidebarView === 'category' && activeCategoryId) {
			const cat = $categories.find((c) => c.id === activeCategoryId);
			return cat?.name || 'Category';
		}
		return 'All Articles';
	});
</script>

<svelte:head>
	<title>{pageTitle()} - FeedNest</title>
</svelte:head>

<div class="flex h-screen" style="background: var(--color-surface);">
	<!-- Mobile sidebar overlay -->
	{#if mobileMenuOpen}
		<div class="fixed inset-0 z-40 lg:hidden">
			<button
				class="absolute inset-0 bg-black/60 backdrop-blur-sm"
				onclick={() => (mobileMenuOpen = false)}
				aria-label="Close menu"
			></button>
			<div class="relative z-50 h-full w-64">
				<Sidebar
					collapsed={false}
					activeFeed={activeFeedId}
					activeCategory={activeCategoryId}
					activeView={sidebarView}
					onSelectAll={selectAll}
					onSelectStarred={selectStarred}
					onSelectFeed={selectFeed}
					onSelectCategory={selectCategory}
					onAddFeed={openAddFeed}
				/>
			</div>
		</div>
	{/if}

	<!-- Desktop sidebar -->
	<div class="hidden lg:block flex-shrink-0">
		<Sidebar
			collapsed={sidebarCollapsed}
			activeFeed={activeFeedId}
			activeCategory={activeCategoryId}
			activeView={sidebarView}
			onSelectAll={selectAll}
			onSelectStarred={selectStarred}
			onSelectFeed={selectFeed}
			onSelectCategory={selectCategory}
			onAddFeed={openAddFeed}
		/>
	</div>

	<!-- Main content -->
	<div class="flex-1 flex flex-col min-w-0">
		<!-- Frosted glass toolbar -->
		<header class="sticky top-0 z-30 glass">
			<div class="flex items-center justify-between px-4 py-3">
				<div class="flex items-center gap-4">
					<!-- Mobile hamburger -->
					<button
						class="lg:hidden p-1.5 rounded-lg hover:bg-[var(--color-elevated)] transition-colors"
						onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
						aria-label="Toggle sidebar"
					>
						<svg class="w-5 h-5 text-[var(--color-text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
						</svg>
					</button>

					<!-- Desktop sidebar toggle -->
					<button
						class="hidden lg:block p-1.5 rounded-lg hover:bg-[var(--color-elevated)] transition-colors"
						onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
						aria-label="Toggle sidebar"
					>
						<svg class="w-5 h-5 text-[var(--color-text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
						</svg>
					</button>

					<!-- Filter tabs with gradient underline -->
					<div class="flex items-center gap-1">
						{#each filterTabs as tab}
							<button
								onclick={() => (filterTab = tab.value)}
								class="relative px-3 py-1.5 text-sm font-medium transition-colors
									{filterTab === tab.value
										? 'text-[var(--color-text-primary)] accent-underline'
										: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}"
							>
								{tab.label}
							</button>
						{/each}
					</div>
				</div>

				<div class="flex items-center gap-3">
					<!-- Sort -->
					<select
						bind:value={sortOption}
						class="text-sm rounded-lg px-3 py-1.5 border transition-colors cursor-pointer
							bg-[var(--color-card)] border-[var(--color-border)] text-[var(--color-text-secondary)]
							focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent"
					>
						<option value="smart">Smart</option>
						<option value="newest">Newest</option>
						<option value="oldest">Oldest</option>
					</select>

					<!-- View toggle (3 modes) -->
					<div class="flex items-center gap-0.5 p-0.5 rounded-lg" style="background: var(--color-border);">
						<button
							onclick={() => setViewMode('hybrid')}
							class="p-1.5 rounded-md transition-all {viewMode === 'hybrid' ? 'bg-[var(--color-card)] shadow-sm' : 'hover:bg-[var(--color-card)]/50'}"
							title="Hybrid view"
						>
							<svg class="w-4 h-4 text-[var(--color-text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h5a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
							</svg>
						</button>
						<button
							onclick={() => setViewMode('cards')}
							class="p-1.5 rounded-md transition-all {viewMode === 'cards' ? 'bg-[var(--color-card)] shadow-sm' : 'hover:bg-[var(--color-card)]/50'}"
							title="Card view"
						>
							<svg class="w-4 h-4 text-[var(--color-text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							</svg>
						</button>
						<button
							onclick={() => setViewMode('list')}
							class="p-1.5 rounded-md transition-all {viewMode === 'list' ? 'bg-[var(--color-card)] shadow-sm' : 'hover:bg-[var(--color-card)]/50'}"
							title="List view"
						>
							<svg class="w-4 h-4 text-[var(--color-text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
							</svg>
						</button>
					</div>

					<ThemeToggle />
				</div>
			</div>
		</header>

		<!-- Article content -->
		<main class="flex-1 overflow-y-auto">
			{#if $articles.loading && !initialized}
				<SkeletonLoader mode={viewMode} />
			{:else if $articles.articles.length === 0}
				<!-- Empty state -->
				<div class="flex flex-col items-center justify-center h-64 text-center px-4 fade-in-up">
					<div class="w-20 h-20 rounded-2xl accent-gradient opacity-10 flex items-center justify-center mb-4">
						<svg class="w-10 h-10 text-[var(--color-text-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
						</svg>
					</div>
					<h2 class="text-lg font-semibold text-[var(--color-text-primary)] mb-1">No articles found</h2>
					<p class="text-sm text-[var(--color-text-secondary)] mb-4">
						{#if $feeds.length === 0}
							Add a feed to get started.
						{:else}
							Try changing your filters or check back later.
						{/if}
					</p>
					{#if $feeds.length === 0}
						<button
							onclick={openAddFeed}
							class="px-5 py-2.5 text-sm font-medium text-white rounded-xl accent-gradient hover:opacity-90 transition-opacity shadow-lg shadow-blue-500/25"
						>
							Add Your First Feed
						</button>
					{/if}
				</div>
			{:else}
				<!-- Hybrid view: featured heroes + dense list -->
				{#if viewMode === 'hybrid'}
					{#if featuredArticles.length > 0}
						<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
							{#each featuredArticles as article, i (article.id)}
								<ArticleCard {article} selected={$articles.articles.indexOf(article) === selectedIndex} index={i} />
							{/each}
						</div>
					{/if}
					<div style="background: var(--color-card);" class="rounded-t-2xl mx-2 mt-2">
						{#each listArticles as article, i (article.id)}
							<ArticleList {article} selected={$articles.articles.indexOf(article) === selectedIndex} index={i} />
						{/each}
					</div>

				<!-- All cards view -->
				{:else if viewMode === 'cards'}
					<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
						{#each $articles.articles as article, i (article.id)}
							<ArticleCard {article} selected={i === selectedIndex} index={i} />
						{/each}
					</div>

				<!-- All list view -->
				{:else}
					<div style="background: var(--color-card);" class="m-2 rounded-2xl overflow-hidden">
						{#each $articles.articles as article, i (article.id)}
							<ArticleList {article} selected={i === selectedIndex} index={i} />
						{/each}
					</div>
				{/if}

				<!-- Loading indicator for filter changes -->
				{#if $articles.loading && initialized}
					<div class="flex items-center justify-center py-4">
						<div class="w-5 h-5 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
					</div>
				{/if}

				<!-- Article count footer -->
				{#if $articles.articles.length > 0}
					<div class="text-center py-4 text-sm text-[var(--color-text-tertiary)]">
						Showing {$articles.articles.length} of {$articles.total} articles
					</div>
				{/if}
			{/if}
		</main>
	</div>
</div>

<!-- Add Feed Modal -->
{#if showAddFeedModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button
			class="absolute inset-0 bg-black/60 backdrop-blur-sm"
			onclick={() => (showAddFeedModal = false)}
			aria-label="Close modal"
		></button>
		<div class="relative glass rounded-2xl shadow-2xl w-full max-w-md p-6 space-y-4 fade-in-up">
			<h2 class="text-lg font-semibold text-[var(--color-text-primary)]">Add Feed</h2>

			{#if addFeedError}
				<div class="p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20">
					{addFeedError}
				</div>
			{/if}

			<div>
				<label for="feed-url" class="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
					Feed URL
				</label>
				<input
					id="feed-url"
					type="url"
					bind:value={feedUrl}
					placeholder="https://example.com/feed.xml"
					class="w-full px-4 py-2.5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]
						text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
				/>
			</div>

			<div>
				<label for="feed-category" class="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
					Category (optional)
				</label>
				<select
					id="feed-category"
					bind:value={feedCategoryId}
					class="w-full px-4 py-2.5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]
						text-[var(--color-text-primary)]
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
				>
					<option value={undefined}>None</option>
					{#each $categories as cat}
						<option value={cat.id}>{cat.name}</option>
					{/each}
				</select>
			</div>

			<div class="flex justify-end gap-3 pt-2">
				<button
					onclick={() => (showAddFeedModal = false)}
					class="px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)]
						hover:bg-[var(--color-elevated)] rounded-xl transition-colors"
				>
					Cancel
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
			</div>
		</div>
	</div>
{/if}
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: redesign dashboard with hero cards, glassmorphism toolbar, adaptive dark"
```

---

### Task 8: Redesign the Sidebar

**Context:** Polish the sidebar with design tokens, feed favicons, glow active states, and smooth collapse animation. Keep it simple — no new features, just visual upgrade.

**Files:**
- Modify: `frontend/src/lib/components/Sidebar.svelte`

**Step 1: Rewrite Sidebar.svelte**

Replace the entire file. Key changes:
- Use `var(--color-base)` for background (darkest layer)
- Feed favicons via `feed.icon_url` with first-letter fallback
- Active state uses `glow-active` class
- Logo area uses gradient text
- Smooth width transition on collapse

```svelte
<script lang="ts">
	import { feeds, categories, type Feed, type Category } from '$lib/stores/feeds';
	import { auth } from '$lib/stores/auth';

	let {
		collapsed = false,
		activeFeed = null as number | null,
		activeCategory = null as number | null,
		activeView = 'all' as 'all' | 'starred' | 'feed' | 'category',
		onSelectAll = () => {},
		onSelectStarred = () => {},
		onSelectFeed = (_id: number) => {},
		onSelectCategory = (_id: number) => {},
		onAddFeed = () => {},
	}: {
		collapsed?: boolean;
		activeFeed?: number | null;
		activeCategory?: number | null;
		activeView?: 'all' | 'starred' | 'feed' | 'category';
		onSelectAll?: () => void;
		onSelectStarred?: () => void;
		onSelectFeed?: (id: number) => void;
		onSelectCategory?: (id: number) => void;
		onAddFeed?: () => void;
	} = $props();

	function groupByCategory(feedList: Feed[], catList: Category[]) {
		const uncategorized: Feed[] = [];
		const grouped: { category: Category; feeds: Feed[] }[] = [];
		const catMap = new Map<number, Feed[]>();
		for (const cat of catList) catMap.set(cat.id, []);
		for (const feed of feedList) {
			if (feed.category_id && catMap.has(feed.category_id)) {
				catMap.get(feed.category_id)!.push(feed);
			} else {
				uncategorized.push(feed);
			}
		}
		for (const cat of catList) {
			grouped.push({ category: cat, feeds: catMap.get(cat.id) || [] });
		}
		return { grouped, uncategorized };
	}

	let feedsByCategory = $derived(groupByCategory($feeds, $categories));

	function totalUnread(feedList: Feed[]): number {
		return feedList.reduce((sum, f) => sum + f.unread_count, 0);
	}

	let allUnread = $derived(totalUnread($feeds));
</script>

<aside
	class="flex flex-col h-full border-r border-[var(--color-border)] transition-all duration-300 ease-in-out overflow-hidden"
	style="background: var(--color-base); width: {collapsed ? '0px' : '16rem'};"
>
	<!-- Logo -->
	<div class="flex items-center gap-2 px-4 py-4 border-b border-[var(--color-border)]">
		<h1 class="text-lg font-bold accent-gradient-text whitespace-nowrap tracking-tight">FeedNest</h1>
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto py-2">
		<!-- All Articles -->
		<button
			onclick={onSelectAll}
			class="flex items-center justify-between w-full px-4 py-2.5 text-sm text-left transition-all rounded-lg mx-1 mr-2
				{activeView === 'all'
					? 'glow-active text-[var(--color-accent)]'
					: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
		>
			<span class="flex items-center gap-2.5">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
				</svg>
				All Articles
			</span>
			{#if allUnread > 0}
				<span class="px-2 py-0.5 text-xs font-semibold rounded-full accent-gradient text-white">
					{allUnread}
				</span>
			{/if}
		</button>

		<!-- Starred -->
		<button
			onclick={onSelectStarred}
			class="flex items-center gap-2.5 w-full px-4 py-2.5 text-sm text-left transition-all rounded-lg mx-1 mr-2
				{activeView === 'starred'
					? 'glow-active text-[var(--color-accent)]'
					: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
			</svg>
			Starred
		</button>

		<div class="my-2 mx-4 border-t border-[var(--color-border)]"></div>

		<!-- Categorized feeds -->
		{#each feedsByCategory.grouped as { category, feeds: catFeeds }}
			{@const catUnread = totalUnread(catFeeds)}
			<div class="mt-1">
				<button
					onclick={() => onSelectCategory(category.id)}
					class="flex items-center justify-between w-full px-4 py-1.5 text-left transition-colors
						{activeView === 'category' && activeCategory === category.id
							? 'text-[var(--color-accent)]'
							: 'hover:text-[var(--color-text-primary)]'}"
				>
					<span class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">
						{category.name}
					</span>
					{#if catUnread > 0}
						<span class="px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
							{catUnread}
						</span>
					{/if}
				</button>
				{#each catFeeds as feed}
					<button
						onclick={() => onSelectFeed(feed.id)}
						class="flex items-center justify-between w-full pl-5 pr-4 py-1.5 text-sm text-left transition-all rounded-r-lg
							{activeView === 'feed' && activeFeed === feed.id
								? 'glow-active text-[var(--color-accent)]'
								: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
					>
						<span class="flex items-center gap-2 truncate">
							{#if feed.icon_url}
								<img src={feed.icon_url} alt="" class="w-4 h-4 rounded-full flex-shrink-0" />
							{:else}
								<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
									{feed.title?.charAt(0)?.toUpperCase() || '?'}
								</span>
							{/if}
							<span class="truncate">{feed.title}</span>
						</span>
						{#if feed.unread_count > 0}
							<span class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
								{feed.unread_count}
							</span>
						{/if}
					</button>
				{/each}
			</div>
		{/each}

		<!-- Uncategorized feeds -->
		{#if feedsByCategory.uncategorized.length > 0}
			<div class="mt-1">
				<span class="block px-4 py-1.5 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">
					Feeds
				</span>
				{#each feedsByCategory.uncategorized as feed}
					<button
						onclick={() => onSelectFeed(feed.id)}
						class="flex items-center justify-between w-full pl-5 pr-4 py-1.5 text-sm text-left transition-all rounded-r-lg
							{activeView === 'feed' && activeFeed === feed.id
								? 'glow-active text-[var(--color-accent)]'
								: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
					>
						<span class="flex items-center gap-2 truncate">
							{#if feed.icon_url}
								<img src={feed.icon_url} alt="" class="w-4 h-4 rounded-full flex-shrink-0" />
							{:else}
								<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
									{feed.title?.charAt(0)?.toUpperCase() || '?'}
								</span>
							{/if}
							<span class="truncate">{feed.title}</span>
						</span>
						{#if feed.unread_count > 0}
							<span class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
								{feed.unread_count}
							</span>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</nav>

	<!-- Bottom actions -->
	<div class="border-t border-[var(--color-border)] p-3 space-y-1">
		<button
			onclick={onAddFeed}
			class="flex items-center gap-2.5 w-full px-3 py-2 text-sm font-medium rounded-xl accent-gradient text-white hover:opacity-90 transition-opacity shadow-lg shadow-blue-500/20"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			Add Feed
		</button>
		<button
			onclick={() => auth.logout()}
			class="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-tertiary)] rounded-lg hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-secondary)] transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
			</svg>
			Sign Out
		</button>
	</div>
</aside>
```

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/Sidebar.svelte
git commit -m "feat: redesign sidebar with gradient branding, favicons, glow active states"
```

---

### Task 9: Redesign the Article Reader

**Context:** Upgrade the article reader page with adaptive dark colors, improved typography, and consistent design tokens. The reader already has frosted glass header — refine it with the new design system.

**Files:**
- Modify: `frontend/src/routes/article/[id]/+page.svelte`

**Step 1: Update the article reader**

Update background colors, text colors, and styling to use CSS custom properties. Key changes:
- Replace `bg-gray-50 dark:bg-gray-900` with `style="background: var(--color-surface);"`
- Replace `bg-white/80 dark:bg-gray-800/80` header with `glass` class
- Replace hardcoded text colors with `text-[var(--color-text-*)]`
- Star button gets bounce animation
- Back button uses accent gradient on hover
- Loading state uses skeleton shimmer instead of spinner

This is a search-and-replace task on the existing file. Replace the entire file with an updated version using the design token system. The structure stays the same, just colors and classes change.

**Step 2: Verify the build**

```bash
cd frontend && npm run build
```

**Step 3: Commit**

```bash
git add frontend/src/routes/article/[id]/+page.svelte
git commit -m "feat: redesign article reader with design tokens and shimmer loading"
```

---

### Task 10: Redesign Auth pages and layout

**Context:** Upgrade login, register, and the layout loading state. Auth pages get an animated gradient background with frosted glass card. Layout loading gets a branded spinner.

**Files:**
- Modify: `frontend/src/routes/auth/login/+page.svelte`
- Modify: `frontend/src/routes/auth/register/+page.svelte`
- Modify: `frontend/src/routes/+layout.svelte`
- Modify: `frontend/src/lib/components/ThemeToggle.svelte`

**Step 1: Update login page**

Replace the login page background with `auth-bg` class (animated gradient), and the form card with `glass` styling:

Key changes:
- Outer div: `class="min-h-screen flex items-center justify-center auth-bg"`
- Card: add `glass` class, replace `bg-white dark:bg-gray-800` with transparent background from glass
- Title: use `accent-gradient-text` for "FeedNest"
- Submit button: use `accent-gradient` class
- Input fields: use design token colors

**Step 2: Update register page**

Same changes as login page.

**Step 3: Update layout loading state**

Replace the plain "Loading..." text with a branded loading screen:

```svelte
{#if $auth.loading}
	<div class="min-h-screen flex flex-col items-center justify-center" style="background: var(--color-surface);">
		<h1 class="text-2xl font-bold accent-gradient-text mb-4">FeedNest</h1>
		<div class="w-6 h-6 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
	</div>
{:else}
	{@render children()}
{/if}
```

**Step 4: Update ThemeToggle**

Replace hardcoded colors with design tokens:

```svelte
<div class="flex items-center gap-0.5 p-0.5 rounded-lg" style="background: var(--color-border);">
	{#each themes as theme}
		<button
			onclick={() => settings.setTheme(theme.value)}
			class="px-2 py-1 text-sm rounded-md transition-all
				{$settings.theme === theme.value ? 'bg-[var(--color-card)] shadow-sm' : 'hover:bg-[var(--color-card)]/50'}
				text-[var(--color-text-secondary)]"
			title={theme.value}
		>
			{theme.label}
		</button>
	{/each}
</div>
```

**Step 5: Verify the build**

```bash
cd frontend && npm run build
```

**Step 6: Commit**

```bash
git add frontend/src/routes/auth/login/+page.svelte frontend/src/routes/auth/register/+page.svelte frontend/src/routes/+layout.svelte frontend/src/lib/components/ThemeToggle.svelte
git commit -m "feat: redesign auth pages with animated gradient, glass cards, branded loading"
```

---

### Task 11: Docker rebuild and final verification

**Context:** Rebuild the Docker images with the updated frontend and backend, verify everything works end-to-end.

**Files:** None (Docker operations only)

**Step 1: Run backend tests**

```bash
cd backend && go test ./...
```

**Step 2: Build frontend**

```bash
cd frontend && npm run build
```

**Step 3: Rebuild and restart Docker**

```bash
cd /mnt/d/git/feednest && docker compose build && docker compose up -d
```

**Step 4: Verify both services respond**

```bash
curl -s http://localhost:8082/api/health
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000
```

**Step 5: Commit any remaining changes and push**

```bash
git add -A && git status
git push origin main
```

---

## Task Dependency Graph

```
Task 1 (backend snippet) ─┐
Task 2 (CSS foundation) ──┤─→ Task 3 (Article type) ─→ Task 4 (HeroCard) ─┐
                           │                           Task 5 (ListRow)  ──┤
                           │                           Task 6 (Skeleton) ──┤
                           │                                               ├─→ Task 7 (Dashboard) ─→ Task 11
                           ├─→ Task 8 (Sidebar) ──────────────────────────┘
                           └─→ Task 9 (Reader) ──→ Task 10 (Auth+Layout) ─→ Task 11
```

**Parallelizable groups:**
- Tasks 1 + 2 (backend + CSS foundation — independent)
- Tasks 4 + 5 + 6 + 8 + 9 (after Tasks 2 + 3 complete — all independent component rewrites)
- Task 10 (after Task 2 — independent of other frontend tasks)
- Task 7 (depends on Tasks 4, 5, 6 being done — imports those components)
- Task 11 (final — depends on everything)

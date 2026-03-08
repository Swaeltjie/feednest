# Psychological UX Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add three psychological wellbeing features — content aging, session framing, and session summaries — to reduce reading anxiety and create satisfying endpoints.

**Architecture:** Frontend-heavy changes with minimal backend additions. Content aging is purely visual (CSS opacity based on article age). Session framing uses existing WPM + article data to build a time-budgeted queue client-side. Session summary tracks articles read during a session in-memory and displays on reader close. One new backend endpoint for reading stats.

**Tech Stack:** SvelteKit 5 (Svelte 5 runes), TailwindCSS 4, Go/Chi backend, SQLite

---

## Task 1: Content Aging — Visual Fade for Old Unread Articles

Old unread articles should visually fade based on age, signaling "it's OK to skip these." This is purely a frontend CSS change.

**Files:**
- Modify: `frontend/src/routes/+page.svelte` (article rendering sections)
- Modify: `frontend/src/lib/components/ArticleCard.svelte`
- Modify: `frontend/src/lib/components/ArticleList.svelte`

**Step 1: Add age-based opacity helper**

In `+page.svelte`, add a helper function in the `<script>` block (after the existing helpers around line 60):

```typescript
function articleAgeOpacity(article: Article): number {
	if (article.is_read) return 1; // read articles already have their own opacity
	if (!article.published_at) return 1;
	const ageHours = (Date.now() - new Date(article.published_at).getTime()) / (1000 * 60 * 60);
	if (ageHours < 24) return 1;
	if (ageHours < 48) return 0.85;
	if (ageHours < 72) return 0.7;
	if (ageHours < 168) return 0.55; // 7 days
	return 0.4;
}
```

**Step 2: Apply aging to ArticleCard**

In `ArticleCard.svelte`, add an `ageOpacity` prop:

```typescript
let {
	article,
	selected = false,
	index = 0,
	ageOpacity = 1,
	onOpen = (_id: number) => {},
}: { article: Article; selected?: boolean; index?: number; ageOpacity?: number; onOpen?: (id: number) => void } = $props();
```

Apply it to the outer `<a>` element's style attribute (append to existing style):

```
style="... opacity: {ageOpacity};"
```

**Step 3: Apply aging to ArticleList**

In `ArticleList.svelte`, add an `ageOpacity` prop similarly and apply to the outer `<div>` style:

```typescript
let {
	article,
	selected = false,
	index = 0,
	ageOpacity = 1,
	onOpen = (_id: number) => {},
	onToggleRead,
	onToggleStar,
}: {
	article: Article;
	selected?: boolean;
	index?: number;
	ageOpacity?: number;
	onOpen?: (id: number) => void;
	onToggleRead?: (id: number, isRead: boolean) => void;
	onToggleStar?: (id: number, isStarred: boolean) => void;
} = $props();
```

Apply to the outer div style (append): `opacity: {article.is_read ? undefined : ageOpacity};`
Note: ArticleList already has `article.is_read ? 'opacity-60' : 'opacity-100'` in the class — override with inline style only when ageOpacity < 1 and article is NOT read.

**Step 4: Pass ageOpacity from +page.svelte**

In `+page.svelte`, everywhere `ArticleCard` and `ArticleList` are rendered, pass `ageOpacity={articleAgeOpacity(article)}`.

Search for `<ArticleCard` and `<ArticleList` in the file and add the prop to each instance.

**Step 5: Commit**

```bash
git add frontend/src/routes/+page.svelte frontend/src/lib/components/ArticleCard.svelte frontend/src/lib/components/ArticleList.svelte
git commit -m "feat: add visual content aging for old unread articles"
```

---

## Task 2: Catch-Up Banner for Stale Unread Articles

When the user has many old unread articles, show a gentle banner offering to clear them.

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Uses existing: `POST /api/articles/catch-up` backend endpoint (already implemented)

**Step 1: Add catch-up state and detection**

In `+page.svelte` script block, add state variables:

```typescript
let showCatchUpBanner = $state(false);
let catchUpCount = $state(0);
```

Add a derived/effect that checks after articles load:

```typescript
$effect(() => {
	if ($articles.articles.length > 0 && !openArticleId) {
		const twoDaysAgo = Date.now() - 2 * 24 * 60 * 60 * 1000;
		const oldUnread = $articles.articles.filter(
			a => !a.is_read && a.published_at && new Date(a.published_at).getTime() < twoDaysAgo
		);
		if (oldUnread.length >= 10) {
			catchUpCount = oldUnread.length;
			showCatchUpBanner = true;
		} else {
			showCatchUpBanner = false;
		}
	}
});
```

**Step 2: Add catch-up handler functions**

```typescript
async function handleCatchUp(strategy: 'older_than' | 'keep_newest', value?: string, count?: number) {
	try {
		const body: Record<string, unknown> = { strategy };
		if (strategy === 'older_than') body.value = value || '2d';
		if (strategy === 'keep_newest') body.count = count || 20;
		if (activeFeedId) body.feed_id = activeFeedId;
		if (activeCategoryId) body.category_id = activeCategoryId;
		const res = await api.post<{ affected: number }>('/api/articles/catch-up', body);
		showCatchUpBanner = false;
		// Reload articles and feed counts
		articles.load(currentFilters);
		feeds.load();
	} catch (err) {
		console.error('Catch-up failed:', err);
	}
}
```

**Step 3: Add catch-up banner UI**

Place this in the template, just above the article grid/list area (after the toolbar, around line 703, before the session estimate):

```svelte
{#if showCatchUpBanner && !openArticleId}
	<div class="mx-4 mb-4 p-4 rounded-2xl glass border border-[var(--color-border)] fade-in-up">
		<div class="flex items-start gap-3">
			<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center flex-shrink-0">
				<svg class="w-5 h-5 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium text-[var(--color-text-primary)]">
					You have {catchUpCount} unread articles older than 2 days
				</p>
				<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
					It's OK to let go. You can always search for them later.
				</p>
				<div class="flex flex-wrap gap-2 mt-3">
					<button
						onclick={() => handleCatchUp('keep_newest', undefined, 20)}
						class="px-3 py-1.5 text-xs font-medium rounded-lg bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
					>
						Keep newest 20
					</button>
					<button
						onclick={() => handleCatchUp('older_than', '2d')}
						class="px-3 py-1.5 text-xs font-medium rounded-lg bg-[var(--color-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
					>
						Clear older than 2 days
					</button>
					<button
						onclick={() => showCatchUpBanner = false}
						class="px-3 py-1.5 text-xs font-medium rounded-lg text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
					>
						Dismiss
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
```

**Step 4: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: add catch-up banner for stale unread articles"
```

---

## Task 3: Session Framing — Time Budget Prompt

Show a gentle session prompt when opening the app with many unread articles. Let users pick a time budget that curates a finite reading queue.

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/lib/stores/settings.ts` (add sessionBudget setting)

**Step 1: Add session state to settings store**

In `settings.ts`, this is NOT persisted — it's session-only state. Instead, add it directly to `+page.svelte`:

```typescript
let sessionBudget = $state<number | null>(null); // minutes, null = no limit
let sessionStartTime = $state<number>(Date.now());
let sessionArticlesRead = $state<number[]>([]); // article IDs read this session
let showSessionPrompt = $state(false);
```

**Step 2: Add session prompt trigger**

After initial article load completes in onMount (around line 282), check if there are enough unread articles to warrant a prompt:

```typescript
// After articles.load(currentFilters) resolves:
const totalUnread = $feeds.reduce((sum, f) => sum + (f.unread_count || 0), 0);
if (totalUnread >= 5) {
	showSessionPrompt = true;
}
```

**Step 3: Add session budget helper**

```typescript
let sessionBudgetArticles = $derived.by(() => {
	if (sessionBudget === null) return null; // no limit
	let remaining = sessionBudget;
	const queue: number[] = [];
	for (const a of $articles.articles) {
		if (a.is_read || sessionArticlesRead.includes(a.id)) continue;
		const time = a.reading_time || 3;
		if (remaining - time < -2) break; // allow slight overshoot
		queue.push(a.id);
		remaining -= time;
	}
	return queue;
});

let sessionTimeElapsed = $state(0);
```

Add a timer effect that updates elapsed time every 30s:

```typescript
$effect(() => {
	if (sessionBudget === null) return;
	const interval = setInterval(() => {
		sessionTimeElapsed = Math.floor((Date.now() - sessionStartTime) / 1000 / 60);
	}, 30000);
	return () => clearInterval(interval);
});
```

**Step 4: Track articles read in session**

In the existing `openArticle(id)` function (around line 100), add:

```typescript
if (!sessionArticlesRead.includes(id)) {
	sessionArticlesRead = [...sessionArticlesRead, id];
}
```

**Step 5: Add session prompt UI**

Place this as an overlay/modal when `showSessionPrompt` is true, before the main content area:

```svelte
{#if showSessionPrompt}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm fade-in">
		<div class="bg-[var(--color-card)] rounded-2xl p-6 shadow-2xl max-w-sm mx-4 fade-in-up border border-[var(--color-border)]">
			<h2 class="text-lg font-bold text-[var(--color-text-primary)]">
				How much time do you have?
			</h2>
			<p class="text-sm text-[var(--color-text-secondary)] mt-1">
				{$feeds.reduce((s, f) => s + (f.unread_count || 0), 0)} unread articles waiting
			</p>
			<div class="grid grid-cols-2 gap-2 mt-4">
				{#each [{ label: '5 min', value: 5 }, { label: '15 min', value: 15 }, { label: '30 min', value: 30 }, { label: 'No limit', value: null }] as option}
					<button
						onclick={() => { sessionBudget = option.value; sessionStartTime = Date.now(); showSessionPrompt = false; }}
						class="px-4 py-3 rounded-xl text-sm font-medium transition-all
							{option.value === null
								? 'bg-[var(--color-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)]'
								: 'bg-[var(--color-accent)]/10 text-[var(--color-accent)] hover:bg-[var(--color-accent)]/20'}"
					>
						{option.label}
					</button>
				{/each}
			</div>
			<button
				onclick={() => showSessionPrompt = false}
				class="w-full mt-3 text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
			>
				Skip
			</button>
		</div>
	</div>
{/if}
```

**Step 6: Add session timer indicator in toolbar**

When a session budget is active, show a subtle timer in the toolbar area (near the refresh countdown, around line 640):

```svelte
{#if sessionBudget !== null}
	<div class="flex items-center gap-1.5 text-xs text-[var(--color-text-tertiary)]" title="Session timer">
		<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
		</svg>
		<span>{sessionTimeElapsed} / {sessionBudget} min</span>
	</div>
{/if}
```

**Step 7: Add session budget nudge**

When sessionTimeElapsed >= sessionBudget, show a gentle nudge banner (non-blocking):

```svelte
{#if sessionBudget !== null && sessionTimeElapsed >= sessionBudget}
	<div class="mx-4 mb-3 p-3 rounded-xl bg-[var(--color-accent)]/5 border border-[var(--color-accent)]/20 fade-in-up">
		<div class="flex items-center justify-between">
			<p class="text-sm text-[var(--color-text-secondary)]">
				Your {sessionBudget} minutes are up. You read {sessionArticlesRead.length} article{sessionArticlesRead.length !== 1 ? 's' : ''}.
			</p>
			<div class="flex gap-2">
				<button
					onclick={() => { sessionBudget = sessionBudget! + 10; }}
					class="px-3 py-1 text-xs font-medium rounded-lg text-[var(--color-accent)] hover:bg-[var(--color-accent)]/10 transition-colors"
				>
					+10 min
				</button>
				<button
					onclick={() => { sessionBudget = null; }}
					class="px-3 py-1 text-xs font-medium rounded-lg text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
				>
					Done
				</button>
			</div>
		</div>
	</div>
{/if}
```

**Step 8: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: add session framing with time budget prompt"
```

---

## Task 4: Session Summary on Reader Close

Track articles read during a session and show a summary card when the user closes the reader after reading 2+ articles.

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/lib/stores/articles.ts` (expose article cache for summary data)

**Step 1: Add session summary state**

In `+page.svelte`, add:

```typescript
let showSessionSummary = $state(false);
let sessionSummaryData = $state<{
	articlesRead: number;
	totalMinutes: number;
	starred: number;
	longestTitle: string;
	longestMinutes: number;
} | null>(null);
```

**Step 2: Track read article details in session**

Expand `sessionArticlesRead` to track reading time per article. Replace the simple ID array with a map:

```typescript
let sessionReadDetails = $state<Map<number, { title: string; readingTime: number; starred: boolean }>>(new Map());
```

Update `openArticle()` to record details:

```typescript
function openArticle(id: number) {
	openArticleId = id;
	const art = $articles.articles.find(a => a.id === id);
	if (art) {
		if (art && !art.is_read) {
			articles.toggleRead(id, true);
			art.is_read = true;
			// Update feed unread count
			const feed = $feeds.find(f => f.id === art.feed_id);
			if (feed && feed.unread_count > 0) feed.unread_count--;
		}
		sessionReadDetails.set(id, {
			title: art.title,
			readingTime: art.reading_time || 3,
			starred: art.is_starred,
		});
	}
}
```

**Step 3: Build summary on reader close**

Modify `closeArticle()`:

```typescript
function closeArticle() {
	// Build session summary if 2+ articles read
	if (sessionReadDetails.size >= 2) {
		let totalMinutes = 0;
		let starred = 0;
		let longestTitle = '';
		let longestMinutes = 0;
		for (const [, detail] of sessionReadDetails) {
			totalMinutes += detail.readingTime;
			if (detail.starred) starred++;
			if (detail.readingTime > longestMinutes) {
				longestMinutes = detail.readingTime;
				longestTitle = detail.title;
			}
		}
		sessionSummaryData = {
			articlesRead: sessionReadDetails.size,
			totalMinutes,
			starred,
			longestTitle,
			longestMinutes,
		};
		showSessionSummary = true;
	}

	openArticleId = null;
	focusMode = false;
	articles.load(currentFilters);
}
```

**Step 4: Add session summary overlay UI**

Place after the reader pane section:

```svelte
{#if showSessionSummary && sessionSummaryData}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm fade-in"
		onclick={() => showSessionSummary = false}>
		<div class="bg-[var(--color-card)] rounded-2xl p-6 shadow-2xl max-w-sm mx-4 fade-in-up border border-[var(--color-border)]"
			onclick={(e) => e.stopPropagation()}>
			<div class="text-center">
				<div class="w-12 h-12 rounded-2xl accent-gradient opacity-20 flex items-center justify-center mx-auto mb-3">
					<svg class="w-6 h-6 text-[var(--color-accent)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<h2 class="text-lg font-bold text-[var(--color-text-primary)]">Reading Complete</h2>
				<div class="flex items-center justify-center gap-3 mt-3 text-sm text-[var(--color-text-secondary)]">
					<span>{sessionSummaryData.articlesRead} articles</span>
					<span class="opacity-40">·</span>
					<span>~{sessionSummaryData.totalMinutes} min</span>
					{#if sessionSummaryData.starred > 0}
						<span class="opacity-40">·</span>
						<span>{sessionSummaryData.starred} starred</span>
					{/if}
				</div>
				{#if sessionSummaryData.longestTitle}
					<p class="text-xs text-[var(--color-text-tertiary)] mt-3 italic line-clamp-2">
						"{sessionSummaryData.longestTitle}" was your longest read ({sessionSummaryData.longestMinutes} min)
					</p>
				{/if}
			</div>
			<button
				onclick={() => { showSessionSummary = false; sessionReadDetails = new Map(); }}
				class="w-full mt-5 px-4 py-2.5 text-sm font-medium text-white rounded-xl accent-gradient hover:opacity-90 transition-opacity"
			>
				Done
			</button>
		</div>
	</div>
{/if}
```

**Step 5: Update starred status in session tracking**

When user stars/unstars an article, update the session tracking map. In the existing star toggle handler (wherever `articles.toggleStar` is called in `+page.svelte`):

```typescript
// After toggling star, update session tracking
const detail = sessionReadDetails.get(articleId);
if (detail) {
	sessionReadDetails.set(articleId, { ...detail, starred: !detail.starred });
}
```

**Step 6: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat: add session summary on reader close"
```

---

## Task 5: Weekly Reading Stats in Sidebar

Add a small stats widget to the bottom of the sidebar showing weekly reading activity.

**Files:**
- Modify: `backend/internal/api/handlers/settings.go` (add reading stats endpoint)
- Modify: `backend/internal/store/article_store.go` (add reading stats query)
- Modify: `backend/internal/api/router.go` (add route)
- Modify: `frontend/src/lib/components/Sidebar.svelte`

**Step 1: Add backend reading stats query**

In `article_store.go`, add:

```go
type ReadingStats struct {
	ArticlesRead  int     `json:"articles_read"`
	TotalMinutes  int     `json:"total_minutes"`
	FeedsRead     int     `json:"feeds_read"`
}

func (q *Queries) GetWeeklyReadingStats(userID int64) (*ReadingStats, error) {
	row := q.db.QueryRow(`
		SELECT
			COUNT(*) as articles_read,
			COALESCE(SUM(a.reading_time), 0) as total_minutes,
			COUNT(DISTINCT a.feed_id) as feeds_read
		FROM articles a
		JOIN feeds f ON f.id = a.feed_id AND f.user_id = ?
		WHERE a.is_read = 1
		AND a.read_at >= datetime('now', '-7 days')
	`, userID)

	stats := &ReadingStats{}
	err := row.Scan(&stats.ArticlesRead, &stats.TotalMinutes, &stats.FeedsRead)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
```

**Step 2: Add handler**

In `settings.go`, add:

```go
func (h *SettingsHandler) GetReadingStats(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.UserID(r.Context())
	stats, err := h.store.GetWeeklyReadingStats(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
```

**Step 3: Add route**

In `router.go`, add alongside the other settings routes (around line 206):

```go
r.Get("/api/settings/reading-stats", settingsH.GetReadingStats)
```

**Step 4: Add sidebar stats widget**

In `Sidebar.svelte`, add state and fetch:

```typescript
let readingStats = $state<{ articles_read: number; total_minutes: number; feeds_read: number } | null>(null);

$effect(() => {
	api.get<{ articles_read: number; total_minutes: number; feeds_read: number }>('/api/settings/reading-stats')
		.then(s => { readingStats = s; })
		.catch(() => {});
});
```

Add the widget at the bottom of the sidebar, just before the settings dropdown (around line 440):

```svelte
{#if readingStats && readingStats.articles_read > 0 && !collapsed}
	<div class="px-4 py-3 border-t border-[var(--color-border)]">
		<p class="text-[10px] font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1.5">This week</p>
		<div class="flex items-center gap-3 text-xs text-[var(--color-text-secondary)]">
			<span>{readingStats.articles_read} read</span>
			<span class="opacity-40">·</span>
			<span>{readingStats.total_minutes} min</span>
			<span class="opacity-40">·</span>
			<span>{readingStats.feeds_read} feeds</span>
		</div>
	</div>
{/if}
```

**Step 5: Commit**

```bash
git add backend/internal/store/article_store.go backend/internal/api/handlers/settings.go backend/internal/api/router.go frontend/src/lib/components/Sidebar.svelte
git commit -m "feat: add weekly reading stats to sidebar"
```

---

## Task 6: Final Integration & Build

**Step 1: Run frontend type check**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.json
```

Expected: 0 errors (warnings OK)

**Step 2: Run backend build**

```bash
cd backend && go build ./...
```

Expected: builds cleanly

**Step 3: Run backend tests**

```bash
cd backend && go test ./...
```

Expected: all pass

**Step 4: Final commit if any fixes needed**

**Step 5: Push and rebuild**

```bash
git push
docker compose up --build -d
```

---

## Summary of Changes

| Feature | Backend | Frontend | UX Impact |
|---------|---------|----------|-----------|
| Content aging | None | Opacity fades old articles | Reduces guilt about old unread articles |
| Catch-up banner | Uses existing `/api/articles/catch-up` | Banner UI with 2 strategies | One-click inbox zero for stale content |
| Session framing | None | Time budget prompt + timer | Creates intentional reading sessions |
| Session summary | None | In-memory tracking + overlay | Satisfying reading endpoints (Peak-End Rule) |
| Weekly stats | New `GET /api/settings/reading-stats` | Sidebar widget | Reading self-awareness without shame |

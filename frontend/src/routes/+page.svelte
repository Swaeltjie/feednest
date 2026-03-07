<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ArticleCard from '$lib/components/ArticleCard.svelte';
	import ArticleList from '$lib/components/ArticleList.svelte';
	import ArticleReader from '$lib/components/ArticleReader.svelte';
	import SkeletonLoader from '$lib/components/SkeletonLoader.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import KeyboardHints from '$lib/components/KeyboardHints.svelte';
	import { articles, type ArticleFilters } from '$lib/stores/articles';
	import { feeds, categories } from '$lib/stores/feeds';
	import { api } from '$lib/api/client';
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
	let newCategoryName = $state('');
	let addFeedError = $state('');
	let searchQuery = $state('');
	let debouncedSearch = $state('');
	let initialized = $state(false);
	let selectedIndex = $state(-1);
	let cleanupKeyboard: (() => void) | undefined;
	let openArticleId = $state<number | null>(null);
	let refreshCountdown = $state(300);
	let refreshInterval: ReturnType<typeof setInterval> | undefined;
	let countdownInterval: ReturnType<typeof setInterval> | undefined;
	let commandPaletteOpen = $state(false);
	let keyboardHintsOpen = $state(false);
	let scrollY = $state(0);
	let headerCompact = $derived(scrollY > 80);
	let focusMode = $state(false);

	const FEATURED_COUNT = 3;

	let currentFilters = $derived<ArticleFilters>({
		status: filterTab === 'unread' ? 'unread' : filterTab === 'starred' ? 'starred' : undefined,
		sort: sortOption,
		feed: sidebarView === 'feed' && activeFeedId ? activeFeedId : undefined,
		category: sidebarView === 'category' && activeCategoryId ? activeCategoryId : undefined,
		search: debouncedSearch || undefined,
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

	function openArticle(id: number) {
		openArticleId = id;
		const a = $articles.articles.find((a) => a.id === id);
		if (a && !a.is_read) {
			articles.toggleRead(id, true);
		}
	}

	function closeArticle() {
		openArticleId = null;
		focusMode = false;
		articles.load(currentFilters);
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
		newCategoryName = '';
		addFeedError = '';
		mobileMenuOpen = false;
	}

	async function handleAddFeed() {
		if (!feedUrl.trim()) return;
		addingFeed = true;
		addFeedError = '';
		try {
			const body: Record<string, unknown> = { url: feedUrl.trim() };
			if (feedCategoryId) body.category_id = feedCategoryId;
			if (newCategoryName.trim()) body.new_category = newCategoryName.trim();
			await api.post('/api/feeds', body);
			await feeds.load();
			await categories.load();
			showAddFeedModal = false;
			await articles.load(currentFilters);
		} catch (err) {
			addFeedError = err instanceof Error ? err.message : 'Failed to add feed';
		} finally {
			addingFeed = false;
		}
	}

	function scrollSelectedIntoView() {
		requestAnimationFrame(() => {
			const el = document.querySelector(`[data-article-index="${selectedIndex}"]`);
			el?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		});
	}

	onMount(async () => {
		try {
			await Promise.all([feeds.load(), categories.load()]);
			await articles.load(currentFilters);
		} finally {
			initialized = true;
		}

		// Auto-refresh every 60 seconds
		refreshCountdown = 300;
		countdownInterval = setInterval(() => {
			refreshCountdown = Math.max(0, refreshCountdown - 1);
		}, 1000);
		refreshInterval = setInterval(async () => {
			refreshCountdown = 300;
			await feeds.load();
			await articles.load(currentFilters);
		}, 300000);

		cleanupKeyboard = setupKeyboardShortcuts({
			j: (e) => {
				const articleList = $articles.articles;
				if (articleList.length > 0) {
					selectedIndex = Math.min(selectedIndex + 1, articleList.length - 1);
					scrollSelectedIntoView();
				}
			},
			k: (e) => {
				if (selectedIndex > 0) {
					selectedIndex = selectedIndex - 1;
					scrollSelectedIntoView();
				}
			},
			enter: (e) => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					openArticle(articleList[selectedIndex].id);
				}
			},
			escape: (e) => {
				if (openArticleId) {
					closeArticle();
				}
			},
			f: (e) => {
				if (openArticleId) {
					focusMode = !focusMode;
				}
			},
			s: (e) => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					const a = articleList[selectedIndex];
					articles.toggleStar(a.id, !a.is_starred);
				}
			},
			m: (e) => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					const a = articleList[selectedIndex];
					articles.toggleRead(a.id, !a.is_read);
				}
			},
			d: (e) => {
				const articleList = $articles.articles;
				if (selectedIndex >= 0 && selectedIndex < articleList.length) {
					articles.dismiss(articleList[selectedIndex].id);
					if (selectedIndex >= articleList.length - 1) {
						selectedIndex = Math.max(0, articleList.length - 2);
					}
				}
			},
			v: (e) => {
				const modes: ViewMode[] = ['hybrid', 'cards', 'list'];
				const current = modes.indexOf(viewMode);
				setViewMode(modes[(current + 1) % modes.length]);
			},
			'/': (e) => {
				const searchInput = document.querySelector<HTMLInputElement>(
					'input[type="search"], input[placeholder*="earch"]'
				);
				if (searchInput) {
					searchInput.focus();
				}
			},
			'cmd+k': (e) => {
				commandPaletteOpen = !commandPaletteOpen;
			},
			'?': (e) => {
				keyboardHintsOpen = !keyboardHintsOpen;
			},
			gg: (e) => {
				selectedIndex = 0;
				scrollSelectedIntoView();
			},
			'G': (e) => {
				const articleList = $articles.articles;
				if (articleList.length > 0) {
					selectedIndex = articleList.length - 1;
					scrollSelectedIntoView();
				}
			},
			r: (e) => {
				refreshCountdown = 300;
				feeds.load();
				articles.load(currentFilters);
			},
			'1': (e) => setViewMode('hybrid'),
			'2': (e) => setViewMode('cards'),
			'3': (e) => setViewMode('list'),
		});
	});

	onDestroy(() => {
		cleanupKeyboard?.();
		clearInterval(refreshInterval);
		clearInterval(countdownInterval);
	});

	// Debounce search input
	$effect(() => {
		const q = searchQuery;
		const timer = setTimeout(() => { debouncedSearch = q; }, 300);
		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (!initialized) return;
		const filters = currentFilters;
		articles.load(filters);
	});

	let pageTitle = $derived.by(() => {
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
	<title>{pageTitle} - FeedNest</title>
</svelte:head>

<svelte:window bind:scrollY={scrollY} />

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
		<header class="sticky top-0 z-30 glass" style="transition: padding var(--duration-snappy) var(--spring-snappy);">
			<div class="flex items-center justify-between px-4 {headerCompact ? 'py-2' : 'py-3'}" style="transition: padding var(--duration-snappy) var(--spring-snappy);">
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
					<div
						class="transition-all overflow-hidden"
						style="max-height: {headerCompact ? '0px' : '60px'}; opacity: {headerCompact ? 0 : 1}; transition: max-height var(--duration-snappy) var(--spring-snappy), opacity var(--duration-snappy) var(--spring-snappy);"
					>
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
				</div>

				<!-- Search -->
				<div class="hidden sm:flex items-center flex-1 max-w-xs mx-4">
					<div class="relative w-full">
						<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
						<input
							type="search"
							bind:value={searchQuery}
							placeholder="Search articles..."
							class="w-full pl-10 pr-4 py-1.5 text-sm rounded-lg
								bg-[var(--color-card)] border border-[var(--color-border)]
								text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
						/>
						{#if searchQuery}
							<button
								onclick={() => (searchQuery = '')}
								aria-label="Clear search"
								class="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
							>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						{/if}
					</div>
				</div>

				<div class="flex items-center gap-3">
					<!-- Refresh countdown -->
					<button
						onclick={async () => { refreshCountdown = 300; await feeds.load(); await articles.load(currentFilters); }}
						class="group flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all
							hover:bg-[var(--color-elevated)] text-[var(--color-text-tertiary)] hover:text-[var(--color-accent)]"
						title="Click to refresh now"
					>
						<svg class="w-3.5 h-3.5 transition-transform group-hover:rotate-180 duration-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
						</svg>
						<span class="tabular-nums w-4 text-center">{refreshCountdown}</span>
						<svg class="w-3 h-3 opacity-60" viewBox="0 0 36 36">
							<circle cx="18" cy="18" r="15" fill="none" stroke="var(--color-border)" stroke-width="3" />
							<circle cx="18" cy="18" r="15" fill="none" stroke="var(--color-accent)" stroke-width="3"
								stroke-dasharray={2 * Math.PI * 15}
								stroke-dashoffset={2 * Math.PI * 15 * (1 - refreshCountdown / 300)}
								stroke-linecap="round"
								transform="rotate(-90 18 18)"
								class="transition-all duration-1000 ease-linear"
							/>
						</svg>
					</button>

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
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
							</svg>
						</button>
					</div>

					<ThemeToggle />
				</div>
			</div>
		</header>

		<!-- Content area: list + optional reading pane -->
		<div class="flex-1 flex overflow-hidden">
			<!-- Article list pane -->
			<div
				class="overflow-y-auto transition-all duration-300
					{openArticleId
						? focusMode
							? 'w-0 min-w-0 opacity-0 overflow-hidden'
							: 'hidden lg:block lg:w-[350px] lg:min-w-[350px] lg:border-r lg:border-[var(--color-border)]'
						: 'w-full'}"
			>
				{#if $articles.loading && !initialized}
					<SkeletonLoader mode={viewMode} />
				{:else if $articles.articles.length === 0}
					<!-- Empty state -->
					<div class="flex flex-col items-center justify-center py-20 text-center px-4 fade-in-up">
						<div class="animate-float mb-6">
							<div class="w-20 h-20 rounded-2xl accent-gradient flex items-center justify-center shadow-lg">
								<svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
								</svg>
							</div>
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
					<!-- When article is open, always show compact list view -->
					{#if openArticleId}
						<div style="background: var(--color-card);">
							{#each $articles.articles as article, i (article.id)}
								<div data-article-index={i}>
									<ArticleList {article} selected={article.id === openArticleId} index={i} onOpen={openArticle} onToggleRead={(id, isRead) => articles.toggleRead(id, isRead)} onToggleStar={(id, isStarred) => articles.toggleStar(id, isStarred)} />
								</div>
							{/each}
						</div>
					{:else}
						<!-- Normal view modes when no article open -->
						{#if viewMode === 'hybrid'}
							{#if featuredArticles.length > 0}
								<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
									{#each featuredArticles as article, i (article.id)}
										<div data-article-index={$articles.articles.indexOf(article)}>
											<ArticleCard {article} selected={$articles.articles.indexOf(article) === selectedIndex} index={i} onOpen={openArticle} />
										</div>
									{/each}
								</div>
							{/if}
							<div style="background: var(--color-card);" class="rounded-t-2xl mx-2 mt-2">
								{#each listArticles as article, i (article.id)}
									<div data-article-index={$articles.articles.indexOf(article)}>
										<ArticleList {article} selected={$articles.articles.indexOf(article) === selectedIndex} index={i} onOpen={openArticle} onToggleRead={(id, isRead) => articles.toggleRead(id, isRead)} onToggleStar={(id, isStarred) => articles.toggleStar(id, isStarred)} />
									</div>
								{/each}
							</div>
						{:else if viewMode === 'cards'}
							<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
								{#each $articles.articles as article, i (article.id)}
									<div data-article-index={i}>
										<ArticleCard {article} selected={i === selectedIndex} index={i} onOpen={openArticle} />
									</div>
								{/each}
							</div>
						{:else}
							<div style="background: var(--color-card);" class="m-2 rounded-2xl overflow-hidden">
								{#each $articles.articles as article, i (article.id)}
									<div data-article-index={i}>
										<ArticleList {article} selected={i === selectedIndex} index={i} onOpen={openArticle} onToggleRead={(id, isRead) => articles.toggleRead(id, isRead)} onToggleStar={(id, isStarred) => articles.toggleStar(id, isStarred)} />
									</div>
								{/each}
							</div>
						{/if}

						{#if $articles.loading && initialized}
							<div class="flex items-center justify-center py-4">
								<div class="w-5 h-5 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
							</div>
						{/if}

						{#if $articles.articles.length > 0}
							<div class="text-center py-4 text-sm text-[var(--color-text-tertiary)]">
								Showing {$articles.articles.length} of {$articles.total} articles
							</div>
						{/if}
					{/if}
				{/if}
			</div>

			<!-- Reading pane (inline, not overlay) -->
			{#if openArticleId}
				<div class="flex-1 min-w-0" style="background: var(--color-surface);">
					<ArticleReader
						articleId={openArticleId}
						onClose={closeArticle}
						articleIds={$articles.articles.map(a => a.id)}
						onNavigate={(id) => { openArticleId = id; }}
						inline={true}
						{focusMode}
						onToggleFocus={() => { focusMode = !focusMode; }}
					/>
				</div>
			{/if}
		</div>
	</div>
</div>

<!-- Command Palette -->
<CommandPalette
	bind:open={commandPaletteOpen}
	onSelectFeed={selectFeed}
	onSelectCategory={selectCategory}
	onSelectAll={selectAll}
	onSelectStarred={selectStarred}
	onRefresh={async () => { refreshCountdown = 300; await feeds.load(); await articles.load(currentFilters); }}
/>

<!-- Keyboard Hints -->
<KeyboardHints bind:open={keyboardHintsOpen} />

<!-- Add Feed Modal -->
{#if showAddFeedModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button
			class="absolute inset-0 bg-black/60 backdrop-blur-sm"
			onclick={() => (showAddFeedModal = false)}
			aria-label="Close modal"
		></button>
		<div class="relative rounded-2xl shadow-2xl w-full max-w-md p-6 space-y-4 fade-in-up border border-[var(--color-border)]" style="background: var(--color-card);">
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
					class="w-full px-4 py-2.5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border-hover)]
						text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
				/>
			</div>

			<div>
				<label for="feed-category" class="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
					Category
				</label>
				<select
					id="feed-category"
					bind:value={feedCategoryId}
					disabled={!!newCategoryName.trim()}
					class="w-full px-4 py-2.5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border-hover)]
						text-[var(--color-text-primary)]
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all
						disabled:opacity-50 appearance-none cursor-pointer"
				>
					<option value={undefined}>None</option>
					{#each $categories as cat}
						<option value={cat.id}>{cat.name}</option>
					{/each}
				</select>
				<div class="text-center text-xs text-[var(--color-text-tertiary)] my-2">or create new</div>
				<input
					type="text"
					bind:value={newCategoryName}
					placeholder="New category name"
					class="w-full px-4 py-2.5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border-hover)]
						text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
				/>
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

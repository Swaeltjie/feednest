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

<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ArticleCard from '$lib/components/ArticleCard.svelte';
	import ArticleList from '$lib/components/ArticleList.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { articles, type ArticleFilters } from '$lib/stores/articles';
	import { feeds, categories } from '$lib/stores/feeds';
	import { setupKeyboardShortcuts } from '$lib/utils/keyboard';

	type ViewMode = 'cards' | 'list';
	type FilterTab = 'all' | 'unread' | 'starred';
	type SortOption = 'smart' | 'newest' | 'oldest';
	type SidebarView = 'all' | 'starred' | 'feed' | 'category';

	let sidebarCollapsed = $state(false);
	let mobileMenuOpen = $state(false);
	let viewMode = $state<ViewMode>(
		(typeof localStorage !== 'undefined' && (localStorage.getItem('feednest_view') as ViewMode)) ||
			'cards'
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

	let currentFilters = $derived<ArticleFilters>({
		status: filterTab === 'unread' ? 'unread' : filterTab === 'starred' ? 'starred' : undefined,
		sort: sortOption,
		feed: sidebarView === 'feed' && activeFeedId ? activeFeedId : undefined,
		category: sidebarView === 'category' && activeCategoryId ? activeCategoryId : undefined,
	});

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
				setViewMode(viewMode === 'cards' ? 'list' : 'cards');
			},
			'/': () => {
				const searchInput = document.querySelector<HTMLInputElement>('input[type="search"], input[placeholder*="earch"]');
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
		// Access the derived filters to track changes
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

<div class="flex h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Mobile sidebar overlay -->
	{#if mobileMenuOpen}
		<div class="fixed inset-0 z-40 lg:hidden">
			<button
				class="absolute inset-0 bg-black/50"
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
		<!-- Toolbar -->
		<header
			class="sticky top-0 z-30 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700"
		>
			<div class="flex items-center justify-between px-4 py-3">
				<div class="flex items-center gap-3">
					<!-- Mobile hamburger -->
					<button
						class="lg:hidden p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
						onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
						aria-label="Toggle sidebar"
					>
						<svg
							class="w-5 h-5 text-gray-600 dark:text-gray-300"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M4 6h16M4 12h16M4 18h16"
							/>
						</svg>
					</button>

					<!-- Desktop sidebar toggle -->
					<button
						class="hidden lg:block p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
						onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
						aria-label="Toggle sidebar"
					>
						<svg
							class="w-5 h-5 text-gray-600 dark:text-gray-300"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M4 6h16M4 12h16M4 18h16"
							/>
						</svg>
					</button>

					<!-- Filter tabs -->
					<div class="flex items-center gap-1 bg-gray-100 dark:bg-gray-700 rounded-lg p-0.5">
						<button
							onclick={() => (filterTab = 'all')}
							class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors {filterTab ===
							'all'
								? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
								: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'}"
						>
							All
						</button>
						<button
							onclick={() => (filterTab = 'unread')}
							class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors {filterTab ===
							'unread'
								? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
								: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'}"
						>
							Unread
						</button>
						<button
							onclick={() => (filterTab = 'starred')}
							class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors {filterTab ===
							'starred'
								? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
								: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'}"
						>
							Starred
						</button>
					</div>
				</div>

				<div class="flex items-center gap-3">
					<!-- Sort dropdown -->
					<select
						bind:value={sortOption}
						class="text-sm bg-gray-100 dark:bg-gray-700 border-0 rounded-lg px-3 py-1.5 text-gray-700 dark:text-gray-300 focus:ring-2 focus:ring-blue-500"
					>
						<option value="smart">Smart</option>
						<option value="newest">Newest</option>
						<option value="oldest">Oldest</option>
					</select>

					<!-- View toggle -->
					<div class="flex items-center gap-1 bg-gray-100 dark:bg-gray-700 rounded-lg p-0.5">
						<button
							onclick={() => setViewMode('cards')}
							class="p-1.5 rounded-md transition-colors {viewMode === 'cards'
								? 'bg-white dark:bg-gray-600 shadow-sm'
								: ''}"
							title="Card view"
						>
							<svg
								class="w-4 h-4 text-gray-600 dark:text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
								/>
							</svg>
						</button>
						<button
							onclick={() => setViewMode('list')}
							class="p-1.5 rounded-md transition-colors {viewMode === 'list'
								? 'bg-white dark:bg-gray-600 shadow-sm'
								: ''}"
							title="List view"
						>
							<svg
								class="w-4 h-4 text-gray-600 dark:text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M4 6h16M4 12h16M4 18h16"
								/>
							</svg>
						</button>
					</div>

					<!-- Theme toggle -->
					<ThemeToggle />
				</div>
			</div>
		</header>

		<!-- Article content -->
		<main class="flex-1 overflow-y-auto">
			{#if $articles.loading && !initialized}
				<div class="flex items-center justify-center h-64">
					<div class="text-gray-500 dark:text-gray-400">Loading articles...</div>
				</div>
			{:else if $articles.articles.length === 0}
				<!-- Empty state -->
				<div class="flex flex-col items-center justify-center h-64 text-center px-4">
					<svg
						class="w-16 h-16 text-gray-300 dark:text-gray-600 mb-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.5"
							d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z"
						/>
					</svg>
					<h2 class="text-lg font-medium text-gray-900 dark:text-white mb-1">No articles found</h2>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						{#if $feeds.length === 0}
							Add a feed to get started.
						{:else}
							Try changing your filters or check back later.
						{/if}
					</p>
					{#if $feeds.length === 0}
						<button
							onclick={openAddFeed}
							class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
						>
							Add Your First Feed
						</button>
					{/if}
				</div>
			{:else if viewMode === 'cards'}
				<!-- Card grid view -->
				<div
					class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4"
				>
					{#each $articles.articles as article, i (article.id)}
						<ArticleCard {article} selected={i === selectedIndex} />
					{/each}
				</div>
			{:else}
				<!-- List view -->
				<div class="bg-white dark:bg-gray-800">
					{#each $articles.articles as article, i (article.id)}
						<ArticleList {article} selected={i === selectedIndex} />
					{/each}
				</div>
			{/if}

			<!-- Loading indicator for filter changes -->
			{#if $articles.loading && initialized}
				<div class="flex items-center justify-center py-4">
					<div
						class="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"
					></div>
				</div>
			{/if}

			<!-- Article count footer -->
			{#if $articles.articles.length > 0}
				<div class="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
					Showing {$articles.articles.length} of {$articles.total} articles
				</div>
			{/if}
		</main>
	</div>
</div>

<!-- Add Feed Modal -->
{#if showAddFeedModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button
			class="absolute inset-0 bg-black/50"
			onclick={() => (showAddFeedModal = false)}
			aria-label="Close modal"
		></button>
		<div
			class="relative bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6 space-y-4"
		>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Add Feed</h2>

			{#if addFeedError}
				<div
					class="p-3 text-sm text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded-lg"
				>
					{addFeedError}
				</div>
			{/if}

			<div>
				<label
					for="feed-url"
					class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					Feed URL
				</label>
				<input
					id="feed-url"
					type="url"
					bind:value={feedUrl}
					placeholder="https://example.com/feed.xml"
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
				/>
			</div>

			<div>
				<label
					for="feed-category"
					class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					Category (optional)
				</label>
				<select
					id="feed-category"
					bind:value={feedCategoryId}
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
					class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleAddFeed}
					disabled={addingFeed || !feedUrl.trim()}
					class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{addingFeed ? 'Adding...' : 'Add Feed'}
				</button>
			</div>
		</div>
	</div>
{/if}

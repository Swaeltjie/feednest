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
		for (const cat of catList) {
			catMap.set(cat.id, []);
		}

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
	class="flex flex-col h-full bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transition-all duration-200 {collapsed
		? 'w-0 overflow-hidden'
		: 'w-64'}"
>
	<!-- Header -->
	<div class="flex items-center gap-2 px-4 py-4 border-b border-gray-200 dark:border-gray-700">
		<h1 class="text-lg font-bold text-gray-900 dark:text-white whitespace-nowrap">FeedNest</h1>
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto py-2">
		<!-- All Articles -->
		<button
			onclick={onSelectAll}
			class="flex items-center justify-between w-full px-4 py-2 text-sm text-left transition-colors {activeView ===
			'all'
				? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
				: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700/50'}"
		>
			<span class="flex items-center gap-2">
				<svg
					class="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					xmlns="http://www.w3.org/2000/svg"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z"
					/>
				</svg>
				All Articles
			</span>
			{#if allUnread > 0}
				<span
					class="px-1.5 py-0.5 text-xs font-medium rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300"
				>
					{allUnread}
				</span>
			{/if}
		</button>

		<!-- Starred -->
		<button
			onclick={onSelectStarred}
			class="flex items-center gap-2 w-full px-4 py-2 text-sm text-left transition-colors {activeView ===
			'starred'
				? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
				: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700/50'}"
		>
			<svg
				class="w-4 h-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
				/>
			</svg>
			Starred
		</button>

		<div class="my-2 border-t border-gray-200 dark:border-gray-700"></div>

		<!-- Categorized feeds -->
		{#each feedsByCategory.grouped as { category, feeds: catFeeds }}
			{@const catUnread = totalUnread(catFeeds)}
			<div class="mt-1">
				<button
					onclick={() => onSelectCategory(category.id)}
					class="flex items-center justify-between w-full px-4 py-1.5 text-left transition-colors {activeView ===
						'category' && activeCategory === category.id
						? 'bg-blue-50 dark:bg-blue-900/30'
						: 'hover:bg-gray-100 dark:hover:bg-gray-700/50'}"
				>
					<span
						class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
					>
						{category.name}
					</span>
					{#if catUnread > 0}
						<span
							class="px-1.5 py-0.5 text-xs font-medium rounded-full bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300"
						>
							{catUnread}
						</span>
					{/if}
				</button>
				{#each catFeeds as feed}
					<button
						onclick={() => onSelectFeed(feed.id)}
						class="flex items-center justify-between w-full pl-8 pr-4 py-1.5 text-sm text-left transition-colors {activeView ===
							'feed' && activeFeed === feed.id
							? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
							: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700/50'}"
					>
						<span class="truncate">{feed.title}</span>
						{#if feed.unread_count > 0}
							<span
								class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300"
							>
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
				<span
					class="block px-4 py-1.5 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
				>
					Uncategorized
				</span>
				{#each feedsByCategory.uncategorized as feed}
					<button
						onclick={() => onSelectFeed(feed.id)}
						class="flex items-center justify-between w-full pl-8 pr-4 py-1.5 text-sm text-left transition-colors {activeView ===
							'feed' && activeFeed === feed.id
							? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
							: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700/50'}"
					>
						<span class="truncate">{feed.title}</span>
						{#if feed.unread_count > 0}
							<span
								class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300"
							>
								{feed.unread_count}
							</span>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</nav>

	<!-- Bottom actions -->
	<div class="border-t border-gray-200 dark:border-gray-700 p-3 space-y-2">
		<button
			onclick={onAddFeed}
			class="flex items-center gap-2 w-full px-3 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
		>
			<svg
				class="w-4 h-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 4v16m8-8H4"
				/>
			</svg>
			Add Feed
		</button>
		<button
			onclick={() => auth.logout()}
			class="flex items-center gap-2 w-full px-3 py-2 text-sm text-gray-500 dark:text-gray-400 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700/50 transition-colors"
		>
			<svg
				class="w-4 h-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
				/>
			</svg>
			Sign Out
		</button>
	</div>
</aside>

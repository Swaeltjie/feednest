<script lang="ts">
	import { feeds, categories, type Feed, type Category } from '$lib/stores/feeds';
	import { auth } from '$lib/stores/auth';
	import { getFaviconUrl } from '$lib/utils/favicon';
	import AnimatedCount from './AnimatedCount.svelte';

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

	let contextMenu = $state<{ feed: Feed; x: number; y: number } | null>(null);
	let deletingFeedId = $state<number | null>(null);
	// Start with all categories collapsed; populated once categories load
	let collapsedCategories = $state<Set<number>>(new Set());
	let initializedCollapsed = $state(false);

	// Drag and drop state
	let draggedFeed = $state<Feed | null>(null);
	let dragOverCategoryId = $state<number | null>(null);
	let dragOverUncategorized = $state(false);

	function toggleCategory(e: Event, categoryId: number) {
		e.stopPropagation();
		const next = new Set(collapsedCategories);
		if (next.has(categoryId)) {
			next.delete(categoryId);
		} else {
			next.add(categoryId);
		}
		collapsedCategories = next;
	}

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

	// Collapse all categories by default on first load
	$effect(() => {
		if (!initializedCollapsed && $categories.length > 0) {
			collapsedCategories = new Set($categories.map(c => c.id));
			initializedCollapsed = true;
		}
	});

	function totalUnread(feedList: Feed[]): number {
		return feedList.reduce((sum, f) => sum + f.unread_count, 0);
	}

	let allUnread = $derived(totalUnread($feeds));

	function handleContextMenu(e: MouseEvent, feed: Feed) {
		e.preventDefault();
		contextMenu = { feed, x: e.clientX, y: e.clientY };
	}

	function closeContextMenu() {
		contextMenu = null;
	}

	async function handleDeleteFeed(feed: Feed) {
		contextMenu = null;
		deletingFeedId = feed.id;
		try {
			await feeds.remove(feed.id);
		} finally {
			deletingFeedId = null;
		}
	}

	function feedIcon(feed: Feed): string | null {
		return getFaviconUrl(feed.icon_url, feed.site_url, feed.url);
	}

	// Drag handlers
	function handleDragStart(e: DragEvent, feed: Feed) {
		draggedFeed = feed;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', String(feed.id));
		}
	}

	function handleDragEnd() {
		draggedFeed = null;
		dragOverCategoryId = null;
		dragOverUncategorized = false;
	}

	function handleCategoryDragOver(e: DragEvent, categoryId: number) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		dragOverCategoryId = categoryId;
		dragOverUncategorized = false;
	}

	function handleUncategorizedDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		dragOverUncategorized = true;
		dragOverCategoryId = null;
	}

	function handleDragLeave() {
		dragOverCategoryId = null;
		dragOverUncategorized = false;
	}

	async function handleCategoryDrop(e: DragEvent, targetCategoryId: number) {
		e.preventDefault();
		if (!draggedFeed || draggedFeed.category_id === targetCategoryId) {
			handleDragEnd();
			return;
		}

		const oldCategoryId = draggedFeed.category_id;
		await feeds.update(draggedFeed.id, { category_id: targetCategoryId });

		// Auto-delete empty category
		if (oldCategoryId) {
			await checkAndDeleteEmptyCategory(oldCategoryId);
		}

		handleDragEnd();
	}

	async function handleUncategorizedDrop(e: DragEvent) {
		e.preventDefault();
		if (!draggedFeed || !draggedFeed.category_id) {
			handleDragEnd();
			return;
		}

		const oldCategoryId = draggedFeed.category_id;
		await feeds.update(draggedFeed.id, { category_id: null });

		// Auto-delete empty category
		if (oldCategoryId) {
			await checkAndDeleteEmptyCategory(oldCategoryId);
		}

		handleDragEnd();
	}

	async function checkAndDeleteEmptyCategory(categoryId: number) {
		// Reload feeds to get fresh counts, then check if category is now empty
		await feeds.load();
		const remainingFeeds = $feeds.filter(f => f.category_id === categoryId);
		if (remainingFeeds.length === 0) {
			await categories.remove(categoryId);
		}
	}
</script>

<svelte:window onclick={closeContextMenu} />

<aside
	class="flex flex-col h-full border-r border-[var(--color-border)] transition-all duration-300 ease-in-out overflow-hidden"
	style="background: var(--color-base); width: {collapsed ? '0px' : '16rem'};"
>
	<!-- Logo -->
	<div class="flex items-center gap-2 px-4 py-4 border-b border-[var(--color-border)]">
		<div class="w-7 h-7 rounded-lg accent-gradient flex items-center justify-center flex-shrink-0">
			<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 110-2 1 1 0 010 2z" />
			</svg>
		</div>
		<h1 class="text-lg font-bold accent-gradient-text whitespace-nowrap tracking-tight">FeedNest</h1>
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto py-2 px-1.5">
		<!-- All Articles -->
		<button
			onclick={onSelectAll}
			class="flex items-center justify-between w-full px-3 py-2.5 text-sm text-left transition-all rounded-xl
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
				<span class="px-2 py-0.5 text-xs font-semibold rounded-full accent-gradient text-white min-w-[1.25rem] text-center">
					<AnimatedCount value={allUnread} />
				</span>
			{/if}
		</button>

		<!-- Starred -->
		<button
			onclick={onSelectStarred}
			class="flex items-center gap-2.5 w-full px-3 py-2.5 text-sm text-left transition-all rounded-xl
				{activeView === 'starred'
					? 'glow-active text-[var(--color-accent)]'
					: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
			</svg>
			Starred
		</button>

		<div class="my-2.5 mx-2 border-t border-[var(--color-border)]"></div>

		<!-- Categorized feeds -->
		{#each feedsByCategory.grouped as { category, feeds: catFeeds }}
			{@const catUnread = totalUnread(catFeeds)}
			{@const isCollapsed = collapsedCategories.has(category.id)}
			{@const isActive = activeView === 'category' && activeCategory === category.id}
			{@const isDragOver = dragOverCategoryId === category.id}
			<div
				class="mt-1 rounded-lg transition-all duration-200 {isDragOver ? 'bg-[var(--color-accent)]/10 ring-1 ring-[var(--color-accent)]/30' : ''}"
				ondragover={(e) => handleCategoryDragOver(e, category.id)}
				ondragleave={handleDragLeave}
				ondrop={(e) => handleCategoryDrop(e, category.id)}
			>
				<div class="flex items-center gap-0.5">
					<!-- Chevron toggle -->
					<button
						onclick={(e) => toggleCategory(e, category.id)}
						class="p-1 rounded-md hover:bg-[var(--color-elevated)] transition-all flex-shrink-0"
						aria-label={isCollapsed ? 'Expand' : 'Collapse'}
					>
						<svg class="w-3 h-3 text-[var(--color-text-tertiary)] transition-transform duration-200 {isCollapsed ? '-rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
						</svg>
					</button>

					<!-- Category name (clickable to filter) -->
					<button
						onclick={() => onSelectCategory(category.id)}
						class="flex items-center justify-between flex-1 min-w-0 px-2 py-1.5 text-left transition-all rounded-lg cursor-pointer
							{isActive ? 'glow-active' : 'hover:bg-[var(--color-elevated)]'}"
					>
						<span class="flex items-center gap-2 truncate">
							<svg class="w-3.5 h-3.5 flex-shrink-0 {isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-tertiary)]'}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
							</svg>
							<span class="text-xs font-semibold uppercase tracking-wider truncate
								{isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-tertiary)]'}">
								{category.name}
							</span>
						</span>
						{#if catUnread > 0}
							<span class="ml-1 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
								<AnimatedCount value={catUnread} />
							</span>
						{/if}
					</button>
				</div>

				{#if !isCollapsed}
					<div class="ml-3 border-l border-[var(--color-border)] pl-1">
						{#each catFeeds as feed}
							<button
								draggable="true"
								ondragstart={(e) => handleDragStart(e, feed)}
								ondragend={handleDragEnd}
								onclick={() => onSelectFeed(feed.id)}
								oncontextmenu={(e) => handleContextMenu(e, feed)}
								class="flex items-center justify-between w-full pl-3 pr-3 py-1.5 text-sm text-left transition-all rounded-lg cursor-grab active:cursor-grabbing
									{deletingFeedId === feed.id ? 'opacity-40' : ''}
									{draggedFeed?.id === feed.id ? 'opacity-30' : ''}
									{activeView === 'feed' && activeFeed === feed.id
										? 'glow-active text-[var(--color-accent)]'
										: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
							>
								<span class="flex items-center gap-2 truncate">
									{#if feedIcon(feed)}
										<img src={feedIcon(feed)} alt="" class="w-4 h-4 rounded-full flex-shrink-0" onerror={(e) => { const img = e.currentTarget as HTMLImageElement; const parent = img.parentElement; if (parent) { const span = document.createElement('span'); span.className = 'w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold'; span.textContent = img.closest('button')?.textContent?.trim()?.charAt(0)?.toUpperCase() || '?'; parent.replaceChild(span, img); } }} />
									{:else}
										<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
											{feed.title?.charAt(0)?.toUpperCase() || '?'}
										</span>
									{/if}
									<span class="truncate">{feed.title}</span>
								</span>
								{#if feed.unread_count > 0}
									<span class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
										<AnimatedCount value={feed.unread_count} />
									</span>
								{/if}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/each}

		<!-- Uncategorized feeds -->
		{#if feedsByCategory.uncategorized.length > 0 || draggedFeed}
			<div
				class="mt-1 rounded-lg transition-all duration-200 {dragOverUncategorized ? 'bg-[var(--color-accent)]/10 ring-1 ring-[var(--color-accent)]/30' : ''}"
				ondragover={handleUncategorizedDragOver}
				ondragleave={handleDragLeave}
				ondrop={handleUncategorizedDrop}
			>
				<span class="block px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">
					{dragOverUncategorized ? 'Drop to uncategorize' : 'Feeds'}
				</span>
				{#each feedsByCategory.uncategorized as feed}
					<button
						draggable="true"
						ondragstart={(e) => handleDragStart(e, feed)}
						ondragend={handleDragEnd}
						onclick={() => onSelectFeed(feed.id)}
						oncontextmenu={(e) => handleContextMenu(e, feed)}
						class="flex items-center justify-between w-full pl-4 pr-3 py-1.5 text-sm text-left transition-all rounded-lg cursor-grab active:cursor-grabbing
							{deletingFeedId === feed.id ? 'opacity-40' : ''}
							{draggedFeed?.id === feed.id ? 'opacity-30' : ''}
							{activeView === 'feed' && activeFeed === feed.id
								? 'glow-active text-[var(--color-accent)]'
								: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
					>
						<span class="flex items-center gap-2 truncate">
							{#if feedIcon(feed)}
								<img src={feedIcon(feed)} alt="" class="w-4 h-4 rounded-full flex-shrink-0" onerror={(e) => { const img = e.currentTarget as HTMLImageElement; const parent = img.parentElement; if (parent) { const span = document.createElement('span'); span.className = 'w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold'; span.textContent = img.closest('button')?.textContent?.trim()?.charAt(0)?.toUpperCase() || '?'; parent.replaceChild(span, img); } }} />
							{:else}
								<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
									{feed.title?.charAt(0)?.toUpperCase() || '?'}
								</span>
							{/if}
							<span class="truncate">{feed.title}</span>
						</span>
						{#if feed.unread_count > 0}
							<span class="ml-2 flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-elevated)] text-[var(--color-text-secondary)]">
								<AnimatedCount value={feed.unread_count} />
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

<!-- Context menu -->
{#if contextMenu}
	<div
		class="fixed z-[100] py-1.5 rounded-xl shadow-2xl min-w-[160px] glass border border-[var(--color-border-hover)] fade-in-up"
		style="left: {contextMenu.x}px; top: {contextMenu.y}px; animation-duration: 100ms;"
	>
		<button
			onclick={() => contextMenu && handleDeleteFeed(contextMenu.feed)}
			class="flex items-center gap-2.5 w-full px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
			</svg>
			Remove Feed
		</button>
	</div>
{/if}

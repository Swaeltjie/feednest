<script lang="ts">
	import { feeds, categories, type Feed, type Category } from '$lib/stores/feeds';
	import { auth } from '$lib/stores/auth';
	import { getFaviconUrl, handleFaviconError } from '$lib/utils/favicon';
	import AnimatedCount from './AnimatedCount.svelte';
	import { settings } from '$lib/stores/settings';

	let {
		collapsed = false,
		activeFeed = null as number | null,
		activeCategory = null as number | null,
		activeView = 'all' as 'all' | 'starred' | 'today' | 'long_reads' | 'feed' | 'category',
		onSelectAll = () => {},
		onSelectStarred = () => {},
		onSelectToday = () => {},
		onSelectLongReads = () => {},
		onSelectFeed = (_id: number) => {},
		onSelectCategory = (_id: number) => {},
		onAddFeed = () => {},
		onMarkFeedRead = (_id: number) => {},
		onMarkCategoryRead = (_id: number) => {},
	}: {
		collapsed?: boolean;
		activeFeed?: number | null;
		activeCategory?: number | null;
		activeView?: 'all' | 'starred' | 'today' | 'long_reads' | 'feed' | 'category';
		onSelectAll?: () => void;
		onSelectStarred?: () => void;
		onSelectToday?: () => void;
		onSelectLongReads?: () => void;
		onSelectFeed?: (id: number) => void;
		onSelectCategory?: (id: number) => void;
		onAddFeed?: () => void;
		onMarkFeedRead?: (id: number) => void;
		onMarkCategoryRead?: (id: number) => void;
	} = $props();

	let contextMenu = $state<{ feed: Feed; x: number; y: number } | null>(null);
	let categoryContextMenu = $state<{ category: Category; x: number; y: number } | null>(null);
	let deletingFeedId = $state<number | null>(null);
	// Start with all categories collapsed; populated once categories load
	let collapsedCategories = $state<Set<number>>(new Set());
	let initializedCollapsed = $state(false);

	// Drag and drop state
	let showSettings = $state(false);
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
		categoryContextMenu = null;
		contextMenu = { feed, x: e.clientX, y: e.clientY };
	}

	function handleCategoryContextMenu(e: MouseEvent, category: Category) {
		e.preventDefault();
		contextMenu = null;
		categoryContextMenu = { category, x: e.clientX, y: e.clientY };
	}

	function closeContextMenu() {
		contextMenu = null;
		categoryContextMenu = null;
		showSettings = false;
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
			{#if !$settings.calmMode && allUnread > 0}
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

		<!-- Today -->
		<button
			onclick={onSelectToday}
			class="flex items-center gap-2.5 w-full px-3 py-2.5 text-sm text-left transition-all rounded-xl
				{activeView === 'today'
					? 'glow-active text-[var(--color-accent)]'
					: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			Today
		</button>

		<!-- Long Reads -->
		<button
			onclick={onSelectLongReads}
			class="flex items-center gap-2.5 w-full px-3 py-2.5 text-sm text-left transition-all rounded-xl
				{activeView === 'long_reads'
					? 'glow-active text-[var(--color-accent)]'
					: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'}"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
			</svg>
			Long Reads
		</button>

		<div class="my-2.5 mx-2 border-t border-[var(--color-border)]"></div>

		<!-- Categorized feeds -->
		{#each feedsByCategory.grouped as { category, feeds: catFeeds }}
			{@const catUnread = totalUnread(catFeeds)}
			{@const isCollapsed = collapsedCategories.has(category.id)}
			{@const isActive = activeView === 'category' && activeCategory === category.id}
			{@const isDragOver = dragOverCategoryId === category.id}
			<div
				role="group"
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
						oncontextmenu={(e) => handleCategoryContextMenu(e, category)}
						class="flex items-center justify-between flex-1 min-w-0 px-2 py-1.5 text-left transition-all rounded-lg cursor-pointer
							{isActive ? 'glow-active' : 'hover:bg-[var(--color-elevated)]'}"
					>
						<span class="flex items-center gap-2 truncate">
							<svg class="w-3.5 h-3.5 flex-shrink-0 {isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-tertiary)]'}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
							</svg>
							<span class="text-xs font-semibold uppercase tracking-wider truncate
								{isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-tertiary)]'}" title={category.name}>
								{category.name}
							</span>
						</span>
						{#if !$settings.calmMode && catUnread > 0}
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
										<img src={feedIcon(feed)} alt="" class="w-4 h-4 rounded-full flex-shrink-0" onerror={(e) => handleFaviconError(e, feed.site_url, feed.url)} />
									{:else}
										<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
											{feed.title?.charAt(0)?.toUpperCase() || '?'}
										</span>
									{/if}
									<span class="truncate" title={feed.title}>{feed.title}</span>
								</span>
								{#if feed.last_error}
									<span
										class="w-2 h-2 rounded-full bg-orange-500 flex-shrink-0"
										title={feed.last_error}
									></span>
								{/if}
								{#if !$settings.calmMode && feed.unread_count > 0}
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
				role="group"
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
								<img src={feedIcon(feed)} alt="" class="w-4 h-4 rounded-full flex-shrink-0" onerror={(e) => handleFaviconError(e, feed.site_url, feed.url)} />
							{:else}
								<span class="w-4 h-4 rounded-full accent-gradient text-[8px] text-white flex items-center justify-center flex-shrink-0 font-bold">
									{feed.title?.charAt(0)?.toUpperCase() || '?'}
								</span>
							{/if}
							<span class="truncate" title={feed.title}>{feed.title}</span>
						</span>
						{#if feed.last_error}
							<span
								class="w-2 h-2 rounded-full bg-orange-500 flex-shrink-0"
								title={feed.last_error}
							></span>
						{/if}
						{#if !$settings.calmMode && feed.unread_count > 0}
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
		<!-- Settings -->
		<div class="relative">
			<button
				onclick={(e) => { e.stopPropagation(); showSettings = !showSettings; }}
				class="flex items-center gap-2 w-full px-3 py-2 text-sm rounded-lg transition-colors
					{showSettings
						? 'bg-[var(--color-elevated)] text-[var(--color-text-primary)]'
						: 'text-[var(--color-text-tertiary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-secondary)]'}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				Settings
			</button>

			{#if showSettings}
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="absolute bottom-full left-0 mb-1 w-full p-3 rounded-xl shadow-2xl border border-[var(--color-border-hover)] fade-in-up z-50"
					style="background: var(--color-card); animation-duration: 100ms;"
					onclick={(e) => e.stopPropagation()}
				>
					<div class="space-y-3">
						<label class="flex items-center justify-between cursor-pointer">
							<span class="text-xs text-[var(--color-text-secondary)]">Calm Mode</span>
							<button
								onclick={() => settings.setCalmMode(!$settings.calmMode)}
								class="relative w-9 h-5 rounded-full transition-colors duration-200
									{$settings.calmMode ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]'}"
								role="switch"
								aria-checked={$settings.calmMode}
								aria-label="Toggle calm mode"
							>
								<span
									class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform duration-200
										{$settings.calmMode ? 'translate-x-4' : 'translate-x-0'}"
								></span>
							</button>
						</label>
						<label class="flex items-center justify-between cursor-pointer">
							<span class="text-xs text-[var(--color-text-secondary)]">Auto-mark read</span>
							<button
								onclick={() => settings.setAutoMarkReadOnScroll(!$settings.autoMarkReadOnScroll)}
								class="relative w-9 h-5 rounded-full transition-colors duration-200
									{$settings.autoMarkReadOnScroll ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]'}"
								role="switch"
								aria-checked={$settings.autoMarkReadOnScroll}
								aria-label="Toggle auto-mark read on scroll"
							>
								<span
									class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform duration-200
										{$settings.autoMarkReadOnScroll ? 'translate-x-4' : 'translate-x-0'}"
								></span>
							</button>
						</label>
						<label class="flex items-center justify-between cursor-pointer">
							<span class="text-xs text-[var(--color-text-secondary)]">Infinite scroll</span>
							<button
								onclick={() => settings.setInfiniteScroll(!$settings.infiniteScroll)}
								class="relative w-9 h-5 rounded-full transition-colors duration-200
									{$settings.infiniteScroll ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]'}"
								role="switch"
								aria-checked={$settings.infiniteScroll}
								aria-label="Toggle infinite scroll"
							>
								<span
									class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform duration-200
										{$settings.infiniteScroll ? 'translate-x-4' : 'translate-x-0'}"
								></span>
							</button>
						</label>
					</div>
				</div>
			{/if}
		</div>

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
			onclick={() => { if (contextMenu) { onMarkFeedRead(contextMenu.feed.id); closeContextMenu(); } }}
			class="flex items-center gap-2.5 w-full px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			Mark All as Read
		</button>
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

<!-- Category context menu -->
{#if categoryContextMenu}
	<div
		class="fixed z-[100] py-1.5 rounded-xl shadow-2xl min-w-[160px] glass border border-[var(--color-border-hover)] fade-in-up"
		style="left: {categoryContextMenu.x}px; top: {categoryContextMenu.y}px; animation-duration: 100ms;"
	>
		<button
			onclick={() => { if (categoryContextMenu) { onMarkCategoryRead(categoryContextMenu.category.id); closeContextMenu(); } }}
			class="flex items-center gap-2.5 w-full px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			Mark All as Read
		</button>
	</div>
{/if}

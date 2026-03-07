<script lang="ts">
	import { feeds, categories } from '$lib/stores/feeds';
	import { articles } from '$lib/stores/articles';
	import { settings } from '$lib/stores/settings';
	import { api, getAccessToken } from '$lib/api/client';

	interface CommandItem {
		id: string;
		label: string;
		category: 'navigate' | 'action' | 'article' | 'view';
		icon: string;
		shortcut?: string;
		action: () => void;
	}

	let {
		open = $bindable(false),
		onSelectFeed,
		onSelectCategory,
		onSelectAll,
		onSelectStarred,
		onSelectUnread,
		onRefresh,
		onSetViewMode,
		onSetSort,
		onAddFeed,
		onOpenArticle,
		onShowShortcuts,
		onMarkAllRead,
		onOpenExternal,
	}: {
		open: boolean;
		onSelectFeed?: (id: number) => void;
		onSelectCategory?: (id: number) => void;
		onSelectAll?: () => void;
		onSelectStarred?: () => void;
		onSelectUnread?: () => void;
		onRefresh?: () => void;
		onSetViewMode?: (mode: string) => void;
		onSetSort?: (sort: string) => void;
		onAddFeed?: () => void;
		onOpenArticle?: (id: number) => void;
		onShowShortcuts?: () => void;
		onMarkAllRead?: () => void;
		onOpenExternal?: () => void;
	} = $props();

	let query = $state('');
	let selectedIdx = $state(0);
	let inputEl = $state<HTMLInputElement | null>(null);

	let commands = $derived.by(() => {
		const items: CommandItem[] = [];

		// ── Navigation ──
		items.push({
			id: 'nav-all',
			label: 'All Articles',
			category: 'navigate',
			icon: '📰',
			action: () => { onSelectAll?.(); close(); }
		});

		items.push({
			id: 'nav-unread',
			label: 'Unread Articles',
			category: 'navigate',
			icon: '🔵',
			action: () => { onSelectUnread?.(); close(); }
		});

		items.push({
			id: 'nav-starred',
			label: 'Starred Articles',
			category: 'navigate',
			icon: '⭐',
			action: () => { onSelectStarred?.(); close(); }
		});

		for (const cat of $categories) {
			items.push({
				id: `nav-cat-${cat.id}`,
				label: `Category: ${cat.name}`,
				category: 'navigate',
				icon: '📁',
				action: () => { onSelectCategory?.(cat.id); close(); }
			});
		}

		for (const feed of $feeds) {
			items.push({
				id: `nav-feed-${feed.id}`,
				label: feed.title,
				category: 'navigate',
				icon: '🔗',
				action: () => { onSelectFeed?.(feed.id); close(); }
			});
		}

		// ── View switching ──
		items.push({
			id: 'view-hybrid',
			label: 'Hybrid (Magazine) View',
			category: 'view',
			icon: '📐',
			shortcut: '1',
			action: () => { onSetViewMode?.('hybrid'); close(); }
		});

		items.push({
			id: 'view-cards',
			label: 'Card Grid View',
			category: 'view',
			icon: '🃏',
			shortcut: '2',
			action: () => { onSetViewMode?.('cards'); close(); }
		});

		items.push({
			id: 'view-list',
			label: 'Compact List View',
			category: 'view',
			icon: '📋',
			shortcut: '3',
			action: () => { onSetViewMode?.('list'); close(); }
		});

		// ── Sort switching ──
		items.push({
			id: 'sort-smart',
			label: 'Sort: Smart',
			category: 'view',
			icon: '🧠',
			action: () => { onSetSort?.('smart'); close(); }
		});

		items.push({
			id: 'sort-newest',
			label: 'Sort: Newest First',
			category: 'view',
			icon: '🕐',
			action: () => { onSetSort?.('newest'); close(); }
		});

		items.push({
			id: 'sort-oldest',
			label: 'Sort: Oldest First',
			category: 'view',
			icon: '🕰️',
			action: () => { onSetSort?.('oldest'); close(); }
		});

		// ── Actions ──
		items.push({
			id: 'action-refresh',
			label: 'Refresh Feeds',
			category: 'action',
			icon: '🔄',
			shortcut: 'r',
			action: () => { onRefresh?.(); close(); }
		});

		items.push({
			id: 'action-mark-all-read',
			label: 'Mark All as Read',
			category: 'action',
			icon: '✅',
			action: () => { onMarkAllRead?.(); close(); }
		});

		items.push({
			id: 'action-add-feed',
			label: 'Add Feed',
			category: 'action',
			icon: '➕',
			action: () => { onAddFeed?.(); close(); }
		});

		items.push({
			id: 'action-theme',
			label: 'Toggle Dark Mode',
			category: 'action',
			icon: '🎨',
			action: () => {
				const current = document.documentElement.classList.contains('dark') ? 'light' : 'dark';
				settings.setTheme(current);
				close();
			}
		});

		items.push({
			id: 'action-shortcuts',
			label: 'Keyboard Shortcuts',
			category: 'action',
			icon: '⌨️',
			shortcut: '?',
			action: () => { onShowShortcuts?.(); close(); }
		});

		items.push({
			id: 'action-open-external',
			label: 'Open Article in Browser',
			category: 'action',
			icon: '🌐',
			action: () => { onOpenExternal?.(); close(); }
		});

		items.push({
			id: 'action-opml-import',
			label: 'Import OPML',
			category: 'action',
			icon: '📥',
			action: () => { triggerOpmlImport(); close(); }
		});

		items.push({
			id: 'action-opml-export',
			label: 'Export OPML',
			category: 'action',
			icon: '📤',
			action: () => { triggerOpmlExport(); close(); }
		});

		// ── Article search (only when query is typed) ──
		if (query.trim().length >= 2) {
			const q = query.toLowerCase();
			for (const article of $articles.articles) {
				if (article.title.toLowerCase().includes(q)) {
					items.push({
						id: `article-${article.id}`,
						label: article.title,
						category: 'article',
						icon: '📄',
						action: () => { onOpenArticle?.(article.id); close(); }
					});
				}
			}
		}

		if (!query.trim()) return items;
		const q = query.toLowerCase();
		return items.filter((item) =>
			item.label.toLowerCase().includes(q) || item.category.toLowerCase().includes(q)
		);
	});

	$effect(() => {
		if (open) {
			query = '';
			selectedIdx = 0;
			requestAnimationFrame(() => inputEl?.focus());
		}
	});

	$effect(() => {
		commands; // subscribe
		if (selectedIdx >= commands.length) {
			selectedIdx = Math.max(0, commands.length - 1);
		}
	});

	function close() {
		open = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			close();
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIdx = (selectedIdx + 1) % Math.max(1, commands.length);
			scrollSelectedIntoView();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIdx = (selectedIdx - 1 + commands.length) % Math.max(1, commands.length);
			scrollSelectedIntoView();
		} else if (e.key === 'Enter') {
			e.preventDefault();
			commands[selectedIdx]?.action();
		}
	}

	function scrollSelectedIntoView() {
		requestAnimationFrame(() => {
			const el = document.querySelector(`[data-cmd-index="${selectedIdx}"]`);
			el?.scrollIntoView({ block: 'nearest' });
		});
	}

	const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8082';

	function triggerOpmlImport() {
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = '.opml,.xml';
		input.onchange = async () => {
			const file = input.files?.[0];
			if (!file) return;
			if (file.size > 5 * 1024 * 1024) {
				alert('OPML file too large (max 5MB)');
				return;
			}
			const formData = new FormData();
			formData.append('file', file);
			try {
				const token = getAccessToken();
				const res = await fetch(`${API_BASE}/api/opml/import`, {
					method: 'POST',
					headers: token ? { 'Authorization': `Bearer ${token}` } : {},
					body: formData,
				});
				if (!res.ok) {
					const data = await res.json().catch(() => ({ error: 'Import failed' }));
					alert(data.error || 'OPML import failed');
					return;
				}
				onRefresh?.();
			} catch (err) {
				console.error('OPML import failed:', err);
				alert('OPML import failed');
			}
		};
		input.click();
	}

	async function triggerOpmlExport() {
		try {
			const token = getAccessToken();
			const res = await fetch(`${API_BASE}/api/opml/export`, {
				headers: token ? { 'Authorization': `Bearer ${token}` } : {},
			});
			if (!res.ok) {
				alert('OPML export failed');
				return;
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'feednest-feeds.opml';
			a.click();
			setTimeout(() => URL.revokeObjectURL(url), 10000);
		} catch (err) {
			console.error('OPML export failed:', err);
			alert('OPML export failed');
		}
	}

	const categoryLabels: Record<string, string> = {
		navigate: 'Navigation',
		view: 'View',
		action: 'Action',
		article: 'Article',
	};
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-[100] flex items-start justify-center pt-[20vh]"
		onkeydown={handleKeydown}
	>
		<div
			class="absolute inset-0 bg-black/50 backdrop-blur-sm"
			onclick={close}
			role="presentation"
		></div>

		<div
			class="relative w-full max-w-lg mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl overflow-hidden fade-in-up"
			style="animation-duration: var(--duration-snappy);"
		>
			<div class="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)]">
				<svg class="w-5 h-5 text-[var(--color-text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<input
					bind:this={inputEl}
					bind:value={query}
					type="text"
					placeholder="Type a command or search articles..."
					class="flex-1 bg-transparent text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] outline-none text-base"
				/>
				<kbd class="px-2 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]">Esc</kbd>
			</div>

			<div class="max-h-80 overflow-y-auto py-2">
				{#if commands.length === 0}
					<div class="px-4 py-8 text-center text-[var(--color-text-tertiary)]">
						No results found
					</div>
				{:else}
					{#each commands as item, i}
						{#if i === 0 || commands[i - 1].category !== item.category}
							<div class="px-4 pt-3 pb-1 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">
								{categoryLabels[item.category] || item.category}
							</div>
						{/if}
						<button
							data-cmd-index={i}
							class="w-full flex items-center gap-3 px-4 py-2 text-left transition-colors
								{i === selectedIdx
									? 'bg-[var(--color-accent-glow)] text-[var(--color-accent)]'
									: 'text-[var(--color-text-primary)] hover:bg-[var(--color-elevated)]'}"
							onclick={() => item.action()}
							onmouseenter={() => (selectedIdx = i)}
						>
							<span class="text-base w-6 text-center">{item.icon}</span>
							<span class="flex-1 text-sm font-medium truncate">{item.label}</span>
							{#if item.shortcut}
								<kbd class="px-1.5 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]">{item.shortcut}</kbd>
							{/if}
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</div>
{/if}

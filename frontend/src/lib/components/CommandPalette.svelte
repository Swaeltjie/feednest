<script lang="ts">
	import { feeds, categories } from '$lib/stores/feeds';
	import { settings } from '$lib/stores/settings';

	interface CommandItem {
		id: string;
		label: string;
		category: 'navigate' | 'action';
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
		onRefresh
	}: {
		open: boolean;
		onSelectFeed?: (id: number) => void;
		onSelectCategory?: (id: number) => void;
		onSelectAll?: () => void;
		onSelectStarred?: () => void;
		onRefresh?: () => void;
	} = $props();

	let query = $state('');
	let selectedIdx = $state(0);
	let inputEl = $state<HTMLInputElement | null>(null);

	let commands = $derived.by(() => {
		const items: CommandItem[] = [];

		items.push({
			id: 'nav-all',
			label: 'All Articles',
			category: 'navigate',
			icon: '\u{1F4F0}',
			action: () => { onSelectAll?.(); close(); }
		});

		items.push({
			id: 'nav-starred',
			label: 'Starred Articles',
			category: 'navigate',
			icon: '\u2B50',
			shortcut: '',
			action: () => { onSelectStarred?.(); close(); }
		});

		for (const cat of $categories) {
			items.push({
				id: `nav-cat-${cat.id}`,
				label: `Category: ${cat.name}`,
				category: 'navigate',
				icon: '\u{1F4C1}',
				action: () => { onSelectCategory?.(cat.id); close(); }
			});
		}

		for (const feed of $feeds) {
			items.push({
				id: `nav-feed-${feed.id}`,
				label: feed.title,
				category: 'navigate',
				icon: '\u{1F517}',
				action: () => { onSelectFeed?.(feed.id); close(); }
			});
		}

		items.push({
			id: 'action-refresh',
			label: 'Refresh Feeds',
			category: 'action',
			icon: '\u{1F504}',
			shortcut: 'r',
			action: () => { onRefresh?.(); close(); }
		});

		items.push({
			id: 'action-theme',
			label: 'Toggle Dark Mode',
			category: 'action',
			icon: '\u{1F3A8}',
			action: () => {
				const current = document.documentElement.classList.contains('dark') ? 'light' : 'dark';
				settings.setTheme(current);
				close();
			}
		});

		if (!query.trim()) return items;
		const q = query.toLowerCase();
		return items.filter((item) => item.label.toLowerCase().includes(q));
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
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIdx = (selectedIdx - 1 + commands.length) % Math.max(1, commands.length);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			commands[selectedIdx]?.action();
		}
	}
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
					placeholder="Type a command or search..."
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
						<button
							class="w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors
								{i === selectedIdx
									? 'bg-[var(--color-accent-glow)] text-[var(--color-accent)]'
									: 'text-[var(--color-text-primary)] hover:bg-[var(--color-elevated)]'}"
							onclick={() => item.action()}
							onmouseenter={() => (selectedIdx = i)}
						>
							<span class="text-lg">{item.icon}</span>
							<span class="flex-1 text-sm font-medium">{item.label}</span>
							{#if item.shortcut}
								<kbd class="px-1.5 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]">{item.shortcut}</kbd>
							{/if}
							<span class="text-xs text-[var(--color-text-tertiary)] capitalize">{item.category}</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</div>
{/if}

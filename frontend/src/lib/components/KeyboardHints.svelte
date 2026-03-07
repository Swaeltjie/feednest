<script lang="ts">
	let { open = $bindable(false) }: { open: boolean } = $props();

	const groups = [
		{
			title: 'Navigation',
			shortcuts: [
				{ keys: ['j'], desc: 'Next article' },
				{ keys: ['k'], desc: 'Previous article' },
				{ keys: ['o', 'Enter'], desc: 'Open article' },
				{ keys: ['Esc'], desc: 'Close / deselect' },
				{ keys: ['g g'], desc: 'Jump to top' },
				{ keys: ['G'], desc: 'Jump to bottom' },
			]
		},
		{
			title: 'Actions',
			shortcuts: [
				{ keys: ['s'], desc: 'Toggle star' },
				{ keys: ['m'], desc: 'Toggle read/unread' },
				{ keys: ['d'], desc: 'Dismiss article' },
				{ keys: ['r'], desc: 'Refresh feeds' },
			]
		},
		{
			title: 'Views & Search',
			shortcuts: [
				{ keys: ['/'], desc: 'Focus search' },
				{ keys: ['1'], desc: 'Hybrid view' },
				{ keys: ['2'], desc: 'Card view' },
				{ keys: ['3'], desc: 'List view' },
				{ keys: ['Cmd+K'], desc: 'Command palette' },
				{ keys: ['?'], desc: 'Toggle this help' },
			]
		},
	];
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-[100] flex items-center justify-center" onkeydown={(e) => { if (e.key === 'Escape') { e.preventDefault(); open = false; } }}>
		<div class="absolute inset-0 bg-black/50 backdrop-blur-sm" onclick={() => (open = false)} role="presentation"></div>
		<div class="relative w-full max-w-md mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl p-6 fade-in-up" style="animation-duration: var(--duration-snappy);">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-bold text-[var(--color-text-primary)]">Keyboard Shortcuts</h2>
				<button onclick={() => (open = false)} class="p-1 rounded-lg hover:bg-[var(--color-elevated)] transition-colors" aria-label="Close keyboard shortcuts">
					<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
			{#each groups as group}
				<h3 class="text-xs font-semibold text-[var(--color-text-tertiary)] uppercase tracking-wider mt-4 mb-2">{group.title}</h3>
				<div class="space-y-1.5">
					{#each group.shortcuts as shortcut}
						<div class="flex items-center justify-between">
							<span class="text-sm text-[var(--color-text-secondary)]">{shortcut.desc}</span>
							<div class="flex gap-1">
								{#each shortcut.keys as key}
									<kbd class="px-2 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)] font-mono">{key}</kbd>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	</div>
{/if}

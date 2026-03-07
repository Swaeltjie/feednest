<script lang="ts">
	import { settings, type Theme } from '$lib/stores/settings';

	const themes: { value: Theme; label: string }[] = [
		{ value: 'light', label: '☀' },
		{ value: 'dark', label: '☾' },
		{ value: 'system', label: '⚙' },
	];
</script>

<div class="flex items-center gap-0.5 p-0.5 rounded-lg" style="background: var(--color-border);">
	{#each themes as theme}
		<button
			onclick={() => settings.setTheme(theme.value)}
			class="px-2 py-1 text-sm rounded-md transition-all
				{$settings.theme === theme.value ? 'bg-[var(--color-card)] shadow-sm' : 'hover:bg-[var(--color-card)]/50'}
				text-[var(--color-text-secondary)]"
			title={theme.value}
		>
			{theme.label}
		</button>
	{/each}
</div>

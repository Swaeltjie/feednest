<script lang="ts">
	import { settings, type Theme } from '$lib/stores/settings';

	const themes: { value: Theme; label: string }[] = [
		{ value: 'light', label: '☀' },
		{ value: 'dark', label: '☾' },
		{ value: 'system', label: '⚙' },
	];
</script>

<div class="flex items-center gap-1 bg-gray-200 dark:bg-gray-700 rounded-lg p-0.5">
	{#each themes as theme}
		<button
			onclick={() => settings.setTheme(theme.value)}
			class="px-2 py-1 text-sm rounded-md transition-colors {$settings.theme === theme.value ? 'bg-white dark:bg-gray-600 shadow-sm' : ''}"
			title={theme.value}
		>
			{theme.label}
		</button>
	{/each}
</div>

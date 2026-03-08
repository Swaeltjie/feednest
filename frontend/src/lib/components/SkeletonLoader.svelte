<script lang="ts">
	let { mode = 'hybrid' }: { mode?: 'cards' | 'list' | 'hybrid' } = $props();
</script>

{#if mode === 'hybrid'}
	<!-- Hybrid: 3 card skeletons at top + 5 list rows below -->
	<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
		{#each [0, 1, 2] as i}
			<div
				class="rounded-2xl overflow-hidden fade-in-up"
				style="animation-delay: {i * 80}ms; min-height: 280px;"
			>
				<div class="skeleton w-full h-full" style="min-height: 280px;"></div>
				<div class="absolute bottom-0 left-0 right-0 p-5 space-y-3">
					<div class="flex items-center gap-2">
						<div class="skeleton w-3.5 h-3.5 rounded-full"></div>
						<div class="skeleton h-3 w-20"></div>
						<div class="skeleton h-3 w-12"></div>
					</div>
					<div class="skeleton h-5 w-3/4"></div>
					<div class="skeleton h-3 w-full opacity-60"></div>
				</div>
			</div>
		{/each}
	</div>
	<div class="rounded-t-2xl mx-2 mt-2" style="background: var(--color-card);">
		{#each Array(5) as _, i}
			<div
				class="flex items-start gap-4 px-4 py-3.5 border-b border-[var(--color-border)] fade-in-up"
				style="animation-delay: {(i + 3) * 60}ms;"
			>
				<div class="skeleton w-16 h-16 rounded-lg flex-shrink-0"></div>
				<div class="flex-1 min-w-0 space-y-2">
					<div class="skeleton h-4 w-3/4"></div>
					<div class="skeleton h-3 w-full"></div>
					<div class="flex items-center gap-2 mt-1.5">
						<div class="skeleton w-3.5 h-3.5 rounded-full"></div>
						<div class="skeleton h-3 w-24"></div>
						<div class="skeleton h-3 w-16"></div>
					</div>
				</div>
			</div>
		{/each}
	</div>
{:else if mode === 'cards'}
	<!-- Cards: 6 card skeletons in responsive grid -->
	<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-4">
		{#each Array(6) as _, i}
			<div
				class="relative rounded-2xl overflow-hidden fade-in-up"
				style="animation-delay: {i * 80}ms; min-height: 280px;"
			>
				<div class="skeleton w-full h-full" style="min-height: 280px;"></div>
				<div class="absolute bottom-0 left-0 right-0 p-5 space-y-3">
					<div class="flex items-center gap-2">
						<div class="skeleton w-3.5 h-3.5 rounded-full"></div>
						<div class="skeleton h-3 w-20"></div>
						<div class="skeleton h-3 w-12"></div>
					</div>
					<div class="skeleton h-5 w-3/4"></div>
					<div class="skeleton h-3 w-full opacity-60"></div>
				</div>
			</div>
		{/each}
	</div>
{:else}
	<!-- List: 8 list row skeletons -->
	<div class="m-2 rounded-2xl overflow-hidden" style="background: var(--color-card);">
		{#each Array(8) as _, i}
			<div
				class="flex items-start gap-4 px-4 py-3.5 border-b border-[var(--color-border)] fade-in-up"
				style="animation-delay: {i * 50}ms; border-left: 3px solid transparent;"
			>
				<div class="skeleton w-16 h-16 rounded-lg flex-shrink-0"></div>
				<div class="flex-1 min-w-0 space-y-2">
					<div class="skeleton h-4 w-3/4"></div>
					<div class="skeleton h-3 w-full"></div>
					<div class="flex items-center gap-2 mt-1.5">
						<div class="skeleton w-3.5 h-3.5 rounded-full"></div>
						<div class="skeleton h-3 w-24"></div>
						<div class="skeleton h-3 w-16"></div>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}

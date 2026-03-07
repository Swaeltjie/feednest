<script lang="ts">
	let { mode = 'hybrid' }: { mode?: 'cards' | 'list' | 'hybrid' } = $props();
</script>

<div class="p-4 space-y-4">
	{#if mode === 'hybrid' || mode === 'cards'}
		<!-- Featured hero skeletons -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each [0, 1] as i}
				<div class="rounded-2xl overflow-hidden" style="animation-delay: {i * 100}ms;">
					<div class="skeleton h-64 w-full"></div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- List row skeletons -->
	<div class="space-y-1">
		{#each Array(6) as _, i}
			<div
				class="flex items-start gap-4 px-4 py-3.5 fade-in-up"
				style="animation-delay: {(i + 2) * 60}ms;"
			>
				<div class="skeleton w-16 h-16 rounded-lg flex-shrink-0"></div>
				<div class="flex-1 space-y-2">
					<div class="skeleton h-4 w-3/4"></div>
					<div class="skeleton h-3 w-full"></div>
					<div class="skeleton h-3 w-1/3"></div>
				</div>
			</div>
		{/each}
	</div>
</div>

<script lang="ts">
	import type { Feed } from '$lib/stores/feeds';
	import { timeAgo } from '$lib/utils/time';

	let {
		feeds = [],
		onRetry = (_id: number) => {},
		onClose = () => {},
	}: {
		feeds?: Feed[];
		onRetry?: (id: number) => void;
		onClose?: () => void;
	} = $props();

	type Health = 'healthy' | 'warning' | 'dead';

	function healthOf(feed: Feed): Health {
		if (feed.consecutive_failures >= 5) return 'dead';
		if (feed.consecutive_failures >= 1) return 'warning';
		return 'healthy';
	}

	let healthyCount = $derived(
		feeds.filter((f) => f.last_fetch_status === 'success' || !f.last_error).length
	);
	let warningCount = $derived(
		feeds.filter((f) => f.consecutive_failures >= 1 && f.consecutive_failures <= 4).length
	);
	let deadCount = $derived(feeds.filter((f) => f.consecutive_failures >= 5).length);

	// Unhealthy feeds first (dead before warning before healthy), then most failures first.
	const order: Record<Health, number> = { dead: 0, warning: 1, healthy: 2 };
	let sortedFeeds = $derived(
		[...feeds].sort((a, b) => {
			const diff = order[healthOf(a)] - order[healthOf(b)];
			if (diff !== 0) return diff;
			return b.consecutive_failures - a.consecutive_failures;
		})
	);
</script>

<div class="glass-card rounded-2xl overflow-hidden shadow-xl">
	<!-- Header -->
	<div class="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border)]">
		<div class="flex items-center gap-2.5">
			<div class="w-7 h-7 rounded-lg accent-gradient flex items-center justify-center flex-shrink-0">
				<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12h4l3 8 4-16 3 8h4" />
				</svg>
			</div>
			<h2 class="text-base font-bold text-[var(--color-text-primary)] tracking-tight">Feed Health</h2>
		</div>
		<button
			onclick={onClose}
			aria-label="Close"
			class="p-1.5 rounded-lg text-[var(--color-text-tertiary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)] transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	</div>

	<!-- Summary -->
	<div class="grid grid-cols-3 gap-3 px-5 py-4 border-b border-[var(--color-border)]">
		<div class="flex flex-col items-center justify-center gap-1 px-3 py-3 rounded-xl bg-[var(--color-elevated)]">
			<span class="flex items-center gap-1.5 text-2xl font-bold text-emerald-500">
				<span class="w-2.5 h-2.5 rounded-full bg-emerald-500"></span>
				{healthyCount}
			</span>
			<span class="text-xs font-medium text-[var(--color-text-tertiary)]">Healthy</span>
		</div>
		<div class="flex flex-col items-center justify-center gap-1 px-3 py-3 rounded-xl bg-[var(--color-elevated)]">
			<span class="flex items-center gap-1.5 text-2xl font-bold text-amber-500">
				<span class="w-2.5 h-2.5 rounded-full bg-amber-500"></span>
				{warningCount}
			</span>
			<span class="text-xs font-medium text-[var(--color-text-tertiary)]">Warning</span>
		</div>
		<div class="flex flex-col items-center justify-center gap-1 px-3 py-3 rounded-xl bg-[var(--color-elevated)]">
			<span class="flex items-center gap-1.5 text-2xl font-bold text-red-500">
				<span class="w-2.5 h-2.5 rounded-full bg-red-500"></span>
				{deadCount}
			</span>
			<span class="text-xs font-medium text-[var(--color-text-tertiary)]">Dead</span>
		</div>
	</div>

	<!-- Feed list -->
	<div class="max-h-[60vh] overflow-y-auto divide-y divide-[var(--color-border)]">
		{#if sortedFeeds.length === 0}
			<div class="px-5 py-10 text-center text-sm text-[var(--color-text-tertiary)]">
				No feeds to report on.
			</div>
		{:else}
			{#each sortedFeeds as feed (feed.id)}
				{@const health = healthOf(feed)}
				<div class="flex items-start gap-3 px-5 py-3.5 hover:bg-[var(--color-elevated)] transition-colors">
					<!-- Feed icon -->
					{#if feed.icon_url}
						<img src={feed.icon_url} alt="" class="w-5 h-5 rounded-full flex-shrink-0 mt-0.5" />
					{:else}
						<span class="w-5 h-5 rounded-full accent-gradient text-[9px] text-white flex items-center justify-center flex-shrink-0 mt-0.5 font-bold">
							{feed.title?.charAt(0)?.toUpperCase() || '?'}
						</span>
					{/if}

					<!-- Content -->
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<h3 class="text-sm font-semibold text-[var(--color-text-primary)] truncate" title={feed.title}>
								{feed.title}
							</h3>
							{#if health === 'healthy'}
								<span class="flex-shrink-0 px-2 py-0.5 text-xs font-semibold rounded-full bg-emerald-500/15 text-emerald-500">
									Healthy
								</span>
							{:else if health === 'warning'}
								<span class="flex-shrink-0 px-2 py-0.5 text-xs font-semibold rounded-full bg-amber-500/15 text-amber-500">
									{feed.consecutive_failures} {feed.consecutive_failures === 1 ? 'failure' : 'failures'}
								</span>
							{:else}
								<span class="flex-shrink-0 px-2 py-0.5 text-xs font-semibold rounded-full bg-red-500/15 text-red-500">
									Dead
								</span>
							{/if}
						</div>

						<div class="mt-0.5 text-xs text-[var(--color-text-tertiary)]">
							Last success: {feed.last_success ? timeAgo(feed.last_success) : '—'}
						</div>

						{#if feed.last_error}
							<p class="mt-1 text-xs text-[var(--color-text-secondary)] truncate" title={feed.last_error}>
								{feed.last_error}
							</p>
						{/if}
					</div>

					<!-- Retry -->
					{#if health !== 'healthy'}
						<button
							onclick={() => onRetry(feed.id)}
							class="flex-shrink-0 flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-lg text-[var(--color-text-secondary)] hover:bg-[var(--color-card)] hover:text-[var(--color-accent)] border border-[var(--color-border)] hover:border-[var(--color-border-hover)] transition-all"
						>
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
							Retry
						</button>
					{/if}
				</div>
			{/each}
		{/if}
	</div>
</div>

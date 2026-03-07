<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { getFaviconUrl } from '$lib/utils/favicon';
	import { getFeedColor } from '$lib/utils/color';
	import { magneticHover } from '$lib/utils/parallax';
	import { blurUp } from '$lib/utils/blurload';
	import { starBurst } from '$lib/utils/particles';

	let {
		article,
		selected = false,
		index = 0,
		onOpen = (_id: number) => {},
	}: { article: Article; selected?: boolean; index?: number; onOpen?: (id: number) => void } = $props();

	function handleClick(e: Event) {
		e.preventDefault();
		onOpen(article.id);
	}

	function handleStar(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		const wasStarred = article.is_starred;
		articles.toggleStar(article.id, !article.is_starred);
		if (!wasStarred) {
			starBurst(e.clientX, e.clientY);
		}
	}

	let starAnimating = $state(false);
	function handleStarWithBounce(e: MouseEvent) {
		handleStar(e);
		starAnimating = true;
		setTimeout(() => (starAnimating = false), 200);
	}

	let feedIcon = $derived(getFaviconUrl(article.feed_icon_url, article.url, undefined));

	let feedAccentColor = $state('');

	$effect(() => {
		getFeedColor(article.feed_icon_url, article.url).then(c => { feedAccentColor = c; });
	});
</script>

<a
	href="/article/{article.id}"
	onclick={handleClick}
	use:magneticHover={{ strength: 5 }}
	class="group relative block rounded-2xl overflow-hidden glass-card fade-in-up"
	style="animation-delay: {index * 60}ms; min-height: 280px; border-bottom: 2px solid {feedAccentColor || 'transparent'};"
	class:ring-2={selected}
	class:ring-blue-500={selected}
>
	<!-- Background image or gradient fallback -->
	{#if article.thumbnail_url}
		<img
			src={article.thumbnail_url}
			alt=""
			class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
			loading="lazy"
			use:blurUp
		/>
	{:else}
		<div class="absolute inset-0 accent-gradient opacity-20"></div>
	{/if}

	<!-- Gradient overlay -->
	<div class="absolute inset-0 hero-overlay"></div>

	<!-- Content positioned at bottom -->
	<div class="relative h-full flex flex-col justify-end p-5">
		<!-- Frosted glass metadata strip -->
		<div
			class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs text-white/80 mb-3 self-start"
			style="background: rgba(255,255,255,0.1); backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);"
		>
			{#if feedIcon}
				<img src={feedIcon} alt="" class="w-3.5 h-3.5 rounded-full" onerror={(e) => (e.currentTarget as HTMLImageElement).style.display = 'none'} />
			{/if}
			<span>{article.feed_title}</span>
			<span class="opacity-50">·</span>
			<span>{timeAgo(article.published_at)}</span>
			{#if article.reading_time > 0}
				<span class="opacity-50">·</span>
				<span>{article.reading_time} min</span>
			{/if}
		</div>

		<!-- Title -->
		<h3 class="text-xl font-bold text-white leading-snug line-clamp-2 drop-shadow-lg">
			{article.title}
		</h3>

		{#if article.snippet}
			<p class="text-sm text-white/60 line-clamp-1 mt-1.5">
				{article.snippet}
			</p>
		{/if}
	</div>

	<!-- Star button (top-right) -->
	<button
		onclick={handleStarWithBounce}
		aria-label={article.is_starred ? 'Unstar' : 'Star'}
		class="absolute top-3 right-3 p-2 rounded-full transition-all {article.is_starred
			? 'text-yellow-400 bg-yellow-400/20'
			: 'text-white/50 hover:text-white bg-black/20 hover:bg-black/40'} {starAnimating ? 'star-bounce' : ''}"
	>
		<svg class="w-5 h-5" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
			<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
		</svg>
	</button>

	<!-- Unread indicator -->
	{#if !article.is_read}
		<div class="absolute top-3 left-3 w-2.5 h-2.5 rounded-full accent-gradient shadow-lg shadow-blue-500/30"></div>
	{/if}
</a>

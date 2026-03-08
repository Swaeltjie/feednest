<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { getFaviconUrl, handleFaviconError } from '$lib/utils/favicon';
	import { getFeedColor } from '$lib/utils/color';
	import { starBurst } from '$lib/utils/particles';
	import { swipeable } from '$lib/utils/swipe';

	let {
		article,
		selected = false,
		index = 0,
		onOpen = (_id: number) => {},
		onToggleRead,
		onToggleStar,
	}: {
		article: Article;
		selected?: boolean;
		index?: number;
		onOpen?: (id: number) => void;
		onToggleRead?: (id: number, isRead: boolean) => void;
		onToggleStar?: (id: number, isStarred: boolean) => void;
	} = $props();

	let starAnimating = $state(false);

	function handleClick() {
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
		starAnimating = true;
		setTimeout(() => (starAnimating = false), 200);
	}

	let feedIcon = $derived(getFaviconUrl(article.feed_icon_url, article.url, undefined));

	let feedAccentColor = $state('');

	$effect(() => {
		getFeedColor(article.feed_icon_url, article.url).then(c => { feedAccentColor = c; });
	});
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	onclick={handleClick}
	use:swipeable={{
		onSwipeRight: () => onToggleRead?.(article.id, !article.is_read),
		onSwipeLeft: () => onToggleStar?.(article.id, !article.is_starred),
		threshold: 80
	}}
	class="group flex items-start gap-4 px-4 py-3.5 transition-all duration-200 spring-in
		border-b border-[var(--color-border)]
		hover:bg-[var(--color-elevated)] hover:shadow-md
		transition-opacity duration-400 cursor-pointer
		{article.is_read ? 'opacity-60 hover:opacity-90' : 'opacity-100'}
		{selected ? 'bg-[var(--color-accent-glow)] ring-1 ring-inset ring-[var(--color-accent)]/30' : ''}"
	style="view-transition-name: article-{article.id}; animation-delay: {index * 30}ms; border-left: 3px solid {feedAccentColor || 'transparent'};"
>
	<!-- Thumbnail -->
	{#if article.thumbnail_url}
		<div class="flex-shrink-0 w-16 h-16 rounded-lg overflow-hidden bg-[var(--color-border)]">
			<img
				src={article.thumbnail_url}
				alt=""
				class="w-full h-full object-cover group-hover:scale-110 transition-transform duration-300"
				loading="lazy"
			/>
		</div>
	{:else}
		<div class="flex-shrink-0 w-16 h-16 rounded-lg accent-gradient opacity-15 flex items-center justify-center">
			<span class="text-2xl font-bold text-[var(--color-text-primary)] opacity-40">
				{article.feed_title?.charAt(0)?.toUpperCase() || '?'}
			</span>
		</div>
	{/if}

	<!-- Content -->
	<div class="flex-1 min-w-0">
		<h3 class="text-sm font-semibold text-[var(--color-text-primary)] leading-snug line-clamp-1 group-hover:text-[var(--color-accent)] transition-colors">
			{article.title}
		</h3>

		{#if article.snippet}
			<p class="text-xs text-[var(--color-text-secondary)] line-clamp-2 mt-0.5 leading-relaxed">
				{article.snippet}
			</p>
		{/if}

		<div class="flex items-center gap-2 mt-1.5 text-xs text-[var(--color-text-tertiary)]">
			{#if feedIcon}
				<img src={feedIcon} alt="" class="w-3.5 h-3.5 rounded-full" onerror={(e) => handleFaviconError(e, article.url)} />
			{/if}
			<span class="font-medium text-[var(--color-text-secondary)]">{article.feed_title}</span>
			<span class="opacity-40">·</span>
			<span>{timeAgo(article.published_at)}</span>
			{#if article.reading_time > 0}
				<span class="opacity-40">·</span>
				<span>{article.reading_time} min</span>
			{/if}
		</div>
	</div>

	<!-- Star -->
	<button
		onclick={handleStar}
		aria-label={article.is_starred ? 'Unstar' : 'Star'}
		class="flex-shrink-0 p-1.5 rounded-lg transition-all
			{article.is_starred
				? 'text-yellow-400 hover:text-yellow-300'
				: 'text-[var(--color-text-tertiary)] hover:text-yellow-400 opacity-0 group-hover:opacity-100'}
			{starAnimating ? 'star-bounce' : ''}"
	>
		<svg class="w-4 h-4" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
			<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
		</svg>
	</button>
</div>

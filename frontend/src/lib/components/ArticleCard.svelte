<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';

	let { article }: { article: Article } = $props();

	function handleStar(e: Event) {
		e.preventDefault();
		e.stopPropagation();
		articles.toggleStar(article.id, !article.is_starred);
	}
</script>

<a
	href="/article/{article.id}"
	class="block bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-lg transition-shadow group {article.is_read ? 'opacity-60' : ''}"
>
	{#if article.thumbnail_url}
		<div class="aspect-video bg-gray-100 dark:bg-gray-700 overflow-hidden">
			<img
				src={article.thumbnail_url}
				alt=""
				class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
				loading="lazy"
			/>
		</div>
	{/if}

	<div class="p-4">
		<h3 class="font-semibold text-gray-900 dark:text-white leading-snug line-clamp-2 mb-2">
			{article.title}
		</h3>

		<div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
			<div class="flex items-center gap-1.5 min-w-0">
				<span class="truncate">{article.feed_title}</span>
				<span>·</span>
				<span class="whitespace-nowrap">{timeAgo(article.published_at)}</span>
				{#if article.reading_time > 0}
					<span>·</span>
					<span class="whitespace-nowrap">{article.reading_time} min</span>
				{/if}
			</div>

			<button
				onclick={handleStar}
				class="ml-2 p-1 hover:text-yellow-500 transition-colors flex-shrink-0 {article.is_starred ? 'text-yellow-500' : ''}"
			>
				{article.is_starred ? '★' : '☆'}
			</button>
		</div>
	</div>
</a>

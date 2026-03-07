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
	class="flex items-center gap-3 px-4 py-3 border-b border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors {article.is_read ? 'opacity-60' : ''}"
>
	<div class="w-2 h-2 rounded-full flex-shrink-0 {article.is_read ? 'bg-transparent' : 'bg-blue-500'}"></div>

	<div class="flex-1 min-w-0">
		<h3 class="text-sm text-gray-900 dark:text-white truncate {article.is_read ? 'font-normal' : 'font-medium'}">
			{article.title}
		</h3>
	</div>

	<span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap flex-shrink-0">
		{article.feed_title}
	</span>

	<span class="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap flex-shrink-0 w-16 text-right">
		{timeAgo(article.published_at)}
	</span>

	<button
		onclick={handleStar}
		class="p-1 hover:text-yellow-500 transition-colors flex-shrink-0 {article.is_starred ? 'text-yellow-500' : 'text-gray-400'}"
	>
		{article.is_starred ? '★' : '☆'}
	</button>
</a>

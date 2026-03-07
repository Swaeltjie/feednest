<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { articles, type Article } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';

	let article: Article | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let startTime = Date.now();

	onMount(async () => {
		const id = Number(page.params.id);
		try {
			article = await articles.getArticle(id);
			if (article && !article.is_read) {
				await articles.toggleRead(article.id, true);
				article.is_read = true;
			}
			api.post('/api/events', { article_id: id, event_type: 'click', duration_seconds: 0 });
		} catch (e) {
			error = 'Article not found';
		} finally {
			loading = false;
		}
	});

	function trackReadTime() {
		if (article) {
			const duration = Math.floor((Date.now() - startTime) / 1000);
			if (duration > 5) {
				api.post('/api/events', {
					article_id: article.id,
					event_type: 'read',
					duration_seconds: duration,
				});
			}
		}
	}

	function handleStar() {
		if (article) {
			article.is_starred = !article.is_starred;
			articles.toggleStar(article.id, article.is_starred);
		}
	}

	function handleBack() {
		trackReadTime();
		goto('/');
	}

	function formatDate(dateStr: string | null): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric',
		});
	}
</script>

<svelte:head>
	<title>{article ? article.title : 'Loading...'} - FeedNest</title>
</svelte:head>

<svelte:window onbeforeunload={trackReadTime} />

{#if loading}
	<div class="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900">
		<div class="flex flex-col items-center gap-3">
			<div
				class="w-8 h-8 border-3 border-blue-500 border-t-transparent rounded-full animate-spin"
			></div>
			<span class="text-sm text-gray-500 dark:text-gray-400">Loading article...</span>
		</div>
	</div>
{:else if error}
	<div class="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900">
		<div class="flex flex-col items-center gap-4 text-center px-4">
			<svg
				class="w-16 h-16 text-gray-300 dark:text-gray-600"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
				/>
			</svg>
			<h2 class="text-lg font-medium text-gray-900 dark:text-white">{error}</h2>
			<button
				onclick={handleBack}
				class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
			>
				Back to Feed
			</button>
		</div>
	</div>
{:else if article}
	<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
		<!-- Sticky header -->
		<header
			class="sticky top-0 z-30 bg-white/80 dark:bg-gray-800/80 backdrop-blur-lg border-b border-gray-200 dark:border-gray-700"
		>
			<div class="max-w-3xl mx-auto flex items-center justify-between px-4 py-3">
				<button
					onclick={handleBack}
					class="flex items-center gap-2 text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors group"
				>
					<svg
						class="w-5 h-5 transition-transform group-hover:-translate-x-0.5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M15 19l-7-7 7-7"
						/>
					</svg>
					Back
				</button>

				<div class="flex items-center gap-2">
					<!-- Star button -->
					<button
						onclick={handleStar}
						class="p-2 rounded-lg transition-colors {article.is_starred
							? 'text-yellow-500 hover:text-yellow-600 bg-yellow-50 dark:bg-yellow-900/20'
							: 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
						title={article.is_starred ? 'Unstar article' : 'Star article'}
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
							/>
						</svg>
					</button>

					<!-- Open original -->
					<a
						href={article.url}
						target="_blank"
						rel="noopener noreferrer"
						class="flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						title="Open original article"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
							/>
						</svg>
						<span class="hidden sm:inline">Original</span>
					</a>
				</div>
			</div>
		</header>

		<!-- Article content -->
		<article class="max-w-[680px] mx-auto px-4 sm:px-6 py-8 sm:py-12">
			<!-- Title -->
			<h1
				class="text-3xl md:text-4xl font-bold text-gray-900 dark:text-white leading-tight tracking-tight"
			>
				{article.title}
			</h1>

			<!-- Metadata -->
			<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-4 mb-8 text-sm text-gray-500 dark:text-gray-400">
				{#if article.feed_title}
					<span class="flex items-center gap-1.5">
						{#if article.feed_icon_url}
							<img
								src={article.feed_icon_url}
								alt=""
								class="w-4 h-4 rounded"
								loading="lazy"
							/>
						{/if}
						<span class="font-medium text-gray-700 dark:text-gray-300">{article.feed_title}</span>
					</span>
					<span class="text-gray-300 dark:text-gray-600">|</span>
				{/if}

				{#if article.author}
					<span>{article.author}</span>
					<span class="text-gray-300 dark:text-gray-600">|</span>
				{/if}

				{#if article.published_at}
					<span title={formatDate(article.published_at)}>
						{timeAgo(article.published_at)}
					</span>
					<span class="text-gray-300 dark:text-gray-600">|</span>
				{/if}

				{#if article.reading_time}
					<span>{article.reading_time} min read</span>
				{/if}
			</div>

			<!-- Tags -->
			{#if article.tags && article.tags.length > 0}
				<div class="flex flex-wrap gap-2 mb-8">
					{#each article.tags as tag}
						<span
							class="px-2.5 py-0.5 text-xs font-medium bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full"
						>
							{tag}
						</span>
					{/each}
				</div>
			{/if}

			<!-- Article body -->
			{#if article.content_clean || article.content_raw}
				<div
					class="prose prose-lg dark:prose-invert
						prose-headings:font-bold prose-headings:tracking-tight
						prose-a:text-blue-600 dark:prose-a:text-blue-400 prose-a:no-underline hover:prose-a:underline
						prose-img:rounded-lg prose-img:shadow-md
						prose-pre:bg-gray-100 dark:prose-pre:bg-gray-800
						prose-blockquote:border-l-blue-500 prose-blockquote:bg-blue-50/50 dark:prose-blockquote:bg-blue-900/10 prose-blockquote:py-1 prose-blockquote:px-4 prose-blockquote:rounded-r-lg
						max-w-none"
				>
					{@html article.content_clean || article.content_raw}
				</div>
			{:else}
				<div
					class="flex flex-col items-center justify-center py-16 text-center"
				>
					<svg
						class="w-12 h-12 text-gray-300 dark:text-gray-600 mb-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.5"
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
						/>
					</svg>
					<p class="text-gray-500 dark:text-gray-400 mb-4">No content available for this article.</p>
					<a
						href={article.url}
						target="_blank"
						rel="noopener noreferrer"
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
					>
						Read on Original Site
					</a>
				</div>
			{/if}

			<!-- Bottom actions -->
			<div
				class="flex items-center justify-between mt-12 pt-8 border-t border-gray-200 dark:border-gray-700"
			>
				<button
					onclick={handleBack}
					class="flex items-center gap-2 text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M15 19l-7-7 7-7"
						/>
					</svg>
					Back to Feed
				</button>

				<a
					href={article.url}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-1.5 text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
				>
					Open Original
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
						/>
					</svg>
				</a>
			</div>
		</article>
	</div>
{/if}

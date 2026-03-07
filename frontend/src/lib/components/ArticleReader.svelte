<script lang="ts">
	import { articles, type Article } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { getFaviconUrl } from '$lib/utils/favicon';
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';
	import DOMPurify from 'isomorphic-dompurify';

	let {
		articleId,
		onClose = () => {},
	}: {
		articleId: number;
		onClose?: () => void;
	} = $props();

	let article: Article | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let startTime = Date.now();
	let starAnimating = $state(false);
	let visible = $state(false);

	onMount(async () => {
		// Trigger slide-in animation
		requestAnimationFrame(() => { visible = true; });

		try {
			article = await articles.getArticle(articleId);
			if (article && !article.is_read) {
				await articles.toggleRead(article.id, true);
				article.is_read = true;
			}
			api.post('/api/events', { article_id: articleId, event_type: 'click', duration_seconds: 0 });
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
			starAnimating = true;
			setTimeout(() => (starAnimating = false), 200);
		}
	}

	function handleClose() {
		trackReadTime();
		visible = false;
		setTimeout(onClose, 300);
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

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- Backdrop -->
<div
	class="fixed inset-0 z-40 transition-opacity duration-300 {visible ? 'bg-black/40 backdrop-blur-sm' : 'bg-transparent pointer-events-none'}"
	role="button"
	tabindex="-1"
	onclick={handleClose}
	onkeydown={(e) => e.key === 'Enter' && handleClose()}
></div>

<!-- Reader panel -->
<div
	class="fixed inset-y-0 right-0 z-50 w-full max-w-2xl flex flex-col transition-transform duration-300 ease-out
		{visible ? 'translate-x-0' : 'translate-x-full'}"
	style="background: var(--color-surface);"
>
	{#if loading}
		<div class="flex-1 flex items-center justify-center">
			<div class="flex flex-col items-center gap-4 fade-in-up px-8 w-full max-w-md">
				<div class="skeleton w-full h-8 rounded-lg"></div>
				<div class="skeleton w-48 h-4 rounded-lg"></div>
				<div class="space-y-3 w-full mt-8">
					<div class="skeleton h-4 w-full"></div>
					<div class="skeleton h-4 w-5/6"></div>
					<div class="skeleton h-4 w-4/5"></div>
					<div class="skeleton h-4 w-full"></div>
					<div class="skeleton h-4 w-3/4"></div>
				</div>
			</div>
		</div>
	{:else if error}
		<div class="flex-1 flex items-center justify-center">
			<div class="flex flex-col items-center gap-4 text-center px-4 fade-in-up">
				<div class="w-16 h-16 rounded-2xl accent-gradient opacity-10 flex items-center justify-center">
					<svg class="w-8 h-8 text-[var(--color-text-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z" />
					</svg>
				</div>
				<h2 class="text-lg font-medium text-[var(--color-text-primary)]">{error}</h2>
				<button onclick={handleClose} class="px-5 py-2.5 text-sm font-medium text-white rounded-xl accent-gradient hover:opacity-90 transition-opacity">
					Close
				</button>
			</div>
		</div>
	{:else if article}
		<!-- Sticky header -->
		<header class="flex-shrink-0 glass border-b border-[var(--color-border)]">
			<div class="flex items-center justify-between px-5 py-3">
				<button
					onclick={handleClose}
					class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors group"
				>
					<svg class="w-5 h-5 transition-transform group-hover:-translate-x-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
					Close
				</button>

				<div class="flex items-center gap-1.5">
					<button
						onclick={handleStar}
						class="p-2 rounded-lg transition-all {starAnimating ? 'star-bounce' : ''}
							{article.is_starred
								? 'text-yellow-500 bg-yellow-500/10'
								: 'text-[var(--color-text-tertiary)] hover:text-yellow-500 hover:bg-[var(--color-elevated)]'}"
						title={article.is_starred ? 'Unstar' : 'Star'}
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
							<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
						</svg>
					</button>

					<a
						href={article.url}
						target="_blank"
						rel="noopener noreferrer"
						class="flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-[var(--color-text-secondary)]
							hover:text-[var(--color-accent)] hover:bg-[var(--color-elevated)] rounded-lg transition-colors"
						title="Open original"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
						</svg>
						<span class="hidden sm:inline">Original</span>
					</a>
				</div>
			</div>
		</header>

		<!-- Scrollable content -->
		<div class="flex-1 overflow-y-auto">
			<!-- Hero image -->
			{#if article.thumbnail_url}
				<div class="relative w-full h-56 overflow-hidden">
					<img src={article.thumbnail_url} alt="" class="w-full h-full object-cover" />
					<div class="absolute inset-0 hero-overlay"></div>
				</div>
			{/if}

			<article class="px-6 sm:px-8 py-6 sm:py-8">
				<!-- Title -->
				<h1 class="text-2xl sm:text-3xl font-bold text-[var(--color-text-primary)] leading-tight" style="letter-spacing: -0.02em;">
					{article.title}
				</h1>

				<!-- Metadata -->
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-4 mb-6 text-sm text-[var(--color-text-secondary)]">
					{#if article.feed_title}
						<span class="flex items-center gap-1.5">
							{#if getFaviconUrl(article.feed_icon_url, article.url, undefined)}
								<img src={getFaviconUrl(article.feed_icon_url, article.url, undefined)} alt="" class="w-4 h-4 rounded" loading="lazy" onerror={(e) => (e.currentTarget as HTMLImageElement).style.display = 'none'} />
							{/if}
							<span class="font-medium text-[var(--color-text-primary)]">{article.feed_title}</span>
						</span>
						<span class="text-[var(--color-text-tertiary)]">·</span>
					{/if}

					{#if article.author}
						<span>{article.author}</span>
						<span class="text-[var(--color-text-tertiary)]">·</span>
					{/if}

					{#if article.published_at}
						<span title={formatDate(article.published_at)}>{timeAgo(article.published_at)}</span>
						<span class="text-[var(--color-text-tertiary)]">·</span>
					{/if}

					{#if article.reading_time}
						<span>{article.reading_time} min read</span>
					{/if}
				</div>

				<!-- Tags -->
				{#if article.tags && article.tags.length > 0}
					<div class="flex flex-wrap gap-2 mb-6">
						{#each article.tags as tag}
							<span class="px-2.5 py-0.5 text-xs font-medium rounded-full bg-[var(--color-accent-glow)] text-[var(--color-accent)]">
								{tag}
							</span>
						{/each}
					</div>
				{/if}

				<!-- Article body -->
				{#if article.content_clean || article.content_raw}
					<div
						class="prose prose-lg dark:prose-invert
							prose-headings:font-bold prose-headings:tracking-tight prose-headings:text-[var(--color-text-primary)]
							prose-a:text-[var(--color-accent)] prose-a:no-underline hover:prose-a:underline
							prose-img:rounded-xl prose-img:shadow-md
							prose-pre:bg-[var(--color-elevated)] prose-pre:border prose-pre:border-[var(--color-border)] prose-pre:rounded-xl
							prose-blockquote:border-l-[var(--color-accent)] prose-blockquote:bg-[var(--color-accent-glow)] prose-blockquote:py-1 prose-blockquote:px-4 prose-blockquote:rounded-r-xl
							prose-p:text-[var(--color-text-primary)] prose-li:text-[var(--color-text-primary)]
							max-w-none"
						style="line-height: 1.75; font-size: 17px;"
					>
						{@html DOMPurify.sanitize(article.content_clean || article.content_raw)}
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-16 text-center">
						<div class="w-16 h-16 rounded-2xl accent-gradient opacity-10 flex items-center justify-center mb-4">
							<svg class="w-8 h-8 text-[var(--color-text-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
						</div>
						<p class="text-[var(--color-text-secondary)] mb-4">No content available for this article.</p>
						<a
							href={article.url}
							target="_blank"
							rel="noopener noreferrer"
							class="px-5 py-2.5 text-sm font-medium text-white rounded-xl accent-gradient hover:opacity-90 transition-opacity shadow-lg shadow-blue-500/25"
						>
							Read on Original Site
						</a>
					</div>
				{/if}
			</article>
		</div>
	{/if}
</div>

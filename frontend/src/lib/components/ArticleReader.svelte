<script lang="ts">
	import { articles, type Article } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { getFaviconUrl, handleFaviconError } from '$lib/utils/favicon';
	import { api } from '$lib/api/client';
	import DOMPurify from 'isomorphic-dompurify';
	import { isSafeUrl } from '$lib/api/client';
	import ReaderSettings from './ReaderSettings.svelte';
	import {
		settings,
		READER_FONT_SIZE_MAP,
		READER_FONT_FAMILY_MAP,
		READER_LINE_HEIGHT_MAP,
		READER_CONTENT_WIDTH_MAP,
	} from '$lib/stores/settings';

	// Harden DOMPurify: add noreferrer to target=_blank links, strip style attrs
	DOMPurify.addHook('afterSanitizeAttributes', (node: Element) => {
		if (node.tagName === 'A') {
			const href = node.getAttribute('href');
			if (href && !isSafeUrl(href)) {
				node.removeAttribute('href');
			}
			if (node.getAttribute('target') === '_blank') {
				node.setAttribute('rel', 'noopener noreferrer');
			}
		}
		node.removeAttribute('style');
	});
	import { blurUp } from '$lib/utils/blurload';

	let {
		articleId,
		onClose = () => {},
		articleIds = [],
		onNavigate,
		inline = false,
		focusMode = false,
		onToggleFocus,
	}: {
		articleId: number;
		onClose?: () => void;
		articleIds?: number[];
		onNavigate?: (id: number) => void;
		inline?: boolean;
		focusMode?: boolean;
		onToggleFocus?: () => void;
	} = $props();

	let article: Article | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let startTime = $state(Date.now());
	let starAnimating = $state(false);
	let readerSettingsOpen = $state(false);

	// Task 8: Reading progress
	let readingProgress = $state(0);
	let readerScrollY = $state(0);
	let lastScrollY = 0;

	// Task 9: Collapsing header
	let readerHeaderCompact = $state(false);

	// Task 10: Article navigation
	let contentEl: HTMLElement | undefined = $state();
	let slideDirection = $state<'left' | 'right' | null>(null);

	// Reactive article fetching - runs on mount and when articleId changes
	$effect(() => {
		const id = articleId;
		let cancelled = false;
		loading = true;
		error = '';
		articles.getArticle(id).then((a) => {
			if (cancelled) return;
			article = a;
			if (a && !a.is_read) {
				articles.toggleRead(a.id, true);
				a.is_read = true;
			}
			api.post('/api/events', { article_id: id, event_type: 'click', duration_seconds: 0 }).catch(() => {});
		}).catch(() => {
			if (cancelled) return;
			error = 'Article not found';
		}).finally(() => {
			if (!cancelled) loading = false;
		});
		return () => { cancelled = true; };
	});

	// Reset state on article change (navigation)
	$effect(() => {
		if (articleId) {
			if (contentEl) contentEl.scrollTop = 0;
			readerHeaderCompact = false;
			readingProgress = 0;
			lastScrollY = 0;
			startTime = Date.now();

			if (slideDirection) {
				const t = setTimeout(() => { slideDirection = null; }, 300);
				return () => clearTimeout(t);
			}
		}
	});

	function handleReaderScroll(e: Event) {
		const target = e.target as HTMLElement;
		readerScrollY = target.scrollTop;

		// Task 8: Progress bar
		const scrollHeight = target.scrollHeight - target.clientHeight;
		readingProgress = scrollHeight > 0 ? Math.min(100, (readerScrollY / scrollHeight) * 100) : 0;

		// Task 9: Header collapse
		if (readerScrollY > 120 && readerScrollY > lastScrollY) {
			readerHeaderCompact = true;
		} else if (readerScrollY < lastScrollY) {
			readerHeaderCompact = false;
		}
		lastScrollY = readerScrollY;
	}

	function trackReadTime() {
		if (article) {
			const duration = Math.floor((Date.now() - startTime) / 1000);
			if (duration > 5) {
				api.post('/api/events', {
					article_id: article.id,
					event_type: 'read',
					duration_seconds: duration,
				}).catch(() => {});
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
		onClose();
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
		if (e.key === 'Escape') {
			if (focusMode) {
				onToggleFocus?.();
			} else {
				handleClose();
			}
		} else if (e.key === 'f') {
			onToggleFocus?.();
		} else if ((e.key === 'j' || e.key === 'k') && articleIds.length > 0) {
			const currentIdx = articleIds.indexOf(articleId);
			if (currentIdx === -1) return;
			const nextIdx = e.key === 'j' ? currentIdx + 1 : currentIdx - 1;
			if (nextIdx >= 0 && nextIdx < articleIds.length) {
				e.preventDefault();
				slideDirection = e.key === 'j' ? 'left' : 'right';
				onNavigate?.(articleIds[nextIdx]);
			}
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Reader panel -->
<div
	class="flex flex-col h-full"
	style="background: var(--color-surface);"
>
	<!-- Task 8: Reading progress bar -->
	{#if !loading && article}
		<div
			class="absolute top-0 left-0 z-10 h-0.5 reading-progress"
			style="width: {readingProgress}%; transition: width 100ms linear;"
		></div>
	{/if}

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
		<!-- Task 9: Collapsing header -->
		<header class="flex-shrink-0 glass border-b border-[var(--color-border)] transition-all"
			style="transition: padding var(--duration-snappy) var(--spring-snappy);">
			<div class="flex items-center justify-between px-5 {readerHeaderCompact ? 'py-1.5' : 'py-3'}"
				style="transition: padding var(--duration-snappy) var(--spring-snappy);">
				<button
					onclick={handleClose}
					class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors group min-w-0"
				>
					<svg class="w-5 h-5 flex-shrink-0 transition-transform group-hover:-translate-x-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
					{#if readerHeaderCompact}
						<span class="truncate max-w-[200px] text-[var(--color-text-primary)] font-medium">{article.title}</span>
					{:else}
						Back
					{/if}
				</button>

				<div class="flex items-center gap-1.5">
					<div class="relative">
						<button
							onclick={(e) => { e.stopPropagation(); readerSettingsOpen = !readerSettingsOpen; }}
							class="p-2 rounded-lg transition-all text-[var(--color-text-tertiary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-elevated)]"
							title="Reader settings"
						>
							<span class="text-sm font-bold">Aa</span>
						</button>
						<ReaderSettings bind:open={readerSettingsOpen} />
					</div>

					<button
						onclick={handleStar}
						class="p-2 rounded-lg transition-all {starAnimating ? 'star-bounce' : ''}
							{article.is_starred
								? 'text-yellow-400 bg-yellow-400/10'
								: 'text-[var(--color-text-tertiary)] hover:text-yellow-400 hover:bg-[var(--color-elevated)]'}"
						title={article.is_starred ? 'Unstar' : 'Star'}
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill={article.is_starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
							<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
						</svg>
					</button>

					{#if article.url && isSafeUrl(article.url)}
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
					{/if}

					<button
						onclick={() => onToggleFocus?.()}
						class="p-2 rounded-lg transition-all text-[var(--color-text-tertiary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-elevated)]"
						title={focusMode ? 'Exit focus mode (f)' : 'Focus mode (f)'}
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							{#if focusMode}
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" />
							{:else}
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
							{/if}
						</svg>
					</button>
				</div>
			</div>
		</header>

		<!-- Task 10: Scrollable content with slide animation -->
		<div class="flex-1 overflow-y-auto {slideDirection === 'left' ? 'slide-from-right' : slideDirection === 'right' ? 'slide-from-left' : ''}"
			bind:this={contentEl}
			onscroll={handleReaderScroll}>
			<!-- Hero image -->
			{#if article.thumbnail_url}
				<div class="relative w-full h-56 overflow-hidden">
					<img src={article.thumbnail_url} alt="" class="w-full h-full object-cover" use:blurUp />
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
								<img src={getFaviconUrl(article.feed_icon_url, article.url, undefined)} alt="" class="w-4 h-4 rounded" loading="lazy" onerror={(e) => handleFaviconError(e, article?.url)} />
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
						<span>{article.reading_time} min</span>
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
						style="font-size: {READER_FONT_SIZE_MAP[$settings.readerFontSize]};
							font-family: {READER_FONT_FAMILY_MAP[$settings.readerFontFamily]};
							line-height: {READER_LINE_HEIGHT_MAP[$settings.readerLineHeight]};
							max-width: {READER_CONTENT_WIDTH_MAP[$settings.readerContentWidth]}; margin: 0 auto;"
					>
						{@html DOMPurify.sanitize(article.content_clean || article.content_raw, { FORBID_TAGS: ['form', 'input', 'textarea', 'select', 'button'], FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'] })}
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-16 text-center">
						<div class="w-16 h-16 rounded-2xl accent-gradient opacity-10 flex items-center justify-center mb-4">
							<svg class="w-8 h-8 text-[var(--color-text-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
						</div>
						<p class="text-[var(--color-text-secondary)] mb-4">No content available for this article.</p>
						{#if article.url && isSafeUrl(article.url)}
						<a
							href={article.url}
							target="_blank"
							rel="noopener noreferrer"
							class="px-5 py-2.5 text-sm font-medium text-white rounded-xl accent-gradient hover:opacity-90 transition-opacity shadow-lg shadow-blue-500/25"
						>
							Read on Original Site
						</a>
						{/if}
					</div>
				{/if}
			</article>
		</div>
	{/if}
</div>

<script lang="ts">
	import { api } from '$lib/api/client';

	interface Source {
		id: number;
		title: string;
		url: string;
		feed_title: string;
	}

	let {
		open = $bindable(false),
		onOpenArticle = (_id: number) => {},
	}: {
		open?: boolean;
		onOpenArticle?: (id: number) => void;
	} = $props();

	let question = $state('');
	let loading = $state(false);
	let answer = $state('');
	let sources = $state<Source[]>([]);
	let error = $state('');
	let hasAsked = $state(false);
	let inputEl = $state<HTMLInputElement | null>(null);

	const examples = [
		'What did my feeds say about AI this week?',
		'Any news about a new release?',
		'Summarize the security stories.',
	];

	// Focus the input when opened; reset everything when closed.
	$effect(() => {
		if (open) {
			requestAnimationFrame(() => inputEl?.focus());
		} else {
			question = '';
			answer = '';
			sources = [];
			error = '';
			loading = false;
			hasAsked = false;
		}
	});

	async function ask() {
		const q = question.trim();
		if (!q || loading) return;
		loading = true;
		error = '';
		answer = '';
		sources = [];
		hasAsked = true;
		try {
			const data = await api.post<{ answer: string; sources: Source[] }>('/api/ask', { question: q });
			answer = data.answer;
			sources = data.sources || [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			loading = false;
		}
	}

	function close() {
		open = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			close();
		}
	}

	function openSource(s: Source) {
		onOpenArticle(s.id);
		close();
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]"
		onkeydown={handleKeydown}
	>
		<div
			class="absolute inset-0 bg-black/50 backdrop-blur-sm"
			onclick={close}
			role="presentation"
		></div>

		<div
			class="relative w-full max-w-xl mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl overflow-hidden fade-in-up"
			style="animation-duration: var(--duration-snappy);"
		>
			<!-- Header / input -->
			<div class="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)]">
				<svg class="w-5 h-5 text-[var(--color-accent)] flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
				</svg>
				<input
					bind:this={inputEl}
					bind:value={question}
					type="text"
					placeholder="Ask your feeds anything..."
					onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); ask(); } }}
					class="flex-1 bg-transparent text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] outline-none text-base"
				/>
				<button
					onclick={ask}
					disabled={loading || !question.trim()}
					class="px-3 py-1.5 text-sm font-medium text-white rounded-lg accent-gradient hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition-opacity flex-shrink-0"
				>
					{loading ? '…' : 'Ask'}
				</button>
			</div>

			<!-- Body -->
			<div class="max-h-[60vh] overflow-y-auto px-4 py-4">
				{#if loading}
					<div class="flex items-center gap-3 text-sm text-[var(--color-text-secondary)] py-6 justify-center">
						<div class="w-4 h-4 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
						Searching your feeds and thinking…
					</div>
				{:else if error}
					<div class="p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20">
						{error}
					</div>
				{:else if answer}
					<p class="text-sm leading-relaxed text-[var(--color-text-primary)] whitespace-pre-wrap">{answer}</p>

					{#if sources.length > 0}
						<div class="mt-4 pt-3 border-t border-[var(--color-border)]">
							<div class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] mb-2">Sources</div>
							<div class="space-y-1">
								{#each sources as source, i}
									<button
										onclick={() => openSource(source)}
										class="w-full flex items-start gap-2 px-2 py-1.5 text-left rounded-lg hover:bg-[var(--color-elevated)] transition-colors group"
									>
										<span class="text-xs font-semibold text-[var(--color-accent)] mt-0.5 flex-shrink-0">[{i + 1}]</span>
										<span class="flex-1 min-w-0">
											<span class="block text-sm text-[var(--color-text-primary)] truncate group-hover:text-[var(--color-accent)] transition-colors">{source.title}</span>
											{#if source.feed_title}
												<span class="block text-xs text-[var(--color-text-tertiary)] truncate">{source.feed_title}</span>
											{/if}
										</span>
									</button>
								{/each}
							</div>
						</div>
					{/if}
				{:else if !hasAsked}
					<div class="text-sm text-[var(--color-text-tertiary)]">
						<p class="mb-3">Ask a question and get an answer grounded in your own subscriptions, with links to the source articles.</p>
						<div class="flex flex-wrap gap-2">
							{#each examples as ex}
								<button
									onclick={() => { question = ex; ask(); }}
									class="px-3 py-1.5 text-xs rounded-full border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)] transition-colors"
								>
									{ex}
								</button>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<div class="flex items-center justify-between px-4 py-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)]">
				<span>Answers are generated from your feeds and may be imperfect.</span>
				<kbd class="px-2 py-0.5 rounded bg-[var(--color-surface)] border border-[var(--color-border)]">Esc</kbd>
			</div>
		</div>
	</div>
{/if}

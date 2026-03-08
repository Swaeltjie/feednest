<script lang="ts">
	import { api } from '$lib/api/client';
	import { feeds } from '$lib/stores/feeds';

	interface FilterRule {
		id: number;
		name: string;
		feed_id: number | null;
		field: string;
		operator: string;
		value: string;
		action: string;
		enabled: boolean;
	}

	let { open = $bindable(false) }: { open: boolean } = $props();

	let rules = $state<FilterRule[]>([]);
	let loading = $state(true);
	let showForm = $state(false);
	let formName = $state('');
	let formFeedId = $state<number | null>(null);
	let formField = $state('title');
	let formOperator = $state('contains');
	let formValue = $state('');
	let formAction = $state('hide');
	let formError = $state('');
	let saving = $state(false);

	const fieldLabels: Record<string, string> = {
		title: 'title',
		author: 'author',
		content: 'content',
	};

	const operatorLabels: Record<string, string> = {
		contains: 'contains',
		not_contains: 'does not contain',
		regex: 'matches regex',
	};

	const actionLabels: Record<string, string> = {
		hide: 'Hide',
		mark_read: 'Auto mark read',
		star: 'Auto star',
	};

	$effect(() => {
		if (open) {
			loadRules();
		}
	});

	async function loadRules() {
		loading = true;
		try {
			rules = await api.get<FilterRule[]>('/api/rules') || [];
		} catch {
			rules = [];
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		formName = '';
		formFeedId = null;
		formField = 'title';
		formOperator = 'contains';
		formValue = '';
		formAction = 'hide';
		formError = '';
	}

	async function createRule() {
		formError = '';
		if (!formName.trim()) {
			formError = 'Name is required';
			return;
		}
		if (!formValue.trim()) {
			formError = 'Value is required';
			return;
		}
		if (formOperator === 'regex') {
			try {
				new RegExp(formValue);
			} catch {
				formError = 'Invalid regular expression';
				return;
			}
		}

		saving = true;
		try {
			const body: Record<string, unknown> = {
				name: formName.trim(),
				feed_id: formFeedId,
				field: formField,
				operator: formOperator,
				value: formValue.trim(),
				action: formAction,
				enabled: true,
			};
			await api.post('/api/rules', body);
			resetForm();
			showForm = false;
			await loadRules();
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to create rule';
		} finally {
			saving = false;
		}
	}

	async function deleteRule(id: number) {
		try {
			await api.del(`/api/rules/${id}`);
			rules = rules.filter(r => r.id !== id);
		} catch (err) {
			console.error('Failed to delete rule:', err);
		}
	}

	async function toggleRule(rule: FilterRule) {
		try {
			await api.put(`/api/rules/${rule.id}`, { ...rule, enabled: !rule.enabled });
			rules = rules.map(r => r.id === rule.id ? { ...r, enabled: !r.enabled } : r);
		} catch (err) {
			console.error('Failed to toggle rule:', err);
		}
	}

	function describeRule(rule: FilterRule): string {
		const op = operatorLabels[rule.operator] || rule.operator;
		const field = fieldLabels[rule.field] || rule.field;
		const action = actionLabels[rule.action] || rule.action;
		return `${action} articles where ${field} ${op} "${rule.value}"`;
	}

	function getFeedName(feedId: number | null): string {
		if (feedId === null) return 'All feeds';
		const feed = $feeds.find(f => f.id === feedId);
		return feed?.title || `Feed #${feedId}`;
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-[100] flex items-center justify-center" onkeydown={(e) => { if (e.key === 'Escape') { e.preventDefault(); open = false; } }}>
		<div class="absolute inset-0 bg-black/50 backdrop-blur-sm" onclick={() => (open = false)} role="presentation"></div>
		<div class="relative w-full max-w-lg mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl overflow-hidden fade-in-up" style="animation-duration: var(--duration-snappy);">
			<!-- Header -->
			<div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border)]">
				<h2 class="text-lg font-bold text-[var(--color-text-primary)]">Filter Rules</h2>
				<div class="flex items-center gap-2">
					<button
						onclick={() => { if (showForm) { resetForm(); showForm = false; } else { showForm = true; } }}
						class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors
							{showForm
								? 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)]'
								: 'text-white accent-gradient hover:opacity-90 shadow-sm'}"
					>
						{showForm ? 'Cancel' : 'Add Rule'}
					</button>
					<button onclick={() => (open = false)} class="p-1 rounded-lg hover:bg-[var(--color-elevated)] transition-colors" aria-label="Close filter rules">
						<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			</div>

			<!-- Add Rule Form -->
			{#if showForm}
				<div class="px-6 py-4 border-b border-[var(--color-border)] space-y-3" style="background: var(--color-elevated);">
					{#if formError}
						<div class="p-2.5 text-xs text-red-400 bg-red-500/10 rounded-lg border border-red-500/20">
							{formError}
						</div>
					{/if}

					<div>
						<label for="rule-name" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Name</label>
						<input
							id="rule-name"
							type="text"
							bind:value={formName}
							placeholder="e.g. Hide sponsored content"
							class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
								text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
						/>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="rule-feed" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Feed</label>
							<select
								id="rule-feed"
								bind:value={formFeedId}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
									text-[var(--color-text-primary)]
									focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all
									appearance-none cursor-pointer"
							>
								<option value={null}>All feeds</option>
								{#each $feeds as feed}
									<option value={feed.id}>{feed.title}</option>
								{/each}
							</select>
						</div>

						<div>
							<label for="rule-field" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Field</label>
							<select
								id="rule-field"
								bind:value={formField}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
									text-[var(--color-text-primary)]
									focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all
									appearance-none cursor-pointer"
							>
								<option value="title">Title</option>
								<option value="author">Author</option>
								<option value="content">Content</option>
							</select>
						</div>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="rule-operator" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Operator</label>
							<select
								id="rule-operator"
								bind:value={formOperator}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
									text-[var(--color-text-primary)]
									focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all
									appearance-none cursor-pointer"
							>
								<option value="contains">Contains</option>
								<option value="not_contains">Does not contain</option>
								<option value="regex">Regex</option>
							</select>
						</div>

						<div>
							<label for="rule-action" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Action</label>
							<select
								id="rule-action"
								bind:value={formAction}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
									text-[var(--color-text-primary)]
									focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all
									appearance-none cursor-pointer"
							>
								<option value="hide">Hide</option>
								<option value="mark_read">Auto mark read</option>
								<option value="star">Auto star</option>
							</select>
						</div>
					</div>

					<div>
						<label for="rule-value" class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Value</label>
						<input
							id="rule-value"
							type="text"
							bind:value={formValue}
							placeholder={formOperator === 'regex' ? 'e.g. sponsor(ed|ship)' : 'e.g. sponsored'}
							class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]
								text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all"
						/>
					</div>

					<div class="flex justify-end pt-1">
						<button
							onclick={createRule}
							disabled={saving}
							class="px-4 py-2 text-sm font-medium text-white rounded-lg accent-gradient
								hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity
								shadow-lg shadow-blue-500/25"
						>
							{saving ? 'Saving...' : 'Create Rule'}
						</button>
					</div>
				</div>
			{/if}

			<!-- Rules List -->
			<div class="max-h-80 overflow-y-auto">
				{#if loading}
					<div class="flex items-center justify-center py-12">
						<div class="w-5 h-5 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
					</div>
				{:else if rules.length === 0}
					<div class="px-6 py-12 text-center">
						<div class="text-3xl mb-3">🔧</div>
						<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">No filter rules yet</p>
						<p class="text-xs text-[var(--color-text-tertiary)]">Create rules to automatically filter, hide, or tag articles.</p>
					</div>
				{:else}
					<div class="divide-y divide-[var(--color-border)]">
						{#each rules as rule (rule.id)}
							<div class="px-6 py-3 flex items-start gap-3 group hover:bg-[var(--color-elevated)] transition-colors">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2">
										<span class="text-sm font-medium text-[var(--color-text-primary)] truncate {!rule.enabled ? 'opacity-50' : ''}">{rule.name}</span>
										{#if rule.feed_id !== null}
											<span class="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--color-accent-glow)] text-[var(--color-accent)] whitespace-nowrap">{getFeedName(rule.feed_id)}</span>
										{/if}
									</div>
									<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 {!rule.enabled ? 'opacity-50' : ''}">{describeRule(rule)}</p>
								</div>

								<div class="flex items-center gap-2 flex-shrink-0">
									<!-- Enable/disable toggle -->
									<button
										onclick={() => toggleRule(rule)}
										class="relative w-9 h-5 rounded-full transition-colors {rule.enabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]'}"
										aria-label="{rule.enabled ? 'Disable' : 'Enable'} rule"
									>
										<span class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform {rule.enabled ? 'translate-x-4' : 'translate-x-0'}"></span>
									</button>

									<!-- Delete -->
									<button
										onclick={() => deleteRule(rule.id)}
										class="p-1 rounded-lg text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 opacity-0 group-hover:opacity-100 transition-all"
										aria-label="Delete rule"
									>
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

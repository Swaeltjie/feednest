<script lang="ts">
	import {
		settings,
		READER_FONT_SIZE_MAP,
		READER_FONT_FAMILY_MAP,
		READER_LINE_HEIGHT_MAP,
		READER_CONTENT_WIDTH_MAP,
		type ReaderFontSize,
		type ReaderFontFamily,
		type ReaderLineHeight,
		type ReaderContentWidth,
	} from '$lib/stores/settings';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	function handleWindowClick(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (open && !target.closest('.reader-settings-popover')) {
			open = false;
		}
	}

	const fontSizes: { value: ReaderFontSize; label: string }[] = [
		{ value: 'small', label: 'S' },
		{ value: 'medium', label: 'M' },
		{ value: 'large', label: 'L' },
		{ value: 'xl', label: 'XL' },
	];

	const fontFamilies: { value: ReaderFontFamily; label: string }[] = [
		{ value: 'sans', label: 'Sans' },
		{ value: 'serif', label: 'Serif' },
		{ value: 'mono', label: 'Mono' },
	];

	const lineHeights: { value: ReaderLineHeight; label: string }[] = [
		{ value: 'compact', label: 'Compact' },
		{ value: 'comfortable', label: 'Comfy' },
		{ value: 'spacious', label: 'Spacious' },
	];

	const contentWidths: { value: ReaderContentWidth; label: string }[] = [
		{ value: 'narrow', label: 'Narrow' },
		{ value: 'medium', label: 'Medium' },
		{ value: 'wide', label: 'Wide' },
	];
</script>

<svelte:window onclick={handleWindowClick} />

{#if open}
	<div class="reader-settings-popover absolute right-0 top-full mt-2 z-50 w-72 rounded-xl shadow-2xl border border-[var(--color-border)] p-4 space-y-4 fade-in-up"
		style="background: var(--color-card); animation-duration: 150ms;">
		<h3 class="text-sm font-semibold text-[var(--color-text-primary)]">Reader Settings</h3>

		<!-- Font Size -->
		<div>
			<label class="block text-xs font-medium text-[var(--color-text-tertiary)] mb-1.5">Font Size</label>
			<div class="flex gap-1 p-0.5 rounded-lg" style="background: var(--color-border);">
				{#each fontSizes as size}
					<button
						onclick={() => settings.setReaderFontSize(size.value)}
						class="flex-1 py-1.5 text-xs font-medium rounded-md transition-all
							{$settings.readerFontSize === size.value
								? 'bg-[var(--color-card)] shadow-sm text-[var(--color-text-primary)]'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}"
					>
						{size.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Font Family -->
		<div>
			<label class="block text-xs font-medium text-[var(--color-text-tertiary)] mb-1.5">Font</label>
			<div class="flex gap-1 p-0.5 rounded-lg" style="background: var(--color-border);">
				{#each fontFamilies as family}
					<button
						onclick={() => settings.setReaderFontFamily(family.value)}
						class="flex-1 py-1.5 text-xs font-medium rounded-md transition-all
							{$settings.readerFontFamily === family.value
								? 'bg-[var(--color-card)] shadow-sm text-[var(--color-text-primary)]'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}"
					>
						{family.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Line Height -->
		<div>
			<label class="block text-xs font-medium text-[var(--color-text-tertiary)] mb-1.5">Spacing</label>
			<div class="flex gap-1 p-0.5 rounded-lg" style="background: var(--color-border);">
				{#each lineHeights as lh}
					<button
						onclick={() => settings.setReaderLineHeight(lh.value)}
						class="flex-1 py-1.5 text-xs font-medium rounded-md transition-all
							{$settings.readerLineHeight === lh.value
								? 'bg-[var(--color-card)] shadow-sm text-[var(--color-text-primary)]'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}"
					>
						{lh.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Content Width -->
		<div>
			<label class="block text-xs font-medium text-[var(--color-text-tertiary)] mb-1.5">Width</label>
			<div class="flex gap-1 p-0.5 rounded-lg" style="background: var(--color-border);">
				{#each contentWidths as cw}
					<button
						onclick={() => settings.setReaderContentWidth(cw.value)}
						class="flex-1 py-1.5 text-xs font-medium rounded-md transition-all
							{$settings.readerContentWidth === cw.value
								? 'bg-[var(--color-card)] shadow-sm text-[var(--color-text-primary)]'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}"
					>
						{cw.label}
					</button>
				{/each}
			</div>
		</div>
	</div>
{/if}

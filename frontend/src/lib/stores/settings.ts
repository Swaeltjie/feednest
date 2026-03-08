import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export type Theme = 'light' | 'dark' | 'system';
export type ReaderFontSize = 'small' | 'medium' | 'large' | 'xl';
export type ReaderFontFamily = 'sans' | 'serif' | 'mono';
export type ReaderLineHeight = 'compact' | 'comfortable' | 'spacious';
export type ReaderContentWidth = 'narrow' | 'medium' | 'wide';

interface SettingsState {
	theme: Theme;
	readerFontSize: ReaderFontSize;
	readerFontFamily: ReaderFontFamily;
	readerLineHeight: ReaderLineHeight;
	readerContentWidth: ReaderContentWidth;
	calmMode: boolean;
	autoMarkReadOnScroll: boolean;
}

export const READER_FONT_SIZE_MAP: Record<ReaderFontSize, string> = {
	small: '15px', medium: '17px', large: '19px', xl: '21px',
};
export const READER_FONT_FAMILY_MAP: Record<ReaderFontFamily, string> = {
	sans: 'ui-sans-serif, system-ui, sans-serif',
	serif: 'ui-serif, Georgia, Cambria, serif',
	mono: 'ui-monospace, SFMono-Regular, monospace',
};
export const READER_LINE_HEIGHT_MAP: Record<ReaderLineHeight, string> = {
	compact: '1.5', comfortable: '1.75', spacious: '2.0',
};
export const READER_CONTENT_WIDTH_MAP: Record<ReaderContentWidth, string> = {
	narrow: '580px', medium: '680px', wide: '820px',
};

function createSettingsStore() {
	function ls(key: string): string | null {
		return typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null;
	}

	function persist(key: string, value: string) {
		if (typeof localStorage !== 'undefined') localStorage.setItem(key, value);
	}

	function apiPersist(key: string, value: string) {
		api.put('/api/settings', { [key]: value }).catch((err) => {
			console.warn('Failed to sync setting to server:', key, err);
		});
	}

	const stored = ls('feednest_theme');
	const initial: SettingsState = {
		theme: (stored === 'light' || stored === 'dark' || stored === 'system') ? stored : 'system',
		readerFontSize: (ls('feednest_reader_font_size') as ReaderFontSize) || 'medium',
		readerFontFamily: (ls('feednest_reader_font_family') as ReaderFontFamily) || 'sans',
		readerLineHeight: (ls('feednest_reader_line_height') as ReaderLineHeight) || 'comfortable',
		readerContentWidth: (ls('feednest_reader_content_width') as ReaderContentWidth) || 'medium',
		calmMode: ls('feednest_calm_mode') === 'true',
		autoMarkReadOnScroll: ls('feednest_auto_mark_read_scroll') === 'true',
	};

	const { subscribe, update } = writable<SettingsState>(initial);

	function applyTheme(theme: Theme) {
		if (typeof document === 'undefined') return;

		const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
		const isDark = theme === 'dark' || (theme === 'system' && prefersDark);

		document.documentElement.classList.add('theme-transitioning');
		document.documentElement.classList.toggle('dark', isDark);
		setTimeout(() => {
			document.documentElement.classList.remove('theme-transitioning');
		}, 350);
	}

	// Apply initial theme
	applyTheme(initial.theme);

	return {
		subscribe,

		setTheme(theme: Theme) {
			update((s) => ({ ...s, theme }));
			persist('feednest_theme', theme);
			applyTheme(theme);
		},

		setReaderFontSize(size: ReaderFontSize) {
			update((s) => ({ ...s, readerFontSize: size }));
			persist('feednest_reader_font_size', size);
			apiPersist('reader_font_size', size);
		},

		setReaderFontFamily(family: ReaderFontFamily) {
			update((s) => ({ ...s, readerFontFamily: family }));
			persist('feednest_reader_font_family', family);
			apiPersist('reader_font_family', family);
		},

		setReaderLineHeight(height: ReaderLineHeight) {
			update((s) => ({ ...s, readerLineHeight: height }));
			persist('feednest_reader_line_height', height);
			apiPersist('reader_line_height', height);
		},

		setReaderContentWidth(width: ReaderContentWidth) {
			update((s) => ({ ...s, readerContentWidth: width }));
			persist('feednest_reader_content_width', width);
			apiPersist('reader_content_width', width);
		},

		setCalmMode(enabled: boolean) {
			update((s) => ({ ...s, calmMode: enabled }));
			persist('feednest_calm_mode', String(enabled));
			apiPersist('calm_mode', String(enabled));
		},

		setAutoMarkReadOnScroll(enabled: boolean) {
			update((s) => ({ ...s, autoMarkReadOnScroll: enabled }));
			persist('feednest_auto_mark_read_scroll', String(enabled));
			apiPersist('auto_mark_read_scroll', String(enabled));
		},
	};
}

export const settings = createSettingsStore();

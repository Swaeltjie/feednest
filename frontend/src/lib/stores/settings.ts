import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark' | 'system';

interface SettingsState {
	theme: Theme;
}

function createSettingsStore() {
	const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('feednest_theme') : null;
	const initial: SettingsState = {
		theme: (stored === 'light' || stored === 'dark' || stored === 'system') ? stored : 'system',
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
			if (typeof localStorage !== 'undefined') {
				localStorage.setItem('feednest_theme', theme);
			}
			applyTheme(theme);
		},
	};
}

export const settings = createSettingsStore();

import { readable } from 'svelte/store';

/**
 * Online/offline status store.
 *
 * Initializes from `navigator.onLine` (defaulting to `true` during SSR or when
 * the Network Information API is unavailable) and stays in sync via the window
 * `online` / `offline` events. All browser-only access lives inside the
 * `readable` start function so the module is safe to import during SSR.
 */
export const online = readable<boolean>(
	typeof navigator !== 'undefined' ? navigator.onLine : true,
	(set) => {
		if (typeof window === 'undefined') return;

		set(navigator.onLine);

		const goOnline = () => set(true);
		const goOffline = () => set(false);

		window.addEventListener('online', goOnline);
		window.addEventListener('offline', goOffline);

		return () => {
			window.removeEventListener('online', goOnline);
			window.removeEventListener('offline', goOffline);
		};
	}
);

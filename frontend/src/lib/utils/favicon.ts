export function getFaviconUrl(iconUrl: string | undefined, siteUrl: string | undefined, feedUrl?: string): string | null {
	const url = siteUrl || feedUrl;
	if (url) {
		try {
			const parsed = new URL(url);
			// favicon.im checks HTML link tags, web manifests, Apple touch icons,
			// /favicon.ico, and falls back to Google — much better coverage than
			// Google's s2/favicons which often returns a generic globe icon.
			return `https://favicon.im/${parsed.host}?larger=true`;
		} catch {
			// fall through
		}
	}

	// Fall back to feed-provided icon
	if (iconUrl) return iconUrl;

	return null;
}

/**
 * Fallback URL to try if the primary favicon fails to load.
 * Uses DuckDuckGo's icon service as secondary source.
 */
export function getFaviconFallback(siteUrl: string | undefined, feedUrl?: string): string | null {
	const url = siteUrl || feedUrl;
	if (url) {
		try {
			const parsed = new URL(url);
			return `https://icons.duckduckgo.com/ip3/${parsed.host}.ico`;
		} catch {
			// fall through
		}
	}
	return null;
}

/**
 * Handle favicon load error: try DuckDuckGo fallback, then hide.
 * Attach to img onerror. Marks the element to avoid infinite retry loops.
 */
export function handleFaviconError(e: Event, siteUrl?: string, feedUrl?: string) {
	const img = e.currentTarget as HTMLImageElement;
	if (img.dataset.faviconRetried) {
		img.style.display = 'none';
		return;
	}
	img.dataset.faviconRetried = '1';
	const fallback = getFaviconFallback(siteUrl, feedUrl);
	if (fallback) {
		img.src = fallback;
	} else {
		img.style.display = 'none';
	}
}

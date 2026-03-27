export function getFaviconUrl(iconUrl: string | undefined, siteUrl: string | undefined, feedUrl?: string): string | null {
	// Prefer feed-provided icon (avoids third-party dependency and privacy leak)
	if (iconUrl) return iconUrl;

	const url = siteUrl || feedUrl;
	if (url) {
		try {
			const parsed = new URL(url);
			return `https://favicon.im/${parsed.host}?larger=true`;
		} catch {
			// fall through
		}
	}

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

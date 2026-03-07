export function getFaviconUrl(iconUrl: string | undefined, siteUrl: string | undefined, feedUrl?: string): string | null {
	// Always prefer Google Favicon API for consistent sizing and availability
	const url = siteUrl || feedUrl;
	if (url) {
		try {
			const parsed = new URL(url);
			return `https://www.google.com/s2/favicons?domain=${parsed.host}&sz=32`;
		} catch {
			// fall through
		}
	}

	// Fall back to feed-provided icon
	if (iconUrl) return iconUrl;

	return null;
}

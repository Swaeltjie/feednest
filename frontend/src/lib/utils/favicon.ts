export function getFaviconUrl(iconUrl: string | undefined, siteUrl: string | undefined, feedUrl?: string): string | null {
	if (iconUrl) return iconUrl;

	const url = siteUrl || feedUrl;
	if (!url) return null;

	try {
		const parsed = new URL(url);
		return `https://www.google.com/s2/favicons?domain=${parsed.host}&sz=32`;
	} catch {
		return null;
	}
}

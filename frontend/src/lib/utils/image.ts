// proxyImage routes a remote image URL through the backend image proxy
// (/api/image), which fetches it server-side with a browser User-Agent and no
// Referer. This defeats the browser-side blocks that leave many feed
// thumbnails broken: Opaque Response Blocking (ERR_BLOCKED_BY_ORB),
// referer-based hotlink protection, and mixed-content. The proxied response is
// same-origin, so it also benefits from normal browser caching.
//
// Non-http(s) values (data: URIs, already-relative paths) are returned as-is.
export function proxyImage(url: string | undefined | null): string {
	if (!url) return '';
	if (url.startsWith('/api/image')) return url; // already proxied
	if (url.startsWith('http://') || url.startsWith('https://')) {
		return `/api/image?url=${encodeURIComponent(url)}`;
	}
	return url;
}

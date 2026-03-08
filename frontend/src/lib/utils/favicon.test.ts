import { describe, it, expect } from 'vitest';
import { getFaviconUrl, getFaviconFallback } from './favicon';

describe('getFaviconUrl', () => {
	it('returns favicon.im URL for valid siteUrl', () => {
		const result = getFaviconUrl(undefined, 'https://example.com/blog');
		expect(result).toBe('https://favicon.im/example.com?larger=true');
	});

	it('prefers siteUrl over feedUrl', () => {
		const result = getFaviconUrl(undefined, 'https://site.com', 'https://feed.com/rss');
		expect(result).toBe('https://favicon.im/site.com?larger=true');
	});

	it('falls back to feedUrl when no siteUrl', () => {
		const result = getFaviconUrl(undefined, undefined, 'https://feed.com/rss');
		expect(result).toBe('https://favicon.im/feed.com?larger=true');
	});

	it('returns iconUrl when no siteUrl or feedUrl', () => {
		const result = getFaviconUrl('https://cdn.example.com/icon.png', undefined);
		expect(result).toBe('https://cdn.example.com/icon.png');
	});

	it('returns null when nothing is provided', () => {
		expect(getFaviconUrl(undefined, undefined)).toBeNull();
	});

	it('handles invalid URLs gracefully', () => {
		const result = getFaviconUrl(undefined, 'not-a-url');
		expect(result).toBeNull();
	});
});

describe('getFaviconFallback', () => {
	it('returns DuckDuckGo icon URL', () => {
		const result = getFaviconFallback('https://example.com');
		expect(result).toBe('https://icons.duckduckgo.com/ip3/example.com.ico');
	});

	it('returns null for no URL', () => {
		expect(getFaviconFallback(undefined)).toBeNull();
	});
});

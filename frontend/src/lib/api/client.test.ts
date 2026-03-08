import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setAccessToken, getAccessToken, isSafeUrl, api } from './client';

describe('isSafeUrl', () => {
	it('returns true for https URLs', () => {
		expect(isSafeUrl('https://example.com')).toBe(true);
	});

	it('returns true for http URLs', () => {
		expect(isSafeUrl('http://example.com')).toBe(true);
	});

	it('returns false for javascript: URLs', () => {
		expect(isSafeUrl('javascript:alert(1)')).toBe(false);
	});

	it('returns false for data: URLs', () => {
		expect(isSafeUrl('data:text/html,<script>alert(1)</script>')).toBe(false);
	});

	it('returns false for invalid URLs', () => {
		expect(isSafeUrl('not a url')).toBe(false);
	});
});

describe('accessToken management', () => {
	beforeEach(() => {
		setAccessToken(null);
	});

	it('starts with null token', () => {
		expect(getAccessToken()).toBeNull();
	});

	it('sets and gets token', () => {
		setAccessToken('test-token');
		expect(getAccessToken()).toBe('test-token');
	});

	it('clears token with null', () => {
		setAccessToken('test-token');
		setAccessToken(null);
		expect(getAccessToken()).toBeNull();
	});
});

describe('api', () => {
	beforeEach(() => {
		setAccessToken(null);
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('makes GET request with correct headers', async () => {
		const mockResponse = { articles: [] };
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify(mockResponse), {
				status: 200,
				headers: { 'Content-Type': 'application/json' },
			})
		);

		const result = await api.get('/api/articles');
		expect(fetch).toHaveBeenCalledWith(
			expect.stringContaining('/api/articles'),
			expect.objectContaining({ method: 'GET' })
		);
		expect(result).toEqual(mockResponse);
	});

	it('includes auth header when token is set', async () => {
		setAccessToken('my-jwt');
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify({}), { status: 200 })
		);

		await api.get('/api/feeds');
		expect(fetch).toHaveBeenCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.objectContaining({
					Authorization: 'Bearer my-jwt',
				}),
			})
		);
	});

	it('throws on non-ok response', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
		);

		await expect(api.get('/api/articles/999')).rejects.toThrow('not found');
	});

	it('returns undefined for 204 responses', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(null, { status: 204 })
		);

		const result = await api.del('/api/feeds/1');
		expect(result).toBeUndefined();
	});

	it('sends JSON body for POST', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify({ id: 1 }), { status: 200 })
		);

		await api.post('/api/feeds', { url: 'https://example.com/rss' });
		expect(fetch).toHaveBeenCalledWith(
			expect.any(String),
			expect.objectContaining({
				method: 'POST',
				body: JSON.stringify({ url: 'https://example.com/rss' }),
			})
		);
	});
});

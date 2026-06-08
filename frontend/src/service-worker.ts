/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

import { build, files, version } from '$service-worker';

// The default `self` in a TS module is typed as `Window & typeof globalThis`,
// which lacks the service-worker APIs we rely on. Re-type it once, here, so the
// rest of the file is fully type-checked with no untyped `self` usage.
const sw = self as unknown as ServiceWorkerGlobalScope;

// Versioned cache name — a new build (new `version`) means a new cache, and the
// old one is purged on `activate`. This is what makes precached build/files
// assets safe to serve cache-first: they can never go stale across deploys.
const CACHE = `feednest-cache-${version}`;

// App shell: hashed build artifacts + static files. Both are immutable for a
// given `version`, so cache-first is correct for them.
const PRECACHE = [...build, ...files];
const PRECACHE_SET = new Set(PRECACHE);

// ---------------------------------------------------------------------------
// Install — precache the app shell, then take over immediately.
// ---------------------------------------------------------------------------
sw.addEventListener('install', (event) => {
	event.waitUntil(
		(async () => {
			try {
				const cache = await caches.open(CACHE);
				await cache.addAll(PRECACHE);
			} catch (err) {
				// A failed precache must not wedge the install — we still skip
				// waiting and fall back to the network at fetch time.
				console.error('[sw] precache failed', err);
			}
			await sw.skipWaiting();
		})()
	);
});

// ---------------------------------------------------------------------------
// Activate — drop caches from previous versions, then claim open clients.
// ---------------------------------------------------------------------------
sw.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			try {
				const keys = await caches.keys();
				await Promise.all(
					keys.map((key) => (key === CACHE ? Promise.resolve(true) : caches.delete(key)))
				);
			} catch (err) {
				console.error('[sw] cache cleanup failed', err);
			}
			await sw.clients.claim();
		})()
	);
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Cache-first: serve from cache, fall back to network (and warm the cache). */
async function cacheFirst(request: Request): Promise<Response> {
	const cached = await caches.match(request);
	if (cached) {
		return cached;
	}
	const response = await fetch(request);
	if (response && response.ok) {
		try {
			const cache = await caches.open(CACHE);
			await cache.put(request, response.clone());
		} catch (err) {
			console.error('[sw] cacheFirst put failed', err);
		}
	}
	return response;
}

/**
 * Network-first: try the network and refresh a runtime cache on success; on
 * failure (offline) serve the last cached copy if we have one. This is what
 * enables offline reading of articles and images that were loaded while online.
 */
async function networkFirst(request: Request): Promise<Response> {
	try {
		const response = await fetch(request);
		if (response && response.ok) {
			try {
				const cache = await caches.open(CACHE);
				await cache.put(request, response.clone());
			} catch (err) {
				console.error('[sw] networkFirst put failed', err);
			}
		}
		return response;
	} catch (err) {
		const cached = await caches.match(request);
		if (cached) {
			return cached;
		}
		throw err;
	}
}

/** Network with cache fallback — like network-first but never writes new entries. */
async function networkWithCacheFallback(request: Request): Promise<Response> {
	try {
		return await fetch(request);
	} catch (err) {
		const cached = await caches.match(request);
		if (cached) {
			return cached;
		}
		throw err;
	}
}

// `GET /api/articles`, `GET /api/articles/{id}` — cache the read endpoints that
// power offline reading. `/api/image*` is handled separately so the leading
// match also covers `/api/image/...` and `/api/image?...`.
function isCacheableArticleRequest(pathname: string): boolean {
	return pathname === '/api/articles' || /^\/api\/articles\/[^/]+$/.test(pathname);
}

function isImageProxyRequest(pathname: string): boolean {
	return pathname === '/api/image' || pathname.startsWith('/api/image/') || pathname.startsWith('/api/image?');
}

// ---------------------------------------------------------------------------
// Fetch — routing.
// ---------------------------------------------------------------------------
sw.addEventListener('fetch', (event) => {
	const { request } = event;

	// Only ever touch GET; let everything else (POST/PUT/DELETE/etc.) hit the
	// network untouched so mutations are never served from cache.
	if (request.method !== 'GET') {
		return;
	}

	let url: URL;
	try {
		url = new URL(request.url);
	} catch {
		// Unparseable URL — leave it to the browser's default handling.
		return;
	}

	// Skip cross-origin requests entirely. The `/api/image` proxy is itself a
	// same-origin endpoint (the backend fetches the remote image), so it is
	// covered by this same-origin gate too — we never cache a true cross-origin
	// response.
	if (url.origin !== sw.location.origin) {
		return;
	}

	const isImageProxy = isImageProxyRequest(url.pathname);

	// NEVER cache or interfere with auth — caching JWT/refresh responses breaks
	// login and silent token refresh. Straight to the network.
	if (url.pathname.startsWith('/api/auth/')) {
		return;
	}

	// Navigations and precached build/files assets → cache-first (safe because
	// build/files are versioned and navigations fall back to the network).
	if (request.mode === 'navigate' || PRECACHE_SET.has(url.pathname)) {
		event.respondWith(cacheFirst(request));
		return;
	}

	// Read endpoints + image proxy → network-first with cache fallback, so
	// already-loaded articles and images remain readable offline.
	if (isCacheableArticleRequest(url.pathname) || isImageProxy) {
		event.respondWith(networkFirst(request));
		return;
	}

	// Everything else same-origin → try network, fall back to cache.
	event.respondWith(networkWithCacheFallback(request));
});

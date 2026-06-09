const API_BASE = import.meta.env.VITE_API_URL || (typeof window !== 'undefined' ? '' : 'http://localhost:8082');

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
	accessToken = token;
}

export function getAccessToken(): string | null {
	return accessToken;
}

export function isSafeUrl(url: string): boolean {
	try {
		const parsed = new URL(url);
		return parsed.protocol === 'https:' || parsed.protocol === 'http:';
	} catch {
		return false;
	}
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	const headers: Record<string, string> = {};
	if (body !== undefined) {
		headers['Content-Type'] = 'application/json';
	}

	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}

	const res = await fetch(`${API_BASE}${path}`, {
		method,
		headers,
		body: body !== undefined ? JSON.stringify(body) : undefined,
	});

	if (res.status === 401 && accessToken) {
		const result = await refreshTokenFn();
		// Transient refresh failure (network blip / backend 5xx): do NOT destroy
		// the still-valid refresh token. Surface a retryable error so a momentary
		// outage coinciding with token expiry doesn't force a needless re-login.
		if (result === 'transient') {
			throw new Error('Temporary authentication error, please retry');
		}
		if (result === 'ok') {
			headers['Authorization'] = `Bearer ${accessToken}`;
			const retry = await fetch(`${API_BASE}${path}`, {
				method,
				headers,
				body: body !== undefined ? JSON.stringify(body) : undefined,
			});
			// A 401 after a fresh token means the session is genuinely invalid —
			// fall through to the session-expiry handling below.
			if (retry.status !== 401) {
				if (!retry.ok) {
					const err = await retry.json().catch(() => ({ error: `HTTP ${retry.status}` }));
					throw new Error(err.error || `API error: ${retry.status}`);
				}
				if (retry.status === 204) return undefined as T;
				return retry.json();
			}
		}
		// result === 'invalid' (or retry still 401): the refresh token is
		// genuinely rejected — clear auth state and redirect to login.
		if (typeof localStorage !== 'undefined') {
			localStorage.removeItem('feednest_refresh_token');
		}
		accessToken = null;
		if (typeof window !== 'undefined') {
			window.location.href = '/auth/login';
		}
		throw new Error('Session expired');
	}

	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
		throw new Error(err.error || `API error: ${res.status}`);
	}

	if (res.status === 204) return undefined as T;
	return res.json();
}

// RefreshResult distinguishes a definitively invalid session (purge creds and
// log out) from a transient failure (network/5xx — keep creds, let the caller
// retry). Collapsing both to one boolean previously logged users out on a
// momentary blip even though their refresh token was still valid.
type RefreshResult = 'ok' | 'invalid' | 'transient';

let refreshPromise: Promise<RefreshResult> | null = null;

async function refreshTokenFn(): Promise<RefreshResult> {
	if (refreshPromise) return refreshPromise;
	refreshPromise = doRefresh().finally(() => {
		refreshPromise = null;
	});
	return refreshPromise;
}

async function doRefresh(): Promise<RefreshResult> {
	if (typeof localStorage === 'undefined') return 'invalid';
	const refreshTok = localStorage.getItem('feednest_refresh_token');
	if (!refreshTok) return 'invalid';

	try {
		const res = await fetch(`${API_BASE}/api/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshTok }),
		});
		// Only an explicit auth rejection invalidates the session; any other
		// non-ok status is transient and must not clear the token.
		if (res.status === 401 || res.status === 403) return 'invalid';
		if (!res.ok) return 'transient';
		const data = await res.json();
		accessToken = data.access_token;
		if (data.refresh_token) {
			localStorage.setItem('feednest_refresh_token', data.refresh_token);
		}
		return 'ok';
	} catch {
		return 'transient';
	}
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	del: <T>(path: string) => request<T>('DELETE', path),
};

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
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
	};

	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}

	const res = await fetch(`${API_BASE}${path}`, {
		method,
		headers,
		body: body ? JSON.stringify(body) : undefined,
	});

	if (res.status === 401 && accessToken) {
		const refreshed = await refreshTokenFn();
		if (refreshed) {
			headers['Authorization'] = `Bearer ${accessToken}`;
			const retry = await fetch(`${API_BASE}${path}`, {
				method,
				headers,
				body: body ? JSON.stringify(body) : undefined,
			});
			if (!retry.ok) throw new Error(`API error: ${retry.status}`);
			if (retry.status === 204) return undefined as T;
			return retry.json();
		}
	}

	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
		throw new Error(err.error || `API error: ${res.status}`);
	}

	if (res.status === 204) return undefined as T;
	return res.json();
}

let refreshPromise: Promise<boolean> | null = null;

async function refreshTokenFn(): Promise<boolean> {
	if (refreshPromise) return refreshPromise;
	refreshPromise = doRefresh().finally(() => {
		refreshPromise = null;
	});
	return refreshPromise;
}

async function doRefresh(): Promise<boolean> {
	if (typeof localStorage === 'undefined') return false;
	const refreshTok = localStorage.getItem('feednest_refresh_token');
	if (!refreshTok) return false;

	try {
		const res = await fetch(`${API_BASE}/api/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshTok }),
		});
		if (!res.ok) return false;
		const data = await res.json();
		accessToken = data.access_token;
		if (data.refresh_token) {
			localStorage.setItem('feednest_refresh_token', data.refresh_token);
		}
		return true;
	} catch {
		return false;
	}
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	del: <T>(path: string) => request<T>('DELETE', path),
};

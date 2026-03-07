const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
	accessToken = token;
}

export function getAccessToken(): string | null {
	return accessToken;
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

async function refreshTokenFn(): Promise<boolean> {
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

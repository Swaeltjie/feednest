import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { api, setAccessToken, getAccessToken } from '$lib/api/client';

interface User {
	id: number;
	username: string;
	email: string;
}

interface AuthState {
	user: User | null;
	isAuthenticated: boolean;
	loading: boolean;
}

function createAuthStore() {
	const { subscribe, set } = writable<AuthState>({
		user: null,
		isAuthenticated: false,
		loading: true,
	});

	return {
		subscribe,

		async login(username: string, password: string) {
			const data = await api.post<{ access_token: string; refresh_token: string; user: User }>(
				'/api/auth/login',
				{ username, password }
			);
			setAccessToken(data.access_token);
			if (browser) localStorage.setItem('feednest_refresh_token', data.refresh_token);
			set({ user: data.user, isAuthenticated: true, loading: false });
		},

		async register(username: string, email: string, password: string) {
			const data = await api.post<{ access_token: string; refresh_token: string; user: User }>(
				'/api/auth/register',
				{ username, email, password }
			);
			setAccessToken(data.access_token);
			if (browser) localStorage.setItem('feednest_refresh_token', data.refresh_token);
			set({ user: data.user, isAuthenticated: true, loading: false });
		},

		logout() {
			setAccessToken(null);
			if (browser) localStorage.removeItem('feednest_refresh_token');
			set({ user: null, isAuthenticated: false, loading: false });
		},

		async checkAuth() {
			if (!browser) {
				set({ user: null, isAuthenticated: false, loading: false });
				return;
			}
			const refreshTok = localStorage.getItem('feednest_refresh_token');
			if (!refreshTok) {
				set({ user: null, isAuthenticated: false, loading: false });
				return;
			}

			try {
				const data = await api.post<{ access_token: string; user: User }>('/api/auth/refresh', {
					refresh_token: refreshTok,
				});
				setAccessToken(data.access_token);
				set({ user: data.user, isAuthenticated: true, loading: false });
			} catch {
				set({ user: null, isAuthenticated: false, loading: false });
			}
		},

		async getUserCount(): Promise<number> {
			const data = await api.get<{ count: number }>('/api/auth/user-count');
			return data.count;
		},
	};
}

export const auth = createAuthStore();

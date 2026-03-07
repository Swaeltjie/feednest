import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface Article {
	id: number;
	feed_id: number;
	title: string;
	url: string;
	author: string;
	content_clean: string;
	content_raw: string;
	snippet: string;
	thumbnail_url: string;
	published_at: string | null;
	word_count: number;
	reading_time: number;
	is_read: boolean;
	is_starred: boolean;
	score: number;
	feed_title: string;
	feed_icon_url: string;
	tags: string[];
}

interface ArticlesResponse {
	articles: Article[];
	total: number;
	page: number;
	limit: number;
}

export interface ArticleFilters {
	status?: string;
	sort?: string;
	feed?: number;
	category?: number;
	tag?: string;
	search?: string;
	page?: number;
}

function createArticlesStore() {
	const { subscribe, set, update } = writable<{
		articles: Article[];
		total: number;
		loading: boolean;
	}>({ articles: [], total: 0, loading: false });

	return {
		subscribe,

		async load(filters: ArticleFilters = {}) {
			update((s) => ({ ...s, loading: true }));
			const params = new URLSearchParams();
			if (filters.status) params.set('status', filters.status);
			params.set('sort', filters.sort || 'smart');
			if (filters.feed) params.set('feed', String(filters.feed));
			if (filters.category) params.set('category', String(filters.category));
			if (filters.tag) params.set('tag', filters.tag);
			if (filters.search) params.set('search', filters.search);
			if (filters.page) params.set('page', String(filters.page));

			const data = await api.get<ArticlesResponse>(`/api/articles?${params}`);
			set({ articles: data.articles || [], total: data.total, loading: false });
		},

		async toggleRead(id: number, isRead: boolean) {
			await api.put(`/api/articles/${id}`, { is_read: isRead });
			update((s) => ({
				...s,
				articles: s.articles.map((a) => (a.id === id ? { ...a, is_read: isRead } : a)),
			}));
		},

		async toggleStar(id: number, isStarred: boolean) {
			await api.put(`/api/articles/${id}`, { is_starred: isStarred });
			update((s) => ({
				...s,
				articles: s.articles.map((a) => (a.id === id ? { ...a, is_starred: isStarred } : a)),
			}));
		},

		async dismiss(id: number) {
			await api.post(`/api/articles/${id}/dismiss`);
			update((s) => ({
				...s,
				articles: s.articles.filter((a) => a.id !== id),
				total: s.total - 1,
			}));
		},

		async getArticle(id: number): Promise<Article> {
			return api.get<Article>(`/api/articles/${id}`);
		},
	};
}

export const articles = createArticlesStore();

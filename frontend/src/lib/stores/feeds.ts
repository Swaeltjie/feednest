import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface Feed {
	id: number;
	url: string;
	title: string;
	site_url: string;
	icon_url: string;
	category_id: number | null;
	unread_count: number;
	last_error: string | null;
}

export interface Category {
	id: number;
	name: string;
	position: number;
}

function createFeedsStore() {
	const { subscribe, set } = writable<Feed[]>([]);

	return {
		subscribe,
		async load() {
			const feeds = await api.get<Feed[]>('/api/feeds');
			set(feeds || []);
		},
		async add(url: string, categoryId?: number) {
			await api.post('/api/feeds', { url, category_id: categoryId });
			await this.load();
		},
		async remove(id: number) {
			await api.del(`/api/feeds/${id}`);
			await this.load();
		},
		async update(id: number, data: { title?: string; category_id?: number | null }) {
			await api.put(`/api/feeds/${id}`, data);
			await this.load();
		},
		async retry(id: number) {
			await api.post(`/api/feeds/${id}/retry`);
		},
	};
}

function createCategoriesStore() {
	const { subscribe, set } = writable<Category[]>([]);

	return {
		subscribe,
		async load() {
			const cats = await api.get<Category[]>('/api/categories');
			set(cats || []);
		},
		async add(name: string) {
			await api.post('/api/categories', { name });
			await this.load();
		},
		async remove(id: number) {
			await api.del(`/api/categories/${id}`);
			await this.load();
		},
	};
}

export const feeds = createFeedsStore();
export const categories = createCategoriesStore();

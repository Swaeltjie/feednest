<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();

	onMount(async () => {
		await auth.checkAuth();
	});

	$effect(() => {
		if (!$auth.loading) {
			const isAuthPage = page.url.pathname.startsWith('/auth');
			if (!$auth.isAuthenticated && !isAuthPage) {
				goto('/auth/login');
			}
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if $auth.loading}
	<div class="min-h-screen flex flex-col items-center justify-center" style="background: var(--color-surface);">
		<h1 class="text-2xl font-bold accent-gradient-text mb-4">FeedNest</h1>
		<div class="w-6 h-6 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
	</div>
{:else}
	{@render children()}
{/if}

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
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-gray-500 dark:text-gray-400">Loading...</div>
	</div>
{:else}
	{@render children()}
{/if}

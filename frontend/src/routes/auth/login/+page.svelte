<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin() {
		error = '';
		loading = true;
		try {
			await auth.login(username, password);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="w-full max-w-md p-8 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
		<h1 class="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8">Sign in to FeedNest</h1>

		{#if error}
			<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg text-sm">{error}</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
				<input id="username" type="text" bind:value={username} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
				<input id="password" type="password" bind:value={password} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<button type="submit" disabled={loading}
				class="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg disabled:opacity-50 transition-colors">
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
			Don't have an account? <a href="/auth/register" class="text-blue-600 hover:underline">Register</a>
		</p>
	</div>
</div>

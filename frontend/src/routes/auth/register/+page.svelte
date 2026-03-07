<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleRegister() {
		error = '';
		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}

		loading = true;
		try {
			await auth.register(username, email, password);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Registration failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="w-full max-w-md p-8 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
		<h1 class="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8">Create your FeedNest account</h1>

		{#if error}
			<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg text-sm">{error}</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
				<input id="username" type="text" bind:value={username} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
				<input id="email" type="email" bind:value={email} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
				<input id="password" type="password" bind:value={password} required minlength="8"
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="confirm" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm Password</label>
				<input id="confirm" type="password" bind:value={confirmPassword} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<button type="submit" disabled={loading}
				class="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg disabled:opacity-50 transition-colors">
				{loading ? 'Creating account...' : 'Create Account'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
			Already have an account? <a href="/auth/login" class="text-blue-600 hover:underline">Sign in</a>
		</p>
	</div>
</div>

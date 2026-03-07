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

<div class="min-h-screen flex items-center justify-center auth-bg">
	<div class="w-full max-w-md p-8 glass rounded-2xl shadow-2xl fade-in-up">
		<h1 class="text-2xl font-bold text-center mb-2">
			<span class="accent-gradient-text">FeedNest</span>
		</h1>
		<p class="text-center text-sm text-white/60 mb-8">Sign in to your account</p>

		{#if error}
			<div class="mb-4 p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20">{error}</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-white/70 mb-1.5">Username</label>
				<input id="username" type="text" bind:value={username} required
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-white/70 mb-1.5">Password</label>
				<input id="password" type="password" bind:value={password} required
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<button type="submit" disabled={loading}
				class="w-full py-2.5 px-4 text-white font-medium rounded-xl accent-gradient
					hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity
					shadow-lg shadow-blue-500/25">
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-white/40">
			Don't have an account? <a href="/auth/register" class="text-[var(--color-accent)] hover:underline">Register</a>
		</p>
	</div>
</div>

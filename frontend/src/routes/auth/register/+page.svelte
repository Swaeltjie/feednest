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

<div class="min-h-screen flex items-center justify-center auth-bg">
	<div class="w-full max-w-md p-8 glass rounded-2xl shadow-2xl fade-in-up">
		<h1 class="text-2xl font-bold text-center mb-2">
			<span class="accent-gradient-text">FeedNest</span>
		</h1>
		<p class="text-center text-sm text-white/60 mb-8">Create your account</p>

		{#if error}
			<div class="mb-4 p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20">{error}</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-white/70 mb-1.5">Username</label>
				<input id="username" type="text" bind:value={username} required
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-white/70 mb-1.5">Email</label>
				<input id="email" type="email" bind:value={email} required
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-white/70 mb-1.5">Password</label>
				<input id="password" type="password" bind:value={password} required minlength="8"
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<div>
				<label for="confirm" class="block text-sm font-medium text-white/70 mb-1.5">Confirm Password</label>
				<input id="confirm" type="password" bind:value={confirmPassword} required
					class="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30
						focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-all" />
			</div>

			<button type="submit" disabled={loading}
				class="w-full py-2.5 px-4 text-white font-medium rounded-xl accent-gradient
					hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity
					shadow-lg shadow-blue-500/25">
				{loading ? 'Creating account...' : 'Create Account'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-white/40">
			Already have an account? <a href="/auth/login" class="text-[var(--color-accent)] hover:underline">Sign in</a>
		</p>
	</div>
</div>

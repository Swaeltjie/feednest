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

<div class="min-h-screen flex items-center justify-center auth-bg relative overflow-x-hidden overflow-y-auto py-8">
	<!-- Animated background orbs -->
	<div class="absolute inset-0 overflow-hidden pointer-events-none">
		<div class="absolute -top-40 -left-40 w-80 h-80 rounded-full bg-blue-500/10 blur-3xl animate-float-slow"></div>
		<div class="absolute top-1/4 -right-20 w-96 h-96 rounded-full bg-indigo-500/8 blur-3xl animate-float-slower"></div>
		<div class="absolute -bottom-32 left-1/3 w-72 h-72 rounded-full bg-cyan-500/8 blur-3xl animate-float-medium"></div>
	</div>

	<!-- Grid pattern overlay -->
	<div class="absolute inset-0 opacity-[0.03]" style="background-image: radial-gradient(circle at 1px 1px, white 1px, transparent 0); background-size: 40px 40px;"></div>

	<div class="relative z-10 w-full max-w-md px-4">
		<!-- Logo section -->
		<div class="text-center mb-8 fade-in-up">
			<div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl accent-gradient shadow-2xl shadow-blue-500/30 mb-4">
				<svg class="w-9 h-9 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 110-2 1 1 0 010 2z" />
				</svg>
			</div>
			<h1 class="text-3xl font-bold tracking-tight">
				<span class="accent-gradient-text">FeedNest</span>
			</h1>
			<p class="text-sm text-white/40 mt-1">Your intelligent feed reader</p>
		</div>

		<!-- Login card -->
		<div class="p-8 rounded-2xl border border-white/10 shadow-2xl fade-in-up" style="background: rgba(255,255,255,0.03); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px); animation-delay: 100ms;">
			<h2 class="text-lg font-semibold text-white/90 mb-1">Welcome back</h2>
			<p class="text-sm text-white/40 mb-6">Sign in to continue reading</p>

			{#if error}
				<div class="mb-4 p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20 flex items-center gap-2">
					<svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
					</svg>
					{error}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-5">
				<div>
					<label for="username" class="block text-xs font-medium uppercase tracking-wider text-white/50 mb-2">Username</label>
					<div class="relative">
						<div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
							<svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
							</svg>
						</div>
						<input id="username" type="text" bind:value={username} required placeholder="Enter username"
							class="w-full pl-10 pr-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/20
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent focus:bg-white/8 transition-all text-sm" />
					</div>
				</div>

				<div>
					<label for="password" class="block text-xs font-medium uppercase tracking-wider text-white/50 mb-2">Password</label>
					<div class="relative">
						<div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
							<svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
							</svg>
						</div>
						<input id="password" type="password" bind:value={password} required placeholder="Enter password"
							class="w-full pl-10 pr-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/20
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent focus:bg-white/8 transition-all text-sm" />
					</div>
				</div>

				<button type="submit" disabled={loading}
					class="w-full py-3 px-4 text-white font-semibold rounded-xl accent-gradient
						hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-all
						shadow-lg shadow-blue-500/25 hover:shadow-xl hover:shadow-blue-500/30 hover:-translate-y-0.5
						active:translate-y-0 text-sm tracking-wide">
					{#if loading}
						<span class="flex items-center justify-center gap-2">
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Signing in...
						</span>
					{:else}
						Sign In
					{/if}
				</button>
			</form>

			<div class="mt-6 pt-6 border-t border-white/5 text-center">
				<p class="text-sm text-white/30">
					Don't have an account? <a href="/auth/register" class="text-[var(--color-accent)] hover:text-[var(--color-accent-end)] transition-colors font-medium">Create one</a>
				</p>
			</div>
		</div>

		<!-- Footer -->
		<p class="text-center text-xs text-white/20 mt-8 fade-in-up" style="animation-delay: 200ms;">
			Self-hosted RSS reader with smart ranking
		</p>
	</div>
</div>

<style>
	@keyframes float-slow {
		0%, 100% { transform: translate(0, 0) scale(1); }
		33% { transform: translate(30px, -30px) scale(1.05); }
		66% { transform: translate(-20px, 20px) scale(0.95); }
	}
	@keyframes float-slower {
		0%, 100% { transform: translate(0, 0) scale(1); }
		50% { transform: translate(-40px, 30px) scale(1.1); }
	}
	@keyframes float-medium {
		0%, 100% { transform: translate(0, 0); }
		50% { transform: translate(30px, -40px); }
	}
	:global(.animate-float-slow) { animation: float-slow 20s ease-in-out infinite; }
	:global(.animate-float-slower) { animation: float-slower 25s ease-in-out infinite; }
	:global(.animate-float-medium) { animation: float-medium 18s ease-in-out infinite; }
</style>

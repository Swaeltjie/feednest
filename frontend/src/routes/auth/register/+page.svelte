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

<div class="min-h-screen flex items-center justify-center auth-bg relative overflow-hidden">
	<!-- Animated background orbs -->
	<div class="absolute inset-0 overflow-hidden pointer-events-none">
		<div class="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-indigo-500/10 blur-3xl animate-float-slow"></div>
		<div class="absolute top-1/3 -left-20 w-96 h-96 rounded-full bg-blue-500/8 blur-3xl animate-float-slower"></div>
		<div class="absolute -bottom-32 right-1/3 w-72 h-72 rounded-full bg-cyan-500/8 blur-3xl animate-float-medium"></div>
	</div>

	<div class="absolute inset-0 opacity-[0.03]" style="background-image: radial-gradient(circle at 1px 1px, white 1px, transparent 0); background-size: 40px 40px;"></div>

	<div class="relative z-10 w-full max-w-md px-4">
		<!-- Logo -->
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

		<!-- Register card -->
		<div class="p-8 rounded-2xl border border-white/10 shadow-2xl fade-in-up" style="background: rgba(255,255,255,0.03); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px); animation-delay: 100ms;">
			<h2 class="text-lg font-semibold text-white/90 mb-1">Create your account</h2>
			<p class="text-sm text-white/40 mb-6">Start reading smarter today</p>

			{#if error}
				<div class="mb-4 p-3 text-sm text-red-400 bg-red-500/10 rounded-xl border border-red-500/20 flex items-center gap-2">
					<svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
					</svg>
					{error}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4">
				<div>
					<label for="username" class="block text-xs font-medium uppercase tracking-wider text-white/50 mb-2">Username</label>
					<div class="relative">
						<div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
							<svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
							</svg>
						</div>
						<input id="username" type="text" bind:value={username} required placeholder="Choose a username"
							class="w-full pl-10 pr-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/20
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent focus:bg-white/8 transition-all text-sm" />
					</div>
				</div>

				<div>
					<label for="email" class="block text-xs font-medium uppercase tracking-wider text-white/50 mb-2">Email</label>
					<div class="relative">
						<div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
							<svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
							</svg>
						</div>
						<input id="email" type="email" bind:value={email} required placeholder="you@example.com"
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
						<input id="password" type="password" bind:value={password} required minlength="8" placeholder="Min 8 characters"
							class="w-full pl-10 pr-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/20
								focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent focus:bg-white/8 transition-all text-sm" />
					</div>
				</div>

				<div>
					<label for="confirm" class="block text-xs font-medium uppercase tracking-wider text-white/50 mb-2">Confirm Password</label>
					<div class="relative">
						<div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
							<svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
							</svg>
						</div>
						<input id="confirm" type="password" bind:value={confirmPassword} required placeholder="Repeat password"
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
							Creating account...
						</span>
					{:else}
						Create Account
					{/if}
				</button>
			</form>

			<div class="mt-6 pt-6 border-t border-white/5 text-center">
				<p class="text-sm text-white/30">
					Already have an account? <a href="/auth/login" class="text-[var(--color-accent)] hover:text-[var(--color-accent-end)] transition-colors font-medium">Sign in</a>
				</p>
			</div>
		</div>
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

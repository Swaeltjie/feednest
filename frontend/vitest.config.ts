import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [svelte({ hot: !process.env.VITEST })],
	test: {
		environment: 'jsdom',
		setupFiles: ['./vitest-setup.ts'],
		include: ['src/**/*.test.ts'],
		globals: true,
		alias: {
			'$lib': '/src/lib',
			'$app/environment': '/src/lib/__mocks__/app-environment.ts',
		},
	},
});

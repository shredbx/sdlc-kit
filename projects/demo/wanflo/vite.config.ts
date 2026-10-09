import { sveltekit } from '@sveltejs/kit/vite';
// from vitest/config, not vite — it is the variant that types the `test` block below
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// Port from the workspace port manager (wanflo-web = 4006); worktrees override via PORT.
		port: Number(process.env.PORT) || 4006,
		host: true
	},
	preview: {
		port: Number(process.env.PORT) || 4006,
		host: true
	},
	test: {
		// Unit tests only. tests/e2e/ belongs to Playwright — without this, vitest collects
		// those specs too and they fail on "Playwright Test did not expect test() to be
		// called here", which looks like a broken suite but is just the wrong runner.
		include: ['src/**/*.{test,spec}.{js,ts}'],
		exclude: ['tests/**', 'node_modules/**']
	}
});

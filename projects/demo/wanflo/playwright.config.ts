import { defineConfig, devices } from '@playwright/test';

// Playwright artifacts (traces, videos, failure and evidence screenshots) all land under
// <repo-root>/.playwright/wanflo-web/ — never beside this app. See scripts/playwright/artifacts.mjs
// for why that is resolved by walking up to .git rather than by counting ../ segments.
import { artifacts } from '../../../../../../../scripts/playwright/artifacts.mjs';

const ARTIFACTS = artifacts(import.meta.url, 'wanflo-web');
// Exported to the specs, which read SHOT_DIR for deliberate captures.
process.env.SHOT_DIR ??= ARTIFACTS.shotsDir;

// Webkit project gated behind WEBKIT=1 || CI — workspace convention (#882 WebKit quality gate).
const withWebkit = !!process.env.WEBKIT || !!process.env.CI;
const port = Number(process.env.PORT) || 4006;

export default defineConfig({
	outputDir: ARTIFACTS.outputDir,
	testDir: './tests/e2e',
	fullyParallel: true,
	reporter: 'list',
	use: {
		baseURL: `http://localhost:${port}`,
		trace: 'retain-on-failure'
	},
	// BAKE_ONLY skips the app build+preview: the asset bakes (dark wordmark, OG images,
	// favicons in screenshots.spec) render from file:// and need no server. This also breaks
	// the chicken-and-egg where the build prerenders a page that links an asset the bake is
	// about to generate — bake first with BAKE_ONLY, then run the full suite against the built app.
	webServer: process.env.BAKE_ONLY
		? undefined
		: {
				command: 'pnpm run build && pnpm run preview',
				port,
				reuseExistingServer: true,
				timeout: 180_000
			},
	projects: [
		{ name: 'chromium', use: { ...devices['Desktop Chrome'] } },
		...(withWebkit ? [{ name: 'webkit', use: { ...devices['Desktop Safari'] } }] : [])
	]
});

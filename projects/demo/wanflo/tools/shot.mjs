// Screenshot one section of the RUNNING site, both themes, at 2x.
//
//   pnpm exec node tools/shot.mjs <css-selector> <name> [path]
//
// Exists because reasoning about CSS is not evidence. The tech row's icons are cross-origin
// SVG masks served from media.wanflo.com; whether a browser actually paints them is a
// question only a browser answers.
import { chromium } from '@playwright/test';
const [sel, name, path = '/'] = process.argv.slice(2);
if (!sel || !name) { console.error('usage: shot.mjs <selector> <name> [path]'); process.exit(1); }
const base = process.env.BASE || 'http://localhost:4006';
const br = await chromium.launch();
for (const theme of ['light', 'dark']) {
	const p = await br.newPage({ viewport: { width: 1280, height: 1000 }, deviceScaleFactor: 2 });
	const failed = [];
	p.on('requestfailed', (r) => failed.push(r.url()));
	p.on('response', (r) => { if (r.status() >= 400) failed.push(`${r.status()} ${r.url()}`); });
	await p.addInitScript((t) => localStorage.setItem('wanflo-theme', t), theme);
	// Pre-answer the cookie banner unless we are deliberately looking at it — otherwise it
	// covers the bottom of the page and every footer screenshot is a photo of the banner.
	if (!process.env.BANNER)
		await p.addInitScript(() =>
			localStorage.setItem(
				'wanflo-analytics-consent',
				JSON.stringify({ accepted: true, version: 1, timestamp: new Date().toISOString() })
			)
		);
	await p.goto(base + path, { waitUntil: 'networkidle' });
	const el = p.locator(sel).first();
	await el.scrollIntoViewIfNeeded();
	await el.screenshot({ path: `/tmp/${name}-${theme}.png` });
	console.log(`/tmp/${name}-${theme}.png` + (failed.length ? `  ⚠ ${failed.length} failed requests: ${failed.slice(0,3)}` : '  (no failed requests)'));
	await p.close();
}
await br.close();

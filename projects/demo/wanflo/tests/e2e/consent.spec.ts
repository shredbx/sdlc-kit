// Cookie-consent + GA4 wiring. The behavior checks (banner shows / hides / persists,
// decline blocks tracking) run always. The "hit actually fires" check needs the real
// gtag.js from googletagmanager.com, so it's gated behind VERIFY_GA=1 (opt-in, keeps the
// default suite offline-safe). We intercept /g/collect so real gtag.js runs but the hit is
// stubbed 204 — proves the pipeline without polluting the live GA property.
import { test, expect } from '@playwright/test';

const MEASUREMENT_ID = 'G-K40Z39EBE4';

test('banner shows on first visit, hides + persists after a choice', async ({ page }) => {
	await page.goto('/');
	const banner = page.getByRole('dialog');
	await expect(banner).toBeVisible();
	await page.getByRole('button', { name: /accept/i }).click();
	await expect(banner).toBeHidden();
	// Choice persists — no banner on reload.
	await page.reload();
	await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('cookie policy page renders + reflects the stored choice', async ({ page }) => {
	// Pre-set consent so the settings panel shows "allowed".
	await page.goto('/');
	await page.getByRole('button', { name: /accept/i }).click();
	await page.goto('/cookies');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByText(/analytics allowed/i)).toBeVisible();
	// Withdraw flips it.
	await page.getByRole('button', { name: /withdraw|decline/i }).click();
	await expect(page.getByText(/analytics declined/i)).toBeVisible();
});

test('VERIFY_GA: accept → a real /g/collect hit fires with the measurement id', async ({ page }) => {
	test.skip(!process.env.VERIFY_GA, 'set VERIFY_GA=1 (needs network to googletagmanager.com)');
	// The only test here that talks to the real network for every step: it downloads
	// gtag.js (~435 KB) and then waits on five separate round trips. Run alongside three
	// other workers it comfortably outruns the default budget — which is contention, not a
	// site defect. Triple the timeout and give each poll room rather than chase a flake.
	test.slow();
	const collectHits: string[] = [];
	// Let real gtag.js load; capture + stub the outbound hit so nothing reaches GA.
	await page.route('**/g/collect*', (route) => {
		collectHits.push(route.request().url());
		route.fulfill({ status: 204, body: '' });
	});
	await page.route('**/collect?*', (route) => {
		collectHits.push(route.request().url());
		route.fulfill({ status: 204, body: '' });
	});

	await page.goto('/');
	await page.getByRole('button', { name: /accept/i }).click();
	// Navigation triggers a consent-granted page_view.
	await page.goto('/ru');
	await expect
		.poll(() => collectHits.filter((u) => u.includes(MEASUREMENT_ID)).length, { timeout: 20000 })
		.toBeGreaterThan(0);
});

test('VERIFY_GA: decline → no measurement hit fires', async ({ page }) => {
	test.skip(!process.env.VERIFY_GA, 'set VERIFY_GA=1 (needs network)');
	const collectHits: string[] = [];
	await page.route('**/g/collect*', (route) => {
		collectHits.push(route.request().url());
		route.fulfill({ status: 204, body: '' });
	});
	await page.goto('/');
	await page.getByRole('button', { name: /decline/i }).click();
	await page.goto('/th');
	await page.waitForTimeout(2500);
	// Consent Mode may emit a cookieless consent ping, but never a page_view event hit.
	expect(collectHits.filter((u) => u.includes('en=page_view')).length).toBe(0);
});

// The conversion events. Until this pass the site defined four event names and fired NONE
// of them: only page_view reached GA4, so "did anyone actually try to contact us" was an
// unanswerable question. These assert the wiring end to end against real gtag.js.
// This one test depends on a THIRD-PARTY network service for every step: it downloads the
// real gtag.js and then waits on five separate round trips to Google. Run alongside the
// other workers it intermittently misses the budget — verified flaky, not broken: it passes
// consistently on its own. Retries are the honest answer for a network-bound check; the
// assertions themselves stay strict, so a genuine regression still fails all three times.
test.describe.configure({ retries: 2 });

test('VERIFY_GA: the conversion events actually reach the wire', async ({ page, context }) => {
	test.skip(!process.env.VERIFY_GA, 'set VERIFY_GA=1 (needs network to googletagmanager.com)');

	const hits: string[] = [];
	const capture = (route: import('@playwright/test').Route) => {
		hits.push(route.request().url() + '|' + (route.request().postData() ?? ''));
		route.fulfill({ status: 204, body: '' });
	};
	await page.route('**/g/collect*', capture);
	await page.route('**/collect?*', capture);
	// The outbound links open in a new tab; stub them so the test never leaves the site.
	await context.route('**/wa.me/**', (r) => r.fulfill({ status: 200, body: 'ok' }));
	await context.route('**://be-in.ru/**', (r) => r.fulfill({ status: 200, body: 'ok' }));

	const fired = (name: string) => hits.filter((h) => h.includes(`en=${name}`)).length;

	await page.goto('/');
	await page.getByRole('button', { name: /accept/i }).click();

	// Everything else happens on ONE page. gtag.js loads asynchronously, so a click issued
	// straight after a navigation can queue an event the next navigation then discards —
	// which is a test artefact, not a site defect. Waiting for this page's own page_view
	// proves the pipeline is live before anything is measured against it.
	await page.goto('/website');
	await expect.poll(() => fired('page_view'), { timeout: 20000 }).toBeGreaterThan(0);

	// contact_click, from the floating dock
	await page.locator('.dock').click({ modifiers: ['Shift'] });
	await expect.poll(() => fired('contact_click'), { timeout: 20000 }).toBeGreaterThan(0);

	// package_click, from a package card
	await page.locator('.pack a[href*="wa.me"]').first().click({ modifiers: ['Shift'] });
	await expect.poll(() => fired('package_click'), { timeout: 20000 }).toBeGreaterThan(0);

	// work_visit, following a card out to a live client site
	await page.locator('.work a[href^="http"]').first().click({ modifiers: ['Shift'] });
	await expect.poll(() => fired('work_visit'), { timeout: 20000 }).toBeGreaterThan(0);

	// lang_switch, from the header switcher
	await page.locator('.lang summary').click();
	await page.locator('.lang-menu a[hreflang="th"]').click();
	await expect.poll(() => fired('lang_switch'), { timeout: 20000 }).toBeGreaterThan(0);
});

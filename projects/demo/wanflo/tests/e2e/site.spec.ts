// Wanflo v2 — browser-observable behaviour (task 2608-005).
//
// These assert STRUCTURE and INVARIANTS, never specific copy strings, so editing
// content/*.yml — which is the whole point of the content spine — never turns them red.
import { expect, test } from '@playwright/test';

const PAGES = ['/', '/website', '/mobile', '/contacts'];

// ── TC8 / TC17 — locale routing through the [[lang]] group ───────────────────
test.describe('routing', () => {
	test('every public page resolves in all three locales', async ({ request }) => {
		for (const prefix of ['', '/th', '/ru']) {
			for (const page of PAGES) {
				const url = prefix + (page === '/' ? '' : page) || '/';
				const res = await request.get(url);
				expect(res.status(), `${url} should be 200`).toBe(200);
			}
		}
	});

	test('an unknown language prefix 404s rather than rendering an empty locale', async ({
		request
	}) => {
		for (const bad of ['/de', '/fr', '/en']) {
			expect((await request.get(bad)).status(), `${bad} should 404`).toBe(404);
		}
	});

	// TC12 — the retired surfaces must be gone from the public site
	test('retired surfaces are unreachable', async ({ request }) => {
		for (const gone of ['/packages', '/admin-sitemap.txt']) {
			expect((await request.get(gone)).status(), `${gone} should be gone`).toBe(404);
		}
	});
});

// ── TC9 / TC18 — the header ──────────────────────────────────────────────────
test.describe('navigation', () => {
	test('the packages dropdown opens and links to both anchors', async ({ page }) => {
		await page.goto('/');
		const drop = page.locator('.drop');
		await drop.locator('summary').click();
		await expect(drop.locator('a[href="/website#packages"]')).toBeVisible();
		await expect(drop.locator('a[href="/mobile#packages"]')).toBeVisible();
	});

	test('the dropdown is a native details, so it works without JavaScript', async ({
		browser
	}) => {
		const ctx = await browser.newContext({ javaScriptEnabled: false });
		const page = await ctx.newPage();
		await page.goto('/');
		// No JS at all: opening the disclosure is a browser behaviour, not a script.
		await page.locator('.drop summary').click();
		await expect(page.locator('.drop a[href="/website#packages"]')).toBeVisible();
		await ctx.close();
	});

	test('below 760px the links collapse to a burger, language and theme stay in the bar', async ({
		page
	}) => {
		await page.setViewportSize({ width: 375, height: 760 });
		await page.goto('/');
		await expect(page.locator('.nav-links')).toBeHidden();
		await expect(page.locator('.burger')).toBeVisible();
		await expect(page.locator('.lang summary')).toBeVisible();
		await expect(page.locator('button.theme-toggle')).toBeVisible();

		await page.locator('.burger summary').click();
		await expect(page.locator('.burger-menu a[href="/website"]')).toBeVisible();
		await expect(page.locator('.burger-menu a[href="/contacts"]')).toBeVisible();
	});
});

// ── TC5 — home counts are derived, never typed ───────────────────────────────
test.describe('home', () => {
	test('door badges report the real number of shipped projects', async ({ page, request }) => {
		await page.goto('/');
		const llms = await (await request.get('/llms.txt')).text();
		// llms.txt states the counts from the same source; the badges must agree with it.
		const webCount = Number(llms.match(/shipped work — websites \((\d+)\)/i)?.[1]);
		const mobileCount = Number(llms.match(/shipped work — mobile apps \((\d+)\)/i)?.[1]);
		expect(webCount).toBeGreaterThan(0);
		expect(mobileCount).toBeGreaterThan(0);

		const badges = page.locator('.chip');
		await expect(badges.first()).toContainText(String(webCount));
		await expect(badges.nth(1)).toContainText(String(mobileCount));
	});

	test('the retired v1 platform bands are gone', async ({ page }) => {
		await page.goto('/');
		for (const id of ['#problem', '#journey', '#connect', '#inside', '#re']) {
			await expect(page.locator(id)).toHaveCount(0);
		}
	});
});

// ── TC6 / TC7 / TC10 — the portfolio edge cases ──────────────────────────────
test.describe('work cards', () => {
	test('a project without a url renders no anchor, and none is empty', async ({ page }) => {
		await page.goto('/website');
		const cards = page.locator('.work');
		await expect(cards.first()).toBeVisible();

		// Whatever the content says, no card may contain a link with an empty or "#" href.
		const hrefs = await page.locator('.work a').evaluateAll((els) =>
			els.map((e) => (e as HTMLAnchorElement).getAttribute('href'))
		);
		for (const h of hrefs) {
			expect(h, 'no work card may render an empty anchor').toBeTruthy();
			expect(h).not.toBe('#');
		}
	});

	test('every card shows either real screenshots or a placeholder, and no image 404s', async ({
		page
	}) => {
		const failed: string[] = [];
		page.on('response', (r) => {
			if (r.request().resourceType() === 'image' && r.status() >= 400) failed.push(r.url());
		});
		for (const path of ['/website', '/mobile']) {
			await page.goto(path);
			await page.evaluate(() =>
				document.querySelectorAll('img').forEach((i) => (i.loading = 'eager'))
			);
			await page.waitForLoadState('networkidle');

			// Every card has EITHER at least one real shot or the placeholder — never neither.
			for (const card of await page.locator('.work').all()) {
				const shots = await card.locator('.shots img').count();
				const empty = await card.locator('.shot-empty').count();
				expect(shots + empty, `${path}: a card rendered no shot and no placeholder`)
					.toBeGreaterThan(0);
			}
		}
		expect(failed, `broken images: ${failed.join(', ')}`).toHaveLength(0);
	});

	// The gap that started this pass: the portfolio described every project and the site
	// described none of them.
	test('every card carries a description', async ({ page }) => {
		for (const path of ['/website', '/mobile']) {
			await page.goto(path);
			const cards = await page.locator('.work').count();
			expect(await page.locator('.work .note').count(), `${path} is missing descriptions`)
				.toBe(cards);
		}
	});

	// "Archived" and "has no public link" are different facts. Labelling every linkless app
	// archived states something untrue, so the chip must be rarer than the missing links.
	test('the archived chip appears only where the content says archived', async ({ page }) => {
		await page.goto('/website');
		const archived = await page.locator('.work .chip-quiet').count();
		expect(archived).toBeGreaterThan(0);

		await page.goto('/mobile');
		// No mobile app is archived; they simply have no store link yet.
		expect(await page.locator('.work .chip-quiet').count()).toBe(0);
		const linkless = await page.locator('.work').count() -
			(await page.locator('.work a[href^="http"]').count());
		expect(linkless, 'the mobile apps have no public links, and none is archived')
			.toBeGreaterThan(0);
	});

	// A card holding two shots must not spill them outside its own padding box.
	test('a card with two screenshots keeps both inside the card', async ({ page }) => {
		await page.goto('/mobile');
		await page.evaluate(() => document.querySelectorAll('img').forEach((i) => (i.loading = 'eager')));
		await page.waitForLoadState('networkidle');

		let pairs = 0;
		for (const card of await page.locator('.work').all()) {
			const imgs = await card.locator('.shots img').all();
			if (imgs.length < 2) continue;
			pairs++;
			const box = await card.boundingBox();
			for (const img of imgs) {
				const b = await img.boundingBox();
				expect(b!.x, 'a shot escaped the card on the left').toBeGreaterThanOrEqual(box!.x);
				expect(b!.x + b!.width, 'a shot escaped the card on the right')
					.toBeLessThanOrEqual(box!.x + box!.width + 1);
			}
		}
		expect(pairs, 'no card rendered a pair of screenshots').toBeGreaterThan(0);
	});

	test('a mobile screenshot is never rendered wider than its natural width', async ({ page }) => {
		await page.goto('/mobile');
		await page.evaluate(() => document.querySelectorAll('img').forEach((i) => (i.loading = 'eager')));
		await page.waitForLoadState('networkidle');
		const imgs = page.locator('.work.framed .shots img');
		for (let i = 0; i < (await imgs.count()); i++) {
			const img = imgs.nth(i);
			const natural = await img.evaluate((e) => (e as HTMLImageElement).naturalWidth);
			const rendered = (await img.boundingBox())?.width ?? 0;
			if (natural > 0) {
				expect(rendered, 'a portfolio shot must never be upscaled').toBeLessThanOrEqual(
					natural + 1
				);
			}
		}
	});
});

// ── TC11 / TC22 — contact ────────────────────────────────────────────────────
test.describe('contacts', () => {
	test('the page collects nothing — no form, no input', async ({ page }) => {
		await page.goto('/contacts');
		await expect(page.locator('form')).toHaveCount(0);
		await expect(page.locator('input')).toHaveCount(0);
		await expect(page.locator('textarea')).toHaveCount(0);
	});

	test('every WhatsApp CTA targets wa.me with a prefilled greeting', async ({ page }) => {
		for (const path of PAGES) {
			await page.goto(path);
			const links = page.locator('a[href*="wa.me"]');
			expect(await links.count(), `${path} should offer WhatsApp`).toBeGreaterThan(0);
			const href = await links.first().getAttribute('href');
			expect(href).toMatch(/^https:\/\/wa\.me\/\d+\?text=.+/);
		}
	});

	test('the greeting is localized per locale', async ({ page }) => {
		await page.goto('/');
		const en = await page.locator('a[href*="wa.me"]').first().getAttribute('href');
		await page.goto('/th');
		const th = await page.locator('a[href*="wa.me"]').first().getAttribute('href');
		// Same number in both; the text parameter is whatever that locale's copy says.
		expect(en?.split('?')[0]).toBe(th?.split('?')[0]);
		expect(th).toContain('?text=');
	});

	test('the email stays reachable with JavaScript disabled', async ({ browser }) => {
		const ctx = await browser.newContext({ javaScriptEnabled: false });
		const page = await ctx.newPage();
		await page.goto('/contacts');
		await expect(page.locator('a[href^="mailto:"]').first()).toBeVisible();
		await ctx.close();
	});
});

// ── TC13 / TC14 / TC20 / TC23 — crawl surfaces ───────────────────────────────
test.describe('seo', () => {
	test('every page has a unique title and ProfessionalService structured data', async ({
		page
	}) => {
		const titles = new Set<string>();
		for (const path of PAGES) {
			await page.goto(path);
			const title = await page.title();
			expect(title.length, `${path} needs a title`).toBeGreaterThan(10);
			titles.add(title);

			const ld = await page.locator('script[type="application/ld+json"]').first().textContent();
			expect(ld).toContain('ProfessionalService');

			const canonical = await page.locator('link[rel=canonical]').getAttribute('href');
			expect(canonical).toContain('https://www.wanflo.com');
		}
		expect(titles.size, 'each page needs its OWN title').toBe(PAGES.length);
	});

	test('the offer pages carry Service and FAQPage matching what is visible', async ({ page }) => {
		for (const path of ['/website', '/mobile']) {
			await page.goto(path);
			const blocks = await page
				.locator('script[type="application/ld+json"]')
				.allTextContents();
			const all = blocks.join('');
			expect(all).toContain('"FAQPage"');
			expect(all).toContain('"Service"');

			// the visible FAQ count must equal the marked-up question count
			const visible = await page.locator('.qa').count();
			const marked = (all.match(/"@type":"Question"/g) ?? []).length;
			expect(marked).toBe(visible);
		}
	});

	// TC14 — a free-text price is not machine-readable; a fabricated one is worse than none
	test('no price or offer is ever emitted into structured data', async ({ page }) => {
		for (const path of PAGES) {
			await page.goto(path);
			const all = (
				await page.locator('script[type="application/ld+json"]').allTextContents()
			).join('');
			expect(all).not.toContain('"price"');
			expect(all).not.toContain('"offers"');
			expect(all).not.toContain('priceCurrency');
		}
	});

	test('the sitemap lists exactly the public pages per locale, and nothing retired', async ({
		request
	}) => {
		const xml = await (await request.get('/sitemap.xml')).text();
		const locs = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);

		// PAGES + /cookies, in each of the three locales.
		expect(locs).toHaveLength((PAGES.length + 1) * 3);
		expect(locs).toContain('https://www.wanflo.com/');
		expect(locs).toContain('https://www.wanflo.com/website');
		expect(locs).toContain('https://www.wanflo.com/th/mobile');
		for (const retired of ['/packages', 'admin-sitemap']) {
			expect(locs.some((l) => l.includes(retired))).toBe(false);
		}
		for (const lang of ['en', 'ru', 'th', 'x-default']) {
			expect(xml).toContain(`hreflang="${lang}"`);
		}
	});

	test('robots and llms describe the current company, not the retired platform', async ({
		request
	}) => {
		const robots = await (await request.get('/robots.txt')).text();
		expect(robots).toContain('User-agent: *');
		expect(robots).toContain('Sitemap: https://www.wanflo.com/sitemap.xml');

		const llms = await (await request.get('/llms.txt')).text();
		expect(llms).toContain('web and mobile development company');
		// the retired positioning must not survive anywhere on the crawl surface
		expect(llms.toLowerCase()).not.toContain('business operating system');
	});
});

// ── the floating contact dock ────────────────────────────────────────────────
// It is the site's only always-visible conversion affordance, and it is the slot the AI
// chat panel will occupy later, so its RULES are what these assert — not its styling.
test.describe('contact dock', () => {
	// Accepting once is what a RETURNING visitor sees. The first-visit case — banner still
	// up — is asserted separately below, and is the case that matters most: it is the one
	// an earlier version of this suite got backwards.
	const seen = async (page: import('@playwright/test').Page) =>
		page.addInitScript(() =>
			localStorage.setItem(
				'wanflo-analytics-consent',
				JSON.stringify({ accepted: true, version: 1, timestamp: new Date().toISOString() })
			)
		);

	test('appears on every page except the one it points at', async ({ page }) => {
		await seen(page);
		for (const path of ['/', '/website', '/mobile', '/th', '/ru/mobile']) {
			await page.goto(path);
			await expect(page.locator('.dock'), `${path} should offer the dock`).toBeVisible();
		}
		await page.goto('/contacts');
		await expect(page.locator('.dock'), '/contacts must not shortcut to itself').toHaveCount(0);
	});

	// REGRESSION. The dock used to HIDE itself whenever the cookie banner was up, so the
	// site's only always-available contact link was invisible to every first-time visitor —
	// which is everyone who matters. The old test asserted the hiding and therefore passed
	// while the feature was broken. This asserts the opposite, deliberately.
	test('stays VISIBLE on a first visit, above the cookie banner, never behind it', async ({
		page
	}) => {
		await page.goto('/'); // no stored consent → banner shows
		const banner = page.locator('.cookie-banner');
		const dock = page.locator('.dock');
		await expect(banner).toBeVisible();
		await expect(dock, 'a first-time visitor must still be able to message').toBeVisible();

		// Above, not overlapping: the dock's bottom edge clears the banner's top edge.
		const d = (await dock.boundingBox())!;
		const b = (await banner.boundingBox())!;
		expect(d.y + d.height, 'the dock is sitting on the cookie banner').toBeLessThanOrEqual(
			b.y + 1
		);

		// And it drops back down once the banner is answered.
		await page.locator('.cookie-banner .c-accept').click();
		await expect(banner).toHaveCount(0);
		await expect(dock).toBeVisible();
		const after = (await dock.boundingBox())!;
		expect(after.y, 'the dock should return to the corner').toBeGreaterThan(d.y);
	});

	// The dock is a real link on first paint, before any JS runs — so it must be in the
	// server-rendered HTML, not conjured by hydration.
	test('is in the server-rendered HTML, not added by hydration', async ({ request }) => {
		const html = await (await request.get('/website')).text();
		expect(html, 'the dock must be prerendered').toContain('class="dock');
	});

	test('is a real WhatsApp link, present without JavaScript', async ({ browser }) => {
		const ctx = await browser.newContext({ javaScriptEnabled: false });
		const page = await ctx.newPage();
		await page.goto('/website');
		const href = await page.locator('.dock').getAttribute('href');
		expect(href, 'the dock must work with JS off').toMatch(/^https:\/\/wa\.me\/\d+\?text=.+/);
		await ctx.close();
	});

	test('keeps an accessible name and a 44px tap target on a phone', async ({ page }) => {
		await seen(page);
		await page.setViewportSize({ width: 375, height: 760 });
		await page.goto('/website');
		const dock = page.locator('.dock');
		expect(await dock.getAttribute('aria-label')).toBeTruthy();
		const box = await dock.boundingBox();
		expect(box!.height, 'below the accessible minimum tap size').toBeGreaterThanOrEqual(44);
		// …and it must not sit on top of the footer content it overlaps.
		expect(box!.y + box!.height).toBeLessThanOrEqual(760);
	});
});

// ── the language switcher ────────────────────────────────────────────────────
test.describe('language', () => {
	test('keeps you on the page you are reading', async ({ page }) => {
		for (const [from, to, expected] of [
			['/website', 'th', '/th/website'],
			['/mobile', 'ru', '/ru/mobile'],
			['/th/contacts', 'en', '/contacts']
		] as const) {
			await page.goto(from);
			await page.locator('.lang summary').click();
			await page.locator(`.lang-menu a[hreflang="${to}"]`).click();
			await expect(page).toHaveURL(new RegExp(`${expected}$`));
		}
	});
});

// ── head / SEO surfaces ──────────────────────────────────────────────────────
test.describe('head', () => {
	test('declares a manifest, an icon and a theme colour', async ({ page, request }) => {
		await page.goto('/');
		await expect(page.locator('link[rel="manifest"]')).toHaveCount(1);
		await expect(page.locator('link[rel="apple-touch-icon"]')).toHaveCount(1);
		const theme = await page.locator('meta[name="theme-color"]').getAttribute('content');
		expect(theme, 'theme-color must be a real colour, not an unfilled placeholder').toMatch(
			/^#[0-9a-f]{6}$/i
		);

		const res = await request.get('/manifest.webmanifest');
		expect(res.status()).toBe(200);
		const manifest = await res.json();
		expect(manifest.name).toBeTruthy();
		expect(manifest.icons.length).toBeGreaterThan(1);
		for (const icon of manifest.icons) {
			expect((await request.get(icon.src)).status(), `${icon.src} 404s`).toBe(200);
		}
	});

	test('gives social scrapers the image dimensions they lay out with', async ({ page }) => {
		await page.goto('/website');
		await expect(page.locator('meta[property="og:image:width"]')).toHaveAttribute(
			'content',
			'1200'
		);
		await expect(page.locator('meta[property="og:image:height"]')).toHaveAttribute(
			'content',
			'630'
		);
		const alt = await page.locator('meta[property="og:image:alt"]').getAttribute('content');
		expect(alt).toBeTruthy();
	});

	test('never ships an unresolved template placeholder', async ({ request }) => {
		for (const path of ['/', '/th', '/ru/mobile', '/contacts']) {
			const html = await (await request.get(path)).text();
			expect(html, `${path} leaked a placeholder`).not.toMatch(/%wanflo\.[a-zA-Z]+%/);
			expect(html, `${path} leaked an unresolved copy variable`).not.toContain('{name}');
		}
	});
});

// ── packages ─────────────────────────────────────────────────────────────────
test.describe('packages', () => {
	test('every package card offers a way to act on it', async ({ page }) => {
		for (const path of ['/website', '/mobile']) {
			await page.goto(path);
			const cards = page.locator('.pack');
			const n = await cards.count();
			expect(n, `${path} renders no packages`).toBeGreaterThan(0);
			for (let i = 0; i < n; i++) {
				const link = cards.nth(i).locator('a[href*="wa.me"]');
				await expect(link, `${path} package ${i} is a dead end`).toHaveCount(1);
				// The message must already name the package, so the reply has context.
				const href = decodeURIComponent((await link.getAttribute('href')) ?? '');
				const name = (await cards.nth(i).locator('h3').textContent())?.trim() ?? '';
				expect(href, 'the prefill should name the package').toContain(name);
			}
		}
	});
});

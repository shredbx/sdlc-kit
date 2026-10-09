// Regenerate the per-locale Open Graph share images.
//
//   make -f Makefile.d/workflows/check.mk run \
//        DIR=clients/wanflo/projects/wanflo/apps/web/svelte CMD="pnpm exec node tools/og/shoot.mjs"
//
// WHY IT RENDERS RATHER THAN DRAWS. The previous og.png/og-ru.png/og-th.png were v1
// artefacts: the old cyan mark, "BUSINESS OPERATING SYSTEM", "Run your whole business from
// one back-office", a "Book a walkthrough" CTA that exists nowhere on this site, and the
// FlowWire graphic that was deleted in 2026-08. Nobody noticed because nothing rendered
// them — they were hand-made once and then carried forward as "existing assets".
//
// This script removes that failure mode. It composes the card from THE SITE'S OWN sources —
// the real lockup PNG from static/brand, the real woff2 files from static/fonts, the real
// palette tokens from app.css, and the real headline strings from content/copy/<lang>.yml —
// and screenshots it with the Playwright already in this repo. Change the tagline in the
// copy deck, re-run, and the share image follows. It can no longer drift on its own.
//
// FORMAT: PNG, deliberately. LinkedIn does not list WebP among its supported og:image
// types, and a share image that fails to render on one network is worth more bytes than a
// smaller file that does.
import { chromium } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { parse } from 'yaml';
import { readFileSync, writeFileSync, mkdirSync, statSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const APP = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const read = (p) => readFileSync(resolve(APP, p), 'utf8');
const b64 = (p) => readFileSync(resolve(APP, p)).toString('base64');

const site = parse(read('content/site.yml'));
const copy = Object.fromEntries(
	['en', 'th', 'ru'].map((l) => [l, parse(read(`content/copy/${l}.yml`))])
);

// Everything is inlined as a data: URI so the page has zero network dependency and cannot
// screenshot a half-loaded font — the classic way an OG generator ships a broken image.
const FONTS = {
	latin: b64('static/fonts/nunito-latin.woff2'),
	cyrillic: b64('static/fonts/nunito-cyrillic.woff2'),
	thai: b64('static/fonts/noto-thai.woff2')
};
// The lockup, cut straight out of the finished brand board (see tools/cut-assets.sh). It
// arrives WITH its own flat navy background, which is why the card's background is that same
// navy: the crop then has no visible edge at all. The previous source here was
// logo-lockup-dark.png, trimmed off the transparent sheet on an alpha threshold — it carried
// a dark halo around the mark that was obvious at 1:1 and got these images rejected.
const LOCKUP = b64('static/brand/og-lockup.png');

// Sampled from the brand board's own top panel — srgb(5..7,19..21,45..48) across the whole
// area. The card background MUST be this, not site.yml's themeColor.dark (#0a1626): the
// lockup crop brings its own navy with it, and any other value would draw a rectangle
// around it.
const NAVY = '#06132f';
const ACCENT = '#4d93ff';

const card = (locale) => {
	const t = (k) => (copy[locale][k] ?? copy.en[k] ?? '').replaceAll('{name}', site.name);
	// Thai sets taller than Latin at the same size, and its headline is the longest of the
	// three; Russian is longer than English. One size per script, so no locale ever wraps to
	// a fourth line or overflows the card.
	const h1 = locale === 'th' ? 46 : locale === 'ru' ? 50 : 52;
	return `<!doctype html><html lang="${locale}"><head><meta charset="utf-8"><style>
@font-face{font-family:'Nunito';font-weight:400 900;font-display:block;
  src:url(data:font/woff2;base64,${FONTS.latin}) format('woff2');
  unicode-range:U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,
    U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD;}
@font-face{font-family:'Nunito';font-weight:400 900;font-display:block;
  src:url(data:font/woff2;base64,${FONTS.cyrillic}) format('woff2');
  unicode-range:U+0301,U+0400-045F,U+0490-0491,U+04B0-04B1,U+2116;}
@font-face{font-family:'Noto Sans Thai';font-weight:400 800;font-display:block;
  src:url(data:font/woff2;base64,${FONTS.thai}) format('woff2');}
*{margin:0;padding:0;box-sizing:border-box}
body{width:1200px;height:630px;overflow:hidden;background:${NAVY};
  font-family:'Nunito','Noto Sans Thai',system-ui,sans-serif;-webkit-font-smoothing:antialiased}
.card{position:relative;width:1200px;height:630px;background:${NAVY};
  display:grid;grid-template-columns:500px 1fr;align-items:center;gap:24px;padding:0 40px}
/* the accent rule from the brand board, used as the card's own top edge */
.rule{position:absolute;top:0;left:0;right:0;height:5px;
  background:linear-gradient(90deg,${ACCENT},rgba(77,147,255,0) 70%)}
/* The crop's navy IS the card's navy, so it has no edge. No border, no shadow, no rounding
   — anything drawn around it would reveal the rectangle that is currently invisible. */
.lockup{width:500px;height:auto;display:block;margin-left:-8px}
/* A hairline echoing the divider the brand board itself uses between mark and wordmark. */
.sep{position:absolute;left:556px;top:150px;bottom:150px;width:1px;
  background:linear-gradient(180deg,rgba(255,255,255,0),rgba(255,255,255,.22),rgba(255,255,255,0))}
h1{font-size:${h1}px;line-height:1.16;font-weight:800;color:#fff;letter-spacing:-.4px}
.lead{margin-top:18px;font-size:23px;line-height:1.5;font-weight:600;color:#9fb1cc}
.dom{position:absolute;right:72px;bottom:42px;font-size:21px;font-weight:800;color:${ACCENT}}
</style></head><body><div class="card">
  <div class="rule"></div>
  <img class="lockup" src="data:image/png;base64,${LOCKUP}" alt="">
  <div class="sep"></div>
  <div>
    <h1>${t('home.hero.title')}</h1>
    <p class="lead">${t('home.hero.lead')}</p>
  </div>
  <div class="dom">${site.domain}</div>
</div></body></html>`;
};

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1200, height: 630 }, deviceScaleFactor: 1 });
// Output goes to cdn-upload/, the single local home of everything media.wanflo.com
// serves. There is deliberately no copy under static/.
mkdirSync(resolve(APP, 'cdn-upload/og'), { recursive: true });

for (const [locale, file] of Object.entries(site.og)) {
	const html = card(locale);
	// Per-locale temp file so each card stays inspectable after the run.
	const tmp = `/tmp/wanflo-og-${locale}.html`;
	writeFileSync(tmp, html);
	await page.goto(`file://${tmp}`);
	await page.evaluate(() => document.fonts.ready);
	const out = resolve(APP, 'cdn-upload' + file);
	await page.screenshot({ path: out, type: 'png' });
	// Playwright's PNG carries metadata chunks and a fast-but-loose compression level.
	// Stripping and recompressing is lossless and takes ~10% off. It lives HERE rather than
	// in a separate manual pass so one command always produces the final artefact — the
	// exact drift that let the v1 share images survive three redesigns.
	execFileSync('magick', [out, '-strip', '-define', 'png:compression-level=9', out]);
	console.log(`${file}  ${locale}  ${Math.round(statSync(out).size / 1024)}K`);
}
await browser.close();

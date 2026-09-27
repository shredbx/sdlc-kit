// @vitest-environment jsdom
//
// ChatMessage security-hardening contract (NFR — XSS sink). Assistant/system content is
// LLM output rendered through `@humanspeak/svelte-markdown`; user content stays on the
// literal `<p>` path. This suite proves the three protections layered on the markdown
// sink, at BOTH levels:
//
//   • Pure level — `makeImageSrcSanitizer` (the image-only host allowlist on top of the
//     library's default scheme allowlist) and `BLOCKED_HTML_TAGS`, tested as plain
//     functions (no DOM) so the policy is asserted directly.
//   • Rendered level — the real component mounted in jsdom (Svelte 5 mount/flushSync,
//     no @testing-library, matching Checkbox/TabbedPageShell): the hostile bytes must be
//     ABSENT from / neutralized in the produced DOM, and benign markdown must still render.
//
// Same-origin is read from the live `window.location.host` so the test is independent of
// the jsdom default. A markdown `![]()` image is lazy in the library renderer, but with
// no IntersectionObserver in jsdom it resolves `src` during onMount — which Svelte 5 fires
// synchronously inside mount() — so `src` is observable right after flushSync.

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import { defaultSanitizeUrl } from '@humanspeak/svelte-markdown';
import type { SanitizeContext } from '@humanspeak/svelte-markdown';
import { UnsupportedHTML } from '@humanspeak/svelte-markdown';
import ChatMessage from './ChatMessage.svelte';
import {
	BLOCKED_HTML_TAGS,
	buildBlockedHtmlRenderers,
	makeImageSrcSanitizer
} from './ChatMessage.security';

// ── Pure: image host allowlist ──────────────────────────────────────────────────────
describe('makeImageSrcSanitizer — image host allowlist', () => {
	const IMG: SanitizeContext = { type: 'image', tag: 'img' };
	const HTML_IMG: SanitizeContext = { type: 'html', tag: 'img' };
	const LINK: SanitizeContext = { type: 'link', tag: 'a' };
	const ORIGIN = window.location.host; // jsdom's current host (same-origin)

	it('allows relative image URLs (same-origin by construction)', () => {
		const s = makeImageSrcSanitizer();
		expect(s('/media/cover.png', IMG)).toBe('/media/cover.png');
		expect(s('./thumb.jpg', IMG)).toBe('./thumb.jpg');
		expect(s('img.png', IMG)).toBe('img.png');
	});

	it('allows an absolute same-origin image URL', () => {
		const s = makeImageSrcSanitizer();
		const url = `https://${ORIGIN}/media/cover.png`;
		expect(s(url, IMG)).toBe(url);
	});

	it('DROPS an off-origin image URL not on the allowlist (→ empty src)', () => {
		const s = makeImageSrcSanitizer();
		expect(s('https://evil.example/beacon.png', IMG)).toBe('');
		// protocol-relative off-origin host is also blocked
		expect(s('//evil.example/beacon.png', IMG)).toBe('');
	});

	it('allows an off-origin image URL whose host IS on the allowlist', () => {
		const s = makeImageSrcSanitizer(['cdn.example.com']);
		const url = 'https://cdn.example.com/p/1.jpg';
		expect(s(url, IMG)).toBe(url);
		// case-insensitive + tolerates a scheme/trailing slash in the allowlist entry
		const s2 = makeImageSrcSanitizer(['https://CDN.example.com/']);
		expect(s2('https://cdn.example.com/p/1.jpg', IMG)).toBe('https://cdn.example.com/p/1.jpg');
	});

	it('applies the host allowlist to raw <img> (type:html) too', () => {
		const s = makeImageSrcSanitizer(['cdn.example.com']);
		expect(s('https://evil.example/x.png', HTML_IMG)).toBe('');
		expect(s('https://cdn.example.com/x.png', HTML_IMG)).toBe('https://cdn.example.com/x.png');
	});

	it('does NOT host-restrict non-image URLs (links keep the default scheme check only)', () => {
		const s = makeImageSrcSanitizer(); // empty allowlist
		const link = 'https://anywhere.example/page';
		expect(s(link, LINK)).toBe(link); // off-origin link still allowed
	});

	it('keeps the default scheme allowlist — javascript:/data:/vbscript: collapse to ""', () => {
		const s = makeImageSrcSanitizer(['cdn.example.com']);
		// images
		expect(s('javascript:alert(1)', IMG)).toBe('');
		expect(s('data:text/html;base64,AAAA', IMG)).toBe('');
		// links — scheme block applies regardless of host policy
		expect(s('javascript:alert(1)', LINK)).toBe('');
		expect(s('vbscript:msgbox(1)', LINK)).toBe('');
		// sanity: matches the library default for these
		expect(s('javascript:alert(1)', IMG)).toBe(defaultSanitizeUrl('javascript:alert(1)', IMG));
	});
});

describe('BLOCKED_HTML_TAGS / buildBlockedHtmlRenderers', () => {
	it('lists the structural raw-HTML tags stripped from LLM output', () => {
		expect([...BLOCKED_HTML_TAGS]).toEqual(['iframe', 'embed', 'object']);
	});

	it('maps EVERY blocked tag — incl. the no-default-renderer "object" — to the placeholder', () => {
		const html = buildBlockedHtmlRenderers();
		// All three are present as keys (so the Parser takes the placeholder branch, not the
		// live <svelte:element> fast-path) and point at the escaped-text component.
		for (const tag of BLOCKED_HTML_TAGS) {
			expect(html[tag]).toBe(UnsupportedHTML);
		}
		// A normal tag keeps a (non-placeholder) renderer so benign HTML still renders.
		expect(html['strong']).toBeTruthy();
		expect(html['strong']).not.toBe(UnsupportedHTML);
	});
});

// ── Rendered: the real component in jsdom ───────────────────────────────────────────
let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

function markdown(): HTMLElement | null {
	return target.querySelector('.message-markdown');
}

beforeEach(() => {
	target = document.createElement('div');
	document.body.appendChild(target);
});

afterEach(() => {
	if (component) {
		try {
			unmount(component);
		} catch {
			/* already torn down */
		}
		component = null;
	}
	target.remove();
});

describe('ChatMessage — rendered markdown security (assistant role)', () => {
	it('drops an off-allowlist <img>: the hostile src is absent from the DOM', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content: '![beacon](https://evil.example/beacon.png)',
				showActions: false
			}
		});
		flushSync();
		// No element anywhere carries the hostile host (neither src nor data-src).
		expect(target.innerHTML).not.toContain('evil.example');
		const imgs = Array.from(target.querySelectorAll('img'));
		for (const img of imgs) {
			expect(img.getAttribute('src') ?? '').not.toContain('evil.example');
			expect(img.getAttribute('data-src') ?? '').not.toContain('evil.example');
		}
	});

	it('renders an allowed-host <img> with its src intact', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content: '![ok](https://cdn.example.com/p/1.jpg)',
				allowedImageHosts: ['cdn.example.com'],
				showActions: false
			}
		});
		flushSync();
		const img = target.querySelector('img');
		expect(img).toBeTruthy();
		expect(img?.getAttribute('src')).toBe('https://cdn.example.com/p/1.jpg');
	});

	it('renders a same-origin <img> with its src intact (default allowlist)', () => {
		const url = `https://${window.location.host}/media/cover.png`;
		component = mount(ChatMessage, {
			target,
			props: { role: 'assistant', content: `![cover](${url})`, showActions: false }
		});
		flushSync();
		const img = target.querySelector('img');
		expect(img).toBeTruthy();
		expect(img?.getAttribute('src')).toBe(url);
	});

	it('strips raw <iframe>/<embed>/<object> — none reach the DOM as live elements', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content:
					'before\n\n<iframe src="https://evil.example/frame"></iframe>\n<embed src="https://evil.example/e">\n<object data="https://evil.example/o"></object>\n\nafter',
				showActions: false
			}
		});
		flushSync();
		// None of the structural tags reach the DOM as LIVE elements (they collapse to the
		// escaped-text placeholder), so no frame/plugin is created and no off-site resource
		// is requested. The placeholder shows the original tag as inert, escaped text — which
		// is the intended "a tag was here but was suppressed" affordance — so the off-site URL
		// survives only as plain text, never as a fetchable attribute.
		expect(target.querySelector('iframe')).toBeNull();
		expect(target.querySelector('embed')).toBeNull();
		expect(target.querySelector('object')).toBeNull();
		// No element carries a live URL-bearing attribute pointing off-site.
		expect(target.querySelector('[src*="evil.example"]')).toBeNull();
		expect(target.querySelector('[data*="evil.example"]')).toBeNull();
		// The placeholder escaped the markup (so it's text, not parsed HTML).
		expect(markdown()?.innerHTML).toContain('&lt;iframe');
		// The surrounding prose still renders.
		expect(markdown()?.textContent).toContain('before');
		expect(markdown()?.textContent).toContain('after');
	});

	it('neutralizes a javascript: link (no live href survives) — default sanitizer', () => {
		component = mount(ChatMessage, {
			target,
			props: { role: 'assistant', content: '[click](javascript:alert(1))', showActions: false }
		});
		flushSync();
		const anchors = Array.from(target.querySelectorAll('a'));
		for (const a of anchors) {
			expect((a.getAttribute('href') ?? '').toLowerCase()).not.toContain('javascript:');
		}
		expect(target.innerHTML.toLowerCase()).not.toContain('javascript:');
	});

	it('does not let an onerror handler on a raw <img> survive — default sanitizer', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				// even a same-origin src so the <img> is allowed to render; the handler must still be stripped
				content: `<img src="/ok.png" onerror="alert(1)">`,
				showActions: false
			}
		});
		flushSync();
		const img = target.querySelector('img');
		// the element may render (allowed host) but must carry NO event handler
		expect(img?.getAttribute('onerror')).toBeNull();
		expect((img as HTMLImageElement | null)?.onerror).toBeFalsy();
		expect(target.innerHTML.toLowerCase()).not.toContain('onerror');
	});

	it('still renders normal markdown: bold, list, link, and a safe same-origin image', () => {
		const safeImg = `https://${window.location.host}/img/ok.png`;
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content: `Options:\n\n- **Buy** a villa\n- [Book](https://example.com)\n\n![ok](${safeImg})`,
				showActions: false
			}
		});
		flushSync();
		const md = markdown();
		expect(md).toBeTruthy();
		expect(md?.querySelector('strong')?.textContent).toBe('Buy');
		expect(md?.querySelectorAll('li').length).toBeGreaterThanOrEqual(2);
		const link = md?.querySelector('a');
		expect(link?.getAttribute('href')).toBe('https://example.com');
		expect(md?.querySelector('img')?.getAttribute('src')).toBe(safeImg);
	});

	it('keeps user messages on the literal <p> path (no markdown sink, inert content)', () => {
		component = mount(ChatMessage, {
			target,
			props: { role: 'user', content: '**not bold** ![x](https://evil.example/x.png)', showActions: false }
		});
		flushSync();
		// User input is rendered literally: the markdown sink is never used, so the image
		// syntax is INERT text — no <img> element is created and no request is issued.
		// (The hostile host appears only as plain text characters inside a <p>, which is
		// safe; the security guarantee is "no live element", not "characters absent".)
		expect(markdown()).toBeNull();
		expect(target.querySelector('img')).toBeNull();
		expect(target.querySelector('a')).toBeNull();
		const p = target.querySelector('.message-content p');
		expect(p?.textContent).toBe('**not bold** ![x](https://evil.example/x.png)');
	});
});

// ── Rendered: result cards (WP-D, 2607-116) ─────────────────────────────────────────
// The assistant attaches structured `cards` (property results) to a turn. This suite
// (TC3 → SC3) proves the render: content, the whole-card link (defaultAction), action
// buttons, a stack of cards, the empty case — and the URL scheme-guard (a hostile card
// can never smuggle javascript:/data: into an img src or href). Card URLs arrive already
// RESOLVED to full hrefs by the consumer (BR prefixes image R2 keys with the media host);
// this renderer trusts the scheme, not the host.
describe('ChatMessage — result cards (assistant role)', () => {
	const propertyCard = {
		image: {
			url: 'https://media.example.com/cdn-cgi/image/width=400/properties/abc/g/1.webp',
			alt: 'Sea View Villa'
		},
		title: 'Sea View Villa',
		subtitle: '13,300,000 THB · House · Koh Phangan',
		buttons: [{ kind: 'link' as const, label: 'View listing', url: 'https://site.example.com/p/abc' }],
		defaultAction: { url: 'https://site.example.com/p/abc' }
	};

	it('renders a property card with image, title, subtitle, a detail link, and a button', () => {
		component = mount(ChatMessage, {
			target,
			props: { role: 'assistant', content: 'Here are some options', cards: [propertyCard], showActions: false }
		});
		flushSync();

		expect(target.querySelectorAll('.chat-card').length).toBe(1);
		// Image src is used verbatim (already resolved by the consumer to the media host).
		const img = target.querySelector('.chat-card__image');
		expect(img?.getAttribute('src')).toBe(propertyCard.image.url);
		expect(img?.getAttribute('alt')).toBe('Sea View Villa');
		expect(img?.getAttribute('loading')).toBe('lazy');

		expect(target.querySelector('.chat-card__title')?.textContent).toBe('Sea View Villa');
		expect(target.querySelector('.chat-card__subtitle')?.textContent).toContain('13,300,000 THB');

		// Whole-card body links to defaultAction.
		const body = target.querySelector('a.chat-card__body');
		expect(body?.getAttribute('href')).toBe('https://site.example.com/p/abc');

		// Action button is a real link to the listing.
		const btn = target.querySelector('.chat-card__button');
		expect(btn?.tagName).toBe('A');
		expect(btn?.getAttribute('href')).toBe('https://site.example.com/p/abc');
		expect(btn?.textContent).toBe('View listing');
	});

	it('renders multiple cards as a stack', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content: '3 results',
				cards: [propertyCard, { ...propertyCard, title: 'Second' }, { ...propertyCard, title: 'Third' }],
				showActions: false
			}
		});
		flushSync();
		expect(target.querySelector('.message-cards')).toBeTruthy();
		expect(target.querySelectorAll('.chat-card').length).toBe(3);
	});

	it('renders no card region when the message has no cards', () => {
		component = mount(ChatMessage, {
			target,
			props: { role: 'assistant', content: 'Just text', showActions: false }
		});
		flushSync();
		expect(target.querySelector('.message-cards')).toBeNull();
	});

	it('scheme-guards hostile card URLs: javascript:/data: never reach the DOM', () => {
		component = mount(ChatMessage, {
			target,
			props: {
				role: 'assistant',
				content: 'x',
				cards: [
					{
						image: { url: 'javascript:alert(1)' },
						title: 'Evil',
						buttons: [{ kind: 'link' as const, label: 'tap', url: 'javascript:alert(1)' }],
						defaultAction: { url: 'data:text/html,evil' }
					}
				],
				showActions: false
			}
		});
		flushSync();
		// Hostile image src → dropped (no <img> rendered).
		expect(target.querySelector('.chat-card__image')).toBeNull();
		// Hostile button href → button not rendered (no dead/dangerous control).
		expect(target.querySelector('.chat-card__button')).toBeNull();
		// Hostile defaultAction → body is an inert <div>, never a live <a>.
		expect(target.querySelector('a.chat-card__body')).toBeNull();
		expect(target.querySelector('.chat-card__body')?.tagName).toBe('DIV');
		// The card still shows its title (the render degrades, it does not blank out).
		expect(target.querySelector('.chat-card__title')?.textContent).toBe('Evil');
		// No dangerous scheme survives anywhere in the produced DOM.
		expect(target.innerHTML.toLowerCase()).not.toContain('javascript:');
		expect(target.innerHTML.toLowerCase()).not.toContain('data:text/html');
	});
});

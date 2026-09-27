import { describe, it, expect } from 'vitest';
import { socialIconPath, ICONS, SOCIAL_ALIASES } from './social-icon-paths';

// #42 — SocialIcon must resolve the names producers actually store. Before the
// alias layer, `x` and `website` missed ICONS and fell back to the github glyph.
describe('socialIconPath', () => {
	it('maps the x alias to the twitter glyph', () => {
		expect(socialIconPath('x')).toBe(ICONS.twitter);
	});

	it('maps website / web / url / homepage to the globe glyph', () => {
		expect(socialIconPath('website')).toBe(ICONS.globe);
		expect(socialIconPath('web')).toBe(ICONS.globe);
		expect(socialIconPath('url')).toBe(ICONS.globe);
		expect(socialIconPath('homepage')).toBe(ICONS.globe);
	});

	it('resolves known platforms unchanged', () => {
		expect(socialIconPath('facebook')).toBe(ICONS.facebook);
		expect(socialIconPath('whatsapp')).toBe(ICONS.whatsapp);
		expect(socialIconPath('line')).toBe(ICONS.line);
		expect(socialIconPath('github')).toBe(ICONS.github);
	});

	it('is case-insensitive (aliases and platforms)', () => {
		expect(socialIconPath('X')).toBe(ICONS.twitter);
		expect(socialIconPath('Website')).toBe(ICONS.globe);
		expect(socialIconPath('Instagram')).toBe(ICONS.instagram);
	});

	it('falls back to globe for an unknown platform (not github)', () => {
		expect(socialIconPath('myspace')).toBe(ICONS.globe);
		expect(socialIconPath('')).toBe(ICONS.globe);
		expect(socialIconPath('myspace')).not.toBe(ICONS.github);
	});

	it('every alias target exists in ICONS', () => {
		for (const target of Object.values(SOCIAL_ALIASES)) {
			expect(ICONS[target], `alias target "${target}" missing from ICONS`).toBeTruthy();
		}
	});

	it('the globe fallback key exists in ICONS (resolver never returns undefined)', () => {
		expect(ICONS.globe).toBeTruthy();
	});
});

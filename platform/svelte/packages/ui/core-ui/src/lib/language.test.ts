import { describe, it, expect } from 'vitest';
import { allLanguages, languageLabel } from './language';

// TC-03 (SC-LANG-03a) — the shared canonical world-language module. The list is the
// admin picker's source; the label resolver is what turns a bare code into a real name
// everywhere (public switcher + admin tabs), so a locale is NEVER shown as a raw code.
// RED until src/lib/language.ts exists.
describe('shared language module', () => {
	it('allLanguages returns the full ISO-639-1 set with names + endonyms', () => {
		const langs = allLanguages();
		expect(langs.length).toBeGreaterThanOrEqual(180);
		const ru = langs.find((l) => l.code === 'ru');
		expect(ru).toBeTruthy();
		expect(ru!.name).toBe('Russian');
		expect(ru!.nativeName).toBe('Русский');
	});

	it('languageLabel resolves a code to its real name, never the raw code', () => {
		expect(languageLabel('ru')).toBe('Русский');
		expect(languageLabel('de')).toBe('Deutsch');
		// An unknown code falls back to itself (can never be saved, so never public).
		expect(languageLabel('zz')).toBe('zz');
	});
});

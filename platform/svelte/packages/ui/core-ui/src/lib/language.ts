/**
 * Shared canonical world-language module (owner ruling 2026-07-13, task 2607-070).
 *
 * One source for "what languages exist" across every project: the ISO-639-1 set
 * from the `iso-639-1` package (CLDR-derived names + endonyms, updated by
 * dependency bump — never a hand-maintained list). A project's `languages`
 * dictionary is its OFFERED subset; this module is the world list behind the
 * picker and the label resolver that turns a bare code into a real name, so a
 * locale is never rendered as a raw code.
 *
 * Server-safe: pure data, no browser APIs.
 */
import ISO6391 from 'iso-639-1';

export interface WorldLanguage {
	/** lowercase ISO-639-1 code, e.g. "ru" */
	code: string;
	/** English name, e.g. "Russian" */
	name: string;
	/** endonym, e.g. "Русский" */
	nativeName: string;
}

/** The full ISO-639-1 language set with English names + endonyms, code-sorted. */
export function allLanguages(): WorldLanguage[] {
	return ISO6391.getLanguages(ISO6391.getAllCodes());
}

/**
 * Human label for a language code — the endonym (what a speaker of that
 * language calls it: ru → "Русский"). Unknown codes fall back to the code
 * itself; canonical write validation means an unknown code can never be a
 * saved locale, so the fallback is a dev-time affordance, not a user state.
 */
export function languageLabel(code: string): string {
	return ISO6391.getNativeName(code) || code;
}

/**
 * English name for a language code (ru → "Russian"); falls back to the code.
 * Pair with languageLabel for "Русский (Russian)" picker rows.
 */
export function languageName(code: string): string {
	return ISO6391.getName(code) || code;
}

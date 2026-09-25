// Shared text-casing primitive (S-FORMAT). The format-knob labels (format-labels.ts) and the
// enum display formatter (enum-formatter.ts) both need to capitalize the first letter of each
// token; this is the single implementation they share so the two never drift.

/**
 * Capitalize the first letter of each token in `value`. Tokens are separated by whitespace,
 * '-' or '_' (the separators are preserved). With `lowerRest` the other letters are lowercased
 * first — proper Title Case ('FOR SALE' → 'For Sale'); without it the rest is preserved, a
 * gentle capitalize that keeps acronyms + existing case ('POA' → 'POA', 'pool-villa' →
 * 'Pool-villa'… wait, the FIRST of each token: 'pool-villa' → 'Pool-Villa'). Empty → ''.
 */
export function capitalizeTokens(value: string, opts: { lowerRest?: boolean } = {}): string {
	const base = opts.lowerRest ? value.toLowerCase() : value;
	// Uppercase the first lowercase letter at the string start or right after a separator,
	// preserving the separator. [\s\-_] == [-_\s]; only LOWERCASE letters are touched, so an
	// already-capital acronym ('POA') passes through unchanged when lowerRest is off.
	return base.replace(/(^|[\s\-_])([a-z])/g, (_match, sep: string, ch: string) => sep + ch.toUpperCase());
}

/** A card's title/link/image_url can come from an agent's structured output - an LLM completion,
 * not code the app controls - so this allowlist is load-bearing, not defensive filler. Blocks
 * javascript:/data: from ever reaching an href or img src. */
export function safeUrl(value: string | null | undefined): string | null {
	if (!value) return null;
	if (value.startsWith('/') && !value.startsWith('//')) return value;
	try {
		const parsed = new URL(value);
		return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? value : null;
	} catch {
		return null;
	}
}

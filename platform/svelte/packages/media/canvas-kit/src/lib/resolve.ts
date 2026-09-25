// Pure resolvers — the heart of the Media Canvas engine. Resolve order is
// BASE → BINDING → ANIMATION@t (design §4); the three are not mutually exclusive.
// Both return a NEW layer (never mutate input) so existing renderers consume them
// unchanged.

import type { Layer } from './types/layer.js';
import type { AnimationTrack } from './types/animation.js';
import type { SourceSnapshot, FormatterRegistry, FormatterDescriptor } from './types/source.js';
import { resolveTemplate, type TokenResolver, type Template } from '@sbx/text-template';
import { ease } from './easing.js';

/**
 * Collapse a layer's animation tracks into concrete property values at time `t` (ms).
 * Numeric properties interpolate between surrounding keyframes using the keyframe's
 * easing curve; non-numeric properties step (hold the previous keyframe). `t` clamps
 * to the first/last keyframe. A layer with no `animations` is returned unchanged.
 */
export function resolveLayerAtTime(layer: Layer, t: number): Layer {
	if (!layer.animations?.length) return layer;
	const out: Layer = { ...layer };
	for (const track of layer.animations) {
		const value = valueAtTime(track, t);
		if (value !== undefined) setPath(out as unknown as Record<string, unknown>, track.property, value);
	}
	return out;
}

function valueAtTime(track: AnimationTrack, t: number): number | string | undefined {
	const kfs = [...track.keyframes].sort((a, b) => a.time - b.time);
	if (kfs.length === 0) return undefined;
	if (kfs.length === 1) return kfs[0].value;
	if (t <= kfs[0].time) return kfs[0].value;
	const last = kfs[kfs.length - 1];
	if (t >= last.time) return last.value;

	for (let i = 0; i < kfs.length - 1; i++) {
		const a = kfs[i];
		const b = kfs[i + 1];
		if (t >= a.time && t < b.time) {
			if (typeof a.value === 'number' && typeof b.value === 'number') {
				const span = b.time - a.time;
				const p = span === 0 ? 0 : (t - a.time) / span;
				return a.value + (b.value - a.value) * ease(a.easing, p);
			}
			return a.value; // non-numeric → step
		}
	}
	return last.value;
}

/**
 * Substitute a layer's bound properties from a resolved source `snapshot`.
 * Resolve precedence (D20): **override → token→formatter → fallback**.
 *   - `binding.override` set (non-null) → used verbatim (the user's fine-tuned final
 *     string / chosen image URL), NOT re-formatted.
 *   - otherwise the dotted `token` is read from the snapshot and formatted via the
 *     matching `formatters` entry (selected by the `format` hint's name).
 *   - a missing/empty token value uses `fallback` (or '' — never undefined, so layout
 *     never breaks).
 * A layer with no `bindings` is returned unchanged. Input is never mutated.
 */
export function resolveBindings(
	layer: Layer,
	snapshot: SourceSnapshot,
	formatters: FormatterRegistry = {}
): Layer {
	const hasTemplate = !!layer.content_template?.length;
	if (!layer.bindings?.length && !hasTemplate) return layer;
	const out: Layer = { ...layer };
	for (const binding of layer.bindings ?? []) {
		let display: string;
		if (binding.override !== undefined && binding.override !== null) {
			// Manual fine-tune wins, used as-is (already the final value).
			display = binding.override;
		} else {
			const raw = getPath(snapshot, binding.token);
			// Empty value → fallback; otherwise the shared token→formatter step (override wins).
			display =
				raw === undefined || raw === null || raw === ''
					? (binding.fallback ?? '')
					: formatTokenValue(raw, binding.format, binding.formatOptions, formatters);
		}
		setPath(out as unknown as Record<string, unknown>, binding.property, display);
	}
	// Template content (INC-C) — resolved via the SAME token→formatter step as a binding, taking
	// precedence over any 'content' binding (the field-kind model makes the two exclusive).
	if (hasTemplate) {
		const resolver: TokenResolver = {
			resolve: (token, format, formatOptions) =>
				formatTokenValue(getPath(snapshot, token), format, formatOptions, formatters)
		};
		setPath(
			out as unknown as Record<string, unknown>,
			'content',
			resolveTemplate(layer.content_template!, resolver)
		);
	}
	return out;
}

/**
 * Format one raw token value via the registry — the SHARED step behind both binding resolution
 * and template-token resolution (so the two never diverge). An empty value → '' (the caller
 * supplies any fallback). No `format` ⇒ String(raw). An unregistered formatter ⇒ String(raw).
 * INC-A: string descriptors receive String(raw), numeric descriptors Number(raw).
 */
export function formatTokenValue(
	raw: unknown,
	format: string | undefined,
	formatOptions: Record<string, string> | undefined,
	formatters: FormatterRegistry
): string {
	if (raw === undefined || raw === null || raw === '') return '';
	if (!format) return String(raw);
	const [name, arg] = format.split(':');
	const descriptor = formatters[name];
	if (!descriptor) return String(raw);
	// Merge the descriptor's per-knob defaults with the caller's options (override wins).
	const opts = { ...descriptorDefaults(descriptor), ...formatOptions };
	const value = descriptor.valueType === 'string' ? String(raw) : Number(raw);
	return descriptor.format(value, opts, arg);
}

/** Split a (possibly source-qualified) template token into its source alias + field path. A
 *  leading "<alias>." whose alias is a KNOWN source selects that source; otherwise the whole
 *  token is a field path resolved against `defaultAlias` (back-compat for bare / hand-typed
 *  tokens). So one template can pull '[secondary.price]' AND a legacy bare '[title]'. */
export function splitSourceToken(
	token: string,
	knownAliases: ReadonlySet<string>,
	defaultAlias: string
): { alias: string; field: string } {
	const dot = token.indexOf('.');
	if (dot > 0) {
		const prefix = token.slice(0, dot);
		if (knownAliases.has(prefix)) return { alias: prefix, field: token.slice(dot + 1) };
	}
	return { alias: defaultAlias, field: token };
}

/** Resolve a content template across MULTIPLE sources — each token routed to its source by an
 *  optional "<alias>." prefix (via {@link splitSourceToken}); a bare token resolves against
 *  `defaultAlias`. Uses the SAME token→formatter step as a binding ({@link formatTokenValue}),
 *  so a multi-source template never diverges from the binding path. */
export function resolveTemplateContent(
	template: Template,
	byAlias: Map<string, SourceSnapshot>,
	formatters: FormatterRegistry,
	defaultAlias: string
): string {
	const known = new Set(byAlias.keys());
	const resolver: TokenResolver = {
		resolve: (token, format, formatOptions) => {
			const { alias, field } = splitSourceToken(token, known, defaultAlias);
			const snapshot = byAlias.get(alias) ?? byAlias.get(defaultAlias) ?? {};
			return formatTokenValue(getPath(snapshot, field), format, formatOptions, formatters);
		}
	};
	return resolveTemplate(template, resolver);
}

/** The descriptor's per-knob defaults as a flat map (key → default choice) — the base
 *  the binding's `formatOptions` override at resolve time. Exported so the editor's
 *  Inspector merges the SAME way the resolver does (one merge contract, no divergence). */
export function descriptorDefaults(descriptor: FormatterDescriptor): Record<string, string> {
	const out: Record<string, string> = {};
	for (const opt of descriptor.options) out[opt.key] = opt.default;
	return out;
}

/** Read a dotted path from an object; undefined if any segment is absent. */
function getPath(obj: unknown, path: string): unknown {
	return path.split('.').reduce<unknown>((acc, key) => {
		if (acc === null || acc === undefined || typeof acc !== 'object') return undefined;
		return (acc as Record<string, unknown>)[key];
	}, obj);
}

/** Write a dotted path, cloning intermediate objects so the input is never mutated. */
function setPath(root: Record<string, unknown>, path: string, value: unknown): void {
	const keys = path.split('.');
	if (keys.length === 1) {
		root[keys[0]] = value;
		return;
	}
	let cur = root;
	for (let i = 0; i < keys.length - 1; i++) {
		const key = keys[i];
		const next = cur[key];
		cur[key] = next !== null && typeof next === 'object' ? { ...(next as object) } : {};
		cur = cur[key] as Record<string, unknown>;
	}
	cur[keys[keys.length - 1]] = value;
}

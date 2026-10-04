import type { JsonSchema } from './types';

/** Resolves a Pydantic-style $ref ("#/$defs/PropertyResult") against the root schema's $defs.
 * A schema with no $ref passes through unchanged. */
export function deref(schema: JsonSchema, defs: Record<string, JsonSchema>): JsonSchema {
	if (schema.$ref) {
		const key = schema.$ref.replace('#/$defs/', '');
		const target = defs[key];
		if (target) return deref(target, defs);
	}
	return schema;
}

/** anyOf shows up as the optional-field wrapper (e.g. `str | None`) - unwrap to the real,
 * non-null branch, resolving $ref along the way, so callers see the type that actually matters. */
export function resolve(schema: JsonSchema, defs: Record<string, JsonSchema> = {}): JsonSchema {
	const s = deref(schema, defs);
	if (s.anyOf) {
		const real = s.anyOf.map((branch) => deref(branch, defs)).find((branch) => branch.type !== 'null');
		return real ? resolve(real, defs) : s;
	}
	return s;
}

export type InputKind = 'select' | 'number' | 'boolean' | 'date' | 'text';

/** Which form input a property should render as - used by SchemaForm (editable). */
export function kindOf(schema: JsonSchema, defs: Record<string, JsonSchema> = {}): InputKind {
	const s = resolve(schema, defs);
	if (s.enum) return 'select';
	if (s.type === 'integer' || s.type === 'number') return 'number';
	if (s.type === 'boolean') return 'boolean';
	if (s.format === 'date') return 'date';
	return 'text';
}

/** A short, human-readable type label for a property's row - "array of PropertyResult",
 * "array of string", "'sale' | 'rent' | ...", "integer". For a nested object/array-of-objects,
 * this only names it; SchemaFields renders its actual fields as real nested rows below, this
 * isn't trying to fully describe it inline. */
export function describeType(schema: JsonSchema, defs: Record<string, JsonSchema> = {}): string {
	const s = resolve(schema, defs);
	if (s.enum) return s.enum.map((v) => `'${v}'`).join(' | ');
	if (s.type === 'array') {
		if (!s.items) return 'array';
		const item = resolve(s.items, defs);
		return item.type === 'object' ? `array of ${item.title ?? 'object'}` : `array of ${describeType(s.items, defs)}`;
	}
	if (s.format) return `${s.type ?? 'string'} (${s.format})`;
	return s.type ?? 'any';
}

/** The nested object schema to recurse into for a property, or null if it's a plain field -
 * either the property itself is an object with properties, or it's an array of such objects
 * (Pydantic's list[SomeModel] shape). Used by SchemaFields to decide when to render nested rows
 * instead of a one-line type description. */
export function nestedObjectSchema(schema: JsonSchema, defs: Record<string, JsonSchema> = {}): JsonSchema | null {
	const s = resolve(schema, defs);
	if (s.type === 'object' && s.properties) return s;
	if (s.type === 'array' && s.items) {
		const item = resolve(s.items, defs);
		if (item.type === 'object' && item.properties) return item;
	}
	return null;
}

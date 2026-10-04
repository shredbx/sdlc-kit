<script lang="ts">
	import type { JsonSchema } from './types';
	import { describeType, nestedObjectSchema } from './schemaKind';
	import SchemaFields from './SchemaFields.svelte';

	interface Props {
		/** An object-typed JSON Schema (has `.properties`, optionally `.required`) - works for
		 * either a tool's input (parameters_json_schema) or output (output_json_schema). */
		schema: JsonSchema;
		/** $defs to resolve $ref against - omit at the top level (falls back to schema.$defs,
		 * which a tool's own output_json_schema root really does carry); recursive calls into a
		 * nested model pass it explicitly, since only the root schema carries $defs. */
		defs?: Record<string, JsonSchema>;
		depth?: number;
	}

	let { schema, defs, depth = 0 }: Props = $props();

	const resolvedDefs = $derived(defs ?? schema.$defs ?? {});
	const properties = $derived(Object.entries(schema.properties ?? {}));
	const required = $derived(new Set(schema.required ?? []));
</script>

{#if properties.length === 0}
	<p class="no-fields">(no fields)</p>
{:else}
	<div class="fields" style="margin-left: {depth * 1.25}rem">
		{#each properties as [name, propSchema] (name)}
			{@const nested = nestedObjectSchema(propSchema, resolvedDefs)}
			<div class="row">
				<span class="name">{name}{#if required.has(name)}<span class="required">*</span>{/if}</span>
				<span class="type">{describeType(propSchema, resolvedDefs)}</span>
				{#if propSchema.description}<span class="desc">{propSchema.description}</span>{/if}
			</div>
			{#if nested}
				<SchemaFields schema={nested} defs={resolvedDefs} depth={depth + 1} />
			{/if}
		{/each}
	</div>
{/if}

<style>
	.no-fields {
		font-size: 0.85rem;
		color: #666;
		margin: 0;
	}
	.fields {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.row {
		display: grid;
		grid-template-columns: minmax(8ch, auto) minmax(10ch, auto) 1fr;
		gap: 0.6rem;
		align-items: baseline;
		font-size: 0.82rem;
		padding: 0.3rem 0;
		border-bottom: 1px solid #222;
	}
	.name {
		font-weight: 600;
	}
	.required {
		color: #e66;
		margin-left: 0.1rem;
	}
	.type {
		color: #8ab;
		font-family: monospace;
		font-size: 0.78rem;
	}
	.desc {
		color: #999;
	}
</style>

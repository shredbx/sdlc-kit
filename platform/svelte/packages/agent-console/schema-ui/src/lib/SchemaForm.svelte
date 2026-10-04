<script lang="ts">
	import type { ToolLike } from './types';
	import { resolve, kindOf } from './schemaKind';

	interface Props {
		tool: ToolLike;
		ontrigger: (name: string, args: Record<string, unknown>) => void;
	}

	let { tool, ontrigger }: Props = $props();

	const properties = $derived(Object.entries(tool.parameters_json_schema.properties ?? {}));
	const required = $derived(new Set(tool.parameters_json_schema.required ?? []));

	let values = $state<Record<string, string | boolean>>({});

	function submit(event: SubmitEvent) {
		event.preventDefault();
		const args: Record<string, unknown> = {};
		for (const [name, schema] of properties) {
			const raw = values[name];
			const kind = kindOf(schema);
			if (kind === 'boolean') {
				args[name] = Boolean(raw);
				continue;
			}
			if (raw === undefined || raw === '') continue; // omit unset - required-but-empty deliberately
			// surfaces the tool's own real validation error, which is itself a useful test.
			args[name] = kind === 'number' ? Number(raw) : raw;
		}
		ontrigger(tool.name, args);
	}
</script>

<form onsubmit={submit} class="tool-form">
	<p class="tool-description">{tool.description}</p>
	{#if properties.length === 0}
		<p class="no-params">(no parameters)</p>
	{/if}
	{#each properties as [name, schema] (name)}
		{@const s = resolve(schema)}
		{@const kind = kindOf(schema)}
		<label class="field">
			<span class="field-label">
				{name}
				{#if required.has(name)}<span class="required">*</span>{/if}
			</span>
			{#if kind === 'select'}
				<select bind:value={values[name]}>
					<option value="">-</option>
					{#each s.enum ?? [] as option (option)}
						<option value={option}>{option}</option>
					{/each}
				</select>
			{:else if kind === 'boolean'}
				<input type="checkbox" bind:checked={values[name] as unknown as boolean} />
			{:else if kind === 'number'}
				<input type="number" bind:value={values[name]} />
			{:else if kind === 'date'}
				<input type="date" bind:value={values[name]} />
			{:else}
				<input type="text" bind:value={values[name]} />
			{/if}
			{#if schema.description}<span class="field-help">{schema.description}</span>{/if}
		</label>
	{/each}
	<button type="submit">Trigger tool:{tool.name}</button>
</form>

<style>
	.tool-form {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 0.75rem;
		border: 1px solid #333;
		border-radius: 6px;
	}
	.tool-description {
		font-size: 0.85rem;
		color: #aaa;
		margin: 0 0 0.25rem;
		white-space: pre-wrap;
	}
	.no-params {
		font-size: 0.85rem;
		color: #666;
		margin: 0;
	}
	.field {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.field-label {
		font-size: 0.8rem;
		font-weight: 600;
	}
	.required {
		color: #e66;
	}
	.field-help {
		font-size: 0.75rem;
		color: #888;
	}
	button {
		margin-top: 0.25rem;
		padding: 0.4rem 0.75rem;
		cursor: pointer;
	}
</style>

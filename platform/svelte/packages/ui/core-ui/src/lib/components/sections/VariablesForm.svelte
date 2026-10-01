<script lang="ts">
	/**
	 * VariablesForm — one labeled input per declared `$variable` for a run (dumb). Each row shows the
	 * `$name` token, an optional description, and a "needs input" dot when the variable has neither a
	 * value nor a default. Inputs are fixed-height multiline textareas (manually resizable) — they do
	 * NOT auto-grow, so filling one never reflows the page. The variable's default shows as the
	 * placeholder. Edits emit onChange(name, value). Imports no store/API.
	 */
	interface Field {
		name: string;
		value?: string;
		hint?: string;
		/** The variable's persisted default — shown as the input placeholder. */
		placeholder?: string;
		/** True when the variable has neither a value nor a default (must be filled to run cleanly). */
		needsInput?: boolean;
	}
	interface Props {
		fields?: Field[];
		onChange?: (name: string, value: string) => void;
	}
	let { fields = [], onChange }: Props = $props();
</script>

<div class="vars" data-testid="variables-form">
	{#each fields as f (f.name)}
		<div class="row">
			<div class="fldlabel">
				<span class="var">${f.name}</span>
				{#if f.hint}<span class="hint">{f.hint}</span>{/if}
				{#if f.needsInput}<span class="need" title="No value or default — fill to run">needs input</span>{/if}
			</div>
			<textarea
				class="fld"
				rows="2"
				aria-label={f.name}
				placeholder={f.placeholder || ''}
				value={f.value ?? ''}
				oninput={(e) => onChange?.(f.name, e.currentTarget.value)}
			></textarea>
		</div>
	{/each}
</div>

<style>
	.vars {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.fldlabel {
		font-size: 11px;
		font-weight: 700;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 6px;
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.fldlabel .var {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		color: var(--color-accent-ink);
		background: var(--color-accent-soft);
		border-radius: 5px;
		padding: 1px 6px;
		text-transform: none;
		letter-spacing: 0;
	}
	.fldlabel .hint {
		font-weight: 500;
		text-transform: none;
		letter-spacing: 0;
		color: var(--color-faint);
	}
	.fldlabel .need {
		margin-left: auto;
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.04em;
		color: var(--color-amber);
		background: var(--color-amber-soft);
		border-radius: 5px;
		padding: 2px 7px;
	}
	.fld {
		width: 100%;
		border: 1px solid var(--color-line-2);
		border-radius: 9px;
		padding: 9px 11px;
		font-size: 13.5px;
		line-height: 1.5;
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
		display: block;
		resize: vertical;
		min-height: 42px;
	}
	.fld::placeholder {
		color: var(--color-faint);
	}
	.fld:focus {
		border-color: var(--color-primary);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 14%, transparent);
		outline: none;
	}
</style>

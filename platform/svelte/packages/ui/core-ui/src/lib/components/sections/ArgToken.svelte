<script lang="ts">
	/**
	 * ArgToken — an inline `$variable` reference chip rendered inside a section's content (dumb).
	 * The host tokenizes the content text and drops one ArgToken per `$name` occurrence; this
	 * component only paints the chip. `name` is the bare variable name (no leading `$`). When the host
	 * passes `onclick`, the chip becomes an interactive button (opens the variable editor); otherwise
	 * it is inert text. Imports no store/API.
	 */
	interface Props {
		name: string;
		onclick?: (name: string) => void;
	}
	let { name, onclick }: Props = $props();
</script>

{#if onclick}
	<button
		type="button"
		class="arg arg--btn"
		data-testid="arg-token-{name}"
		title="Edit variable ${name}"
		onclick={() => onclick(name)}
	>${name}</button>
{:else}
	<span class="arg" data-testid="arg-token-{name}">${name}</span>
{/if}

<style>
	.arg {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 0.92em;
		color: var(--color-accent-ink);
		background: var(--color-accent-soft);
		border-radius: 5px;
		padding: 1px 5px;
		font-weight: 600;
	}
	.arg--btn {
		border: 0;
		cursor: pointer;
		line-height: inherit;
		transition: filter 0.12s, box-shadow 0.12s;
	}
	.arg--btn:hover {
		filter: brightness(0.97);
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary) 45%, transparent);
	}
	.arg--btn:focus-visible {
		outline: none;
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 20%, transparent);
	}
</style>

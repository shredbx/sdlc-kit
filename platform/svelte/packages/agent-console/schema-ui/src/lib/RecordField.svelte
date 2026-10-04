<script lang="ts">
	interface Props {
		label: string;
		value: string;
		rows?: number;
		editable?: boolean;
	}

	let { label, value, rows = 2, editable = false }: Props = $props();
</script>

<div class="field">
	<span class="field-label">{label}</span>
	{#if editable}
		<textarea {rows} bind:value></textarea>
	{:else}
		<textarea readonly {rows} value={value}></textarea>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}
	.field-label {
		font-size: 0.7rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		color: #888;
	}
	textarea {
		width: 100%;
		box-sizing: border-box;
		background: #1a1a1a;
		color: #eee;
		border: 1px solid #333;
		border-radius: 4px;
		padding: 0.5rem;
		font-family: inherit;
		font-size: 0.85rem;
		line-height: 1.4;
		resize: vertical;
	}
	textarea[readonly] {
		cursor: default;
	}
</style>

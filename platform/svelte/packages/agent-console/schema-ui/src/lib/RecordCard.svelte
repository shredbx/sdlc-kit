<script lang="ts">
	import RecordField from './RecordField.svelte';
	import type { RecordFieldSpec } from './types';

	interface Props {
		fields: RecordFieldSpec[];
	}

	let { fields }: Props = $props();
</script>

<div class="record">
	{#each fields as f (f.label)}
		<RecordField label={f.label} value={f.value} rows={f.rows} />
	{/each}
</div>

<style>
	.record {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		border: 1px solid #2a2a2a;
		border-radius: 6px;
		padding: 0.6rem 0.75rem;
		background: #161616;
	}
</style>

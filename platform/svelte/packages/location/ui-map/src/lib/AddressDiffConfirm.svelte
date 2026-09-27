<script lang="ts">
	import type { AddressDiffField } from './types.js';

	interface Props {
		open: boolean;
		fields: AddressDiffField[];
		onconfirm: () => void;
		oncancel: () => void;
		title?: string;
		confirmLabel?: string;
		cancelLabel?: string;
		// When true, unchanged rows are hidden. When false, they're rendered
		// muted so the user can confirm "nothing else is changing".
		hideUnchanged?: boolean;
	}

	let {
		open,
		fields,
		onconfirm,
		oncancel,
		title = 'Apply address from map?',
		confirmLabel = 'Apply',
		cancelLabel = 'Cancel',
		hideUnchanged = false
	}: Props = $props();

	const changes = $derived(fields.filter((f) => f.isChange));
	const unchanged = $derived(hideUnchanged ? [] : fields.filter((f) => !f.isChange));
	const hasChanges = $derived(changes.length > 0);

	function placeholder(value: string): string {
		return value.length === 0 ? '—' : value;
	}
</script>

{#if open}
	<div
		class="ui-map-diff__backdrop"
		role="dialog"
		aria-modal="true"
		aria-labelledby="ui-map-diff-title"
		onclick={(e) => { if (e.target === e.currentTarget) oncancel(); }}
		onkeydown={(e) => {
			if (e.key === 'Escape') oncancel();
		}}
		tabindex="-1"
	>
		<!-- Panel: no onclick needed — the backdrop only closes on a click that
		     lands on itself (e.target === e.currentTarget), so a click inside the
		     panel never bubbles to a close. Avoids a non-interactive element with
		     a click handler (a11y). -->
		<div class="ui-map-diff__panel" role="document">
			<header class="ui-map-diff__header">
				<h2 id="ui-map-diff-title" class="ui-map-diff__title">{title}</h2>
				<p class="ui-map-diff__summary">
					{#if hasChanges}
						{changes.length}
						{changes.length === 1 ? 'field' : 'fields'} will change
					{:else}
						No changes — the form already matches this address.
					{/if}
				</p>
			</header>

			<div class="ui-map-diff__body">
				{#if hasChanges}
					<table class="ui-map-diff__table">
						<thead>
							<tr>
								<th scope="col">Field</th>
								<th scope="col">Current</th>
								<th scope="col">After</th>
							</tr>
						</thead>
						<tbody>
							{#each changes as f (f.key)}
								<tr class="ui-map-diff__row ui-map-diff__row--change">
									<th scope="row" class="ui-map-diff__field-label">{f.label}</th>
									<td class="ui-map-diff__value ui-map-diff__value--current">
										{placeholder(f.current)}
									</td>
									<td class="ui-map-diff__value ui-map-diff__value--next">
										{placeholder(f.next)}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}

				{#if unchanged.length > 0}
					<details class="ui-map-diff__unchanged">
						<summary>{unchanged.length} unchanged {unchanged.length === 1 ? 'field' : 'fields'}</summary>
						<table class="ui-map-diff__table ui-map-diff__table--unchanged">
							<tbody>
								{#each unchanged as f (f.key)}
									<tr class="ui-map-diff__row">
										<th scope="row" class="ui-map-diff__field-label">{f.label}</th>
										<td class="ui-map-diff__value" colspan="2">{placeholder(f.current)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</details>
				{/if}
			</div>

			<footer class="ui-map-diff__footer">
				<button type="button" class="ui-map-diff__btn ui-map-diff__btn--cancel" onclick={oncancel}>
					{cancelLabel}
				</button>
				<button
					type="button"
					class="ui-map-diff__btn ui-map-diff__btn--confirm"
					onclick={onconfirm}
					disabled={!hasChanges}
				>
					{confirmLabel}
				</button>
			</footer>
		</div>
	</div>
{/if}

<style>
	.ui-map-diff__backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 15, 20, 0.45);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		z-index: 1000;
	}
	.ui-map-diff__panel {
		width: min(640px, 100%);
		max-height: min(80vh, 720px);
		display: flex;
		flex-direction: column;
		background: var(--color-surface, #fff);
		border-radius: 0.5rem;
		box-shadow: 0 24px 64px rgba(0, 0, 0, 0.2);
		overflow: hidden;
	}
	.ui-map-diff__header {
		padding: 1rem 1.25rem 0.75rem;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}
	.ui-map-diff__title {
		margin: 0 0 0.25rem;
		font-size: 1.125rem;
		font-weight: 600;
	}
	.ui-map-diff__summary {
		margin: 0;
		font-size: 0.875rem;
		color: var(--color-muted, #6b7280);
	}
	.ui-map-diff__body {
		padding: 0.75rem 1.25rem;
		overflow-y: auto;
		flex: 1;
	}
	.ui-map-diff__table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}
	.ui-map-diff__table thead th {
		text-align: left;
		font-weight: 500;
		color: var(--color-muted, #6b7280);
		padding: 0.5rem 0.5rem 0.5rem 0;
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}
	.ui-map-diff__row {
		border-bottom: 1px solid var(--color-border-faint, #f3f4f6);
	}
	.ui-map-diff__row:last-child {
		border-bottom: 0;
	}
	.ui-map-diff__field-label {
		text-align: left;
		font-weight: 500;
		padding: 0.5rem 0.5rem 0.5rem 0;
		width: 30%;
		vertical-align: top;
	}
	.ui-map-diff__value {
		padding: 0.5rem 0.5rem 0.5rem 0;
		vertical-align: top;
		word-break: break-word;
	}
	.ui-map-diff__value--current {
		color: var(--color-muted, #6b7280);
		text-decoration: line-through;
	}
	.ui-map-diff__value--next {
		color: var(--color-success-strong, #047857);
		font-weight: 500;
	}
	.ui-map-diff__unchanged {
		margin-top: 0.75rem;
		font-size: 0.8125rem;
		color: var(--color-muted, #6b7280);
	}
	.ui-map-diff__unchanged summary {
		cursor: pointer;
		padding: 0.5rem 0;
	}
	.ui-map-diff__table--unchanged {
		font-size: 0.8125rem;
		color: var(--color-muted, #6b7280);
	}
	.ui-map-diff__footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		padding: 0.875rem 1.25rem;
		border-top: 1px solid var(--color-border, #e5e7eb);
		background: var(--color-surface-muted, #fafafa);
	}
	.ui-map-diff__btn {
		padding: 0.5rem 1rem;
		border-radius: 0.375rem;
		font: inherit;
		font-weight: 500;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.ui-map-diff__btn--cancel {
		background: transparent;
		border-color: var(--color-border, #d1d5db);
		color: var(--color-text, #111827);
	}
	.ui-map-diff__btn--cancel:hover {
		background: var(--color-hover, #f4f4f5);
	}
	.ui-map-diff__btn--confirm {
		background: var(--color-brand, #e5392b);
		color: #fff;
	}
	.ui-map-diff__btn--confirm:hover:not(:disabled) {
		filter: brightness(0.95);
	}
	.ui-map-diff__btn--confirm:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}
</style>

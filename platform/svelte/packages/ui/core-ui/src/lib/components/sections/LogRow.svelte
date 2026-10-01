<script lang="ts">
	/**
	 * LogRow — one run-history entry (dumb): when · model · status · preview · chevron. Clicking fires
	 * onOpen so the host can show the full request/response. Imports no store/API.
	 */
	interface Props {
		when?: string;
		model?: string;
		status?: string | number;
		kind?: 'ok' | 'err';
		preview?: string;
		onOpen?: () => void;
	}
	let { when = '', model = '', status = '', kind = 'ok', preview = '', onOpen }: Props = $props();
</script>

<button class="logrow" data-testid="log-row" onclick={() => onOpen?.()}>
	<span class="when">{when}</span>
	<span class="lmodel">{model}</span>
	<span class="status {kind}">{status}</span>
	<span class="prev">{preview}</span>
	<span class="go" aria-hidden="true">›</span>
</button>

<style>
	.logrow {
		display: grid;
		grid-template-columns: 96px 128px 58px 1fr 20px;
		align-items: center;
		gap: 14px;
		padding: 13px 14px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		margin-bottom: 8px;
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: border-color 0.12s, box-shadow 0.12s;
	}
	.logrow:hover {
		border-color: var(--color-line-strong);
		box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04), 0 1px 3px rgba(16, 24, 40, 0.06);
	}
	.when {
		font-size: 12.5px;
		color: var(--color-text-muted);
	}
	.lmodel {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12px;
		color: var(--color-text);
	}
	.status {
		font-size: 11px;
		font-weight: 700;
		padding: 2px 8px;
		border-radius: 999px;
		text-align: center;
	}
	.status.ok {
		color: var(--color-green);
		background: var(--color-green-soft);
	}
	.status.err {
		color: var(--color-red);
		background: var(--color-red-soft);
	}
	.prev {
		font-size: 12.5px;
		color: var(--color-text-muted);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-family: var(--font-family-mono, ui-monospace, monospace);
	}
	.go {
		color: var(--color-faint);
	}
</style>

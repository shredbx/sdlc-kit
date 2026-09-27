<script lang="ts">
	/**
	 * ResponseView — a run's outcome (dumb, static). Four states, no motion:
	 *   idle    — nothing run yet: a quiet centered placeholder + the model that would run.
	 *   running — a plain "running…" line (no spinner).
	 *   done    — a status header (status · latency · tokens) over the response body.
	 *   error   — the error message in place of the body, header reads "error".
	 * Presentational twin of the run result — imports no store/API.
	 */
	interface Props {
		status?: string | number;
		latency?: string;
		tokens?: string | number;
		text?: string;
		error?: string | null;
		pending?: boolean;
		/** Idle-state copy + the model that will run (shown only before the first run). */
		placeholder?: string;
		/** Idle: the model that WOULD run. Done: the model the backend reports it served. */
		model?: string;
		/** Done-state extras — each renders in the status line only when provided. */
		stopReason?: string | null;
		tokensIn?: number;
		tokensOut?: number;
		/** Preformatted cost label (e.g. "$0.0041") — the host decides provider vs computed. */
		cost?: string | null;
		/** When set, a Copy button appears on a completed response and copies this text. */
		copyText?: string;
	}
	let {
		status,
		latency,
		tokens,
		text = '',
		error = null,
		pending = false,
		placeholder = 'Run to see the response',
		model = '',
		stopReason = null,
		tokensIn,
		tokensOut,
		cost = null,
		copyText
	}: Props = $props();

	const idle = $derived(!pending && !error && !text && (status === undefined || status === ''));
	// in→out split when the host provides it; the single total otherwise (back-compat).
	const tokenLabel = $derived(
		tokensIn != null && tokensOut != null
			? `${tokensIn}→${tokensOut} tok`
			: tokens != null
				? `${tokens} tok`
				: null
	);
	const statusLabel = $derived(
		[status, model, stopReason, latency, tokenLabel, cost]
			.filter((x) => x != null && x !== '')
			.join(' · ')
	);

	let copied = $state(false);
	function copy() {
		if (!copyText) return;
		try {
			void navigator.clipboard?.writeText(copyText);
		} catch {
			/* clipboard unavailable — the button is best-effort */
		}
		copied = true;
		setTimeout(() => (copied = false), 1200);
	}
</script>

<div class="response-view" data-testid="response-view">
	{#if idle}
		<div class="idle" data-testid="response-idle">
			<p class="idle-main">{placeholder}</p>
			{#if model}<p class="idle-sub">{model}</p>{/if}
		</div>
	{:else if pending}
		<div class="head"><span class="stat">running…</span></div>
		{#if text}<div class="resp">{text}</div>{/if}
	{:else if error}
		<div class="head"><span class="stat stat--err">error</span></div>
		<div class="resp resp--err">{error}</div>
	{:else}
		<div class="head">
			<span class="stat stat--ok">{statusLabel || '200'}</span>
			{#if copyText}
				<button type="button" class="copy" data-testid="response-copy" onclick={copy}>
					{copied ? 'Copied ✓' : 'Copy'}
				</button>
			{/if}
		</div>
		<div class="resp">{text}</div>
	{/if}
</div>

<style>
	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		margin-bottom: 10px;
	}
	.copy {
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text-muted);
		font-size: 11.5px;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		border-radius: 7px;
		padding: 3px 10px;
		cursor: pointer;
	}
	.copy:hover {
		color: var(--color-text);
		border-color: var(--color-line-strong);
	}
	.stat {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 12px;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		color: var(--color-text-muted);
		background: var(--color-subtle);
		border-radius: 999px;
		padding: 3px 11px;
	}
	.stat--ok {
		color: var(--color-green);
		background: var(--color-green-soft);
	}
	.stat--err {
		color: var(--color-red);
		background: var(--color-red-soft);
	}
	.resp {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12.5px;
		line-height: 1.62;
		background: var(--color-surface-2);
		color: var(--color-code);
		border: 1px solid var(--color-border);
		border-radius: 9px;
		padding: 14px 15px;
		white-space: pre-wrap;
		word-break: break-word;
	}
	.resp--err {
		color: var(--color-red);
	}
	.idle {
		min-height: 200px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 6px;
		text-align: center;
	}
	.idle-main {
		margin: 0;
		font-size: 13.5px;
		color: var(--color-text-muted);
	}
	.idle-sub {
		margin: 0;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12px;
		color: var(--color-faint);
	}
</style>

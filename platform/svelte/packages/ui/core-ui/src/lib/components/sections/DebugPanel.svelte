<script lang="ts">
	/**
	 * DebugPanel — everything sent & returned for a run (dumb): five collapsible sections over
	 * parameters, resolved variables, the raw request/response JSON, and usage/timing. Native
	 * <details> so open/close needs no JS. Imports no store/API.
	 */
	interface KV {
		key: string;
		value: string;
	}
	interface Props {
		parameters?: KV[];
		resolvedVars?: KV[];
		rawRequest?: string;
		rawResponse?: string;
		usage?: KV[];
		labels?: {
			parameters?: string;
			resolvedVars?: string;
			rawRequest?: string;
			rawResponse?: string;
			usageTiming?: string;
		};
	}
	let { parameters = [], resolvedVars = [], rawRequest = '', rawResponse = '', usage = [], labels = {} }: Props = $props();
	const L = $derived({
		parameters: labels.parameters ?? 'Parameters',
		resolvedVars: labels.resolvedVars ?? 'Resolved variables',
		rawRequest: labels.rawRequest ?? 'Raw request',
		rawResponse: labels.rawResponse ?? 'Raw response',
		usageTiming: labels.usageTiming ?? 'Usage & timing'
	});
</script>

<div class="debug-panel" data-testid="debug-panel">
	<details class="dbg" open>
		<summary><span>{L.parameters}</span><span class="tag">{parameters.length}</span></summary>
		<div class="dc"><dl class="kv">{#each parameters as p}<dt>{p.key}</dt><dd>{p.value}</dd>{/each}</dl></div>
	</details>
	<details class="dbg">
		<summary><span>{L.resolvedVars}</span><span class="tag">{resolvedVars.length}</span></summary>
		<div class="dc"><dl class="kv">{#each resolvedVars as v}<dt>${v.key}</dt><dd>"{v.value}"</dd>{/each}</dl></div>
	</details>
	<details class="dbg">
		<summary><span>{L.rawRequest}</span><span class="tag">json</span></summary>
		<div class="dc"><pre>{rawRequest}</pre></div>
	</details>
	<details class="dbg">
		<summary><span>{L.rawResponse}</span><span class="tag">json</span></summary>
		<div class="dc"><pre>{rawResponse}</pre></div>
	</details>
	<details class="dbg">
		<summary><span>{L.usageTiming}</span><span class="tag">{usage.length}</span></summary>
		<div class="dc"><dl class="kv">{#each usage as u}<dt>{u.key}</dt><dd>{u.value}</dd>{/each}</dl></div>
	</details>
</div>

<style>
	.dbg {
		border: 1px solid var(--color-border);
		border-radius: 9px;
		margin-bottom: 9px;
		background: var(--color-surface);
		overflow: hidden;
	}
	.dbg > summary {
		list-style: none;
		cursor: pointer;
		padding: 10px 13px;
		font-size: 12.5px;
		font-weight: 600;
		color: var(--color-text);
		display: flex;
		align-items: center;
		gap: 9px;
	}
	.dbg > summary::-webkit-details-marker {
		display: none;
	}
	.dbg > summary::before {
		content: '▸';
		color: var(--color-faint);
		font-size: 10px;
		transition: transform 0.12s;
	}
	.dbg[open] > summary::before {
		transform: rotate(90deg);
	}
	.dbg > summary .tag {
		margin-left: auto;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 11px;
		color: var(--color-faint);
		font-weight: 500;
	}
	.dc {
		padding: 0 13px 13px;
	}
	pre {
		margin: 0;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12px;
		line-height: 1.6;
		background: var(--color-surface-2);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 12px;
		color: var(--color-code);
		overflow-x: auto;
		white-space: pre-wrap;
	}
	.kv {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 6px 16px;
		font-size: 12.5px;
	}
	.kv dt {
		color: var(--color-text-muted);
		font-family: var(--font-family-mono, ui-monospace, monospace);
	}
	.kv dd {
		margin: 0;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		color: var(--color-text);
	}
</style>

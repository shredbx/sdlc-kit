<script lang="ts">
	/**
	 * ModelControls — the Run pane's model config (dumb). Provider/model selects, a temperature
	 * slider with a mono readout, max-tokens + response-format, and the Run button. Seeds local state
	 * from props and emits every change via callbacks; imports no store/API.
	 */
	interface Props {
		provider?: string;
		providers?: string[];
		model?: string;
		models?: string[];
		temperature?: number;
		maxTokens?: number;
		format?: string;
		formats?: string[];
		labels?: {
			provider?: string;
			model?: string;
			temperature?: string;
			maxTokens?: string;
			respFormat?: string;
			run?: string;
		};
		onProviderChange?: (v: string) => void;
		onModelChange?: (v: string) => void;
		onTemperatureChange?: (v: number) => void;
		onMaxTokensChange?: (v: number) => void;
		onFormatChange?: (v: string) => void;
		onRun?: () => void;
	}

	let {
		provider = '',
		providers = [],
		model = '',
		models = [],
		temperature = 0.2,
		maxTokens = 1024,
		format = 'JSON object',
		formats = ['JSON object', 'JSON schema', 'Text'],
		labels = {},
		onProviderChange,
		onModelChange,
		onTemperatureChange,
		onMaxTokensChange,
		onFormatChange,
		onRun
	}: Props = $props();

	const L = $derived({
		provider: labels.provider ?? 'Provider',
		model: labels.model ?? 'Model',
		temperature: labels.temperature ?? 'Temperature',
		maxTokens: labels.maxTokens ?? 'Max tokens',
		respFormat: labels.respFormat ?? 'Response format',
		run: labels.run ?? 'Run'
	});

	// Seed local editable state from the initial props. This form remounts per selection (the Run pane
	// is keyed on the open document), so a one-time seed is correct — no prop→state sync effect needed.
	let temp = $state(temperature);
	let maxTok = $state(maxTokens);
</script>

<div class="model-controls stack" data-testid="model-controls">
	<div class="grid2">
		<div>
			<div class="fldlabel">{L.provider}</div>
			<select class="sel" aria-label={L.provider} value={provider} onchange={(e) => onProviderChange?.(e.currentTarget.value)}>
				{#each providers as p}<option>{p}</option>{/each}
			</select>
		</div>
		<div>
			<div class="fldlabel">{L.model}</div>
			<select class="sel" aria-label={L.model} value={model} onchange={(e) => onModelChange?.(e.currentTarget.value)}>
				{#each models as m}<option>{m}</option>{/each}
			</select>
		</div>
	</div>

	<div>
		<div class="fldlabel">{L.temperature}</div>
		<div class="rng">
			<input
				type="range"
				min="0"
				max="1"
				step="0.05"
				aria-label={L.temperature}
				bind:value={temp}
				oninput={() => onTemperatureChange?.(+temp)}
			/>
			<span class="val">{(+temp).toFixed(2)}</span>
		</div>
	</div>

	<div class="grid2">
		<div>
			<div class="fldlabel">{L.maxTokens}</div>
			<input
				class="fld"
				aria-label={L.maxTokens}
				bind:value={maxTok}
				oninput={() => onMaxTokensChange?.(+maxTok)}
			/>
		</div>
		<div>
			<div class="fldlabel">{L.respFormat}</div>
			<select class="sel" aria-label={L.respFormat} value={format} onchange={(e) => onFormatChange?.(e.currentTarget.value)}>
				{#each formats as f}<option>{f}</option>{/each}
			</select>
		</div>
	</div>

	<button class="run-btn" onclick={() => onRun?.()}>▶ {L.run}</button>
</div>

<style>
	.stack {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.grid2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 14px;
	}
	.fldlabel {
		font-size: 11px;
		font-weight: 700;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 6px;
	}
	.fld,
	.sel {
		width: 100%;
		border: 1px solid var(--color-line-2);
		border-radius: 9px;
		padding: 9px 11px;
		font-size: 13.5px;
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
	}
	.sel {
		appearance: none;
		-webkit-appearance: none;
		padding-right: 30px;
	}
	.fld:focus,
	.sel:focus {
		border-color: var(--color-primary);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 14%, transparent);
		outline: none;
	}
	.rng {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.rng input[type='range'] {
		flex: 1;
		accent-color: var(--color-primary);
	}
	.rng .val {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 13px;
		color: var(--color-text);
		min-width: 34px;
		text-align: right;
	}
	.run-btn {
		border: 1px solid var(--color-primary);
		background: var(--color-primary);
		color: #fff;
		font-size: 13px;
		font-weight: 600;
		padding: 9px 13px;
		border-radius: 8px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 7px;
	}
	.run-btn:hover {
		filter: brightness(1.06);
	}
</style>

<script lang="ts">
	// EventTypePicker — pick an event_type, OR create one on the fly (D7 / S2).
	// PRESENTATION ONLY: it renders the options a DictionaryPort gives it and emits
	// the picked code; it imports no data source. The create affordance is INLINE
	// (a reveal-in-place input, never a redirect) so a host form never loses its
	// unsaved state — and auto-selects the new value. Generic over `dict`, so the
	// same control later serves person_type / relation pickers.
	//
	// Controlled: the host owns `value` and reacts to `onSelect`. No `any`.

	import { onMount, untrack } from 'svelte';
	import type { DictionaryPort, DictName, DictOption } from './types.js';

	interface Props {
		/** The currently-selected dict code (controlled by the host). */
		value: string;
		/** The dictionary source — the mock now, an HTTP port later. */
		port: DictionaryPort;
		/** Emitted with the picked (or freshly created) code. */
		onSelect: (code: string) => void;
		/** Which dictionary to pick from. Default `event_type`. */
		dict?: DictName;
		/** `id` for host `<label for>` association. */
		id?: string;
		/** Accessible name when the host doesn't wrap this in a labelled Field. */
		ariaLabel?: string;
		/** Disable the whole control (e.g. while the form submits). */
		disabled?: boolean;
	}

	let {
		value,
		port,
		onSelect,
		dict = 'event_type',
		id,
		ariaLabel = 'Event type',
		disabled = false
	}: Props = $props();

	let options = $state<DictOption[]>([]);
	let loading = $state(true);

	// Mirror of `value` the <select> binds to. Synced one-way from the prop so the
	// host stays the source of truth; `onchange` pushes the new code back out.
	let selected = $state(untrack(() => value));
	$effect(() => {
		const v = value;
		untrack(() => {
			if (v !== selected) selected = v;
		});
	});

	onMount(async () => {
		try {
			options = await port.options(dict);
		} finally {
			loading = false;
		}
	});

	function handleChange(): void {
		onSelect(selected);
	}

	// ── Create-on-the-fly (inline, no redirect) ──────────────────────────────────
	let creating = $state(false);
	let draftLabel = $state('');
	let busy = $state(false);
	let inputEl = $state<HTMLInputElement | null>(null);

	function openCreate(): void {
		creating = true;
		draftLabel = '';
		// Focus the input once it renders.
		queueMicrotask(() => inputEl?.focus());
	}
	function cancelCreate(): void {
		creating = false;
		draftLabel = '';
	}

	async function submitCreate(): Promise<void> {
		const label = draftLabel.trim();
		if (!label || busy) return;
		busy = true;
		try {
			// The port dedupes/normalizes (A8): an existing label returns its option.
			const opt = await port.create(dict, label);
			if (!options.some((o) => o.code === opt.code)) {
				options = [...options, opt];
			}
			selected = opt.code;
			onSelect(opt.code);
			creating = false;
			draftLabel = '';
		} finally {
			busy = false;
		}
	}

	function onInputKeydown(event: KeyboardEvent): void {
		if (event.key === 'Enter') {
			event.preventDefault();
			void submitCreate();
		} else if (event.key === 'Escape') {
			event.preventDefault();
			cancelCreate();
		}
	}
</script>

<div class="etp">
	<div class="etp-row">
		<select
			{id}
			class="etp-select"
			aria-label={ariaLabel}
			bind:value={selected}
			onchange={handleChange}
			disabled={disabled || loading}
		>
			{#if loading}
				<option value={selected}>Loading…</option>
			{:else}
				{#each options as opt (opt.code)}
					<option value={opt.code}>{opt.label}</option>
				{/each}
				<!-- When the current value isn't in the option list (e.g. a code created
				     elsewhere), keep it selectable rather than silently dropping it. -->
				{#if selected && !options.some((o) => o.code === selected)}
					<option value={selected}>{selected}</option>
				{/if}
			{/if}
		</select>

		<button
			type="button"
			class="etp-new"
			onclick={openCreate}
			disabled={disabled || creating}
		>
			+ New
		</button>
	</div>

	{#if creating}
		<div class="etp-create">
			<input
				bind:this={inputEl}
				bind:value={draftLabel}
				class="etp-input"
				type="text"
				name="new-event-type"
				autocomplete="off"
				placeholder="New type — e.g. Key handover"
				aria-label="New event type name"
				onkeydown={onInputKeydown}
				disabled={busy}
			/>
			<div class="etp-create-actions">
				<button type="button" class="etp-cancel" onclick={cancelCreate} disabled={busy}>
					Cancel
				</button>
				<button
					type="button"
					class="etp-create-btn"
					onclick={submitCreate}
					disabled={busy || !draftLabel.trim()}
				>
					Create
				</button>
			</div>
		</div>
	{/if}
</div>

<style>
	.etp {
		--cal-accent: #e5392b;
		--cal-text: var(--color-text, #1d1d1b);
		--cal-text-muted: var(--color-text-muted, #8a8a85);
		--cal-border: var(--color-border, #e4e4e2);
		--cal-bg: var(--color-bg, #ffffff);

		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.etp-row {
		display: flex;
		gap: 0.5rem;
		align-items: stretch;
	}

	.etp-select {
		flex: 1 1 auto;
		min-width: 0;
		font: inherit;
		font-size: 0.9rem;
		color: var(--cal-text);
		background: var(--cal-bg);
		border: 1px solid var(--cal-border);
		border-radius: 0.4rem;
		padding: 0.45rem 0.6rem;
		cursor: pointer;
	}
	.etp-select:focus-visible {
		outline: 2px solid var(--cal-accent);
		outline-offset: 1px;
	}
	.etp-select:disabled {
		cursor: default;
		opacity: 0.7;
	}

	.etp-new {
		flex: 0 0 auto;
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--cal-text);
		background: transparent;
		border: 1px solid var(--cal-border);
		border-radius: 0.4rem;
		padding: 0.45rem 0.7rem;
		cursor: pointer;
		white-space: nowrap;
	}
	.etp-new:hover:not(:disabled) {
		border-color: var(--cal-accent);
		color: var(--cal-accent);
	}
	.etp-new:focus-visible {
		outline: 2px solid var(--cal-accent);
		outline-offset: 1px;
	}
	.etp-new:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.etp-create {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		padding: 0.5rem;
		border: 1px dashed var(--cal-border);
		border-radius: 0.4rem;
	}
	.etp-input {
		font: inherit;
		font-size: 0.9rem;
		color: var(--cal-text);
		background: var(--cal-bg);
		border: 1px solid var(--cal-border);
		border-radius: 0.4rem;
		padding: 0.45rem 0.6rem;
	}
	.etp-input:focus-visible {
		outline: 2px solid var(--cal-accent);
		outline-offset: 1px;
	}

	/* CTAs right-aligned (house rule). */
	.etp-create-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.4rem;
	}
	.etp-cancel,
	.etp-create-btn {
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		border-radius: 0.4rem;
		padding: 0.35rem 0.8rem;
		cursor: pointer;
	}
	.etp-cancel {
		color: var(--cal-text-muted);
		background: transparent;
		border: 1px solid var(--cal-border);
	}
	.etp-cancel:hover:not(:disabled) {
		color: var(--cal-text);
	}
	.etp-create-btn {
		color: #ffffff;
		background: var(--cal-accent);
		border: 1px solid var(--cal-accent);
	}
	.etp-create-btn:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.etp-cancel:focus-visible,
	.etp-create-btn:focus-visible {
		outline: 2px solid var(--cal-accent);
		outline-offset: 1px;
	}
</style>

<script lang="ts" module>
	import type { SourceProvider, SourceRecord } from '@sbx/canvas-kit';
	import type {
		SourcePickerSelect,
		SourcePickerView,
		SourcePickerCategoryDisplay
	} from './SourcePicker.svelte';

	/** Public prop contract — consumers import this for typed usage. */
	export interface SourceBrowserProps {
		/** Providers to browse across. ONE → the kind chooser is skipped (nothing to choose);
		 *  TWO+ → a kind chooser precedes the picker and a Back returns to it. */
		providers: SourceProvider[];
		/** Open directly on this kind (e.g. a Settings pre-selection). Still backable when >1 provider. */
		initialKind?: string;
		/** 'single' (default) click-and-close · 'multi' checkbox-select + Add(N). Forwarded to the picker. */
		select?: SourcePickerSelect;
		/** Single-select WITH a confirm step (radio + footer) — forwarded; only meaningful with select='single'. */
		confirm?: boolean;
		/** Footer button label for the single-confirm step (default 'Select'). Forwarded. */
		confirmLabel?: string;
		/** Pre-selected record ids for the active kind — multi pre-checks, single+confirm pre-highlights.
		 *  Forwarded; re-seeds on a kind switch. */
		selectedIds?: string[];
		/** Row rendering — 'list' (default) | 'card'. Forwarded. */
		view?: SourcePickerView;
		/** Category control — 'auto' (default) | 'tabs' | 'dropdown'. Forwarded. */
		categoryDisplay?: SourcePickerCategoryDisplay;
		/** select='single' — a record was chosen; carries the chosen kind alongside the record. */
		onpick?: (kind: string, record: SourceRecord) => void;
		/** select='multi' — selection confirmed; carries the chosen kind alongside the records. */
		onconfirm?: (kind: string, records: SourceRecord[]) => void;
	}
</script>

<script lang="ts">
	// Source browser (2606-012) — a thin two-step wrapper over <SourcePicker> that lets ONE
	// dialog span multiple source KINDS (property · guide · service · news). Step 1 is a kind
	// chooser (provider labels); picking one mounts the UNCHANGED single-provider SourcePicker
	// for that kind. The proven picker stays untouched — kind-spanning is purely additive.
	// Selection bubbles up WITH its kind so the caller resolves + maps the verb per kind.
	//
	// ONE provider → the chooser is skipped (nothing to choose). `initialKind` opens straight on
	// a kind; a Back returns to the chooser whenever there is more than one provider.
	import { untrack } from 'svelte';
	import { Icon } from '@sbx/core-ui/components/primitives';
	import SourcePicker from './SourcePicker.svelte';
	// `SourceProvider`/`SourceRecord` (from @sbx/canvas-kit) and `SourceBrowserProps` are
	// declared in the <script module> block above and are in scope here.

	let {
		providers,
		initialKind,
		select = 'single',
		confirm = false,
		confirmLabel = 'Select',
		selectedIds,
		view = 'list',
		categoryDisplay = 'auto',
		onpick,
		onconfirm
	}: SourceBrowserProps = $props();

	// A chooser only makes sense with >1 provider; with one we open straight on it. Derived so
	// the template stays reactive if the provider set ever changes; the one-time seed below
	// reads the length directly inside untrack (intended initial-value capture).
	const canChooseKind = $derived(providers.length > 1);
	let chosenKind = $state<string | null>(
		untrack(() => initialKind ?? (providers.length > 1 ? null : (providers[0]?.kind ?? null)))
	);
	const chosenProvider = $derived(providers.find((p) => p.kind === chosenKind));
</script>

<div class="browser">
	{#if chosenProvider}
		{#if canChooseKind}
			<button type="button" class="back" onclick={() => (chosenKind = null)}>
				<Icon name="chevron-left" size="sm" />
				<span>All sources</span>
			</button>
		{/if}
		<!-- Remount the picker per kind: SourcePicker seeds the provider's facets/categories
		     ONCE at mount, so a kind switch must re-seed (the {#key} forces it). -->
		{#key chosenKind}
			<SourcePicker
				provider={chosenProvider}
				{select}
				{confirm}
				{confirmLabel}
				{selectedIds}
				{view}
				{categoryDisplay}
				onpick={(record) => onpick?.(chosenKind!, record)}
				onconfirm={(records) => onconfirm?.(chosenKind!, records)}
			/>
		{/key}
	{:else}
		<ul class="kinds" role="list" aria-label="Source types">
			{#each providers as provider (provider.kind)}
				<li>
					<button type="button" class="kind" onclick={() => (chosenKind = provider.kind)}>
						<span class="kind__label">{provider.label}</span>
						<span class="kind__chev" aria-hidden="true"><Icon name="chevron-right" size="sm" /></span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.browser {
		display: flex;
		flex-direction: column;
		gap: var(--br-space-sm, 0.5rem);
		min-width: 22rem;
	}

	/* Back to the kind chooser — left-aligned, quiet (navigation affordance, not a CTA). */
	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		align-self: flex-start;
		padding: 0.25rem 0.375rem 0.25rem 0.125rem;
		border: none;
		background: transparent;
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--br-color-primary, #0d4f4f);
		cursor: pointer;
	}

	.back:hover {
		text-decoration: underline;
	}

	.kinds {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.kind {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--br-space-sm, 0.5rem);
		width: 100%;
		padding: 0.75rem 0.875rem;
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: var(--br-radius-sm, 0.375rem);
		background: transparent;
		text-align: left;
		color: var(--br-color-neutral-800, #202018);
		cursor: pointer;
	}

	.kind:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
		border-color: var(--br-color-primary, #0d4f4f);
	}

	.kind__label {
		font-size: 0.9375rem;
		font-weight: 500;
	}

	.kind__chev {
		display: inline-flex;
		color: var(--br-color-neutral-400, #a0a098);
	}
</style>

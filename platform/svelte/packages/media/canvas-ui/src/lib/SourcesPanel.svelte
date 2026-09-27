<script lang="ts">
	// Sources context panel (IA refactor R1) — the 2nd rail item. Replaces the
	// dissolved Settings gear's sources half and grows it into its own panel: the
	// attached source records grouped BY KIND, one CollapsibleSection per registered
	// provider kind. Two add affordances (both, by design):
	//   header  "＋ ▾"        add a source of ANY kind — opens the attach MODAL at the
	//                          kind chooser (EditorShell owns it; null = chooser).
	//   section "＋"          add another source of THIS kind — opens the modal
	//                          pre-selected to the section's kind.
	// Both call back to EditorShell — never a redirect (hard rule). Each attached
	// record renders as the shared SourceCell (thumb + title + detach ✕).
	import { Icon } from '@sbx/core-ui/components/primitives';
	import type { SourceProvider, SourceRef } from '@sbx/canvas-kit';
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import EmptyState from './EmptyState.svelte';
	import SourceCell from './SourceCell.svelte';

	interface Props {
		sources?: SourceRef[];
		sourceProviders?: SourceProvider[];
		/** Open the attach MODAL (never a redirect — hard rule). `null` = kind chooser
		 *  (header add); a kind string = pre-select that kind (per-section add). */
		onaddsource?: (kind: string | null) => void;
		/** Re-pick the record for an attached alias (same-alias replace — bindings follow). */
		onchangesource?: (alias: string) => void;
		/** Detach an attached source by alias (its bindings then fall back). */
		ondetachsource?: (alias: string) => void;
	}

	let { sources = [], sourceProviders = [], onaddsource, onchangesource, ondetachsource }: Props =
		$props();

	/** Attached sources for a provider kind (each section lists its own). */
	function sourcesOfKind(kind: string): SourceRef[] {
		return sources.filter((s) => s.kind === kind);
	}
</script>

<Panel title="Sources" search={false}>
	{#snippet headerAction()}
		{#if sourceProviders.length > 0}
			<button
				class="add-any"
				type="button"
				aria-label="Attach a source"
				title="Attach a source"
				onclick={() => onaddsource?.(null)}
			>
				<Icon name="plus" size="sm" />
				<Icon name="chevron-down" size="xs" />
			</button>
		{/if}
	{/snippet}

	{#if sourceProviders.length === 0}
		<EmptyState text="No source provider is available." />
	{:else}
		{#each sourceProviders as provider (provider.kind)}
			{@const kindSources = sourcesOfKind(provider.kind)}
			<CollapsibleSection title={provider.label} count={kindSources.length} open={kindSources.length > 0}>
				{#snippet actions()}
					<button
						class="add-kind"
						type="button"
						aria-label={`Add ${provider.label}`}
						title={`Add ${provider.label}`}
						onclick={() => onaddsource?.(provider.kind)}
					>
						<Icon name="plus" size="sm" />
					</button>
				{/snippet}
				{#if kindSources.length > 0}
					<ul class="source-list">
						{#each kindSources as src (src.id)}
							<SourceCell source={src} onchange={onchangesource} ondetach={ondetachsource} />
						{/each}
					</ul>
				{:else}
					<EmptyState text={`No ${provider.label.toLowerCase()} attached yet — ＋ to add one.`} />
				{/if}
			</CollapsibleSection>
		{/each}
	{/if}
</Panel>

<style>
	/* Header "add any kind" — plus + chevron (opens the kind chooser modal). */
	.add-any {
		display: inline-flex;
		align-items: center;
		gap: 0.0625rem;
		padding: 0.25rem 0.375rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
	}

	.add-any:hover {
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-900, #111111);
	}

	/* Per-section "add this kind" — compact icon button (right-aligned, hard rule). */
	.add-kind {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.add-kind:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
		color: var(--cv-color-primary, #333333);
	}

	.source-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
</style>

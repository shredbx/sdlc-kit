<script lang="ts">
	// Components context panel (§11.6, redline 2) — drawing / shapes / widgets.
	// Sections:
	//   Drawing  pencil + brush tool buttons (kept — these are tools, not a palette)
	//   Shapes   ComponentGrid of the `shape` presets (rectangle · ellipse · line · triangle)
	//   Widgets  ComponentGrid of the `widget` presets (price tag · spec badge · image callout)
	// Shapes + Widgets use the SHARED ComponentGrid/ComponentCard (Fix 4) reading
	// the D15 preset catalogue — no bespoke card design. Clicking a card inserts the
	// preset at the page centre; dragging it places at the drop point (D-2).
	// element = layer (D6).
	import { Icon } from '@sbx/core-ui/components/primitives';
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import ComponentGrid from './ComponentGrid.svelte';
	import ComponentCard from './ComponentCard.svelte';
	import { presetsForSection } from './palette.js';
	import { startPresetDrag } from './dnd.js';

	interface Tool {
		id: string;
		label: string;
		icon: string;
	}

	interface Props {
		/** Activate a drawing tool (stubbed → wired with the draw-tools iteration). */
		ontool?: (toolId: string) => void;
		/** Insert a shape/widget preset — a card click ARMS the tool (shapes) or drops it
		 *  in the page centre (widgets), per the preset's placement mode (handled by the shell). */
		oninsert?: (presetId: string) => void;
		/** The armed placement tool's preset id — the matching shape card reads as selected. */
		activeTool?: string | null;
	}

	let { ontool, oninsert, activeTool }: Props = $props();

	const tools: Tool[] = [
		{ id: 'pencil', label: 'Pencil', icon: 'pencil' },
		{ id: 'brush', label: 'Brush', icon: 'paintbrush' }
	];

	const shapePresets = presetsForSection('shape');
	const widgetPresets = presetsForSection('widget');
</script>

<Panel title="Components" searchPlaceholder="Search components…">
	<CollapsibleSection title="Drawing">
		<div class="tools">
			{#each tools as tool (tool.id)}
				<button class="tool" type="button" aria-label={tool.label} title={tool.label} onclick={() => ontool?.(tool.id)}>
					<Icon name={tool.icon} size="sm" />
					<span class="tool__label">{tool.label}</span>
				</button>
			{/each}
		</div>
	</CollapsibleSection>

	<CollapsibleSection title="Shapes">
		<ComponentGrid>
			{#each shapePresets as preset (preset.id)}
				<ComponentCard
					label={preset.label}
					icon={preset.icon}
					selected={activeTool === preset.id}
					onclick={() => oninsert?.(preset.id)}
					ondragstart={(e) => startPresetDrag(e, preset.id)}
				/>
			{/each}
		</ComponentGrid>
	</CollapsibleSection>

	<CollapsibleSection title="Widgets">
		<ComponentGrid>
			{#each widgetPresets as preset (preset.id)}
				<ComponentCard
					label={preset.label}
					icon={preset.icon}
					onclick={() => oninsert?.(preset.id)}
					ondragstart={(e) => startPresetDrag(e, preset.id)}
				/>
			{/each}
		</ComponentGrid>
	</CollapsibleSection>
</Panel>

<style>
	.tools {
		display: flex;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.tool {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.75rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
	}

	.tool:hover {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.tool__label {
		font-size: 0.75rem;
		font-weight: 500;
	}
</style>

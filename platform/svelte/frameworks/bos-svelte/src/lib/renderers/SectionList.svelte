<script lang="ts">
	// Renders an ordered list of sections through the registry — dispatch by data, never a
	// hand-written switch on renderer id (docs/proposals/bos-page-designer.md §5). A section
	// whose renderer id isn't registered is skipped with a visible note rather than crashing the
	// page, since content can reference a renderer that hasn't shipped yet.
	import { getRenderer } from './registry';
	import type { SectionData } from './types';

	let { sections }: { sections: SectionData[] } = $props();
</script>

{#each sections as section, i (i)}
	{#if !section.hidden}
		{@const renderer = getRenderer(section.kind)}
		{#if renderer}
			<renderer.component content={(section[section.kind] as Record<string, unknown>) ?? {}} />
		{:else}
			<p class="renderer-missing">Unknown renderer: {section.kind}</p>
		{/if}
	{/if}
{/each}

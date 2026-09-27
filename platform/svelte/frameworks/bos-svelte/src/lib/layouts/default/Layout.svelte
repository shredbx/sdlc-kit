<script lang="ts">
	import { SectionList } from '../../renderers';
	import type { SectionData } from '../../renderers';

	let { sections }: { sections: SectionData[] } = $props();

	// Placement by SectionKind, not a stored slot (see ../types.ts's doc comment):
	// the first-class distinction this layout offers is hero vs. everything else.
	const hero = $derived(sections.filter((s) => s.kind === 'hero'));
	const main = $derived(sections.filter((s) => s.kind !== 'hero'));
</script>

{#if hero.length}
	<div class="layout-region layout-region--hero">
		<SectionList sections={hero} />
	</div>
{/if}
<div class="layout-region layout-region--main">
	<SectionList sections={main} />
</div>

<style>
	.layout-region--main {
		max-width: 48rem;
		margin: 0 auto;
		padding: 0 1.5rem 4rem;
	}
</style>

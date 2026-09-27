<script lang="ts">
	import { SectionList } from '../../renderers';
	import type { SectionData } from '../../renderers';

	let { sections }: { sections: SectionData[] } = $props();

	// No hero region exists for this layout — a hero-kind section is structurally
	// excluded rather than merely discouraged, so a legal page can never visually
	// produce a hero no matter what its data contains (bos-layout-presets.md §3).
	const main = $derived(sections.filter((s) => s.kind !== 'hero'));
</script>

<div class="layout-region layout-region--main layout-region--legal">
	<SectionList sections={main} />
</div>

<style>
	.layout-region--legal {
		max-width: 40rem;
		margin: 0 auto;
		padding: 3rem 1.5rem 4rem;
	}
</style>

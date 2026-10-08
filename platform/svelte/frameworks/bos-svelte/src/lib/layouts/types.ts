import type { Component } from 'svelte';
import type { SectionData } from '../renderers';

/** One entry in the layout registry (docs/proposals/bos-layout-presets.md §3, corrected: a
 * layout's component partitions the `sections` it's given by SectionKind — e.g. "hero"-kind
 * sections go in the hero region — rather than reading a separate stored slot field, since the
 * cms kit's Section envelope has no such field). `regions` is metadata only (labels the admin's
 * per-region section lists); the placement logic itself lives in the component. */
export interface RegisteredLayout {
	id: string;
	label: string;
	regions: string[];
	component: Component<{ sections: SectionData[] }>;
}

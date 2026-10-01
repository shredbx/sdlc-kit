import type { IPresenter } from '../../presenter/IPresenter';
import PresetPill from './PresetPill.svelte';

const presenter: IPresenter = {
	name: 'PresetPill',
	slug: 'preset-pill',
	category: 'sections',
	presenterType: 'component',
	component: PresetPill,
	description: 'Provenance chip for a prompt section — linked (iris) / edited (amber) / custom (muted).',
	variants: [
		{ name: 'Linked', props: { status: 'linked', presetName: 'evaluator' } },
		{ name: 'Edited', props: { status: 'edited', presetName: 'warm' } },
		{ name: 'Custom', props: { status: 'custom' } }
	]
};

export default presenter;

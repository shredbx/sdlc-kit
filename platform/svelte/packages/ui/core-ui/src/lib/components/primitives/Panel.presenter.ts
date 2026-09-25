import type { IPresenter } from '../../presenter/IPresenter';
import Panel from './Panel.svelte';

const presenter: IPresenter = {
	name: 'Panel',
	slug: 'panel',
	category: 'primitives',
	presenterType: 'component',
	component: Panel,
	description: 'Bordered header+body card — the Run pane’s repeating shell (title + optional sub, padded body).',
	variants: [
		{ name: 'Titled', props: { title: 'Variables', sub: 'values for this run' } },
		{ name: 'Plain', props: { title: 'Model' } }
	]
};

export default presenter;

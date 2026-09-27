import RevealGroup from './RevealGroup.svelte';
import type { IPresenter, IVariant, IPropDef } from '../presenter/IPresenter';

const variants: IVariant[] = [
	{
		name: 'Minimal preset',
		description: 'Default preset — children use fade reveal.',
		props: { preset: 'minimal' }
	},
	{
		name: 'Expressive preset',
		description: 'Children inherit zoom-spring motion.',
		props: { preset: 'expressive' }
	},
	{
		name: 'Directional preset',
		description: 'Children inherit slide-up motion.',
		props: { preset: 'directional' }
	}
];

const props: IPropDef[] = [
	{
		name: 'preset',
		type: 'select',
		description: 'Overrides the transition preset for all child <Reveal> components via Svelte context.',
		options: ['minimal', 'expressive', 'directional'],
		default: 'minimal'
	},
	{ name: 'as', type: 'select', description: 'HTML tag to render.', options: ['div','section','article'], default: 'div' }
];

const presenter: IPresenter = {
	name: 'RevealGroup',
	slug: 'reveal-group',
	category: 'transitions',
	presenterType: 'component',
	component: RevealGroup,
	description: 'Section-level preset override via Svelte context. Wraps child <Reveal> components and provides a unified preset without polluting individual component props. Decision #0220.',
	variants,
	props,
	tags: ['reveal', 'group', 'context', 'preset', 'animation'],
	status: 'stable',
	version: '1.0.0'
};

export default presenter;

import PageTransition from './PageTransition.svelte';
import type { IPresenter, IVariant, IPropDef } from '../presenter/IPresenter';

const variants: IVariant[] = [
	{ name: 'Fade (default)', description: 'Opacity crossfade between routes.', props: { type: 'fade' } },
	{ name: 'Slide', description: 'Horizontal slide between routes.', props: { type: 'slide' } },
	{ name: 'Blur', description: 'Blur-in/out route transition.', props: { type: 'blur' } },
	{ name: 'Mask wipe', description: 'Clip-path wipe between routes.', props: { type: 'mask-wipe' } }
];

const props: IPropDef[] = [
	{
		name: 'type',
		type: 'select',
		description: 'Transition variant. Maps to [data-page-transition] on <html>.',
		options: ['fade', 'slide', 'blur', 'mask-wipe'],
		default: 'fade'
	},
	{
		name: 'duration',
		type: 'number',
		description: 'Duration override in ms. Defaults to --duration-normal token.',
		default: 0
	}
];

const presenter: IPresenter = {
	name: 'PageTransition',
	slug: 'page-transition',
	category: 'transitions',
	presenterType: 'component',
	component: PageTransition,
	description: 'Route-level SvelteKit transition using the native View Transitions API (document.startViewTransition). Wraps +layout.svelte children. Gracefully degrades on browsers without support (~90.5% coverage Apr 2026). Decision #0216.',
	variants,
	props,
	tags: ['page', 'route', 'transition', 'sveltekit', 'view-transitions', 'navigation'],
	status: 'stable',
	version: '1.0.0'
};

export default presenter;

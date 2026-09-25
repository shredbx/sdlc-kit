import DotNav from './DotNav.svelte';
import type { IPresenter, IPropDef } from '../presenter/IPresenter';

const props: IPropDef[] = [
	{
		name: 'items',
		type: 'json',
		description: 'Array of {id, label} section targets. Auto-discovers [data-dot-nav-section] if omitted.',
		default: []
	},
	{ name: 'offset', type: 'number', description: 'Scroll offset in px (for sticky headers).', default: 0 },
	{ name: 'rootMargin', type: 'string', description: 'IntersectionObserver rootMargin.', default: '0px' },
];

const presenter: IPresenter = {
	name: 'DotNav',
	slug: 'dot-nav',
	category: 'navigation',
	presenterType: 'component',
	component: DotNav,
	description: 'Fixed right-rail scroll-spy navigation. Highlights the active section dot as the user scrolls.',
	props,
	tags: ['navigation', 'scroll', 'spy', 'dots', 'fixed'],
	status: 'stable',
	version: '1.0.0'
};

export default presenter;

import type { IPresenter } from '../../presenter/IPresenter';
import ArgToken from './ArgToken.svelte';

const presenter: IPresenter = {
	name: 'ArgToken',
	slug: 'arg-token',
	category: 'sections',
	presenterType: 'component',
	component: ArgToken,
	description: 'Inline `$variable` reference chip rendered within a section’s content.',
	variants: [
		{ name: 'Product', props: { name: 'product' } },
		{ name: 'Audience', props: { name: 'audience' } }
	]
};

export default presenter;

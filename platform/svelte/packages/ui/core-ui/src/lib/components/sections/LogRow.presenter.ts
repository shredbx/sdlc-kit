import type { IPresenter } from '../../presenter/IPresenter';
import LogRow from './LogRow.svelte';

const presenter: IPresenter = {
	name: 'LogRow',
	slug: 'log-row',
	category: 'sections',
	presenterType: 'component',
	component: LogRow,
	description: 'One run-history row — when · model · status · preview · chevron.',
	variants: [
		{
			name: 'OK',
			props: { when: '2 min ago', model: 'sonnet-5', status: 200, kind: 'ok', preview: 'grade reply… → [{ "accuracy": 4 }]' }
		},
		{
			name: 'Error',
			props: { when: '1 hr ago', model: 'sonnet-5', status: 422, kind: 'err', preview: 'grade reply… → error: $product was empty' }
		}
	]
};

export default presenter;

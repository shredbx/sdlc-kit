import type { IPresenter } from '../../presenter/IPresenter';
import ResponseView from './ResponseView.svelte';

const presenter: IPresenter = {
	name: 'ResponseView',
	slug: 'response-view',
	category: 'sections',
	presenterType: 'component',
	component: ResponseView,
	description: 'A run’s outcome — status pill + response body, or the error message on failure.',
	variants: [
		{
			name: 'Success',
			props: {
				status: 200,
				latency: '1.4s',
				tokens: 512,
				text: '[{ "criterion": "accuracy", "score": 4 }]'
			}
		},
		{
			name: 'Success (full metadata + copy)',
			props: {
				status: 200,
				model: 'claude-haiku-4-5',
				stopReason: 'end_turn',
				latency: '1.4s',
				tokensIn: 1512,
				tokensOut: 231,
				cost: '$0.0041',
				copyText: '[{ "criterion": "accuracy", "score": 4 }]',
				text: '[{ "criterion": "accuracy", "score": 4 }]'
			}
		},
		{
			name: 'Error',
			props: { error: 'anthropic rate-limited this request. Try again in a moment.' }
		}
	]
};

export default presenter;

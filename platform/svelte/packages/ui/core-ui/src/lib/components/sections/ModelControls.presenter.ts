import type { IPresenter } from '../../presenter/IPresenter';
import ModelControls from './ModelControls.svelte';

const presenter: IPresenter = {
	name: 'ModelControls',
	slug: 'model-controls',
	category: 'sections',
	presenterType: 'component',
	component: ModelControls,
	description: 'Run-pane model config — provider/model, temperature, max tokens, response format, Run.',
	variants: [
		{
			name: 'Default',
			props: {
				provider: 'Anthropic',
				providers: ['Anthropic', 'OpenAI', 'Google'],
				model: 'Claude Sonnet 5',
				models: ['Claude Sonnet 5', 'Claude Opus 4.8'],
				temperature: 0.2,
				maxTokens: 1024
			}
		}
	]
};

export default presenter;

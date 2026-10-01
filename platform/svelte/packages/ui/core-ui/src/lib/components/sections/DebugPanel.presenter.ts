import type { IPresenter } from '../../presenter/IPresenter';
import DebugPanel from './DebugPanel.svelte';

const presenter: IPresenter = {
	name: 'DebugPanel',
	slug: 'debug-panel',
	category: 'sections',
	presenterType: 'component',
	component: DebugPanel,
	description: 'Everything sent & returned for a run — parameters, resolved vars, raw request/response, usage.',
	variants: [
		{
			name: 'Default',
			props: {
				parameters: [
					{ key: 'provider', value: 'anthropic' },
					{ key: 'model', value: 'claude-sonnet-5' },
					{ key: 'temperature', value: '0.2' }
				],
				resolvedVars: [
					{ key: 'product', value: 'Bestie' },
					{ key: 'audience', value: 'home buyers' }
				],
				rawRequest: '{ "model": "claude-sonnet-5" }',
				rawResponse: '{ "stop_reason": "end_turn" }',
				usage: [{ key: 'input_tokens', value: '318' }]
			}
		}
	]
};

export default presenter;

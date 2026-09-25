import type { IPresenter } from '../../presenter/IPresenter';
import SectionBlock from './SectionBlock.svelte';

const presenter: IPresenter = {
	name: 'SectionBlock',
	slug: 'section-block',
	category: 'sections',
	presenterType: 'component',
	component: SectionBlock,
	description: 'One prompt section on the Edit surface — provenance pill, editable content with inline $arg chips, nested children.',
	variants: [
		{
			name: 'Linked',
			props: {
				node: {
					name: 'role',
					description: 'Sets who the model is — one or two sentences.',
					content: 'You are a meticulous QA reviewer for a $product support team. You grade replies for $audience.',
					status: 'linked',
					presetName: 'evaluator',
					uses: ['product', 'audience']
				}
			}
		},
		{
			name: 'Custom',
			props: {
				node: {
					name: 'task',
					description: 'The single job.',
					content: 'Read the reply and score it against every criterion in $criteria. Return one JSON object.',
					status: 'custom',
					uses: ['criteria']
				}
			}
		},
		{
			name: 'Nested with children',
			props: {
				node: {
					name: 'criteria',
					description: 'The named qualities you are grading.',
					content: '',
					status: 'custom',
					children: [
						{ name: 'accuracy', content: 'Every claim is supported.', status: 'linked', presetName: 'strict' },
						{ name: 'tone', content: 'Warm and human, never dismissive.', status: 'edited', presetName: 'warm' }
					]
				}
			}
		}
	]
};

export default presenter;

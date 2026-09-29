import type { RegisteredRenderer } from '../types';
import Prose from './Prose.svelte';

export const registeredRenderer: RegisteredRenderer = {
	id: 'prose',
	label: 'Prose',
	shape: 'single',
	fields: [{ key: 'markdown', label: 'Body', type: 'markdown' }],
	component: Prose
};

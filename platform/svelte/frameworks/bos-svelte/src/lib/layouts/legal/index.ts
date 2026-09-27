import Layout from './Layout.svelte';
import type { RegisteredLayout } from '../types';

export const registeredLayout: RegisteredLayout = {
	id: 'legal',
	label: 'Legal (main only)',
	regions: ['main'],
	component: Layout
};

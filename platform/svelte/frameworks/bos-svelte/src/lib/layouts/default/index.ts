import Layout from './Layout.svelte';
import type { RegisteredLayout } from '../types';

export const registeredLayout: RegisteredLayout = {
	id: 'default',
	label: 'Default (hero + main)',
	regions: ['hero', 'main'],
	component: Layout
};

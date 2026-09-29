import type { RegisteredFooter } from '../types';
import Footer from './Footer.svelte';

export const registeredChrome: RegisteredFooter = {
	id: 'default',
	label: 'Default',
	component: Footer
};

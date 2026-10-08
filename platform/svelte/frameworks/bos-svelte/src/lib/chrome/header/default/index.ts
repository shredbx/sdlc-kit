import type { RegisteredHeader } from '../types';
import Header from './Header.svelte';

export const registeredChrome: RegisteredHeader = {
	id: 'default',
	label: 'Default',
	component: Header
};

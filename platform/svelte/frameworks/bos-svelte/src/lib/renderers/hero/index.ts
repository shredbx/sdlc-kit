import type { RegisteredRenderer } from '../types';
import Hero from './Hero.svelte';

export const registeredRenderer: RegisteredRenderer = {
	id: 'hero',
	label: 'Hero',
	shape: 'single',
	fields: [
		{ key: 'eyebrow', label: 'Eyebrow', type: 'text' },
		{ key: 'headline', label: 'Headline', type: 'text' },
		{ key: 'sub', label: 'Sub-headline', type: 'text' }
	],
	component: Hero
};

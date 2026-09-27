/**
 * Skeleton Presenter
 *
 * Configuration for Skeleton component in /components browser.
 */

import type { IPresenter } from '../../presenter/IPresenter';
import Skeleton from './Skeleton.svelte';

const presenter: IPresenter = {
	name: 'Skeleton',
	slug: 'skeleton',
	category: 'utilities',
	presenterType: 'component',
	component: Skeleton,

	description: 'Placeholder loading state component for SSR-first pattern. Shows animated placeholders while content loads.',

	variants: [
		{
			name: 'Rectangle (Default)',
			props: { variant: 'rectangle', width: '200px', height: '40px' }
		},
		{
			name: 'Circle',
			props: { variant: 'circle', width: '48px' }
		},
		{
			name: 'Text (3 lines)',
			props: { variant: 'text', lines: 3, width: '300px' }
		},
		{
			name: 'List (3 items)',
			props: { variant: 'list', count: 3 }
		},
		{
			name: 'Card',
			props: { variant: 'card' }
		},
		{
			name: 'Wave Animation',
			props: { variant: 'rectangle', width: '200px', height: '40px', animation: 'wave' }
		},
		{
			name: 'No Animation',
			props: { variant: 'rectangle', width: '200px', height: '40px', animation: 'none' }
		}
	],

	presets: [
		{
			name: 'Avatar',
			props: { variant: 'circle', width: '40px' },
			code: '<Skeleton variant="circle" width="40px" />'
		},
		{
			name: 'Button',
			props: { variant: 'rectangle', width: '100px', height: '36px', rounded: '6px' },
			code: '<Skeleton width="100px" height="36px" rounded="6px" />'
		},
		{
			name: 'Paragraph',
			props: { variant: 'text', lines: 4 },
			code: '<Skeleton variant="text" lines={4} />'
		},
		{
			name: 'Message List',
			props: { variant: 'list', count: 5 },
			code: '<Skeleton variant="list" count={5} />'
		}
	],

	props: [
		{
			name: 'variant',
			type: 'select',
			options: ['rectangle', 'circle', 'text', 'list', 'card'],
			default: 'rectangle',
			description: 'Predefined skeleton layout'
		},
		{
			name: 'width',
			type: 'string',
			default: '100%',
			description: 'Width (CSS value)'
		},
		{
			name: 'height',
			type: 'string',
			default: '20px',
			description: 'Height (CSS value)'
		},
		{
			name: 'animation',
			type: 'select',
			options: ['pulse', 'wave', 'none'],
			default: 'pulse',
			description: 'Animation style'
		},
		{
			name: 'rounded',
			type: 'string',
			default: '4px',
			description: 'Border radius'
		},
		{
			name: 'lines',
			type: 'number',
			default: 3,
			min: 1,
			max: 10,
			description: 'Number of lines (text variant)'
		},
		{
			name: 'count',
			type: 'number',
			default: 3,
			min: 1,
			max: 10,
			description: 'Number of items (list variant)'
		}
	],

	preview: {
		width: 300,
		height: 150,
		background: 'dark'
	},

	tags: ['loading', 'placeholder', 'skeleton', 'ssr', 'utility']
};

export default presenter;

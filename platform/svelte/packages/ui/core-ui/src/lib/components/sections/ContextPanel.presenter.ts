/**
 * ContextPanel Presenter
 *
 * Configuration for ContextPanel component in /components browser.
 * Provides variants for tree, list, and grid views in editor context.
 */

import type { IPresenter } from '../../presenter/IPresenter';
import ContextPanel from './ContextPanel.svelte';

const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'ContextPanel',
	slug: 'context-panel',
	category: 'sections',
	presenterType: 'component',
	component: ContextPanel,

	// Documentation
	description:
		'Editor context panel showing trees, lists, or grids based on nav rail selection. Supports search, resize, and collapse.',

	// Variants - different states/configurations
	variants: [
		{
			name: 'Component Tree',
			props: {
				activeSection: 'components',
				sections: [
					{
						id: 'components',
						type: 'tree',
						header: 'Components',
						searchable: true,
						actions: [
							{ id: 'add', icon: 'plus', tooltip: 'Add component' },
							{ id: 'refresh', icon: 'refresh-cw', tooltip: 'Refresh' }
						],
						items: [
							{
								id: 'primitives',
								label: 'Primitives',
								icon: 'box',
								expanded: true,
								children: [
									{ id: 'button', label: 'Button', icon: 'square' },
									{ id: 'input', label: 'Input', icon: 'text-cursor-input' },
									{ id: 'badge', label: 'Badge', icon: 'tag' }
								]
							},
							{
								id: 'blocks',
								label: 'Blocks',
								icon: 'layout-grid',
								expanded: false,
								children: [
									{ id: 'nav-item', label: 'NavItem', icon: 'navigation' },
									{ id: 'dock', label: 'Dock', icon: 'dock' }
								]
							},
							{
								id: 'sections',
								label: 'Sections',
								icon: 'layout',
								expanded: false,
								children: [
									{ id: 'navbar', label: 'NavBar', icon: 'menu' },
									{ id: 'sidebar', label: 'Sidebar', icon: 'panel-left' }
								]
							}
						]
					}
				]
			}
		},
		{
			name: 'Layer Tree',
			props: {
				activeSection: 'layers',
				sections: [
					{
						id: 'layers',
						type: 'tree',
						header: 'Layers',
						searchable: false,
						actions: [{ id: 'add', icon: 'plus', tooltip: 'Add layer' }],
						items: [
							{
								id: 'frame-1',
								label: 'Frame 1',
								icon: 'square',
								expanded: true,
								children: [
									{
										id: 'header',
										label: 'Header',
										icon: 'rectangle-horizontal',
										children: [
											{ id: 'logo', label: 'Logo', icon: 'image' },
											{ id: 'nav', label: 'Navigation', icon: 'menu' }
										]
									},
									{
										id: 'content',
										label: 'Content',
										icon: 'rectangle-vertical',
										expanded: true,
										children: [
											{ id: 'title', label: 'Title', icon: 'type' },
											{ id: 'button-1', label: 'Button', icon: 'square', selected: true }
										]
									}
								]
							}
						]
					}
				]
			}
		},
		{
			name: 'Asset Grid',
			props: {
				activeSection: 'assets',
				sections: [
					{
						id: 'assets',
						type: 'grid',
						header: 'Assets',
						searchable: true,
						actions: [{ id: 'upload', icon: 'upload', tooltip: 'Upload asset' }],
						items: [
							{ id: 'logo-svg', label: 'logo.svg', icon: 'file-image' },
							{ id: 'hero-png', label: 'hero.png', icon: 'image' },
							{ id: 'icon-set', label: 'icons/', icon: 'folder' },
							{ id: 'bg-pattern', label: 'pattern.svg', icon: 'file-image' },
							{ id: 'avatar', label: 'avatar.jpg', icon: 'user' },
							{ id: 'fonts', label: 'fonts/', icon: 'folder' }
						]
					}
				]
			}
		},
		{
			name: 'Page List',
			props: {
				activeSection: 'pages',
				sections: [
					{
						id: 'pages',
						type: 'list',
						header: 'Pages',
						searchable: true,
						actions: [{ id: 'add', icon: 'plus', tooltip: 'Add page' }],
						items: [
							{ id: 'home', label: 'Home', icon: 'home', selected: true },
							{ id: 'about', label: 'About', icon: 'file-text' },
							{ id: 'contact', label: 'Contact', icon: 'mail' },
							{ id: 'blog', label: 'Blog', icon: 'newspaper', badge: 12 },
							{ id: 'settings', label: 'Settings', icon: 'settings', disabled: true }
						]
					}
				]
			}
		},
		{
			name: 'Collapsed',
			props: {
				activeSection: 'components',
				collapsed: true,
				sections: [
					{
						id: 'components',
						type: 'tree',
						header: 'Components',
						items: []
					}
				]
			}
		}
	],

	// Presets - saved prop combinations
	presets: [
		{
			name: 'Interface Builder Components',
			props: {
				activeSection: 'components',
				width: 240,
				collapsible: true,
				resizable: true,
				sections: [
					{
						id: 'components',
						type: 'tree',
						header: 'Components',
						searchable: true,
						actions: [
							{ id: 'add', icon: 'plus', tooltip: 'Add' },
							{ id: 'filter', icon: 'filter', tooltip: 'Filter' }
						],
						items: []
					},
					{
						id: 'layers',
						type: 'tree',
						header: 'Layers',
						items: []
					},
					{
						id: 'assets',
						type: 'grid',
						header: 'Assets',
						searchable: true,
						items: []
					}
				]
			}
		}
	],

	// Props definition for real-time editing
	props: [
		{
			name: 'activeSection',
			type: 'string',
			default: '',
			description: 'Currently active section ID'
		},
		{
			name: 'width',
			type: 'number',
			default: 240,
			description: 'Panel width in pixels'
		},
		{
			name: 'collapsible',
			type: 'boolean',
			default: true,
			description: 'Allow panel to collapse'
		},
		{
			name: 'resizable',
			type: 'boolean',
			default: true,
			description: 'Allow panel to be resized'
		},
		{
			name: 'collapsed',
			type: 'boolean',
			default: false,
			description: 'Current collapsed state'
		}
	]
};

export default presenter;

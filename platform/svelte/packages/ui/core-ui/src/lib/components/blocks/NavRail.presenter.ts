/**
 * NavRail Presenter
 *
 * Configuration for NavRail component in /components browser.
 * Provides variants and presets for Interface Builder navigation.
 */

import type { IPresenter } from '../../presenter/IPresenter';
import NavRail from './NavRail.svelte';

const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'NavRail',
	slug: 'nav-rail',
	category: 'blocks',
	presenterType: 'component',
	component: NavRail,

	// Documentation
	description:
		'48px vertical icon-only navigation rail for editor layouts. Material Design 3 nav rail pattern.',

	// Variants - different states/configurations
	variants: [
		{
			name: 'Default',
			props: {
				items: [
					{ id: 'components', icon: 'layers', tooltip: 'Components' },
					{ id: 'assets', icon: 'image', tooltip: 'Assets' },
					{ id: 'pages', icon: 'file-text', tooltip: 'Pages' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'components'
			}
		},
		{
			name: 'With Badges',
			props: {
				items: [
					{ id: 'inbox', icon: 'inbox', tooltip: 'Inbox', badge: 5 },
					{ id: 'notifications', icon: 'bell', tooltip: 'Notifications', badge: 12 },
					{ id: 'messages', icon: 'message-circle', tooltip: 'Messages', badge: 3 },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'inbox'
			}
		},
		{
			name: 'With Dividers',
			props: {
				items: [
					{ id: 'home', icon: 'home', tooltip: 'Home' },
					{ id: 'search', icon: 'search', tooltip: 'Search', divider: true },
					{ id: 'components', icon: 'layers', tooltip: 'Components' },
					{ id: 'assets', icon: 'image', tooltip: 'Assets' },
					{ id: 'pages', icon: 'file-text', tooltip: 'Pages', divider: true },
					{ id: 'help', icon: 'help-circle', tooltip: 'Help', position: 'bottom' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'components'
			}
		},
		{
			name: 'Editor Mode',
			props: {
				items: [
					{ id: 'components', icon: 'component', tooltip: 'Components' },
					{ id: 'layers', icon: 'layers', tooltip: 'Layers' },
					{ id: 'assets', icon: 'folder', tooltip: 'Assets' },
					{ id: 'code', icon: 'code', tooltip: 'Code', divider: true },
					{ id: 'plugins', icon: 'puzzle', tooltip: 'Plugins', position: 'bottom' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'components',
				width: 48
			}
		},
		{
			name: 'With Disabled',
			props: {
				items: [
					{ id: 'dashboard', icon: 'layout-dashboard', tooltip: 'Dashboard' },
					{ id: 'analytics', icon: 'bar-chart-2', tooltip: 'Analytics', disabled: true },
					{ id: 'reports', icon: 'file-bar-chart', tooltip: 'Reports', disabled: true },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'dashboard'
			}
		},
		{
			name: 'Wider Rail',
			props: {
				items: [
					{ id: 'home', icon: 'home', tooltip: 'Home' },
					{ id: 'explore', icon: 'compass', tooltip: 'Explore' },
					{ id: 'library', icon: 'library', tooltip: 'Library' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'home',
				width: 56
			}
		}
	],

	// Presets - saved prop combinations
	presets: [
		{
			name: 'Interface Builder',
			props: {
				items: [
					{ id: 'components', icon: 'layers', tooltip: 'Components' },
					{ id: 'tree', icon: 'git-branch', tooltip: 'Layer Tree' },
					{ id: 'assets', icon: 'image', tooltip: 'Assets' },
					{ id: 'code', icon: 'code', tooltip: 'Code' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'components',
				width: 48
			}
		},
		{
			name: 'Dashboard App',
			props: {
				items: [
					{ id: 'dashboard', icon: 'layout-dashboard', tooltip: 'Dashboard' },
					{ id: 'analytics', icon: 'trending-up', tooltip: 'Analytics' },
					{ id: 'users', icon: 'users', tooltip: 'Users' },
					{ id: 'reports', icon: 'file-text', tooltip: 'Reports', divider: true },
					{ id: 'help', icon: 'help-circle', tooltip: 'Help', position: 'bottom' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'dashboard'
			}
		},
		{
			name: 'Minimal',
			props: {
				items: [
					{ id: 'home', icon: 'home', tooltip: 'Home' },
					{ id: 'search', icon: 'search', tooltip: 'Search' },
					{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
				],
				activeId: 'home'
			}
		},
		{
			name: 'Creative Tools',
			props: {
				items: [
					{ id: 'select', icon: 'mouse-pointer-2', tooltip: 'Select' },
					{ id: 'frame', icon: 'square', tooltip: 'Frame' },
					{ id: 'text', icon: 'type', tooltip: 'Text' },
					{ id: 'shapes', icon: 'shapes', tooltip: 'Shapes' },
					{ id: 'pen', icon: 'pen-tool', tooltip: 'Pen', divider: true },
					{ id: 'hand', icon: 'hand', tooltip: 'Hand', position: 'bottom' },
					{ id: 'zoom', icon: 'zoom-in', tooltip: 'Zoom', position: 'bottom' }
				],
				activeId: 'select'
			}
		}
	],

	// Props definition for real-time editing
	props: [
		{
			name: 'items',
			type: 'array',
			default: [],
			description: 'Navigation items with id, icon, tooltip'
		},
		{
			name: 'activeId',
			type: 'string',
			default: '',
			description: 'Currently active item ID'
		},
		{
			name: 'width',
			type: 'number',
			default: 48,
			description: 'Rail width in pixels (48-56 recommended)'
		}
	]
};

export default presenter;

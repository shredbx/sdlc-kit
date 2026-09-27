/**
 * NavBar Presenter
 *
 * Defines presentation metadata for NavBar component.
 * Enables auto-registration in /components page with
 * variants, presets, and live editing support.
 *
 * @see navigation.yml schema for data structure
 * @see IPresenter interface for conformance
 */

import NavBar from './NavBar.svelte';
import type { IPresenter, IVariant, IPreset, IPropDef } from '../../presenter/IPresenter';

// Sample navigation items for previews
const sampleItems = [
	{ id: 'home', label: 'Home', href: '/', icon: '🏠' },
	{ id: 'work', label: 'Work', href: '/work', icon: '💼' },
	{ id: 'about', label: 'About', href: '/about', icon: '👤' },
	{ id: 'contact', label: 'Contact', href: '/contact', icon: '📧' }
];

const sampleItemsWithBadges = [
	{ id: 'dashboard', label: 'Dashboard', href: '/dashboard', icon: '📊' },
	{
		id: 'notifications',
		label: 'Notifications',
		href: '/notifications',
		icon: '🔔',
		badge: { count: 5, variant: 'error' }
	},
	{
		id: 'messages',
		label: 'Messages',
		href: '/messages',
		icon: '💬',
		badge: { count: 12, variant: 'primary' }
	},
	{ id: 'settings', label: 'Settings', href: '/settings', icon: '⚙️' }
];

const sampleItemsWithDisabled = [
	{ id: 'home', label: 'Home', href: '/' },
	{ id: 'lab', label: 'Lab', href: '/lab', disabled: true },
	{ id: 'projects', label: 'Projects', href: '/projects', disabled: true },
	{ id: 'me', label: 'Me', href: '/me' },
	{ id: 'contacts', label: 'Contacts', href: '/contacts', disabled: true }
];

// Variants showcase different configurations
const variants: IVariant[] = [
	{
		name: 'Floating (Default)',
		description: 'Glassmorphism floating nav, centered at top',
		props: {
			items: sampleItems,
			variant: 'floating',
			position: 'top-center',
			logo: 'shredbx'
		}
	},
	{
		name: 'Floating with Badges',
		description: 'Navigation items with notification badges',
		props: {
			items: sampleItemsWithBadges,
			variant: 'floating',
			logo: 'Hub'
		}
	},
	{
		name: 'Floating with Disabled',
		description: 'Mix of active and coming-soon items',
		props: {
			items: sampleItemsWithDisabled,
			variant: 'floating',
			logo: 'shredbx'
		}
	},
	{
		name: 'Fixed Header',
		description: 'Full-width fixed header bar',
		props: {
			items: sampleItems,
			variant: 'fixed',
			logo: 'App'
		}
	},
	{
		name: 'Vertical Sidebar',
		description: 'Vertical layout for sidebar use',
		props: {
			items: sampleItemsWithBadges,
			variant: 'inline',
			style: 'vertical'
		}
	},
	{
		name: 'Top Left',
		description: 'Floating nav positioned top-left',
		props: {
			items: sampleItems.slice(0, 3),
			variant: 'floating',
			position: 'top-left',
			logo: 'Menu'
		}
	},
	{
		name: 'Bottom Center',
		description: 'Dock-style bottom navigation',
		props: {
			items: sampleItems,
			variant: 'floating',
			position: 'bottom-center'
		}
	}
];

// Presets are saved prop combinations for quick reuse
const presets: IPreset[] = [
	{
		name: 'finndollimore.com Style',
		description: 'Purple accent floating nav with glassmorphism',
		props: {
			items: sampleItemsWithDisabled,
			variant: 'floating',
			position: 'top-center',
			logo: 'shredbx'
		}
	},
	{
		name: 'Dashboard Nav',
		description: 'App navigation with badges',
		props: {
			items: sampleItemsWithBadges,
			variant: 'floating',
			logo: 'Hub'
		}
	},
	{
		name: 'Mobile Bottom Nav',
		description: 'Bottom dock for mobile apps',
		props: {
			items: sampleItems.map((i) => ({ ...i, label: '' })),
			variant: 'floating',
			position: 'bottom-center'
		}
	}
];

// Props definitions for real-time editing
const props: IPropDef[] = [
	{
		name: 'items',
		type: 'array',
		description: 'Navigation items array',
		default: []
	},
	{
		name: 'variant',
		type: 'select',
		description: 'Visual variant',
		options: ['floating', 'fixed', 'inline'],
		default: 'floating'
	},
	{
		name: 'position',
		type: 'select',
		description: 'Screen position',
		options: ['top-center', 'top-left', 'top-right', 'bottom-center'],
		default: 'top-center'
	},
	{
		name: 'style',
		type: 'select',
		description: 'Layout direction',
		options: ['horizontal', 'vertical'],
		default: 'horizontal'
	},
	{
		name: 'logo',
		type: 'string',
		description: 'Logo text or emoji',
		default: ''
	},
	{
		name: 'logoHref',
		type: 'string',
		description: 'Logo link destination',
		default: '/'
	},
	{
		name: 'visible',
		type: 'boolean',
		description: 'Show/hide with animation',
		default: true
	}
];

// Fixture references for build-once-reuse pattern
const fixtures = [
	{
		entity: 'navigation',
		fixture: 'hub-main-nav',
		use: 'preview' as const
	}
];

const presenter: IPresenter & Record<string, unknown> = {
	// Required - minimal conformance
	name: 'NavBar',
	slug: 'navbar',
	category: 'sections',
	presenterType: 'component',
	component: NavBar,

	// Optional - enrichment
	description:
		'Floating glassmorphism navigation bar with support for badges, disabled states, and page transitions.',
	variants,
	presets,
	props,
	fixture: {
		default: 'navigation:hub-main-nav'
	},

	// Preview config - use inline variant to prevent fixed positioning in card
	preview: {
		defaultProps: {
			items: [{ id: 'nav', label: 'NavBar', href: '#' }],
			variant: 'inline',
			style: 'horizontal'
		}
	},

	// Source reference
	source: {
		path: 'components/navigation/NavBar.svelte'
	},

	// IView conformance
	viewConfig: {
		decorators: ['dev-mode', 'branding'],
		devMode: {
			selectable: true,
			inspectable: true
		}
	}
};

export default presenter;

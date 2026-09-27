/**
 * NavItem Presenter - Auto-registration for component library
 *
 * Defines how NavItem is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import NavItem from './NavItem.svelte';

/**
 * NavItem property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'href',
		type: 'string',
		description: 'Navigation URL',
		default: '#'
	},
	{
		name: 'label',
		type: 'string',
		description: 'Display label',
		default: 'Navigation'
	},
	{
		name: 'icon',
		type: 'string',
		description: 'Icon name (from Icon component)',
		default: undefined
	},
	{
		name: 'active',
		type: 'boolean',
		description: 'Active/selected state',
		default: false
	},
	{
		name: 'disabled',
		type: 'boolean',
		description: 'Disabled state',
		default: false
	},
	{
		name: 'badge',
		type: 'number',
		description: 'Badge count',
		default: undefined
	},
	{
		name: 'variant',
		type: 'select',
		options: ['default', 'floating', 'sidebar', 'pill'],
		description: 'Visual variant',
		default: 'default'
	},
	{
		name: 'external',
		type: 'boolean',
		description: 'External link (opens in new tab)',
		default: false
	},
	{
		name: 'collapsed',
		type: 'boolean',
		description: 'Collapsed mode (icon only)',
		default: false
	}
];

/**
 * NavItem variants for different states/configurations.
 */
const variants: IVariant[] = [
	// Basic states
	{
		name: 'Default',
		description: 'Standard navigation item',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home'
		}
	},
	{
		name: 'Active',
		description: 'Currently selected item',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			active: true
		}
	},
	{
		name: 'Disabled',
		description: 'Unavailable navigation',
		props: {
			href: '/pro',
			label: 'Pro Features',
			icon: 'star',
			disabled: true
		}
	},
	{
		name: 'With Badge',
		description: 'Item with count badge',
		props: {
			href: '/notifications',
			label: 'Notifications',
			icon: 'bell',
			badge: 5
		}
	},
	{
		name: 'External Link',
		description: 'Opens in new tab',
		props: {
			href: 'https://docs.example.com',
			label: 'Documentation',
			icon: 'book',
			external: true
		}
	},

	// Variant styles
	{
		name: 'Floating Style',
		description: 'Glassmorphism nav bar style (finndollimore.com)',
		props: {
			href: '/work',
			label: 'Work',
			variant: 'floating'
		}
	},
	{
		name: 'Floating Active',
		description: 'Active state in floating nav',
		props: {
			href: '/work',
			label: 'Work',
			variant: 'floating',
			active: true
		}
	},
	{
		name: 'Sidebar Style',
		description: 'Left/right sidebar navigation',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			variant: 'sidebar'
		}
	},
	{
		name: 'Sidebar Active',
		description: 'Active sidebar item',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			variant: 'sidebar',
			active: true
		}
	},
	{
		name: 'Pill Style',
		description: 'Tab-style navigation',
		props: {
			href: '/overview',
			label: 'Overview',
			variant: 'pill'
		}
	},
	{
		name: 'Pill Active',
		description: 'Active pill tab',
		props: {
			href: '/overview',
			label: 'Overview',
			variant: 'pill',
			active: true
		}
	},

	// Collapsed
	{
		name: 'Collapsed',
		description: 'Icon only mode',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			collapsed: true
		}
	},
	{
		name: 'Collapsed Active',
		description: 'Active collapsed item',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			collapsed: true,
			active: true
		}
	}
];

/**
 * NavItem presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'Dashboard Link',
		description: 'Main dashboard navigation',
		props: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			variant: 'sidebar'
		}
	},
	{
		name: 'Settings Link',
		description: 'Settings page navigation',
		props: {
			href: '/settings',
			label: 'Settings',
			icon: 'settings',
			variant: 'sidebar'
		}
	},
	{
		name: 'Notifications with Badge',
		description: 'Notification center with count',
		props: {
			href: '/notifications',
			label: 'Notifications',
			icon: 'bell',
			badge: 12,
			variant: 'sidebar'
		}
	},
	{
		name: 'Top Nav Link',
		description: 'Floating top navigation item',
		props: {
			href: '/work',
			label: 'Work',
			variant: 'floating'
		}
	},
	{
		name: 'Docs External',
		description: 'External documentation link',
		props: {
			href: 'https://docs.example.com',
			label: 'Docs',
			icon: 'book-open',
			external: true,
			variant: 'sidebar'
		}
	}
];

/**
 * NavItem Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'NavItem',
	slug: 'nav-item',
	category: 'blocks',
	presenterType: 'component',

	// Component reference
	component: NavItem,

	// Optional - enrichment
	description:
		'Navigation link with icon, label, badge, and active state. Supports multiple visual variants including floating (finndollimore.com style), sidebar, and pill styles.',

	tags: ['navigation', 'link', 'menu', 'sidebar', 'navbar'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/blocks/NavItem.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 200,
		height: 50,
		background: 'dark',
		defaultProps: {
			href: '/dashboard',
			label: 'Dashboard',
			icon: 'home',
			variant: 'sidebar'
		}
	},

	documentation: `
## NavItem Component

Navigation link with icon, label, and active state indicators.

### Usage

\`\`\`svelte
<script>
  import NavItem from './NavItem.svelte';
</script>

<NavItem href="/dashboard" label="Dashboard" icon="home" />
<NavItem href="/settings" label="Settings" icon="settings" active />
\`\`\`

### Variants

- \`default\` - Basic link with underline animation
- \`floating\` - Glassmorphism style (finndollimore.com inspired)
- \`sidebar\` - Left/right sidebar with background highlight
- \`pill\` - Tab-style with pill shape

### With Badge

\`\`\`svelte
<NavItem
  href="/notifications"
  label="Notifications"
  icon="bell"
  badge={5}
/>
\`\`\`

### Collapsed Mode

For sidebar collapse, use \`collapsed\` prop to show icon only.
`
};

export default presenter;

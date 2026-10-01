/**
 * Sidebar Presenter
 *
 * Defines presentation metadata for Sidebar component.
 * Enables preview in /components page with navigation, sections, and custom content modes.
 *
 * Key Features:
 * - Fixture-driven: Items can come from navigation fixtures
 * - Extensible: Custom content via snippet for filetree, etc.
 * - Position variants: left/right sidebar
 * - Collapsible with configurable widths
 *
 * @see navigation.yml schema for item definitions
 * @see IPresenter interface for conformance
 */

import Sidebar from './Sidebar.svelte';
import type { IPresenter, IVariant, IPreset, IPropDef } from '../../presenter/IPresenter';

// Sample navigation items for preview
const sampleNavItems = [
	{ id: 'dashboard', label: 'Dashboard', icon: '📊', href: '/' },
	{ id: 'workspace', label: 'Workspace', icon: '📁', href: '/workspace' },
	{ id: 'components', label: 'Components', icon: '🧩', href: '/components' },
	{ id: 'packages', label: 'Packages', icon: '📦', href: '/packages' }
];

const sampleWithBadges = [
	{ id: 'dashboard', label: 'Dashboard', icon: '📊', href: '/', badge: { count: 3, variant: 'primary' as const } },
	{ id: 'workspace', label: 'Workspace', icon: '📁', href: '/workspace' },
	{ id: 'components', label: 'Components', icon: '🧩', href: '/components', badge: { label: 'New', variant: 'success' as const } },
	{ id: 'packages', label: 'Packages', icon: '📦', href: '/packages', badge: { count: 12, variant: 'error' as const } }
];

const sampleWithDisabled = [
	{ id: 'dashboard', label: 'Dashboard', icon: '📊', href: '/' },
	{ id: 'workspace', label: 'Workspace', icon: '📁', href: '/workspace' },
	{ id: 'ai', label: 'AI Assistant', icon: '🤖', href: '/ai', disabled: true },
	{ id: 'analytics', label: 'Analytics', icon: '📈', href: '/analytics', disabled: true }
];

const sampleSections = [
	{
		id: 'main',
		title: 'Navigation',
		items: [
			{ id: 'dashboard', label: 'Dashboard', icon: '📊', href: '/' },
			{ id: 'workspace', label: 'Workspace', icon: '📁', href: '/workspace' }
		]
	},
	{
		id: 'tools',
		title: 'Tools',
		items: [
			{ id: 'components', label: 'Components', icon: '🧩', href: '/components' },
			{ id: 'packages', label: 'Packages', icon: '📦', href: '/packages' }
		]
	},
	{
		id: 'admin',
		title: 'Admin',
		items: [
			{ id: 'settings', label: 'Settings', icon: '⚙️', href: '/settings' },
			{ id: 'users', label: 'Users', icon: '👥', href: '/users' }
		]
	}
];

// =============================================================================
// VARIANTS
// =============================================================================

const variants: IVariant[] = [
	{
		name: 'Left Sidebar (Default)',
		description: 'Standard left-positioned navigation sidebar',
		props: {
			position: 'left',
			items: sampleNavItems,
			header: { title: 'Hub', href: '/' }
		}
	},
	{
		name: 'Right Sidebar',
		description: 'Right-positioned sidebar for secondary content',
		props: {
			position: 'right',
			items: sampleNavItems,
			header: { title: 'Tools', href: '/' }
		}
	},
	{
		name: 'With Badges',
		description: 'Navigation items with notification badges',
		props: {
			position: 'left',
			items: sampleWithBadges,
			header: { title: 'Hub', href: '/' }
		}
	},
	{
		name: 'With Disabled Items',
		description: 'Shows coming soon disabled items',
		props: {
			position: 'left',
			items: sampleWithDisabled,
			header: { title: 'Hub', href: '/' }
		}
	},
	{
		name: 'Grouped Sections',
		description: 'Items organized in titled sections',
		props: {
			position: 'left',
			sections: sampleSections,
			header: { title: 'Hub', href: '/' }
		}
	},
	{
		name: 'Collapsed State',
		description: 'Sidebar in collapsed icon-only mode',
		props: {
			position: 'left',
			items: sampleNavItems,
			header: { title: 'Hub', href: '/' },
			collapsed: true
		}
	},
	{
		name: 'No Collapse Button',
		description: 'Sidebar without collapse toggle',
		props: {
			position: 'left',
			items: sampleNavItems,
			header: { title: 'Hub', href: '/' },
			showCollapseButton: false
		}
	},
	{
		name: 'Custom Width',
		description: 'Wider sidebar for more content',
		props: {
			position: 'left',
			items: sampleNavItems,
			header: { title: 'Hub', href: '/' },
			width: 300,
			collapsedWidth: 80
		}
	}
];

// =============================================================================
// PRESETS
// =============================================================================

const presets: IPreset[] = [
	{
		name: 'Hub Navigation',
		description: 'Standard Hub application sidebar with logo and nav items',
		props: {
			position: 'left',
			items: sampleNavItems,
			header: { title: 'Hub', logo: '/hub-logo.svg', href: '/' },
			width: 240
		}
	},
	{
		name: 'Dashboard Sidebar',
		description: 'Admin dashboard with grouped sections',
		props: {
			position: 'left',
			sections: sampleSections,
			header: { title: 'Dashboard', href: '/' },
			width: 260
		}
	},
	{
		name: 'Minimal Sidebar',
		description: 'Compact sidebar without header or footer',
		props: {
			position: 'left',
			items: sampleNavItems,
			showCollapseButton: false,
			width: 200
		}
	},
	{
		name: 'Wide Sidebar (Filetree Ready)',
		description: 'Wider sidebar suitable for file tree content',
		props: {
			position: 'left',
			header: { title: 'Explorer', href: '/' },
			width: 320,
			collapsedWidth: 48
		}
	}
];

// =============================================================================
// PROPS DEFINITIONS
// =============================================================================

const props: IPropDef[] = [
	{
		name: 'position',
		type: 'select',
		description: 'Sidebar position relative to content',
		options: ['left', 'right'],
		default: 'left'
	},
	{
		name: 'collapsed',
		type: 'boolean',
		description: 'Whether sidebar is in collapsed state',
		default: false
	},
	{
		name: 'width',
		type: 'number',
		description: 'Expanded width in pixels',
		default: 240,
		min: 160,
		max: 400
	},
	{
		name: 'collapsedWidth',
		type: 'number',
		description: 'Collapsed width in pixels',
		default: 60,
		min: 40,
		max: 100
	},
	{
		name: 'showCollapseButton',
		type: 'boolean',
		description: 'Show the collapse/expand toggle button',
		default: true
	},
	{
		name: 'items',
		type: 'json',
		description: 'Navigation items array (simple mode)',
		default: []
	},
	{
		name: 'sections',
		type: 'json',
		description: 'Grouped sections array (advanced mode)',
		default: []
	},
	{
		name: 'header',
		type: 'json',
		description: 'Header configuration { title, logo, href }',
		default: {}
	}
];

// =============================================================================
// PRESENTER DEFINITION
// =============================================================================

const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'Sidebar',
	slug: 'sidebar',
	category: 'sections',
	presenterType: 'component',
	component: Sidebar,

	// Optional - enrichment
	description:
		'Generic sidebar component for navigation, sections, or custom content. Supports fixture-driven items, badges, disabled states, and collapsible mode.',
	variants,
	presets,
	props,

	// Source reference
	source: {
		path: 'components/sidebar/Sidebar.svelte'
	},

	// Searchable tags
	tags: ['navigation', 'sidebar', 'menu', 'layout', 'nav', 'drawer'],

	// Documentation
	documentation: `
## Basic Usage

\`\`\`svelte
<script>
  import Sidebar from './Sidebar.svelte';

  const items = [
    { id: 'home', label: 'Home', icon: '🏠', href: '/' },
    { id: 'about', label: 'About', icon: 'ℹ️', href: '/about' }
  ];
</script>

<Sidebar
  {items}
  header={{ title: 'My App', href: '/' }}
/>
\`\`\`

## With Fixture Data

\`\`\`svelte
<script>
  import Sidebar from './Sidebar.svelte';
  import hubNav from '$fixtures/navigation/hub-left-sidebar.yml';

  // Transform fixture data to items
  const items = hubNav.data.groups.value.flatMap(g => g.items);
</script>

<Sidebar {items} header={{ title: 'Hub' }} />
\`\`\`

## Grouped Sections

\`\`\`svelte
<Sidebar
  sections={[
    { id: 'main', title: 'Navigation', items: [...] },
    { id: 'tools', title: 'Tools', items: [...] }
  ]}
/>
\`\`\`

## Custom Content (Filetree)

\`\`\`svelte
<Sidebar position="left" header={{ title: 'Explorer' }}>
  {#snippet content()}
    <FileTree files={files} />
  {/snippet}
</Sidebar>
\`\`\`

## With Badges

\`\`\`svelte
<Sidebar items={[
  { id: 'inbox', label: 'Inbox', icon: '📥', href: '/inbox', badge: { count: 5, variant: 'error' } },
  { id: 'new', label: 'New Feature', icon: '✨', href: '/new', badge: { label: 'New', variant: 'success' } }
]} />
\`\`\`

## Controlled Collapse

\`\`\`svelte
<script>
  let collapsed = $state(false);
</script>

<Sidebar
  {items}
  bind:collapsed
  onCollapseChange={(c) => console.log('Collapsed:', c)}
/>
\`\`\`
`,

	// Status
	status: 'stable',
	version: '1.0.0'
};

export default presenter;

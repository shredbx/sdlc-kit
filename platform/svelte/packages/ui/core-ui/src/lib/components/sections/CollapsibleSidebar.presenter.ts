/**
 * CollapsibleSidebar Presenter
 *
 * Configuration for CollapsibleSidebar component in /components browser.
 * Demonstrates Canva/Figma-style dual-mode sidebar with NavRail (48px) + ContextPanel (240px).
 *
 * Key Features:
 * - NavRail (48px): Icon-only navigation, always visible
 * - ContextPanel (240px): Expandable content area
 * - States: collapsed (rail only), expanded (rail + panel), pinned (stays expanded)
 * - 200ms ease-out transition animation
 * - Expand on hover with configurable delay
 * - Keyboard shortcuts: Cmd+B toggle, Cmd+Shift+P pin
 *
 * @see IPresenter interface for conformance
 */

import type { IPresenter, IVariant, IPreset, IPropDef } from '../../presenter/IPresenter';
import CollapsibleSidebar from './CollapsibleSidebar.svelte';

// =============================================================================
// VARIANTS
// =============================================================================

const variants: IVariant[] = [
	{
		name: 'Expanded (Default)',
		description: 'Standard expanded state showing rail + panel',
		props: {
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			railItems: [
				{ id: 'home', icon: 'home', tooltip: 'Home', shortcut: 'Cmd+1' },
				{ id: 'components', icon: 'component', tooltip: 'Components', shortcut: 'Cmd+2' },
				{ id: 'brand', icon: 'palette', tooltip: 'Brand', shortcut: 'Cmd+3' },
				{ id: 'layers', icon: 'layers', tooltip: 'Layers', shortcut: 'Cmd+4' },
				{ id: 'divider1', icon: '', tooltip: '', divider: true },
				{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
			],
			activeRailItem: 'components',
			contextSections: [
				{
					id: 'components',
					type: 'tree',
					header: 'Components',
					searchable: true,
					items: [
						{
							id: 'primitives',
							label: 'Primitives',
							icon: 'box',
							expanded: true,
							children: [
								{ id: 'button', label: 'Button', icon: 'square' },
								{ id: 'badge', label: 'Badge', icon: 'tag' },
								{ id: 'input', label: 'Input', icon: 'text-cursor-input' }
							]
						},
						{
							id: 'blocks',
							label: 'Blocks',
							icon: 'grid-2x2',
							children: [
								{ id: 'nav-item', label: 'NavItem', icon: 'navigation' },
								{ id: 'dock', label: 'Dock', icon: 'dock' }
							]
						}
					]
				}
			]
		}
	},
	{
		name: 'Collapsed (Rail Only)',
		description: 'Only the 48px rail is visible',
		props: {
			expanded: false,
			railWidth: 48,
			railItems: [
				{ id: 'home', icon: 'home', tooltip: 'Home' },
				{ id: 'projects', icon: 'folder', tooltip: 'Projects' },
				{ id: 'templates', icon: 'layout-template', tooltip: 'Templates' },
				{ id: 'brand', icon: 'palette', tooltip: 'Brand' },
				{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
			],
			activeRailItem: 'home'
		}
	},
	{
		name: 'Pinned State',
		description: 'Panel stays expanded even on mouse leave',
		props: {
			expansionState: 'pinned',
			railWidth: 48,
			contextWidth: 240,
			allowPin: true,
			railItems: [
				{ id: 'files', icon: 'folder', tooltip: 'Files' },
				{ id: 'search', icon: 'search', tooltip: 'Search' },
				{ id: 'git', icon: 'git-branch', tooltip: 'Source Control' }
			],
			activeRailItem: 'files',
			contextSections: [
				{
					id: 'files',
					type: 'tree',
					header: 'Explorer',
					searchable: true,
					items: [
						{ id: 'src', label: 'src', icon: 'folder', expanded: true, children: [
							{ id: 'app', label: 'app.ts', icon: 'file-code' },
							{ id: 'main', label: 'main.ts', icon: 'file-code' }
						]}
					]
				}
			]
		}
	},
	{
		name: 'Expand on Hover',
		description: 'Panel expands when hovering over rail',
		props: {
			expanded: false,
			expandOnHover: true,
			hoverDelay: 200,
			railWidth: 48,
			contextWidth: 240,
			railItems: [
				{ id: 'home', icon: 'home', tooltip: 'Home' },
				{ id: 'docs', icon: 'book-open', tooltip: 'Documentation' },
				{ id: 'api', icon: 'code', tooltip: 'API Reference' }
			],
			activeRailItem: 'home',
			contextSections: [
				{
					id: 'home',
					type: 'list',
					header: 'Quick Start',
					items: [
						{ id: 'getting-started', label: 'Getting Started', icon: 'play' },
						{ id: 'tutorials', label: 'Tutorials', icon: 'graduation-cap' },
						{ id: 'examples', label: 'Examples', icon: 'code-2' }
					]
				}
			]
		}
	},
	{
		name: 'With Header',
		description: 'Includes logo and title in header',
		props: {
			expanded: true,
			header: {
				logo: '/hub-logo.svg',
				title: 'Hub'
			},
			railWidth: 48,
			contextWidth: 240,
			railItems: [
				{ id: 'dashboard', icon: 'layout-dashboard', tooltip: 'Dashboard' },
				{ id: 'workspace', icon: 'briefcase', tooltip: 'Workspace' },
				{ id: 'components', icon: 'component', tooltip: 'Components' }
			],
			activeRailItem: 'dashboard',
			contextSections: [
				{
					id: 'dashboard',
					type: 'list',
					header: 'Dashboard',
					items: [
						{ id: 'overview', label: 'Overview', icon: 'chart-bar' },
						{ id: 'metrics', label: 'Metrics', icon: 'activity' },
						{ id: 'tasks', label: 'Tasks', icon: 'check-square' }
					]
				}
			]
		}
	},
	{
		name: 'Right Position',
		description: 'Sidebar positioned on the right side',
		props: {
			position: 'right',
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			railItems: [
				{ id: 'attributes', icon: 'sliders', tooltip: 'Attributes' },
				{ id: 'styles', icon: 'paintbrush', tooltip: 'Styles' },
				{ id: 'motion', icon: 'zap', tooltip: 'Motion' }
			],
			activeRailItem: 'attributes',
			contextSections: [
				{
					id: 'attributes',
					type: 'list',
					header: 'Attributes',
					items: [
						{ id: 'size', label: 'Size', icon: 'move' },
						{ id: 'color', label: 'Color', icon: 'droplet' },
						{ id: 'spacing', label: 'Spacing', icon: 'space' }
					]
				}
			]
		}
	},
	{
		name: 'With Badges',
		description: 'Rail items with notification badges',
		props: {
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			railItems: [
				{ id: 'inbox', icon: 'inbox', tooltip: 'Inbox', badge: 12 },
				{ id: 'tasks', icon: 'check-circle', tooltip: 'Tasks', badge: 5 },
				{ id: 'messages', icon: 'message-circle', tooltip: 'Messages', badge: 3 },
				{ id: 'alerts', icon: 'alert-triangle', tooltip: 'Alerts', badge: 1 }
			],
			activeRailItem: 'inbox',
			contextSections: [
				{
					id: 'inbox',
					type: 'list',
					header: 'Inbox',
					searchable: true,
					actions: [{ id: 'mark-read', icon: 'check-check', tooltip: 'Mark all read' }],
					items: [
						{ id: 'msg1', label: 'New comment on PR', icon: 'git-pull-request' },
						{ id: 'msg2', label: 'Build completed', icon: 'check', badge: 2 },
						{ id: 'msg3', label: 'Deployment ready', icon: 'rocket' }
					]
				}
			]
		}
	}
];

// =============================================================================
// PRESETS
// =============================================================================

const presets: IPreset[] = [
	{
		name: 'Hub Navigation',
		description: 'Standard Hub application sidebar',
		props: {
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			allowPin: true,
			railItems: [
				{ id: 'dashboard', icon: 'layout-dashboard', tooltip: 'Dashboard', shortcut: 'Cmd+1' },
				{ id: 'workspace', icon: 'briefcase', tooltip: 'Workspace', shortcut: 'Cmd+2' },
				{ id: 'components', icon: 'component', tooltip: 'Components', shortcut: 'Cmd+3' },
				{ id: 'packages', icon: 'package', tooltip: 'Packages', shortcut: 'Cmd+4' },
				{ id: 'fixtures', icon: 'database', tooltip: 'Fixtures', shortcut: 'Cmd+5' },
				{ id: 'divider1', icon: '', tooltip: '', divider: true },
				{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
			],
			showToggle: true
		}
	},
	{
		name: 'Figma Style',
		description: 'Figma-like editor sidebar',
		props: {
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			allowPin: true,
			expandOnHover: false,
			railItems: [
				{ id: 'layers', icon: 'layers', tooltip: 'Layers' },
				{ id: 'components', icon: 'component', tooltip: 'Components' },
				{ id: 'assets', icon: 'folder', tooltip: 'Assets' }
			],
			position: 'left'
		}
	},
	{
		name: 'VS Code Style',
		description: 'VS Code-like activity bar + sidebar',
		props: {
			expanded: true,
			railWidth: 48,
			contextWidth: 240,
			allowPin: true,
			railItems: [
				{ id: 'explorer', icon: 'files', tooltip: 'Explorer' },
				{ id: 'search', icon: 'search', tooltip: 'Search' },
				{ id: 'git', icon: 'git-branch', tooltip: 'Source Control' },
				{ id: 'debug', icon: 'bug', tooltip: 'Run and Debug' },
				{ id: 'extensions', icon: 'puzzle', tooltip: 'Extensions' },
				{ id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
			]
		}
	},
	{
		name: 'Minimal',
		description: 'Minimal collapsed rail only',
		props: {
			expanded: false,
			railWidth: 48,
			showToggle: false,
			allowPin: false,
			railItems: [
				{ id: 'home', icon: 'home', tooltip: 'Home' },
				{ id: 'search', icon: 'search', tooltip: 'Search' },
				{ id: 'settings', icon: 'settings', tooltip: 'Settings' }
			]
		}
	}
];

// =============================================================================
// PROPS DEFINITIONS
// =============================================================================

const props: IPropDef[] = [
	{
		name: 'expanded',
		type: 'boolean',
		default: true,
		description: 'Whether context panel is visible (legacy prop)'
	},
	{
		name: 'expansionState',
		type: 'select',
		options: ['collapsed', 'expanded', 'pinned'],
		default: 'expanded',
		description: 'Current expansion state'
	},
	{
		name: 'railWidth',
		type: 'number',
		default: 48,
		min: 40,
		max: 64,
		description: 'Width of the nav rail in pixels'
	},
	{
		name: 'contextWidth',
		type: 'number',
		default: 240,
		min: 180,
		max: 400,
		description: 'Width of the context panel in pixels'
	},
	{
		name: 'position',
		type: 'select',
		options: ['left', 'right'],
		default: 'left',
		description: 'Sidebar position'
	},
	{
		name: 'showToggle',
		type: 'boolean',
		default: true,
		description: 'Show expand/collapse button in rail'
	},
	{
		name: 'allowPin',
		type: 'boolean',
		default: true,
		description: 'Allow pinning the panel open'
	},
	{
		name: 'expandOnHover',
		type: 'boolean',
		default: false,
		description: 'Expand panel when hovering over rail'
	},
	{
		name: 'hoverDelay',
		type: 'number',
		default: 200,
		min: 0,
		max: 1000,
		description: 'Delay before expanding on hover (ms)'
	}
];

// =============================================================================
// PRESENTER DEFINITION
// =============================================================================

const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'CollapsibleSidebar',
	slug: 'collapsible-sidebar',
	category: 'sections',
	presenterType: 'component',
	component: CollapsibleSidebar,

	// Optional enrichment
	description:
		'Canva/Figma-style collapsible sidebar combining NavRail (48px icons) with ContextPanel (240px expandable content). Supports collapsed, expanded, and pinned states with 200ms ease-out transitions.',

	variants,
	presets,
	props,

	// Source reference
	source: {
		path: 'components/sections/CollapsibleSidebar.svelte'
	},

	// Searchable tags
	tags: ['sidebar', 'navigation', 'rail', 'panel', 'collapsible', 'figma', 'canva', 'editor'],

	// Documentation
	documentation: `
## Basic Usage

\`\`\`svelte
<script>
  import CollapsibleSidebar from './CollapsibleSidebar.svelte';

  const railItems = [
    { id: 'files', icon: 'folder', tooltip: 'Files' },
    { id: 'search', icon: 'search', tooltip: 'Search' },
    { id: 'settings', icon: 'settings', tooltip: 'Settings', position: 'bottom' }
  ];
</script>

<CollapsibleSidebar
  {railItems}
  activeRailItem="files"
  expanded={true}
/>
\`\`\`

## With Context Sections

\`\`\`svelte
<CollapsibleSidebar
  railItems={[...]}
  contextSections={[
    {
      id: 'files',
      type: 'tree',
      header: 'Explorer',
      items: [...]
    }
  ]}
/>
\`\`\`

## Pinned State

\`\`\`svelte
<script>
  let expansionState = $state('pinned');
</script>

<CollapsibleSidebar
  {railItems}
  bind:expansionState
  allowPin={true}
  onstatechange={(s) => console.log('State:', s)}
/>
\`\`\`

## Expand on Hover

\`\`\`svelte
<CollapsibleSidebar
  {railItems}
  expanded={false}
  expandOnHover={true}
  hoverDelay={200}
/>
\`\`\`

## Keyboard Shortcuts

- **Cmd+B**: Toggle sidebar expand/collapse
- **Cmd+Shift+P**: Toggle pin state
- **Cmd+[**: Collapse context panel only
`,

	// Status
	status: 'stable',
	version: '2.0.0'
};

export default presenter;

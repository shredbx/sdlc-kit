/**
 * Dock Presenter - Auto-registration for component library
 *
 * Defines how Dock is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import Dock from './Dock.svelte';

/**
 * Dock property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'items',
		type: 'json',
		description: 'Array of dock items with id, label, icon, tooltip',
		default: []
	},
	{
		name: 'activeId',
		type: 'string',
		description: 'ID of the currently active item',
		default: ''
	},
	{
		name: 'showLabels',
		type: 'boolean',
		description: 'Show text labels alongside icons',
		default: true
	},
	{
		name: 'position',
		type: 'select',
		options: ['bottom', 'top'],
		description: 'Dock position in container',
		default: 'bottom'
	}
];

/**
 * Example items for variants
 */
const sampleItems = [
	{ id: 'all', label: 'All', icon: '📦' },
	{ id: 'ui', label: 'UI', icon: '🎨' },
	{ id: 'utils', label: 'Utils', icon: '🔧' },
	{ id: 'adapters', label: 'Adapters', icon: '🔌' }
];

/**
 * Dock variants for different states/configurations.
 */
const variants: IVariant[] = [
	{
		name: 'Default',
		description: 'Standard dock with labels',
		props: {
			items: sampleItems,
			activeId: 'all',
			showLabels: true,
			position: 'bottom'
		}
	},
	{
		name: 'Icons Only',
		description: 'Compact dock without labels',
		props: {
			items: sampleItems,
			activeId: 'ui',
			showLabels: false,
			position: 'bottom'
		}
	},
	{
		name: 'Top Position',
		description: 'Dock at top of container',
		props: {
			items: sampleItems,
			activeId: 'utils',
			showLabels: true,
			position: 'top'
		}
	},
	{
		name: 'No Selection',
		description: 'Dock without active item',
		props: {
			items: sampleItems,
			activeId: '',
			showLabels: true,
			position: 'bottom'
		}
	}
];

/**
 * Dock presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'Package Filter',
		description: 'Filter dock for packages page',
		props: {
			items: [
				{ id: 'all', label: 'All', icon: '📦' },
				{ id: 'ui', label: 'UI', icon: '🎨' },
				{ id: 'utils', label: 'Utils', icon: '🔧' }
			],
			activeId: 'all',
			showLabels: true
		}
	},
	{
		name: 'Component Filter',
		description: 'Filter dock for components page',
		props: {
			items: [
				{ id: 'all', label: 'All', icon: '🧩' },
				{ id: 'primitives', label: 'Primitives', icon: '🔵' },
				{ id: 'blocks', label: 'Blocks', icon: '🟦' },
				{ id: 'sections', label: 'Sections', icon: '🟪' }
			],
			activeId: 'all',
			showLabels: true
		}
	}
];

/**
 * Dock Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'Dock',
	slug: 'dock',
	category: 'blocks',
	presenterType: 'component',

	// Component reference
	component: Dock,

	// Optional - enrichment
	description:
		'Floating navigation dock for filtering and quick actions. macOS-style dock component with icons, labels, and active state indicator.',

	tags: ['navigation', 'dock', 'filter', 'tabs', 'floating'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/blocks/Dock.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 400,
		height: 80,
		background: 'dark',
		defaultProps: {
			items: sampleItems,
			activeId: 'all',
			showLabels: true,
			position: 'bottom'
		}
	},

	documentation: `
## Dock Component

Floating navigation dock for filtering, section navigation, and quick actions.
Inspired by macOS dock with glassmorphism styling.

### Usage

\`\`\`svelte
<script>
  import Dock from './Dock.svelte';

  const items = [
    { id: 'all', label: 'All', icon: '📦' },
    { id: 'ui', label: 'UI', icon: '🎨' },
    { id: 'utils', label: 'Utils', icon: '🔧' }
  ];

  let activeId = 'all';

  function handleSelect(id: string) {
    activeId = id;
  }
</script>

<Dock {items} {activeId} onSelect={handleSelect} />
\`\`\`

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| items | DockItem[] | [] | Array of dock items |
| activeId | string | '' | Currently active item ID |
| showLabels | boolean | true | Show text labels |
| position | 'bottom' \\| 'top' | 'bottom' | Dock position |
| onSelect | (id: string) => void | - | Selection handler |

### DockItem Interface

\`\`\`typescript
interface DockItem {
  id: string;
  label: string;
  icon?: string;
  tooltip?: string;
}
\`\`\`

### Icons Only Mode

\`\`\`svelte
<Dock {items} {activeId} showLabels={false} />
\`\`\`
`
};

export default presenter;

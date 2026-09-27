/**
 * Badge Presenter - Auto-registration for component library
 *
 * Defines how Badge is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import Badge from './Badge.svelte';

/**
 * Badge property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'count',
		type: 'number',
		description: 'Numeric count to display',
		default: undefined
	},
	{
		name: 'label',
		type: 'string',
		description: 'Text label (alternative to count)',
		default: undefined
	},
	{
		name: 'variant',
		type: 'select',
		options: ['default', 'primary', 'success', 'warning', 'error', 'info'],
		description: 'Visual style variant',
		default: 'default'
	},
	{
		name: 'size',
		type: 'select',
		options: ['xs', 'sm', 'md', 'lg'],
		description: 'Size preset',
		default: 'sm'
	},
	{
		name: 'dot',
		type: 'boolean',
		description: 'Show as dot only (no content)',
		default: false
	},
	{
		name: 'max',
		type: 'number',
		description: 'Maximum count before showing +',
		default: 99
	},
	{
		name: 'showZero',
		type: 'boolean',
		description: 'Show badge when count is zero',
		default: false
	},
	{
		name: 'pulse',
		type: 'boolean',
		description: 'Pulse animation for attention',
		default: false
	},
	{
		name: 'icon',
		type: 'string',
		description: 'Emoji or icon before label',
		default: undefined
	}
];

/**
 * Badge variants for different states/configurations.
 */
const variants: IVariant[] = [
	// Count variants
	{
		name: 'Notification Count',
		description: 'Unread notification indicator',
		props: { count: 5, variant: 'error' }
	},
	{
		name: 'Large Count',
		description: 'Count exceeding max (99+)',
		props: { count: 150, variant: 'error' }
	},
	{
		name: 'Zero Count',
		description: 'Zero with showZero enabled',
		props: { count: 0, showZero: true, variant: 'default' }
	},

	// Label variants
	{
		name: 'New Label',
		description: 'New feature indicator',
		props: { label: 'New', variant: 'success' }
	},
	{
		name: 'Beta Label',
		description: 'Beta feature tag',
		props: { label: 'Beta', variant: 'warning' }
	},
	{
		name: 'Pro Label',
		description: 'Premium feature indicator',
		props: { label: 'Pro', variant: 'primary' }
	},

	// Status variants
	{
		name: 'Online Status',
		description: 'Green dot for online',
		props: { dot: true, variant: 'success' }
	},
	{
		name: 'Offline Status',
		description: 'Gray dot for offline',
		props: { dot: true, variant: 'default' }
	},
	{
		name: 'Busy Status',
		description: 'Red dot for busy/DND',
		props: { dot: true, variant: 'error' }
	},

	// Size variants
	{
		name: 'Extra Small',
		description: 'Minimal size badge',
		props: { count: 3, size: 'xs', variant: 'primary' }
	},
	{
		name: 'Medium',
		description: 'Standard size badge',
		props: { count: 3, size: 'md', variant: 'primary' }
	},
	{
		name: 'Large',
		description: 'Prominent badge',
		props: { count: 3, size: 'lg', variant: 'primary' }
	},

	// Special variants
	{
		name: 'Pulse Animation',
		description: 'Attention-grabbing pulse effect',
		props: { count: 1, variant: 'error', pulse: true }
	},
	{
		name: 'With Icon',
		description: 'Badge with leading icon',
		props: { label: 'Updates', icon: '🔄', variant: 'info' }
	}
];

/**
 * Badge presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'Unread Messages',
		description: 'Notification badge for unread count',
		props: { count: 3, variant: 'error', size: 'sm' }
	},
	{
		name: 'Nav Badge',
		description: 'Navigation item count badge',
		props: { count: 12, variant: 'warning', size: 'sm' }
	},
	{
		name: 'Feature Tag',
		description: 'New/Beta/Pro feature indicator',
		props: { label: 'New', variant: 'success', size: 'xs' }
	},
	{
		name: 'User Status Dot',
		description: 'Online/Offline status indicator',
		props: { dot: true, variant: 'success', size: 'sm' }
	},
	{
		name: 'Alert Badge',
		description: 'Critical notification with pulse',
		props: { count: 1, variant: 'error', pulse: true }
	}
];

/**
 * Badge Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance (4 fields)
	name: 'Badge',
	slug: 'badge',
	category: 'primitives',
	presenterType: 'component',

	// Component reference
	component: Badge,

	// Optional - enrichment
	description:
		'A compact label component for counts, status indicators, or categorization tags. Supports multiple variants, sizes, and animations.',

	tags: ['notification', 'count', 'status', 'label', 'indicator'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/primitives/Badge.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 100,
		height: 50,
		background: 'dark',
		defaultProps: { count: 5, variant: 'error' }
	},

	documentation: `
## Badge Component

A compact label for displaying counts, status indicators, or categorization tags.

### Usage

\`\`\`svelte
<script>
  import Badge from './Badge.svelte';
</script>

<Badge count={5} variant="error" />
<Badge label="New" variant="success" />
<Badge dot variant="success" />
\`\`\`

### Count Display

- Shows numeric count with automatic 99+ formatting
- Use \`max\` prop to customize threshold
- Use \`showZero\` to display zero counts

### Dot Mode

Use \`dot\` prop for status indicators without text.

### Animations

- Use \`pulse\` prop for attention-grabbing effect
`
};

export default presenter;

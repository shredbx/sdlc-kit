/**
 * IconButton Presenter - Auto-registration for component library
 *
 * Defines how IconButton is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import IconButton from './IconButton.svelte';

const props: IPropDef[] = [
	{
		name: 'icon',
		type: 'string',
		description: 'Icon name (Lucide icon name like "bell", "settings", "user")',
		default: undefined
	},
	{
		name: 'label',
		type: 'string',
		description: 'Accessible label (required)',
		default: undefined,
		required: true
	},
	{
		name: 'variant',
		type: 'select',
		options: ['default', 'primary', 'secondary', 'ghost', 'danger', 'success'],
		description: 'Visual variant',
		default: 'default'
	},
	{
		name: 'size',
		type: 'select',
		options: ['xs', 'sm', 'md', 'lg', 'xl'],
		description: 'Size preset',
		default: 'md'
	},
	{
		name: 'shape',
		type: 'select',
		options: ['square', 'rounded', 'circle'],
		description: 'Button shape',
		default: 'rounded'
	},
	{
		name: 'disabled',
		type: 'boolean',
		description: 'Disabled state',
		default: false
	},
	{
		name: 'loading',
		type: 'boolean',
		description: 'Loading state with spinner',
		default: false
	},
	{
		name: 'active',
		type: 'boolean',
		description: 'Active/pressed state',
		default: false
	},
	{
		name: 'tooltip',
		type: 'boolean',
		description: 'Show tooltip on hover',
		default: true
	},
	{
		name: 'tooltipPosition',
		type: 'select',
		options: ['top', 'bottom', 'left', 'right'],
		description: 'Tooltip position',
		default: 'top'
	},
	{
		name: 'badge',
		type: 'number',
		description: 'Badge count',
		default: undefined
	},
	{
		name: 'badgeVariant',
		type: 'select',
		options: ['default', 'error', 'warning', 'success'],
		description: 'Badge variant',
		default: 'error'
	}
];

const variants: IVariant[] = [
	// Variants
	{ name: 'Default', description: 'Standard icon button', props: { icon: 'settings', label: 'Settings' } },
	{ name: 'Primary', description: 'Primary action', props: { icon: 'plus', label: 'Add', variant: 'primary' } },
	{ name: 'Secondary', description: 'Secondary action', props: { icon: 'edit', label: 'Edit', variant: 'secondary' } },
	{ name: 'Ghost', description: 'Minimal style', props: { icon: 'x', label: 'Close', variant: 'ghost' } },
	{ name: 'Danger', description: 'Destructive action', props: { icon: 'trash', label: 'Delete', variant: 'danger' } },
	{ name: 'Success', description: 'Positive action', props: { icon: 'check', label: 'Confirm', variant: 'success' } },

	// Shapes
	{ name: 'Square', description: 'Square shape', props: { icon: 'folder', label: 'Folder', shape: 'square' } },
	{ name: 'Rounded', description: 'Rounded corners', props: { icon: 'folder', label: 'Folder', shape: 'rounded' } },
	{ name: 'Circle', description: 'Circular shape', props: { icon: 'message-circle', label: 'Chat', shape: 'circle', variant: 'primary' } },

	// Sizes
	{ name: 'Extra Small', description: 'Tiny button', props: { icon: 'x', label: 'Close', size: 'xs' } },
	{ name: 'Small', description: 'Small button', props: { icon: 'x', label: 'Close', size: 'sm' } },
	{ name: 'Large', description: 'Large button', props: { icon: 'message-circle', label: 'Chat', size: 'lg', variant: 'primary' } },
	{ name: 'Extra Large', description: 'Very large button', props: { icon: 'message-circle', label: 'Chat', size: 'xl', variant: 'primary' } },

	// States
	{ name: 'Disabled', description: 'Disabled button', props: { icon: 'settings', label: 'Settings', disabled: true } },
	{ name: 'Loading', description: 'Loading spinner', props: { icon: 'save', label: 'Save', loading: true } },
	{ name: 'Active', description: 'Active state', props: { icon: 'bell', label: 'Notifications', active: true } },

	// With badge
	{ name: 'With Badge', description: 'Notification badge', props: { icon: 'bell', label: 'Notifications', badge: 5 } },
	{ name: 'Badge 99+', description: 'Large badge count', props: { icon: 'mail', label: 'Messages', badge: 150 } },
	{ name: 'Warning Badge', description: 'Warning indicator', props: { icon: 'alert-triangle', label: 'Warnings', badge: 3, badgeVariant: 'warning' } }
];

const presets: IPreset[] = [
	{
		name: 'Notification Bell',
		description: 'Header notification button',
		props: { icon: 'bell', label: 'Notifications', badge: 3, size: 'md', variant: 'ghost' }
	},
	{
		name: 'Chat FAB',
		description: 'Floating chat button',
		props: { icon: 'message-circle', label: 'Chat', shape: 'circle', size: 'lg', variant: 'primary' }
	},
	{
		name: 'Close Button',
		description: 'Modal/dialog close',
		props: { icon: 'x', label: 'Close', variant: 'ghost', size: 'sm' }
	},
	{
		name: 'Settings Cog',
		description: 'Settings access',
		props: { icon: 'settings', label: 'Settings', variant: 'default' }
	},
	{
		name: 'Send Message',
		description: 'Chat send button',
		props: { icon: 'send', label: 'Send', variant: 'primary', shape: 'rounded' }
	},
	{
		name: 'Delete Action',
		description: 'Delete with confirmation',
		props: { icon: 'trash', label: 'Delete', variant: 'danger' }
	}
];

const presenter: IPresenter = {
	name: 'IconButton',
	slug: 'icon-button',
	category: 'primitives',
	presenterType: 'component',
	component: IconButton,

	description:
		'A button optimized for icon-only interactions with proper accessibility. Supports badges, loading states, and multiple variants.',

	tags: ['button', 'icon', 'action', 'interactive', 'fab'],
	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/primitives/IconButton.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 80,
		height: 80,
		background: 'dark',
		defaultProps: { icon: 'bell', label: 'Notifications', variant: 'primary', shape: 'circle', size: 'lg' }
	},

	documentation: `
## IconButton Component

A button component optimized for icon-only interactions. Uses Lucide icons.

### Usage

\`\`\`svelte
<script>
  import IconButton from './IconButton.svelte';
</script>

<IconButton icon="bell" label="Notifications" />
<IconButton icon="x" label="Close" variant="ghost" />
<IconButton icon="message-circle" label="Chat" variant="primary" shape="circle" />
\`\`\`

### Accessibility

The \`label\` prop is **required** for screen readers. It's used as:
- \`aria-label\` for the button
- \`title\` attribute for tooltip (when enabled)

### Icon Names

Use Lucide icon names like: bell, settings, user, search, plus, x, check, trash, edit, save, send, mail, message-circle, and more.

### Variants

- \`default\`: Standard tertiary style
- \`primary\`: Highlighted primary action
- \`secondary\`: Outlined style
- \`ghost\`: Minimal, transparent
- \`danger\`: Destructive actions
- \`success\`: Positive confirmations

### Badges

Use \`badge\` prop for notification counts:

\`\`\`svelte
<IconButton icon="bell" label="Notifications" badge={5} />
\`\`\`
`
};

export default presenter;

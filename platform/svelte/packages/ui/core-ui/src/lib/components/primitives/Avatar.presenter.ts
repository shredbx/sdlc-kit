/**
 * Avatar Presenter - Auto-registration for component library
 *
 * Defines how Avatar is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import Avatar from './Avatar.svelte';

const props: IPropDef[] = [
	{
		name: 'src',
		type: 'string',
		description: 'Image source URL',
		default: undefined
	},
	{
		name: 'alt',
		type: 'string',
		description: 'Alt text for image',
		default: undefined
	},
	{
		name: 'name',
		type: 'string',
		description: 'User name for initials fallback',
		default: undefined
	},
	{
		name: 'icon',
		type: 'string',
		description: 'Icon fallback (emoji)',
		default: undefined
	},
	{
		name: 'size',
		type: 'select',
		options: ['xs', 'sm', 'md', 'lg', 'xl', '2xl'],
		description: 'Size preset',
		default: 'md'
	},
	{
		name: 'shape',
		type: 'select',
		options: ['circle', 'square', 'rounded'],
		description: 'Shape variant',
		default: 'circle'
	},
	{
		name: 'status',
		type: 'select',
		options: ['online', 'offline', 'busy', 'away', 'invisible'],
		description: 'Status indicator',
		default: undefined
	},
	{
		name: 'ring',
		type: 'string',
		description: 'Ring/border color',
		default: undefined
	},
	{
		name: 'bgColor',
		type: 'string',
		description: 'Background color for initials/icon',
		default: undefined
	},
	{
		name: 'textColor',
		type: 'string',
		description: 'Text color for initials',
		default: undefined
	},
	{
		name: 'clickable',
		type: 'boolean',
		description: 'Enable button behavior',
		default: false
	},
	{
		name: 'loading',
		type: 'boolean',
		description: 'Loading skeleton state',
		default: false
	}
];

const variants: IVariant[] = [
	// Display modes
	{ name: 'With Image', description: 'User photo', props: { src: 'https://i.pravatar.cc/150?img=1', alt: 'User' } },
	{ name: 'Initials', description: 'Name initials', props: { name: 'John Doe' } },
	{ name: 'Single Initial', description: 'One letter initial', props: { name: 'Alice' } },
	{ name: 'Icon Fallback', description: 'Emoji icon', props: { icon: '👤' } },
	{ name: 'Placeholder', description: 'Default placeholder', props: {} },

	// Sizes
	{ name: 'Extra Small', description: 'Tiny avatar', props: { name: 'XS', size: 'xs' } },
	{ name: 'Small', description: 'Small avatar', props: { name: 'SM', size: 'sm' } },
	{ name: 'Medium', description: 'Standard avatar', props: { name: 'MD', size: 'md' } },
	{ name: 'Large', description: 'Large avatar', props: { name: 'LG', size: 'lg' } },
	{ name: 'Extra Large', description: 'Very large avatar', props: { name: 'XL', size: 'xl' } },
	{ name: '2X Large', description: 'Huge avatar', props: { name: '2X', size: '2xl' } },

	// Shapes
	{ name: 'Circle', description: 'Round avatar', props: { name: 'JD', shape: 'circle' } },
	{ name: 'Square', description: 'Square avatar', props: { name: 'JD', shape: 'square' } },
	{ name: 'Rounded', description: 'Rounded corners', props: { name: 'JD', shape: 'rounded' } },

	// Status
	{ name: 'Online', description: 'Online status', props: { name: 'JD', status: 'online' } },
	{ name: 'Offline', description: 'Offline status', props: { name: 'JD', status: 'offline' } },
	{ name: 'Busy', description: 'Busy/DND status', props: { name: 'JD', status: 'busy' } },
	{ name: 'Away', description: 'Away status', props: { name: 'JD', status: 'away' } },

	// Special
	{ name: 'With Ring', description: 'Highlighted avatar', props: { name: 'JD', ring: '#6366f1' } },
	{ name: 'Custom Colors', description: 'Custom background', props: { name: 'JD', bgColor: '#10b981', textColor: '#fff' } },
	{ name: 'Clickable', description: 'Button behavior', props: { name: 'JD', clickable: true } },
	{ name: 'Loading', description: 'Skeleton loading', props: { loading: true } }
];

const presets: IPreset[] = [
	{
		name: 'User Profile',
		description: 'Main user avatar',
		props: { src: 'https://i.pravatar.cc/150?img=3', size: 'lg', status: 'online' }
	},
	{
		name: 'Navigation Avatar',
		description: 'Small nav avatar',
		props: { name: 'John Doe', size: 'sm', clickable: true }
	},
	{
		name: 'Comment Author',
		description: 'Comment/chat avatar',
		props: { name: 'Jane Smith', size: 'md' }
	},
	{
		name: 'Team Member',
		description: 'Team list avatar',
		props: { name: 'Bob Wilson', size: 'md', status: 'online', ring: '#6366f1' }
	},
	{
		name: 'Placeholder',
		description: 'Empty state avatar',
		props: { icon: '👤', size: 'md' }
	}
];

const presenter: IPresenter = {
	name: 'Avatar',
	slug: 'avatar',
	category: 'primitives',
	presenterType: 'component',
	component: Avatar,

	description:
		'A user representation component with image, initials, or icon fallback. Supports status indicators and multiple shapes.',

	tags: ['avatar', 'user', 'profile', 'image', 'initials'],
	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/primitives/Avatar.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 80,
		height: 80,
		background: 'dark',
		defaultProps: { name: 'John Doe', size: 'lg', status: 'online' }
	},

	documentation: `
## Avatar Component

A user representation with multiple display modes and status indicators.

### Usage

\`\`\`svelte
<script>
  import Avatar from './Avatar.svelte';
</script>

<Avatar src="/user.jpg" alt="John" />
<Avatar name="John Doe" />
<Avatar icon="👤" size="lg" />
<Avatar name="JD" status="online" />
\`\`\`

### Display Modes

1. **Image**: Provide \`src\` for user photo
2. **Initials**: Provide \`name\` to auto-generate initials
3. **Icon**: Provide \`icon\` for emoji fallback
4. **Placeholder**: Default user icon

### Status Indicators

- \`online\`: Green dot
- \`offline\`: Gray dot
- \`busy\`: Red dot
- \`away\`: Yellow dot
- \`invisible\`: No indicator

### Accessibility

Always provide \`alt\` when using \`src\`, or \`name\` for proper labeling.
`
};

export default presenter;

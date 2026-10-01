/**
 * Icon Presenter - Auto-registration for component library
 *
 * Defines how Icon is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import Icon from './Icon.svelte';

const props: IPropDef[] = [
	{
		name: 'name',
		type: 'string',
		description: 'Icon name (lucide icon name, emoji, symbol id, or font icon)',
		default: undefined,
		required: true
	},
	{
		name: 'type',
		type: 'select',
		options: ['lucide', 'emoji', 'symbol', 'font'],
		description: 'Icon type',
		default: 'lucide'
	},
	{
		name: 'size',
		type: 'select',
		options: ['xs', 'sm', 'md', 'lg', 'xl', '2xl'],
		description: 'Size preset',
		default: 'md'
	},
	{
		name: 'color',
		type: 'string',
		description: 'Custom color (CSS value)',
		default: undefined
	},
	{
		name: 'spin',
		type: 'boolean',
		description: 'Continuous spin animation',
		default: false
	},
	{
		name: 'pulse',
		type: 'boolean',
		description: 'Pulse animation',
		default: false
	},
	{
		name: 'flipX',
		type: 'boolean',
		description: 'Flip horizontally',
		default: false
	},
	{
		name: 'flipY',
		type: 'boolean',
		description: 'Flip vertically',
		default: false
	},
	{
		name: 'rotate',
		type: 'number',
		description: 'Rotation in degrees',
		default: 0
	},
	{
		name: 'label',
		type: 'string',
		description: 'Accessible label for screen readers',
		default: undefined
	}
];

const variants: IVariant[] = [
	// Common Lucide icons
	{ name: 'Bell', description: 'Notification bell', props: { name: 'bell', size: 'md' } },
	{ name: 'Settings', description: 'Gear/cog icon', props: { name: 'settings', size: 'md' } },
	{ name: 'Search', description: 'Magnifier icon', props: { name: 'search', size: 'md' } },
	{ name: 'User', description: 'Person icon', props: { name: 'user', size: 'md' } },
	{ name: 'Chat', description: 'Message bubble', props: { name: 'message-circle', size: 'md' } },
	{ name: 'Close', description: 'X close icon', props: { name: 'x', size: 'md' } },
	{ name: 'Home', description: 'Home icon', props: { name: 'home', size: 'md' } },
	{ name: 'Check', description: 'Checkmark', props: { name: 'check', size: 'md' } },
	{ name: 'Plus', description: 'Add icon', props: { name: 'plus', size: 'md' } },
	{ name: 'Trash', description: 'Delete icon', props: { name: 'trash', size: 'md' } },

	// Sizes
	{ name: 'Extra Small', description: 'Tiny icon', props: { name: 'star', size: 'xs' } },
	{ name: 'Small', description: 'Small icon', props: { name: 'star', size: 'sm' } },
	{ name: 'Large', description: 'Large icon', props: { name: 'star', size: 'lg' } },
	{ name: 'Extra Large', description: 'Very large icon', props: { name: 'star', size: 'xl' } },
	{ name: '2X Large', description: 'Huge icon', props: { name: 'star', size: '2xl' } },

	// Animations
	{ name: 'Loading Spin', description: 'Spinning loader', props: { name: 'loader', spin: true } },
	{ name: 'Attention Pulse', description: 'Pulsing alert', props: { name: 'alert-circle', pulse: true } },

	// Transforms
	{ name: 'Flipped X', description: 'Horizontally flipped', props: { name: 'arrow-right', flipX: true } },
	{ name: 'Rotated 45°', description: '45 degree rotation', props: { name: 'arrow-up', rotate: 45 } },
	{ name: 'Rotated 90°', description: '90 degree rotation', props: { name: 'arrow-up', rotate: 90 } },

	// Colored
	{ name: 'Custom Color', description: 'Colored icon', props: { name: 'star', color: '#6366f1' } },

	// Emoji fallback
	{ name: 'Emoji Bell', description: 'Emoji type icon', props: { name: '🔔', type: 'emoji', size: 'md' } },
	{ name: 'Emoji Star', description: 'Emoji star', props: { name: '⭐', type: 'emoji', size: 'md' } }
];

const presets: IPreset[] = [
	{
		name: 'Navigation Icon',
		description: 'Sidebar/nav icon',
		props: { name: 'folder', size: 'md' }
	},
	{
		name: 'Button Icon',
		description: 'Icon for buttons',
		props: { name: 'plus', size: 'sm' }
	},
	{
		name: 'Loading Indicator',
		description: 'Spinning loader',
		props: { name: 'loader', size: 'md', spin: true }
	},
	{
		name: 'Status Icon',
		description: 'Status indicator',
		props: { name: 'check-circle', size: 'sm', color: '#10b981' }
	}
];

const presenter: IPresenter = {
	name: 'Icon',
	slug: 'icon',
	category: 'primitives',
	presenterType: 'component',
	component: Icon,

	description:
		'A flexible icon display supporting Lucide icons, emoji, SVG symbols, and icon fonts. Uses Lucide by default for crisp, consistent icons.',

	tags: ['icon', 'lucide', 'emoji', 'symbol', 'display', 'visual'],
	status: 'stable',
	version: '2.0.0',

	source: {
		path: 'src/lib/components/primitives/Icon.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 60,
		height: 60,
		background: 'dark',
		defaultProps: { name: 'bell', size: 'lg' }
	},

	documentation: `
## Icon Component

A flexible icon display supporting multiple icon sources. Uses Lucide icons by default.

### Usage

\`\`\`svelte
<script>
  import Icon from './Icon.svelte';
</script>

<!-- Lucide icons (default) -->
<Icon name="bell" />
<Icon name="settings" size="lg" />
<Icon name="loader" spin />

<!-- Emoji fallback -->
<Icon name="🔔" type="emoji" />
\`\`\`

### Icon Types

- \`lucide\`: Lucide icons (default, 1500+ icons)
- \`emoji\`: Native emoji characters
- \`symbol\`: SVG symbol sprites
- \`font\`: Icon fonts (Material, FontAwesome)

### Available Lucide Icons

Common icons: bell, settings, user, search, home, plus, x, check, trash, edit, save, download, upload, share, mail, message-circle, phone, calendar, clock, star, heart, eye, lock, key, and many more.

### Animations

- \`spin\`: Continuous rotation (loading states)
- \`pulse\`: Opacity pulse (attention)

### Accessibility

Always provide \`label\` prop when icon conveys meaning.
`
};

export default presenter;

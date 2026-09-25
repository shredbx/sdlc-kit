/**
 * Button Presenter - Demonstrates the IPresenter pattern
 *
 * This file shows how to create a presenter with:
 * - Required fields (minimal conformance)
 * - Variants (different states)
 * - Presets (common use cases)
 * - Props (for real-time editing)
 */

import type { IPresenter } from '../../presenter/IPresenter';
import Button from './Button.svelte';

const ButtonPresenter: IPresenter = {
	// =========================================================================
	// REQUIRED FIELDS (4 fields to be listed)
	// =========================================================================
	name: 'Button',
	slug: 'button',
	category: 'primitives',
	presenterType: 'component',

	// =========================================================================
	// ENRICHMENT FIELDS (optional)
	// =========================================================================
	component: Button,

	description: 'Primary call-to-action button with multiple variants, sizes, and states.',

	source: {
		path: 'src/lib/components/primitives/Button.svelte',
		repo: 'hub'
	},

	status: 'stable',
	version: '1.0.0',

	tags: ['button', 'action', 'cta', 'interactive', 'primitive'],

	// Variants showcase different states
	variants: [
		{
			name: 'Primary',
			description: 'Main call-to-action style',
			props: { variant: 'primary' }
		},
		{
			name: 'Secondary',
			description: 'Secondary action style',
			props: { variant: 'secondary' }
		},
		{
			name: 'Ghost',
			description: 'Minimal visual footprint',
			props: { variant: 'ghost' }
		},
		{
			name: 'Outline',
			description: 'Bordered without fill',
			props: { variant: 'outline' }
		},
		{
			name: 'Destructive',
			description: 'Dangerous action warning',
			props: { variant: 'destructive' }
		},
		{
			name: 'Disabled',
			description: 'Non-interactive state',
			props: { variant: 'primary', disabled: true }
		},
		{
			name: 'Loading',
			description: 'Processing state',
			props: { variant: 'primary', loading: true }
		},
		{
			name: 'With Icon',
			description: 'Button with leading icon',
			props: { variant: 'primary', icon: '✓' }
		}
	],

	// Presets are ready-to-use configurations
	presets: [
		{
			name: 'Submit Button',
			description: 'Standard form submission',
			props: { variant: 'primary' },
			code: '<Button variant="primary">Submit</Button>'
		},
		{
			name: 'Cancel Button',
			description: 'Form cancellation',
			props: { variant: 'ghost' },
			code: '<Button variant="ghost">Cancel</Button>'
		},
		{
			name: 'Delete Button',
			description: 'Dangerous action confirmation',
			props: { variant: 'destructive', icon: '🗑' },
			code: '<Button variant="destructive" icon="🗑">Delete</Button>'
		},
		{
			name: 'Save Button',
			description: 'Save current state',
			props: { variant: 'outline', icon: '💾' },
			code: '<Button variant="outline" icon="💾">Save</Button>'
		}
	],

	// Props for real-time editing
	props: [
		{
			name: 'variant',
			label: 'Variant',
			type: 'select',
			description: 'Visual style of the button',
			default: 'primary',
			options: ['primary', 'secondary', 'ghost', 'outline', 'destructive']
		},
		{
			name: 'size',
			label: 'Size',
			type: 'select',
			description: 'Button size',
			default: 'md',
			options: ['sm', 'md', 'lg']
		},
		{
			name: 'disabled',
			label: 'Disabled',
			type: 'boolean',
			description: 'Whether button is disabled',
			default: false
		},
		{
			name: 'loading',
			label: 'Loading',
			type: 'boolean',
			description: 'Show loading spinner',
			default: false
		},
		{
			name: 'icon',
			label: 'Icon',
			type: 'string',
			description: 'Leading icon (emoji or icon name)',
			default: ''
		}
	],

	preview: {
		width: 150,
		height: 50,
		background: 'dark',
		defaultProps: {}
	},

	documentation: `
## Button Component

The Button component is a fundamental primitive for user interactions.

### Usage

\`\`\`svelte
<script>
  import Button from './Button.svelte';
</script>

<Button variant="primary">Click me</Button>
\`\`\`

### Variants

- **primary** - Main call-to-action (default)
- **secondary** - Secondary actions
- **ghost** - Minimal visual footprint
- **outline** - Bordered without fill
- **destructive** - Dangerous actions

### Sizes

- **sm** - Small buttons for compact UIs
- **md** - Default size (recommended)
- **lg** - Large buttons for emphasis

### Accessibility

- Uses native \`<button>\` element
- Supports \`disabled\` state
- Loading state disables interaction
`
};

export default ButtonPresenter;

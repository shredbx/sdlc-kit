/**
 * Input Presenter - Auto-registration for component library
 *
 * Defines how Input is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import Input from './Input.svelte';

const props: IPropDef[] = [
	{
		name: 'value',
		type: 'string',
		description: 'Current input value (bindable)',
		default: ''
	},
	{
		name: 'type',
		type: 'select',
		options: ['text', 'password', 'email', 'number', 'search', 'tel', 'url'],
		description: 'Input type',
		default: 'text'
	},
	{
		name: 'placeholder',
		type: 'string',
		description: 'Placeholder text',
		default: ''
	},
	{
		name: 'label',
		type: 'string',
		description: 'Visible label above input',
		default: undefined
	},
	{
		name: 'helperText',
		type: 'string',
		description: 'Helper text below input',
		default: undefined
	},
	{
		name: 'error',
		type: 'string',
		description: 'Error message (shows error state)',
		default: undefined
	},
	{
		name: 'size',
		type: 'select',
		options: ['sm', 'md', 'lg'],
		description: 'Size preset',
		default: 'md'
	},
	{
		name: 'variant',
		type: 'select',
		options: ['default', 'filled', 'outlined', 'ghost'],
		description: 'Visual variant',
		default: 'default'
	},
	{
		name: 'disabled',
		type: 'boolean',
		description: 'Disabled state',
		default: false
	},
	{
		name: 'readonly',
		type: 'boolean',
		description: 'Read-only state',
		default: false
	},
	{
		name: 'required',
		type: 'boolean',
		description: 'Required field',
		default: false
	},
	{
		name: 'leftIcon',
		type: 'string',
		description: 'Icon on left side',
		default: undefined
	},
	{
		name: 'rightIcon',
		type: 'string',
		description: 'Icon on right side',
		default: undefined
	},
	{
		name: 'clearable',
		type: 'boolean',
		description: 'Show clear button',
		default: false
	},
	{
		name: 'showCount',
		type: 'boolean',
		description: 'Show character count',
		default: false
	},
	{
		name: 'maxlength',
		type: 'number',
		description: 'Maximum characters',
		default: undefined
	}
];

const variants: IVariant[] = [
	// Basic variants
	{ name: 'Default', description: 'Standard input', props: { placeholder: 'Enter text...' } },
	{
		name: 'With Label',
		description: 'Labeled input',
		props: { label: 'Username', placeholder: 'Enter username' }
	},
	{
		name: 'With Helper',
		description: 'Input with helper text',
		props: { label: 'Email', placeholder: 'you@example.com', helperText: 'We will never share your email' }
	},

	// States
	{
		name: 'Error State',
		description: 'Validation error',
		props: { label: 'Password', error: 'Password must be at least 8 characters' }
	},
	{ name: 'Disabled', description: 'Disabled input', props: { placeholder: 'Disabled...', disabled: true } },
	{
		name: 'Read-only',
		description: 'Read-only input',
		props: { value: 'Cannot edit this', readonly: true }
	},
	{ name: 'Required', description: 'Required field', props: { label: 'Full Name', required: true } },

	// With icons
	{
		name: 'Search Input',
		description: 'Search with icon',
		props: { leftIcon: '🔍', placeholder: 'Search...', clearable: true }
	},
	{
		name: 'Password Input',
		description: 'Password field',
		props: { type: 'password', label: 'Password', rightIcon: '👁️' }
	},
	{
		name: 'Email Input',
		description: 'Email with icon',
		props: { type: 'email', leftIcon: '✉️', placeholder: 'your@email.com' }
	},

	// Sizes
	{ name: 'Small', description: 'Compact input', props: { size: 'sm', placeholder: 'Small...' } },
	{ name: 'Large', description: 'Large input', props: { size: 'lg', placeholder: 'Large...' } },

	// Variants
	{ name: 'Filled', description: 'Filled style', props: { variant: 'filled', placeholder: 'Filled...' } },
	{
		name: 'Outlined',
		description: 'Outlined style',
		props: { variant: 'outlined', placeholder: 'Outlined...' }
	},
	{ name: 'Ghost', description: 'Ghost style', props: { variant: 'ghost', placeholder: 'Ghost...' } },

	// Special
	{
		name: 'With Count',
		description: 'Character counter',
		props: { label: 'Bio', maxlength: 140, showCount: true, placeholder: 'Write something...' }
	},
	{
		name: 'Clearable',
		description: 'With clear button',
		props: { value: 'Clear me', clearable: true }
	}
];

const presets: IPreset[] = [
	{
		name: 'Search Box',
		description: 'Global search input',
		props: { leftIcon: '🔍', placeholder: 'Search...', clearable: true, size: 'md' }
	},
	{
		name: 'Login Username',
		description: 'Login form username',
		props: { label: 'Username', placeholder: 'Enter username', required: true }
	},
	{
		name: 'Login Password',
		description: 'Login form password',
		props: { label: 'Password', type: 'password', required: true }
	},
	{
		name: 'Chat Input',
		description: 'Chat message input',
		props: { placeholder: 'Type a message...', variant: 'filled', size: 'md' }
	},
	{
		name: 'Bio Field',
		description: 'Profile bio with counter',
		props: { label: 'Bio', maxlength: 160, showCount: true, helperText: 'Brief description' }
	}
];

const presenter: IPresenter = {
	name: 'Input',
	slug: 'input',
	category: 'primitives',
	presenterType: 'component',
	component: Input,

	description:
		'A flexible text input with support for labels, validation, icons, and various visual states.',

	tags: ['input', 'text', 'form', 'field', 'search'],
	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/primitives/Input.svelte',
		repo: 'hub'
	},

	props,
	variants,
	presets,

	preview: {
		width: 280,
		height: 80,
		background: 'dark',
		defaultProps: { placeholder: 'Enter text...', label: 'Label' }
	},

	documentation: `
## Input Component

A versatile text input component for forms and user data entry.

### Usage

\`\`\`svelte
<script>
  import Input from './Input.svelte';
  let value = $state('');
</script>

<Input bind:value placeholder="Enter text..." />
<Input label="Email" type="email" required />
<Input leftIcon="🔍" placeholder="Search..." clearable />
\`\`\`

### Features

- **Labels & Helper Text**: Built-in label and helper text support
- **Validation**: Error state with accessible error messages
- **Icons**: Left and right icon slots
- **Clearable**: Optional clear button
- **Character Count**: Built-in character counting

### Accessibility

- Proper label association via \`for\` attribute
- ARIA attributes for errors
- Required field indicator
`
};

export default presenter;

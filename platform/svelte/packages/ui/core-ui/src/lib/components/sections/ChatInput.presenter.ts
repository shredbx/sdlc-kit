/**
 * ChatInput Presenter - Auto-registration for component library
 *
 * Defines how ChatInput is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import ChatInput from './ChatInput.svelte';

/**
 * ChatInput property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'value',
		type: 'string',
		description: 'Current input value',
		default: ''
	},
	{
		name: 'placeholder',
		type: 'string',
		description: 'Placeholder text',
		default: 'Type a message...'
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
		description: 'Loading/sending state',
		default: false
	},
	{
		name: 'maxlength',
		type: 'number',
		description: 'Maximum characters',
		default: undefined
	},
	{
		name: 'multiline',
		type: 'boolean',
		description: 'Allow multi-line input',
		default: false
	},
	{
		name: 'maxRows',
		type: 'number',
		description: 'Maximum rows for multiline',
		default: 4
	},
	{
		name: 'showCount',
		type: 'boolean',
		description: 'Show character count',
		default: false
	},
	{
		name: 'showAttachment',
		type: 'boolean',
		description: 'Show attachment button',
		default: false
	},
	{
		name: 'showVoice',
		type: 'boolean',
		description: 'Show voice input button',
		default: false
	}
];

/**
 * ChatInput variants for different states/configurations.
 */
const variants: IVariant[] = [
	// Basic states
	{
		name: 'Default',
		description: 'Standard chat input',
		props: {
			placeholder: 'Type a message...'
		}
	},
	{
		name: 'With Value',
		description: 'Input with pre-filled text',
		props: {
			value: 'Hello, how are you?',
			placeholder: 'Type a message...'
		}
	},
	{
		name: 'Disabled',
		description: 'Disabled input state',
		props: {
			placeholder: 'Chat disabled',
			disabled: true
		}
	},
	{
		name: 'Loading',
		description: 'Sending message state',
		props: {
			value: 'Sending this message...',
			loading: true
		}
	},

	// Features
	{
		name: 'Multiline',
		description: 'Expandable text area',
		props: {
			placeholder: 'Type a longer message...',
			multiline: true,
			maxRows: 4
		}
	},
	{
		name: 'With Character Count',
		description: 'Shows character limit',
		props: {
			placeholder: 'Type a message...',
			maxlength: 280,
			showCount: true
		}
	},
	{
		name: 'With Attachment',
		description: 'Attachment button enabled',
		props: {
			placeholder: 'Type a message...',
			showAttachment: true
		}
	},
	{
		name: 'With Voice',
		description: 'Voice input button enabled',
		props: {
			placeholder: 'Type or speak...',
			showVoice: true
		}
	},
	{
		name: 'Full Featured',
		description: 'All features enabled',
		props: {
			placeholder: 'Type, attach, or speak...',
			multiline: true,
			showAttachment: true,
			showVoice: true,
			maxlength: 1000,
			showCount: true
		}
	},

	// Character limit states
	{
		name: 'Near Limit',
		description: 'Approaching character limit',
		props: {
			value: 'This message is getting close to the character limit and will show warning',
			maxlength: 80,
			showCount: true
		}
	},
	{
		name: 'Over Limit',
		description: 'Exceeded character limit',
		props: {
			value: 'This message has exceeded the maximum character limit and shows an error state',
			maxlength: 50,
			showCount: true
		}
	}
];

/**
 * ChatInput presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'Simple Chat',
		description: 'Basic chat input',
		props: {
			placeholder: 'Type a message...'
		}
	},
	{
		name: 'AI Assistant',
		description: 'AI chat with multiline',
		props: {
			placeholder: 'Ask anything...',
			multiline: true,
			maxRows: 6
		}
	},
	{
		name: 'Social Media',
		description: 'Tweet-style with limit',
		props: {
			placeholder: "What's happening?",
			maxlength: 280,
			showCount: true
		}
	},
	{
		name: 'Rich Input',
		description: 'Full-featured chat input',
		props: {
			placeholder: 'Type, attach files, or use voice...',
			multiline: true,
			showAttachment: true,
			showVoice: true
		}
	},
	{
		name: 'Feedback Form',
		description: 'Multiline feedback input',
		props: {
			placeholder: 'Share your feedback...',
			multiline: true,
			maxRows: 8,
			maxlength: 2000,
			showCount: true
		}
	}
];

/**
 * ChatInput Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'ChatInput',
	slug: 'chat-input',
	category: 'blocks',
	presenterType: 'component',

	// Component reference
	component: ChatInput,

	// Optional - enrichment
	description:
		'Text input with send button for chat interfaces. Supports multi-line input, character limits, attachment and voice buttons, and loading states.',

	tags: ['chat', 'input', 'message', 'form', 'textarea'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/sections/ChatInput.svelte',
		repo: 'core-ui'
	},

	props,
	variants,
	presets,

	preview: {
		width: 320,
		height: 60,
		background: 'dark',
		defaultProps: {
			placeholder: 'Type a message...'
		}
	},

	documentation: `
## ChatInput Component

Text input for chat interfaces with send functionality.

### Usage

\`\`\`svelte
<script>
  import ChatInput from '@sbx/core-ui/components/sections/ChatInput.svelte';

  function handleSend(message: string) {
    console.log('Sending:', message);
  }
</script>

<ChatInput onsubmit={handleSend} placeholder="Ask anything..." />
\`\`\`

### Multiline Input

Enable auto-expanding textarea:

\`\`\`svelte
<ChatInput multiline maxRows={6} />
\`\`\`

### Character Limit

Show character count with limit:

\`\`\`svelte
<ChatInput maxlength={280} showCount />
\`\`\`

### Additional Actions

Add attachment and voice buttons:

\`\`\`svelte
<ChatInput
  showAttachment
  showVoice
  onattach={() => openFilePicker()}
  onvoice={() => startVoiceInput()}
/>
\`\`\`

### Keyboard Shortcuts

- \`Enter\` - Submit message
- \`Shift+Enter\` - New line (multiline mode)
`
};

export default presenter;

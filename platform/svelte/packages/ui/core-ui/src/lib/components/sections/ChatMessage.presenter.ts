/**
 * ChatMessage Presenter - Auto-registration for component library
 *
 * Defines how ChatMessage is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import ChatMessage from './ChatMessage.svelte';

/**
 * ChatMessage property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'role',
		type: 'select',
		options: ['user', 'assistant', 'system'],
		description: 'Message role/sender type',
		default: 'assistant'
	},
	{
		name: 'content',
		type: 'string',
		description: 'Message content',
		default: 'Hello!'
	},
	{
		name: 'sender',
		type: 'string',
		description: 'Sender name override',
		default: undefined
	},
	{
		name: 'avatar',
		type: 'string',
		description: 'Avatar image URL',
		default: undefined
	},
	{
		name: 'timestamp',
		type: 'string',
		description: 'Timestamp display',
		default: undefined
	},
	{
		name: 'status',
		type: 'select',
		options: ['sending', 'sent', 'delivered', 'read', 'error'],
		description: 'Message status (user messages)',
		default: undefined
	},
	{
		name: 'thinking',
		type: 'boolean',
		description: 'Show thinking indicator',
		default: false
	},
	{
		name: 'streaming',
		type: 'boolean',
		description: 'Streaming/typing animation',
		default: false
	},
	{
		name: 'compact',
		type: 'boolean',
		description: 'Compact display mode',
		default: false
	},
	{
		name: 'showActions',
		type: 'boolean',
		description: 'Show action buttons on hover',
		default: true
	}
];

/**
 * ChatMessage variants for different states/configurations.
 */
const variants: IVariant[] = [
	// Role variants
	{
		name: 'Assistant Message',
		description: 'AI assistant response',
		props: {
			role: 'assistant',
			content: 'How can I help you today?',
			timestamp: '2:30 PM'
		}
	},
	{
		name: 'User Message',
		description: 'User input message',
		props: {
			role: 'user',
			content: "What's the status of my project?",
			timestamp: '2:31 PM',
			status: 'read'
		}
	},
	{
		name: 'System Message',
		description: 'System notification',
		props: {
			role: 'system',
			content: 'Chat session started'
		}
	},

	// Status variants
	{
		name: 'Sending',
		description: 'Message being sent',
		props: {
			role: 'user',
			content: 'This message is sending...',
			status: 'sending'
		}
	},
	{
		name: 'Sent',
		description: 'Message sent',
		props: {
			role: 'user',
			content: 'This message was sent',
			status: 'sent'
		}
	},
	{
		name: 'Delivered',
		description: 'Message delivered',
		props: {
			role: 'user',
			content: 'This message was delivered',
			status: 'delivered'
		}
	},
	{
		name: 'Read',
		description: 'Message read',
		props: {
			role: 'user',
			content: 'This message was read',
			status: 'read'
		}
	},
	{
		name: 'Error',
		description: 'Failed to send',
		props: {
			role: 'user',
			content: 'This message failed',
			status: 'error'
		}
	},

	// Special states
	{
		name: 'Thinking',
		description: 'AI thinking indicator',
		props: {
			role: 'assistant',
			content: '',
			thinking: true
		}
	},
	{
		name: 'Streaming',
		description: 'Live streaming response',
		props: {
			role: 'assistant',
			content: 'I am currently generating this response',
			streaming: true
		}
	},
	{
		name: 'With Error',
		description: 'Message with error state',
		props: {
			role: 'assistant',
			content: '',
			error: 'Failed to generate response'
		}
	},

	// Display modes
	{
		name: 'Compact',
		description: 'Dense display mode',
		props: {
			role: 'assistant',
			content: 'Compact message without avatar',
			compact: true
		}
	},
	{
		name: 'Long Message',
		description: 'Multi-line content',
		props: {
			role: 'assistant',
			content:
				'This is a longer message that spans multiple lines.\n\nIt includes paragraph breaks and demonstrates how the component handles extended content gracefully.',
			timestamp: '2:35 PM'
		}
	},

	// Markdown content
	{
		name: 'Markdown',
		description: 'Assistant message rendered as markdown (lists, links, emphasis)',
		props: {
			role: 'assistant',
			content:
				'Here are a few options:\n\n- **Buy** an off-plan villa\n- *Lease* a condo near the beach\n- [Book a viewing](https://example.com)\n\nLet me know which interests you.',
			timestamp: '2:37 PM'
		}
	},

	// Custom sender
	{
		name: 'Custom Sender',
		description: 'Named sender override',
		props: {
			role: 'assistant',
			content: 'Hello from Claude!',
			sender: 'Claude',
			timestamp: '2:36 PM'
		}
	}
];

/**
 * ChatMessage presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'AI Welcome',
		description: 'Initial AI greeting',
		props: {
			role: 'assistant',
			content: 'Hi! How can I help you with the workspace today?',
			sender: 'Assistant'
		}
	},
	{
		name: 'User Query',
		description: 'Typical user question',
		props: {
			role: 'user',
			content: "What's the status of my recent builds?",
			status: 'read'
		}
	},
	{
		name: 'AI Thinking',
		description: 'Processing indicator',
		props: {
			role: 'assistant',
			content: '',
			thinking: true
		}
	},
	{
		name: 'System Notice',
		description: 'System notification message',
		props: {
			role: 'system',
			content: 'Session started • Connected to workspace'
		}
	},
	{
		name: 'Error Response',
		description: 'AI error with retry',
		props: {
			role: 'assistant',
			content: '',
			error: 'Unable to process request. Please try again.'
		}
	}
];

/**
 * ChatMessage Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'ChatMessage',
	slug: 'chat-message',
	category: 'blocks',
	presenterType: 'component',

	// Component reference
	component: ChatMessage,

	// Optional - enrichment
	description:
		'Individual message in a chat conversation with role-based styling. Supports user, assistant, and system messages with status indicators, thinking animation, markdown rendering, and streaming display.',

	tags: ['chat', 'message', 'conversation', 'ai', 'assistant'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/sections/ChatMessage.svelte',
		repo: 'core-ui'
	},

	props,
	variants,
	presets,

	preview: {
		width: 320,
		height: 100,
		background: 'dark',
		defaultProps: {
			role: 'assistant',
			content: 'How can I help you today?'
		}
	},

	documentation: `
## ChatMessage Component

Individual message bubble for chat interfaces.

### Usage

\`\`\`svelte
<script>
  import ChatMessage from '@sbx/core-ui/components/sections/ChatMessage.svelte';
</script>

<ChatMessage role="assistant" content="Hello!" />
<ChatMessage role="user" content="Hi there!" status="read" />
\`\`\`

### Message Roles

- \`assistant\` - AI/bot responses (left-aligned, neutral bg)
- \`user\` - User messages (right-aligned, accent bg)
- \`system\` - System notifications (centered, subtle)

### Status Indicators

User messages show delivery status:
- \`sending\` - Clock icon
- \`sent\` - Single check
- \`delivered\` - Double check
- \`read\` - Blue double check
- \`error\` - Error icon

### Markdown

Assistant and system messages render their content as markdown (text, lists,
links, images) via \`@humanspeak/svelte-markdown\`. User messages and empty
content stay plain text (literal input + the streaming cursor).

### Thinking State

Show AI thinking with animated dots:

\`\`\`svelte
<ChatMessage role="assistant" thinking />
\`\`\`

### Streaming

Show cursor animation during live generation:

\`\`\`svelte
<ChatMessage role="assistant" content={partialContent} streaming />
\`\`\`
`
};

export default presenter;

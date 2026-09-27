/**
 * ChatWidget Presenter - Auto-registration for component library
 *
 * Defines how ChatWidget is documented and previewed in /components.
 * Follows IPresenter interface for auto-discovery.
 */

import type { IPresenter, IPropDef, IVariant, IPreset } from '../../presenter/IPresenter';
import ChatWidget from './ChatWidget.svelte';

/**
 * Sample messages for preview
 */
const sampleMessages = [
	{
		id: '1',
		role: 'assistant' as const,
		content: 'Hello! How can I help you today?',
		timestamp: '2:30 PM'
	},
	{
		id: '2',
		role: 'user' as const,
		content: 'I have a question about my account.',
		timestamp: '2:31 PM'
	},
	{
		id: '3',
		role: 'assistant' as const,
		content: "Of course! I'd be happy to help with your account. What would you like to know?",
		timestamp: '2:31 PM'
	}
];

/**
 * ChatWidget property definitions for real-time editing.
 */
const props: IPropDef[] = [
	{
		name: 'messages',
		type: 'array',
		description: 'Array of message objects with id, role, content, timestamp',
		default: []
	},
	{
		name: 'placeholder',
		type: 'string',
		description: 'Input placeholder text',
		default: 'Type a message...'
	},
	{
		name: 'expanded',
		type: 'boolean',
		description: 'Whether the chat panel is expanded',
		default: false
	},
	{
		name: 'title',
		type: 'string',
		description: 'Chat panel title',
		default: 'Chat'
	},
	{
		name: 'subtitle',
		type: 'string',
		description: 'Chat panel subtitle',
		default: undefined
	},
	{
		name: 'typing',
		type: 'boolean',
		description: 'Show typing indicator',
		default: false
	},
	{
		name: 'disabled',
		type: 'boolean',
		description: 'Disable input',
		default: false
	},
	{
		name: 'position',
		type: 'select',
		options: ['bottom-right', 'bottom-left'],
		description: 'Widget position on screen',
		default: 'bottom-right'
	},
	{
		name: 'unreadCount',
		type: 'number',
		description: 'Unread message count shown on FAB badge',
		default: undefined
	}
];

/**
 * ChatWidget variants for different states/configurations.
 */
const variants: IVariant[] = [
	{
		name: 'Collapsed (FAB)',
		description: 'Default collapsed state showing only FAB',
		props: {
			expanded: false,
			title: 'Chat'
		}
	},
	{
		name: 'Collapsed with Badge',
		description: 'FAB with unread message badge',
		props: {
			expanded: false,
			unreadCount: 3
		}
	},
	{
		name: 'Expanded Empty',
		description: 'Expanded panel with no messages',
		props: {
			expanded: true,
			title: 'Chat',
			subtitle: 'We typically reply within minutes'
		}
	},
	{
		name: 'Expanded with Messages',
		description: 'Expanded panel with conversation',
		props: {
			expanded: true,
			title: 'Support',
			subtitle: 'Online',
			messages: sampleMessages
		}
	},
	{
		name: 'Typing Indicator',
		description: 'Showing assistant is typing',
		props: {
			expanded: true,
			title: 'AI Assistant',
			messages: sampleMessages.slice(0, 2),
			typing: true
		}
	},
	{
		name: 'Bottom Left Position',
		description: 'Widget positioned on bottom-left',
		props: {
			expanded: false,
			position: 'bottom-left'
		}
	},
	{
		name: 'Disabled Input',
		description: 'Chat input is disabled',
		props: {
			expanded: true,
			title: 'Chat',
			messages: sampleMessages,
			disabled: true
		}
	}
];

/**
 * ChatWidget presets - saved prop combinations for common uses.
 */
const presets: IPreset[] = [
	{
		name: 'Customer Support',
		description: 'Customer support chat widget',
		props: {
			title: 'Support',
			subtitle: 'We typically reply within minutes',
			placeholder: 'Type your question...'
		}
	},
	{
		name: 'AI Assistant',
		description: 'AI-powered assistant widget',
		props: {
			title: 'AI Assistant',
			subtitle: 'Powered by AI',
			placeholder: 'Ask me anything...'
		}
	},
	{
		name: 'Sales Chat',
		description: 'Sales inquiry widget',
		props: {
			title: 'Sales Team',
			subtitle: 'Online now',
			placeholder: 'Have a question about pricing?'
		}
	},
	{
		name: 'Feedback Widget',
		description: 'Collect user feedback',
		props: {
			title: 'Feedback',
			subtitle: 'Help us improve',
			placeholder: 'Share your thoughts...'
		}
	}
];

/**
 * ChatWidget Presenter Export
 */
const presenter: IPresenter = {
	// Required - minimal conformance
	name: 'ChatWidget',
	slug: 'chat-widget',
	category: 'blocks',
	presenterType: 'component',

	// Component reference
	component: ChatWidget,

	// Optional - enrichment
	description:
		'Floating chat FAB with expandable chat panel. Intercom/Drift style widget for customer support, AI assistants, or live chat. Features glassmorphism design, smooth animations, and responsive behavior.',

	tags: ['chat', 'widget', 'fab', 'floating', 'support', 'messenger', 'intercom'],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'src/lib/components/sections/ChatWidget.svelte',
		repo: 'core-ui'
	},

	props,
	variants,
	presets,

	preview: {
		width: 400,
		height: 600,
		background: 'dark',
		defaultProps: {
			expanded: true,
			title: 'Chat',
			messages: sampleMessages
		}
	},

	documentation: `
## ChatWidget Component

Floating chat FAB with expandable chat panel (Intercom/Drift style).

### Usage

\`\`\`svelte
<script>
  import ChatWidget from '@sbx/core-ui/components/sections/ChatWidget.svelte';
  import type { Message } from './ChatPanel.svelte';

  let messages: Message[] = $state([]);

  function handleSend(content: string) {
    // Add user message
    messages = [...messages, {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      timestamp: new Date().toLocaleTimeString()
    }];

    // Send to backend, receive response...
  }
</script>

<ChatWidget
  {messages}
  title="Support"
  subtitle="We typically reply within minutes"
  onsend={handleSend}
/>
\`\`\`

### Controlled Expanded State

Use \`bind:expanded\` to control the widget state:

\`\`\`svelte
<script>
  let isOpen = $state(false);
</script>

<button onclick={() => isOpen = true}>Open Chat</button>
<ChatWidget bind:expanded={isOpen} />
\`\`\`

### Unread Badge

Show unread count on the FAB:

\`\`\`svelte
<ChatWidget unreadCount={3} />
\`\`\`

### Position

Position the widget on bottom-left:

\`\`\`svelte
<ChatWidget position="bottom-left" />
\`\`\`

### Design Features

- **FAB**: 56px circle button with message-circle icon
- **Panel**: 360px x 480px glassmorphism panel
- **Animation**: Scale up from FAB position
- **Responsive**: Adapts to mobile viewports
- **Accessibility**: Proper ARIA labels and keyboard support
`
};

export default presenter;

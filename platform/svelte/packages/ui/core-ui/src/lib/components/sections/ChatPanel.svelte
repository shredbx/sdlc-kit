<script lang="ts">
	/**
	 * ChatPanel Section Component
	 *
	 * Complete chat interface section with message history, input, and header.
	 * Supports floating (popover) and inline modes.
	 *
	 * @layer section
	 * @uses ChatMessage (block)
	 * @uses ChatInput (block)
	 * @uses Icon (primitive)
	 * @uses IconButton (primitive)
	 *
	 * @example
	 * <ChatPanel
	 *   messages={conversationHistory}
	 *   title="AI Assistant"
	 *   onsend={(msg) => sendToAssistant(msg)}
	 * />
	 */

	import { tick, type Snippet } from 'svelte';
	import SvelteMarkdown from '@humanspeak/svelte-markdown';
	import ChatMessage from './ChatMessage.svelte';
	import ChatInput from './ChatInput.svelte';
	import Icon from '../primitives/Icon.svelte';
	import IconButton from '../primitives/IconButton.svelte';
	import Avatar from '../primitives/Avatar.svelte';

	// Message data type
	export interface Message {
		id: string;
		role: 'user' | 'assistant' | 'system';
		content: string;
		timestamp?: string;
		status?: 'sending' | 'sent' | 'delivered' | 'read' | 'error';
		error?: string;
		/** Rich, structured parts (Meta-compatible) the assistant may attach. Rendered
		 *  when the live engine emits them; markdown `content` already covers text,
		 *  images, and links. */
		quickReplies?: ChatQuickReply[];
		cards?: ChatCard[];
	}

	// A suggested-prompt card shown in the welcome (empty) state. Clicking one sends
	// its `prompt` through onsend — a typical AI-chat landing affordance.
	export interface ChatSuggestion {
		/** Stable unique key for list rendering (e.g. the source action's id). Two
		 *  suggestions may share a title/category, so the title is NOT a safe key. */
		id: string;
		/** Small label above the title (e.g. "Popular", "Buying"). */
		category?: string;
		/** The card's headline — what the user sees. */
		title: string;
		/** The message actually sent when the card is clicked. */
		prompt: string;
	}

	// ── Rich message protocol (Meta-compatible superset) ──────────────────────────────
	// The runtime wire shape the assistant emits and external connectors (Messenger /
	// WhatsApp) project to and from. A button is the ONE abstraction both platforms share
	// exactly and which round-trips (a postback payload ↔ a reply id): a link, a postback
	// intent, or a call. The lowest-common-denominator limits — ≤20 label · ≤256 payload ·
	// ≤3 buttons · ≤24 title · ≤72 subtitle — keep a card renderable on every channel.
	// Types only for now; rendering arrives with the live engine that emits them.
	export type ChatButton =
		| { kind: 'link'; label: string; url: string }
		| { kind: 'postback'; label: string; payload: string }
		| { kind: 'call'; label: string; phone: string };

	/** An inline quick reply attached to a message (≤3 shown; label ≤20, payload ≤256). */
	export interface ChatQuickReply {
		label: string;
		payload: string;
	}

	/** A rich card: image + title + subtitle + up to 3 buttons, optionally tappable as a
	 *  whole (defaultAction). A carousel is ChatCard[]. */
	export interface ChatCard {
		image?: { url: string; alt?: string };
		title: string;
		subtitle?: string;
		buttons?: ChatButton[];
		defaultAction?: { url: string };
	}

	type ChatPanelVariant = 'floating' | 'inline' | 'fullscreen';

	interface ChatPanelProps {
		/** Message history */
		messages?: Message[];
		/** Panel title */
		title?: string;
		/** Panel subtitle */
		subtitle?: string;
		/** Assistant avatar */
		assistantAvatar?: string;
		/** User avatar */
		userAvatar?: string;
		/** Panel variant */
		variant?: ChatPanelVariant;
		/** Open state (for floating variant) */
		open?: boolean;
		/** Close handler */
		onclose?: () => void;
		/** Send message handler */
		onsend?: (message: string) => void;
		/** Retry handler for failed messages */
		onretry?: (messageId: string) => void;
		/** Clear history handler */
		onclear?: () => void;
		/** Input placeholder */
		placeholder?: string;
		/** Show typing indicator */
		typing?: boolean;
		/** Turn in flight (send → stream settled) — locks the input so a mid-stream send cannot
		 *  kill the live turn. Distinct from `typing`, which clears on the first token. */
		busy?: boolean;
		/** Disable input */
		disabled?: boolean;
		/** Show attachment button */
		showAttachment?: boolean;
		/** Attachment handler */
		onattach?: () => void;
		/** Show voice input */
		showVoice?: boolean;
		/** Voice input handler */
		onvoice?: () => void;
		/** Maximum height (for inline/floating) */
		maxHeight?: string;
		/** Show header */
		showHeader?: boolean;
		/** Compact mode */
		compact?: boolean;
		/**
		 * Suggested-prompt cards for the welcome state. When provided AND the message
		 * list is empty, they render as clickable cards; clicking one sends its prompt.
		 */
		suggestions?: ChatSuggestion[];
		/**
		 * Optional welcome/greeting (markdown) shown in the empty state, above the
		 * suggestions — replaces the default "Start a conversation" copy when provided.
		 */
		welcomeMessage?: string;
		/**
		 * Optional toolbar region rendered in the footer, directly ABOVE the input
		 * (e.g. a model selector). The shared panel stays unopinionated — the consumer
		 * fills this slot.
		 */
		toolbar?: Snippet;
		/**
		 * Extra hosts allowed for images in rendered assistant/system markdown, forwarded
		 * to each ChatMessage. Assistant text is LLM output: by default only same-origin
		 * images load — pass a media/CDN host here (e.g. a property-image host) so those
		 * images aren't dropped by the markdown sanitizer. No effect on links. Default: [].
		 */
		allowedImageHosts?: string[];
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		messages = [],
		title = 'Chat',
		subtitle,
		assistantAvatar,
		userAvatar,
		variant = 'inline',
		open = true,
		onclose,
		onsend,
		onretry,
		onclear,
		placeholder = 'Type a message...',
		typing = false,
		busy = false,
		disabled = false,
		showAttachment = false,
		onattach,
		showVoice = false,
		onvoice,
		maxHeight = '400px',
		showHeader = true,
		compact = false,
		suggestions,
		welcomeMessage,
		toolbar,
		allowedImageHosts = [],
		class: className = '',
		'data-view-id': viewId,
		...restProps
	}: ChatPanelProps = $props();

	// svelte-ignore non_reactive_update
	let messagesContainer: HTMLDivElement;
	let inputValue = $state('');

	// Auto-scroll to bottom on new messages
	$effect(() => {
		if (messages.length > 0 && messagesContainer) {
			tick().then(() => {
				messagesContainer.scrollTop = messagesContainer.scrollHeight;
			});
		}
	});

	function handleSend(message: string) {
		onsend?.(message);
	}

	function handleRetry(messageId: string) {
		onretry?.(messageId);
	}

	function handleSuggestion(suggestion: ChatSuggestion) {
		if (disabled) return;
		onsend?.(suggestion.prompt);
	}

	let isEmpty = $derived(messages.length === 0);
	let showClearButton = $derived(messages.length > 1 && onclear);
	let hasSuggestions = $derived(!!suggestions && suggestions.length > 0);
</script>

{#if open}
	<div
		class="chat-panel chat-{variant} {className}"
		class:compact
		style:--chat-max-height={maxHeight}
		data-view-id={viewId}
		{...restProps}
	>
		<!-- Header -->
		{#if showHeader}
			<header class="chat-header">
				<div class="header-info">
					{#if assistantAvatar}
						<Avatar src={assistantAvatar} size="sm" status="online" />
					{:else}
						<div class="header-icon">
							<Icon name="bot" size="md" />
						</div>
					{/if}
					<div class="header-text">
						<span class="header-title">{title}</span>
						{#if subtitle}
							<span class="header-subtitle">{subtitle}</span>
						{/if}
					</div>
				</div>

				<div class="header-actions">
					{#if showClearButton}
						<IconButton
							icon="trash-2"
							variant="ghost"
							size="sm"
							onclick={onclear}
							title="Clear history"
						/>
					{/if}
					{#if onclose && variant === 'floating'}
						<IconButton
							icon="x"
							variant="ghost"
							size="sm"
							onclick={onclose}
							title="Close chat"
						/>
					{/if}
				</div>
			</header>
		{/if}

		<!-- Messages area -->
		<div class="chat-messages" bind:this={messagesContainer}>
			{#if isEmpty}
				<!-- Welcome state -->
				<div class="welcome-state">
					<div class="welcome-icon">
						<Icon name="message-circle" size="xl" />
					</div>
					{#if welcomeMessage}
						<!-- Consumer-provided greeting (markdown — the assistant's opening line). -->
						<div class="welcome-message">
							<SvelteMarkdown source={welcomeMessage} />
						</div>
					{:else}
						<p class="welcome-title">Start a conversation</p>
						<p class="welcome-subtitle">Send a message to begin chatting</p>
					{/if}

					{#if hasSuggestions}
						<!-- Suggested-prompt cards — a chat-landing affordance; clicking one
						     sends its prompt straight through onsend. A labelled group of
						     buttons (each card is an action, not a list item). -->
						<div class="suggestions" role="group" aria-label="Suggested prompts">
							{#each suggestions ?? [] as suggestion (suggestion.id)}
								<button
									type="button"
									class="suggestion-card"
									onclick={() => handleSuggestion(suggestion)}
									{disabled}
								>
									{#if suggestion.category}
										<span class="suggestion-category">{suggestion.category}</span>
									{/if}
									<span class="suggestion-title">{suggestion.title}</span>
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{:else}
				<!-- Message list -->
				{#each messages as message (message.id)}
					<ChatMessage
						role={message.role}
						content={message.content}
						timestamp={message.timestamp}
						status={message.status}
						error={message.error}
						avatar={message.role === 'user' ? userAvatar : assistantAvatar}
						{compact}
						{allowedImageHosts}
						cards={message.cards}
						onretry={message.error ? () => handleRetry(message.id) : undefined}
						data-view-id={viewId ? `${viewId}-message-${message.id}` : undefined}
					/>
				{/each}

				<!-- Typing indicator -->
				{#if typing}
					<ChatMessage role="assistant" content="" thinking avatar={assistantAvatar} />
				{/if}
			{/if}
		</div>

		<!-- Input area -->
		<footer class="chat-footer">
			{#if toolbar}
				<!-- Consumer-filled toolbar (e.g. a model selector), directly above input. -->
				<div class="chat-toolbar">
					{@render toolbar()}
				</div>
			{/if}
			<ChatInput
				bind:value={inputValue}
				{placeholder}
				{disabled}
				loading={busy}
				{showAttachment}
				{onattach}
				{showVoice}
				{onvoice}
				onsubmit={handleSend}
				multiline
				maxRows={3}
				data-view-id={viewId ? `${viewId}-input` : undefined}
			/>
		</footer>
	</div>
{/if}

<style>
	.chat-panel {
		display: flex;
		flex-direction: column;
		background: var(--chat-panel-bg, var(--color-bg-secondary, #1a1a2e));
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: 0.75rem;
		overflow: hidden;
	}

	/* Variant: Floating. Width is themeable (--chat-floating-width) so a consumer can
	   size the popover as a proper chat window; the default keeps the historical 320px. */
	.chat-panel.chat-floating {
		position: absolute;
		bottom: 60px;
		right: 0;
		width: var(--chat-floating-width, 320px);
		max-height: var(--chat-max-height, 400px);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
		z-index: 100;
	}

	/* Variant: Inline */
	.chat-panel.chat-inline {
		height: 100%;
		max-height: var(--chat-max-height, 400px);
	}

	/* Variant: Fullscreen */
	.chat-panel.chat-fullscreen {
		position: fixed;
		inset: 0;
		max-height: none;
		border-radius: 0;
		z-index: 200;
	}

	/* Header */
	.chat-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: var(--chat-header-bg, var(--color-bg-tertiary, #252540));
		border-bottom: 1px solid var(--color-border, #2a2a4a);
		flex-shrink: 0;
	}

	.header-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.header-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		background: var(--color-accent, #6366f1);
		border-radius: 0.5rem;
		color: #fff;
	}

	.header-text {
		display: flex;
		flex-direction: column;
	}

	.header-title {
		font-weight: 600;
		font-size: 0.875rem;
		color: var(--color-text, #fff);
	}

	.header-subtitle {
		font-size: 0.75rem;
		color: var(--color-text-muted, #9ca3af);
	}

	.header-actions {
		display: flex;
		gap: 0.25rem;
	}

	/* Messages area */
	.chat-messages {
		flex: 1;
		overflow-y: auto;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		/* Wide panels (a full-width admin tab): the conversation reads as a proper
		   centered chat column instead of bubbles hugging one edge of a huge void.
		   Unset by default — narrow panels (the floating popover) are unaffected. */
		width: 100%;
		max-width: var(--chat-messages-max-width, none);
		margin-inline: auto;
	}

	/* Welcome state */
	.welcome-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
		padding: 2rem;
	}

	.welcome-icon {
		color: var(--color-text-muted, #6b7280);
		opacity: 0.5;
		margin-bottom: 1rem;
	}

	.welcome-title {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-text, #fff);
	}

	.welcome-subtitle {
		margin: 0.5rem 0 0;
		font-size: 0.875rem;
		color: var(--color-text-muted, #9ca3af);
	}

	/* Consumer welcome message (markdown). Centered like the default copy; the prose
	   tightens its paragraph margins so a short greeting reads as one block. */
	.welcome-message {
		max-width: 32rem;
		font-size: 0.9375rem;
		line-height: 1.5;
		color: var(--color-text, #fff);
	}
	.welcome-message :global(p) {
		margin: 0.5rem 0 0;
	}
	.welcome-message :global(p:first-child) {
		margin-top: 0;
	}

	/* Suggested-prompt cards */
	.suggestions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 0.75rem;
		width: 100%;
		max-width: 36rem;
		margin-top: 1.5rem;
	}

	.suggestion-card {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.375rem;
		padding: 0.875rem 1rem;
		text-align: left;
		background: var(--chat-suggestion-bg, var(--color-bg-tertiary, #252540));
		border: 1px solid var(--chat-suggestion-border, var(--color-border, #2a2a4a));
		border-radius: 0.75rem;
		cursor: pointer;
		transition:
			border-color 0.15s ease,
			background 0.15s ease,
			transform 0.15s ease;
	}

	.suggestion-card:hover:not(:disabled) {
		border-color: var(--chat-suggestion-border-hover, var(--color-accent, #6366f1));
		background: var(--chat-suggestion-bg-hover, var(--color-bg-hover, #2a2a4a));
	}

	.suggestion-card:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	.suggestion-card:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.suggestion-category {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--chat-suggestion-category, var(--color-accent, #6366f1));
	}

	.suggestion-title {
		font-size: 0.875rem;
		font-weight: 500;
		line-height: 1.35;
		color: var(--color-text, #fff);
	}

	@media (prefers-reduced-motion: reduce) {
		.suggestion-card {
			transition: none;
		}
	}

	/* Footer/Input area */
	.chat-footer {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 0.75rem;
		border-top: 1px solid var(--color-border, #2a2a4a);
		background: var(--chat-footer-bg, var(--color-bg-tertiary, #252540));
		flex-shrink: 0;
	}

	/* Toolbar row (sits above the input) */
	.chat-toolbar {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	/* Compact mode */
	.chat-panel.compact .chat-header {
		padding: 0.5rem 0.75rem;
	}

	.chat-panel.compact .chat-messages {
		padding: 0.75rem;
	}

	.chat-panel.compact .chat-footer {
		padding: 0.5rem;
	}

	/* Scrollbar styling */
	.chat-messages::-webkit-scrollbar {
		width: 6px;
	}

	.chat-messages::-webkit-scrollbar-track {
		background: transparent;
	}

	.chat-messages::-webkit-scrollbar-thumb {
		background: var(--color-border, #2a2a4a);
		border-radius: 3px;
	}

	.chat-messages::-webkit-scrollbar-thumb:hover {
		background: var(--color-text-muted, #6b7280);
	}
</style>

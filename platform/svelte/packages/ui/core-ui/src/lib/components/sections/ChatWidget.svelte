<script lang="ts">
	/**
	 * ChatWidget Block Component
	 *
	 * Floating chat FAB with expandable chat panel (Intercom/Drift style).
	 * Combines FAB trigger with ChatPanel for a complete chat widget experience.
	 *
	 * @layer block
	 * @uses ChatPanel (section)
	 * @uses IconButton (primitive)
	 *
	 * @example
	 * <ChatWidget
	 *   messages={chatHistory}
	 *   onsend={(msg) => sendMessage(msg)}
	 * />
	 */

	import ChatPanel, { type Message, type ChatSuggestion } from './ChatPanel.svelte';
	import IconButton from '../primitives/IconButton.svelte';

	interface ChatWidgetProps {
		/** Message history */
		messages?: Message[];
		/** Welcome message shown in the empty state, before any turn. */
		welcomeMessage?: string;
		/** Tappable starter openers shown under the welcome — each sends its prompt via onsend. */
		suggestions?: ChatSuggestion[];
		/** Hostnames whose images may render in assistant markdown + cards (XSS allow-list). */
		allowedImageHosts?: string[];
		/** Input placeholder text */
		placeholder?: string;
		/** Send message handler */
		onsend?: (message: string) => void;
		/** Controlled expanded state */
		expanded?: boolean;
		/** Panel title */
		title?: string;
		/** Panel subtitle */
		subtitle?: string;
		/** Show typing indicator */
		typing?: boolean;
		/** Turn in flight — locks the panel input for the whole turn (see ChatPanel.busy). */
		busy?: boolean;
		/** Disabled state */
		disabled?: boolean;
		/** Position: bottom-right or bottom-left */
		position?: 'bottom-right' | 'bottom-left';
		/** Assistant avatar URL */
		assistantAvatar?: string;
		/** User avatar URL */
		userAvatar?: string;
		/** Clear history handler */
		onclear?: () => void;
		/** Retry handler for failed messages */
		onretry?: (messageId: string) => void;
		/** Unread message count (shown on FAB badge) */
		unreadCount?: number;
		/** Max height of the expanded floating panel (any CSS length — a viewport-derived
		 *  value makes the popover a proper chat window). Default keeps the historical 480px. */
		panelMaxHeight?: string;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		messages = [],
		welcomeMessage,
		suggestions,
		allowedImageHosts,
		placeholder = 'Type a message...',
		onsend,
		expanded = $bindable(false),
		title = 'Chat',
		subtitle,
		typing = false,
		busy = false,
		disabled = false,
		position = 'bottom-right',
		assistantAvatar,
		userAvatar,
		onclear,
		onretry,
		unreadCount,
		panelMaxHeight = '480px',
		class: className = '',
		'data-view-id': viewId,
		...restProps
	}: ChatWidgetProps = $props();

	function toggleExpanded() {
		expanded = !expanded;
	}

	function handleClose() {
		expanded = false;
	}

	function handleSend(message: string) {
		onsend?.(message);
	}
</script>

<div
	class="chat-widget chat-widget-{position} {className}"
	class:expanded
	data-view-id={viewId}
	{...restProps}
>
	<!-- Chat Panel (expanded state) -->
	{#if expanded}
		<div class="chat-widget-panel" class:entering={expanded}>
			<ChatPanel
				{messages}
				{welcomeMessage}
				{suggestions}
				{allowedImageHosts}
				{title}
				{subtitle}
				{placeholder}
				{typing}
				{busy}
				{disabled}
				{assistantAvatar}
				{userAvatar}
				{onclear}
				{onretry}
				variant="floating"
				open={true}
				onclose={handleClose}
				onsend={handleSend}
				maxHeight={panelMaxHeight}
				data-view-id={viewId ? `${viewId}-panel` : undefined}
			/>
		</div>
	{/if}

	<!-- FAB Trigger -->
	<div class="chat-widget-fab" class:hidden={expanded}>
		<IconButton
			icon={expanded ? 'x' : 'message-circle'}
			label={expanded ? 'Close chat' : 'Open chat'}
			variant="primary"
			size="xl"
			shape="circle"
			onclick={toggleExpanded}
			badge={!expanded && unreadCount ? unreadCount : undefined}
			badgeVariant="error"
			data-view-id={viewId ? `${viewId}-fab` : undefined}
		/>
	</div>
</div>

<style>
	.chat-widget {
		position: fixed;
		z-index: 1000;
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 1rem;
	}

	/* Position variants */
	.chat-widget-bottom-right {
		bottom: 1rem;
		right: 1rem;
	}

	.chat-widget-bottom-left {
		bottom: 1rem;
		left: 1rem;
		align-items: flex-start;
	}

	/* FAB */
	.chat-widget-fab {
		transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.chat-widget-fab.hidden {
		transform: scale(0);
		opacity: 0;
		pointer-events: none;
	}

	.chat-widget-fab :global(.icon-button-primary) {
		width: 3.5rem;
		height: 3.5rem;
		box-shadow:
			0 8px 24px rgba(99, 102, 241, 0.4),
			0 4px 8px rgba(0, 0, 0, 0.2);
	}

	.chat-widget-fab :global(.icon-button-primary:hover) {
		transform: scale(1.1);
		box-shadow:
			0 12px 32px rgba(99, 102, 241, 0.5),
			0 6px 12px rgba(0, 0, 0, 0.25);
	}

	/* Panel container */
	.chat-widget-panel {
		width: 360px;
		max-width: calc(100vw - 2rem);
		animation: chat-widget-expand 0.3s cubic-bezier(0.4, 0, 0.2, 1);
		transform-origin: bottom right;
	}

	.chat-widget-bottom-left .chat-widget-panel {
		transform-origin: bottom left;
	}

	@keyframes chat-widget-expand {
		from {
			opacity: 0;
			transform: scale(0.8) translateY(1rem);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}

	/* Panel glassmorphism styling override. Background + border are TOKENIZED with the dark
	   glassmorphism as the fallback: a consumer that bridges the chat suite to a light theme
	   (--chat-panel-bg / --chat-input-border) gets its own surface, while a consumer that sets
	   nothing keeps the original dark-by-default look — no regression for existing users. */
	.chat-widget-panel :global(.chat-panel) {
		background: var(--chat-panel-bg, rgba(26, 26, 46, 0.95));
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border: 1px solid var(--chat-input-border, rgba(255, 255, 255, 0.1));
		border-radius: 1rem;
		box-shadow:
			0 24px 48px rgba(0, 0, 0, 0.4),
			0 8px 16px rgba(0, 0, 0, 0.2);
		overflow: hidden;
	}

	/* Mobile: a FULL-SCREEN chat sheet — the default UX for a chat widget on a phone. When
	   expanded, the widget covers the viewport edge-to-edge so the conversation gets the whole
	   screen instead of a cramped floating card. The panel's own header carries the close button
	   (onclose), so there is always a way out of the sheet; the FAB is hidden while expanded (the
	   rule above), so it never floats over it. 100dvh (dynamic viewport) keeps the input above a
	   mobile browser's collapsing URL bar. */
	@media (max-width: 640px) {
		/* Raise above page chrome (e.g. a cookie bar) so the sheet truly covers the screen. */
		.chat-widget.expanded {
			z-index: 2000;
		}
		/* Force the floating panel into a true full-screen sheet: fixed to the VIEWPORT (the
		   floating variant is position:absolute + max-height:var — override both), edge-to-edge,
		   square corners. The panel keeps its own header (with the close button) pinned at the top
		   and the input at the bottom, so a visitor always has a way out. */
		.chat-widget.expanded :global(.chat-panel.chat-floating) {
			position: fixed;
			inset: 0;
			width: 100vw;
			height: 100dvh;
			max-height: none;
			border-radius: 0;
		}
	}

	/* Ensure panel is above FAB */
	.chat-widget-panel {
		position: relative;
		z-index: 1;
	}

	.chat-widget-fab {
		position: relative;
		z-index: 0;
	}

	/* When expanded, hide FAB with animation */
	.chat-widget.expanded .chat-widget-fab {
		transform: scale(0);
		opacity: 0;
		pointer-events: none;
	}
</style>

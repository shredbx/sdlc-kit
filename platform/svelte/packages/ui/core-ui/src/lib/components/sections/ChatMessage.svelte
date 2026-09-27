<script lang="ts">
	/**
	 * ChatMessage Block Component
	 *
	 * Individual message in a chat conversation with role-based styling.
	 * Supports user, assistant, and system message types.
	 *
	 * @example
	 * <ChatMessage role="assistant" content="How can I help?" />
	 * <ChatMessage role="user" content="What's the status?" timestamp="2:30 PM" />
	 */

	import SvelteMarkdown from '@humanspeak/svelte-markdown';
	import Avatar from '../primitives/Avatar.svelte';
	import Icon from '../primitives/Icon.svelte';
	import { buildBlockedHtmlRenderers, makeImageSrcSanitizer } from './ChatMessage.security';
	import type { ChatButton, ChatCard } from './ChatPanel.svelte';

	type MessageRole = 'user' | 'assistant' | 'system';
	type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read' | 'error';

	interface ChatMessageProps {
		/** Message role/sender type */
		role: MessageRole;
		/** Message content */
		content: string;
		/** Sender name (optional) */
		sender?: string;
		/** Avatar URL */
		avatar?: string;
		/** Timestamp display */
		timestamp?: string;
		/** Message status (for user messages) */
		status?: MessageStatus;
		/** Show thinking/typing indicator */
		thinking?: boolean;
		/** Streaming content (partial) */
		streaming?: boolean;
		/** Error message */
		error?: string;
		/** Retry handler for error state */
		onretry?: () => void;
		/** Copy handler */
		oncopy?: () => void;
		/** Compact mode */
		compact?: boolean;
		/** Show actions on hover */
		showActions?: boolean;
		/**
		 * Extra hosts allowed for rendered image `src` in assistant/system markdown.
		 * Assistant text is LLM output: by default only same-origin (relative URLs + the
		 * current host) images load — an off-allowlist host (e.g. a tracking beacon) is
		 * dropped so no request is issued. Pass a media/CDN host here to permit it.
		 * Has no effect on links. Default: same-origin only.
		 */
		allowedImageHosts?: string[];
		/**
		 * Structured result cards the assistant attached to this message (e.g. property
		 * results). Rendered below the text as a stack of image + title + subtitle + up to
		 * 3 action buttons, with an optional whole-card link (defaultAction). URLs are
		 * expected already RESOLVED to full/relative hrefs by the consumer (the media host
		 * for images, the site for links) — this renderer only scheme-guards them.
		 */
		cards?: ChatCard[];
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		role,
		content,
		sender,
		avatar,
		timestamp,
		status,
		thinking = false,
		streaming = false,
		error,
		onretry,
		oncopy,
		compact = false,
		showActions = true,
		allowedImageHosts = [],
		cards = [],
		class: className = '',
		'data-view-id': viewId,
		...restProps
	}: ChatMessageProps = $props();

	// Role-based defaults
	const roleConfig = {
		user: { icon: 'user', name: 'You', align: 'right' },
		assistant: { icon: 'bot', name: 'Assistant', align: 'left' },
		system: { icon: 'info', name: 'System', align: 'center' }
	};

	let config = $derived(roleConfig[role]);
	let displayName = $derived(sender ?? config.name);

	// Markdown is rendered for assistant + system prose (text · lists · images · links,
	// with custom nodes later). User messages and any empty/streaming-from-empty content
	// keep the plain pre-wrap path: user input is literal (never interpret a stray `*`),
	// and an empty string has nothing to parse — so the streaming cursor shows on the
	// plain path until the first token lands, then markdown takes over as tokens stream in.
	let useMarkdown = $derived(role !== 'user' && content.trim().length > 0);

	// Security hardening for the markdown sink (assistant text is LLM output):
	//  - `mdRenderers` strips structural raw-HTML tags (iframe/embed/object) — they
	//    render live by default (off-site loads / clickjacking) — to escaped text.
	//  - `mdSanitizeUrl` keeps the library's default scheme allowlist (javascript:/data:
	//    stay blocked) AND adds an image-only host allowlist: same-origin always, plus
	//    `allowedImageHosts`; an off-allowlist image src collapses to '' so no request
	//    fires. The default `sanitizeAttributes` (strips on* handlers) is left in place.
	const mdRenderers = { html: buildBlockedHtmlRenderers() };
	let mdSanitizeUrl = $derived(makeImageSrcSanitizer(allowedImageHosts));

	async function handleCopy() {
		await navigator.clipboard.writeText(content);
		oncopy?.();
	}

	// Card URLs are structured engine output (DB-backed, resolved by the consumer to known
	// hosts) — not free-form LLM markdown. Still scheme-guard them so a crafted card can
	// never smuggle javascript:/data: into an href or img src (defence in depth): relative,
	// http(s), tel and mailto only; anything else collapses to '' (no link, no image request).
	function safeCardUrl(url: string | undefined): string {
		if (!url) return '';
		const u = url.trim();
		if (/^[/#?]/.test(u)) return u; // root-relative / fragment / query — same-origin
		if (!u.includes(':') && !u.startsWith('//')) return u; // path-relative (no scheme)
		if (/^https?:\/\//i.test(u)) return u;
		if (/^(tel|mailto):/i.test(u)) return u;
		return ''; // javascript:, data:, vbscript:, //protocol-relative, unknown → drop
	}

	// The href for a card action button. Only link + call carry a navigable href; a postback
	// (a payload the engine would act on) needs an onsend/onpostback handler this leaf
	// renderer does not own — the live engine emits only link buttons today, so a postback is
	// skipped rather than rendered as a dead control.
	function cardButtonHref(btn: ChatButton): string {
		if (btn.kind === 'link') return safeCardUrl(btn.url);
		if (btn.kind === 'call') return safeCardUrl(`tel:${btn.phone}`);
		return '';
	}
</script>

<div
	class="chat-message message-{role} {className}"
	class:compact
	class:thinking
	class:streaming
	class:has-error={!!error}
	data-view-id={viewId}
	{...restProps}
>
	<!-- Avatar (not for system messages) -->
	{#if role !== 'system'}
		<div class="message-avatar">
			{#if avatar}
				<Avatar src={avatar} size="sm" />
			{:else}
				<!-- config.icon is an Icon NAME (bot/user/info), not an emoji — render it via
				     <Icon> in Avatar's fallback slot, NOT Avatar's emoji `icon` prop (which would
				     print the literal name). -->
				<Avatar size="sm">
					{#snippet fallback()}
						<Icon name={config.icon} size="sm" />
					{/snippet}
				</Avatar>
			{/if}
		</div>
	{/if}

	<!-- Message bubble -->
	<div class="message-bubble">
		<!-- Header (sender + time) -->
		{#if !compact && (displayName || timestamp)}
			<div class="message-header">
				{#if displayName}
					<span class="message-sender">{displayName}</span>
				{/if}
				{#if timestamp}
					<span class="message-time">{timestamp}</span>
				{/if}
			</div>
		{/if}

		<!-- Content -->
		<div class="message-content">
			{#if thinking}
				<div class="thinking-indicator">
					<span class="dot"></span>
					<span class="dot"></span>
					<span class="dot"></span>
				</div>
			{:else if error}
				<div class="error-content">
					<Icon name="alert-circle" size="sm" />
					<span>{error}</span>
					{#if onretry}
						<button class="retry-btn" onclick={onretry}>Retry</button>
					{/if}
				</div>
			{:else if useMarkdown}
				<!-- Rich prose (assistant/system): markdown render — text, lists, images,
				     links. The trailing streaming cursor still animates while streaming. -->
				<div class="message-markdown" class:streaming>
					<SvelteMarkdown source={content} renderers={mdRenderers} sanitizeUrl={mdSanitizeUrl} />
				</div>
			{:else}
				<!-- Plain path: literal user input, or empty/first-token streaming state. -->
				<p class:streaming>{content}</p>
			{/if}
		</div>

		<!-- Result cards (property results etc.) the assistant attached to this turn.
		     Rendered below the prose; each card is image + title + subtitle + up to 3 action
		     buttons, optionally tappable as a whole via defaultAction. -->
		{#if cards.length > 0}
			<div class="message-cards" role="group" aria-label="Results">
				{#each cards as card, i (card.title + '-' + i)}
					{@const bodyHref = safeCardUrl(card.defaultAction?.url)}
					<div class="chat-card">
						{#if bodyHref}
							<a class="chat-card__body" href={bodyHref}>
								{@render cardInner(card)}
							</a>
						{:else}
							<div class="chat-card__body">
								{@render cardInner(card)}
							</div>
						{/if}
						{#if card.buttons && card.buttons.length > 0}
							<div class="chat-card__buttons">
								{#each card.buttons.slice(0, 3) as btn (btn.label)}
									{@const href = cardButtonHref(btn)}
									{#if href}
										<a class="chat-card__button" href={href}>{btn.label}</a>
									{/if}
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<!-- Footer (status + actions) -->
		<div class="message-footer">
			<!-- Status indicators for user messages -->
			{#if role === 'user' && status}
				<span class="message-status status-{status}">
					{#if status === 'sending'}
						<Icon name="clock" size="xs" />
					{:else if status === 'sent'}
						<Icon name="check" size="xs" />
					{:else if status === 'delivered'}
						<Icon name="check-check" size="xs" />
					{:else if status === 'read'}
						<Icon name="check-check" size="xs" class="read" />
					{:else if status === 'error'}
						<Icon name="alert-circle" size="xs" />
					{/if}
				</span>
			{/if}

			<!-- Actions -->
			{#if showActions && !thinking && !error}
				<div class="message-actions">
					<button class="action-btn" onclick={handleCopy} title="Copy message">
						<Icon name="copy" size="xs" />
					</button>
				</div>
			{/if}
		</div>
	</div>
</div>

{#snippet cardInner(card: ChatCard)}
	{@const imgSrc = safeCardUrl(card.image?.url)}
	{#if imgSrc}
		<img class="chat-card__image" src={imgSrc} alt={card.image?.alt ?? card.title} loading="lazy" />
	{/if}
	<div class="chat-card__text">
		<span class="chat-card__title">{card.title}</span>
		{#if card.subtitle}
			<span class="chat-card__subtitle">{card.subtitle}</span>
		{/if}
	</div>
{/snippet}

<style>
	.chat-message {
		display: flex;
		gap: 0.75rem;
		padding: 0.5rem 0;
		max-width: 85%;
	}

	/* Role alignment */
	.message-user {
		flex-direction: row-reverse;
		margin-left: auto;
	}

	.message-assistant {
		margin-right: auto;
	}

	.message-system {
		max-width: 100%;
		justify-content: center;
	}

	/* Avatar */
	.message-avatar {
		flex-shrink: 0;
		align-self: flex-end;
	}

	/* Bubble */
	.message-bubble {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 0;
	}

	/* Header */
	.message-header {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		padding: 0 0.25rem;
	}

	.message-sender {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-text-muted, #9ca3af);
	}

	.message-time {
		font-size: 0.625rem;
		color: var(--color-text-muted, #6b7280);
	}

	/* Content */
	.message-content {
		padding: 0.75rem 1rem;
		border-radius: 1rem;
		background: var(--message-bg, var(--color-bg-tertiary, #252540));
	}

	/* Role-specific bubble styles */
	.message-user .message-content {
		background: var(--message-user-bg, var(--color-accent, #6366f1));
		color: var(--message-user-text, #fff);
		border-bottom-right-radius: 0.25rem;
	}

	.message-assistant .message-content {
		background: var(--message-assistant-bg, var(--color-bg-tertiary, #252540));
		border-bottom-left-radius: 0.25rem;
	}

	.message-system .message-content {
		background: var(--message-system-bg, rgba(99, 102, 241, 0.1));
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: 0.5rem;
		text-align: center;
		font-size: 0.75rem;
		color: var(--color-text-muted, #9ca3af);
	}

	.message-content p {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.5;
		white-space: pre-wrap;
		word-break: break-word;
	}

	/* Markdown prose — inherits the bubble text colour; spacing tuned for chat density.
	   Uses :global() because the rendered markup is owned by SvelteMarkdown, not this
	   component's scoped DOM. */
	.message-markdown {
		font-size: 0.875rem;
		line-height: 1.5;
		word-break: break-word;
	}

	.message-markdown :global(p) {
		margin: 0 0 0.5rem;
	}

	.message-markdown :global(p:last-child) {
		margin-bottom: 0;
	}

	.message-markdown :global(ul),
	.message-markdown :global(ol) {
		margin: 0 0 0.5rem;
		padding-left: 1.25rem;
	}

	.message-markdown :global(li) {
		margin: 0.125rem 0;
	}

	.message-markdown :global(li > p) {
		margin: 0;
	}

	.message-markdown :global(a) {
		color: var(--message-link-color, var(--color-accent, #6366f1));
		text-decoration: underline;
	}

	.message-markdown :global(img) {
		display: block;
		max-width: 100%;
		height: auto;
		margin: 0.5rem 0;
		border-radius: 0.5rem;
	}

	.message-markdown :global(code) {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 0.8125em;
		padding: 0.1em 0.35em;
		border-radius: 0.25rem;
		background: var(--message-code-bg, var(--color-bg-hover, rgba(0, 0, 0, 0.08)));
	}

	.message-markdown :global(pre) {
		margin: 0 0 0.5rem;
		padding: 0.625rem 0.75rem;
		overflow-x: auto;
		border-radius: 0.5rem;
		background: var(--message-code-bg, var(--color-bg-hover, rgba(0, 0, 0, 0.08)));
	}

	.message-markdown :global(pre code) {
		padding: 0;
		background: transparent;
	}

	.message-markdown :global(:first-child) {
		margin-top: 0;
	}

	/* Streaming cursor — appended after the last rendered block while streaming. */
	.message-content p.streaming::after,
	.message-markdown.streaming::after {
		content: '|';
		display: inline;
		margin-left: 0.05em;
		animation: cursor-blink 1s infinite;
	}

	@keyframes cursor-blink {
		0%, 50% { opacity: 1; }
		51%, 100% { opacity: 0; }
	}

	@media (prefers-reduced-motion: reduce) {
		.message-content p.streaming::after,
		.message-markdown.streaming::after {
			animation: none;
		}
	}

	/* Thinking indicator */
	.thinking-indicator {
		display: flex;
		gap: 0.25rem;
		padding: 0.25rem 0;
	}

	.thinking-indicator .dot {
		width: 0.5rem;
		height: 0.5rem;
		background: var(--color-text-muted, #9ca3af);
		border-radius: 50%;
		animation: thinking-bounce 1.4s infinite ease-in-out;
	}

	.thinking-indicator .dot:nth-child(1) { animation-delay: 0s; }
	.thinking-indicator .dot:nth-child(2) { animation-delay: 0.2s; }
	.thinking-indicator .dot:nth-child(3) { animation-delay: 0.4s; }

	@keyframes thinking-bounce {
		0%, 80%, 100% {
			transform: translateY(0);
			opacity: 0.4;
		}
		40% {
			transform: translateY(-0.375rem);
			opacity: 1;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.thinking-indicator .dot {
			animation: none;
		}
	}

	/* Error state */
	.error-content {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--color-error, #ef4444);
		font-size: 0.875rem;
	}

	.retry-btn {
		padding: 0.25rem 0.5rem;
		background: transparent;
		border: 1px solid currentColor;
		border-radius: 0.25rem;
		color: inherit;
		font-size: 0.75rem;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.retry-btn:hover {
		background: var(--color-error, #ef4444);
		color: #fff;
	}

	/* Footer */
	.message-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.5rem;
		padding: 0 0.25rem;
		min-height: 1.25rem;
	}

	.message-user .message-footer {
		flex-direction: row-reverse;
	}

	/* Status */
	.message-status {
		display: flex;
		align-items: center;
		color: var(--color-text-muted, #6b7280);
	}

	.message-status.status-read :global(.read) {
		color: var(--color-info, #3b82f6);
	}

	.message-status.status-error {
		color: var(--color-error, #ef4444);
	}

	/* Actions */
	.message-actions {
		display: flex;
		gap: 0.25rem;
		opacity: 0;
		transition: opacity 0.15s ease;
	}

	.chat-message:hover .message-actions {
		opacity: 1;
	}

	.action-btn {
		padding: 0.25rem;
		background: transparent;
		border: none;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		border-radius: 0.25rem;
		transition: all 0.15s ease;
	}

	.action-btn:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, #2a2a4a);
	}

	/* Compact mode */
	.chat-message.compact {
		padding: 0.25rem 0;
	}

	.chat-message.compact .message-content {
		padding: 0.5rem 0.75rem;
	}

	.chat-message.compact .message-avatar {
		display: none;
	}

	/* ── Result cards (property results etc.) ──────────────────────────────────────────
	   A HORIZONTAL snap-scroll carousel under the message prose (the Meta-style card rail
	   chat protocols assume — the deferred #36, task 2607-119): fixed-width cards, swipe /
	   scroll sideways, one visible edge hinting there is more. Themed via the same token
	   family the bubbles use (--color-* / --message-* / --chat-card-*), so a consumer's
	   theme bridge (e.g. BR light) maps them without touching this component. NOTE
	   --chat-card-border exists because admin shells commonly repoint --color-border for
	   their own chrome (BR maps it to transparent) — cards need their own say. */
	.message-cards {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.5rem;
		max-width: 100%;
		overflow-x: auto;
		scroll-snap-type: x proximity;
		padding-bottom: 0.375rem;
		scrollbar-width: thin;
	}

	.message-cards::-webkit-scrollbar {
		height: 6px;
	}

	.message-cards::-webkit-scrollbar-thumb {
		background: var(--chat-card-border, var(--color-border, #2a2a4a));
		border-radius: 3px;
	}

	.chat-card {
		display: flex;
		flex-direction: column;
		flex: 0 0 16rem;
		scroll-snap-align: start;
		overflow: hidden;
		border: 1px solid var(--chat-card-border, var(--color-border, #2a2a4a));
		border-radius: 0.75rem;
		background: var(--chat-card-bg, var(--color-bg-secondary, #1a1a2e));
	}

	.chat-card__body {
		display: flex;
		flex-direction: column;
		color: inherit;
		text-decoration: none;
	}

	a.chat-card__body {
		cursor: pointer;
		transition: background 0.15s ease;
	}

	a.chat-card__body:hover {
		background: var(--color-bg-hover, rgba(0, 0, 0, 0.06));
	}

	a.chat-card__body:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: -2px;
	}

	.chat-card__image {
		display: block;
		width: 100%;
		aspect-ratio: 16 / 10;
		object-fit: cover;
		background: var(--color-bg-tertiary, #252540);
	}

	.chat-card__text {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
		padding: 0.625rem 0.75rem;
	}

	.chat-card__title {
		font-size: 0.875rem;
		font-weight: 600;
		line-height: 1.3;
		color: var(--color-text, #fff);
	}

	.chat-card__subtitle {
		font-size: 0.8125rem;
		line-height: 1.35;
		color: var(--color-text-muted, #9ca3af);
	}

	.chat-card__buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.375rem;
		padding: 0 0.75rem 0.75rem;
	}

	.chat-card__button {
		flex: 1 1 auto;
		text-align: center;
		padding: 0.4rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 600;
		text-decoration: none;
		border-radius: 0.5rem;
		border: 1px solid var(--message-link-color, var(--color-accent, #6366f1));
		color: var(--message-link-color, var(--color-accent, #6366f1));
		transition:
			background 0.15s ease,
			color 0.15s ease;
	}

	.chat-card__button:hover {
		background: var(--message-link-color, var(--color-accent, #6366f1));
		color: var(--color-bg-secondary, #fff);
	}

	.chat-card__button:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}
</style>

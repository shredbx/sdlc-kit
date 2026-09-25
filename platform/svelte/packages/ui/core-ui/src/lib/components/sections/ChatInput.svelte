<script lang="ts">
	/**
	 * ChatInput Block Component
	 *
	 * Text input with send button for chat interfaces.
	 * Supports multi-line, attachments, and voice input.
	 *
	 * @example
	 * <ChatInput onsubmit={(msg) => sendMessage(msg)} placeholder="Ask anything..." />
	 */

	import Icon from '../primitives/Icon.svelte';
	import IconButton from '../primitives/IconButton.svelte';

	interface ChatInputProps {
		/** Current input value (bindable) */
		value?: string;
		/** Placeholder text */
		placeholder?: string;
		/** Disabled state */
		disabled?: boolean;
		/** Loading state (sending) */
		loading?: boolean;
		/** Maximum characters */
		maxlength?: number;
		/** Allow multi-line input */
		multiline?: boolean;
		/** Maximum rows for multiline */
		maxRows?: number;
		/** Show character count */
		showCount?: boolean;
		/** Show attachment button */
		showAttachment?: boolean;
		/** Show voice input button */
		showVoice?: boolean;
		/** Submit handler */
		onsubmit?: (message: string) => void;
		/** Attachment handler */
		onattach?: () => void;
		/** Voice input handler */
		onvoice?: () => void;
		/** Input change handler */
		oninput?: (value: string) => void;
		/** Focus state */
		autofocus?: boolean;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		value = $bindable(''),
		placeholder = 'Type a message...',
		disabled = false,
		loading = false,
		maxlength,
		multiline = false,
		maxRows = 4,
		showCount = false,
		showAttachment = false,
		showVoice = false,
		onsubmit,
		onattach,
		onvoice,
		oninput,
		autofocus = false,
		class: className = '',
		'data-view-id': viewId,
		...restProps
	}: ChatInputProps = $props();

	// svelte-ignore non_reactive_update
	let inputRef: HTMLTextAreaElement | HTMLInputElement;
	let rows = $state(1);

	// Auto-resize for multiline
	function handleInput(e: Event) {
		const target = e.target as HTMLTextAreaElement | HTMLInputElement;
		value = target.value;
		oninput?.(value);

		if (multiline && target instanceof HTMLTextAreaElement) {
			// Reset height to calculate scroll height
			target.style.height = 'auto';
			const lineHeight = parseInt(getComputedStyle(target).lineHeight) || 20;
			const newRows = Math.min(Math.ceil(target.scrollHeight / lineHeight), maxRows);
			rows = Math.max(1, newRows);
			target.style.height = `${rows * lineHeight}px`;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			if (multiline && !e.shiftKey) {
				// Multiline: Enter submits, Shift+Enter adds line
				e.preventDefault();
				handleSubmit();
			} else if (!multiline) {
				// Single line: Enter always submits
				e.preventDefault();
				handleSubmit();
			}
		}
	}

	function handleSubmit() {
		if (disabled || loading || !value.trim()) return;
		onsubmit?.(value.trim());
		value = '';
		rows = 1;
		if (inputRef) {
			inputRef.style.height = 'auto';
		}
	}

	let canSend = $derived(!disabled && !loading && value.trim().length > 0);
	let charCount = $derived(value.length);
	let isOverLimit = $derived(maxlength ? charCount > maxlength : false);
</script>

<div
	class="chat-input-wrapper {className}"
	class:disabled
	class:loading
	class:multiline
	data-view-id={viewId}
	{...restProps}
>
	<!-- Leading actions -->
	{#if showAttachment || showVoice}
		<div class="input-actions leading">
			{#if showAttachment}
				<IconButton
					icon="paperclip"
					variant="ghost"
					size="sm"
					onclick={onattach}
					disabled={disabled || loading}
					title="Attach file"
				/>
			{/if}
		</div>
	{/if}

	<!-- Input field -->
	<div class="input-container">
		{#if multiline}
			<textarea
				bind:this={inputRef}
				{value}
				{placeholder}
				{disabled}
				{maxlength}
				{rows}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="chat-input"
				class:over-limit={isOverLimit}
			></textarea>
		{:else}
			<input
				bind:this={inputRef}
				type="text"
				{value}
				{placeholder}
				{disabled}
				{maxlength}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="chat-input"
				class:over-limit={isOverLimit}
			/>
		{/if}

		<!-- Character count -->
		{#if showCount && maxlength}
			<span class="char-count" class:over-limit={isOverLimit}>
				{charCount}/{maxlength}
			</span>
		{/if}
	</div>

	<!-- Trailing actions -->
	<div class="input-actions trailing">
		{#if showVoice}
			<IconButton
				icon="mic"
				variant="ghost"
				size="sm"
				onclick={onvoice}
				disabled={disabled || loading}
				title="Voice input"
			/>
		{/if}

		<!-- Send button -->
		<IconButton
			icon={loading ? 'loader' : 'send'}
			variant={canSend ? 'primary' : 'ghost'}
			size="sm"
			onclick={handleSubmit}
			disabled={!canSend}
			loading={loading}
			title="Send message"
		/>
	</div>
</div>

<style>
	.chat-input-wrapper {
		display: flex;
		align-items: flex-end;
		gap: 0.5rem;
		padding: 0.5rem;
		background: var(--chat-input-bg, var(--color-bg-tertiary, #252540));
		border: 1px solid var(--chat-input-border, var(--color-border, #2a2a4a));
		border-radius: 0.75rem;
		transition: all 0.15s ease;
	}

	.chat-input-wrapper:focus-within {
		border-color: var(--color-accent, #6366f1);
		box-shadow: 0 0 0 3px var(--chat-input-ring, rgba(99, 102, 241, 0.1));
	}

	.chat-input-wrapper.disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	/* Input container */
	.input-container {
		flex: 1;
		display: flex;
		flex-direction: column;
		position: relative;
	}

	/* Input field */
	.chat-input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: transparent;
		border: none;
		color: var(--color-text, #fff);
		font-size: 0.875rem;
		font-family: inherit;
		line-height: 1.5;
		resize: none;
	}

	.chat-input::placeholder {
		color: var(--color-text-muted, #6b7280);
	}

	.chat-input:focus {
		outline: none;
	}

	.chat-input:disabled {
		cursor: not-allowed;
	}

	/* Textarea specific */
	textarea.chat-input {
		min-height: 1.5rem;
		max-height: 6rem;
		overflow-y: auto;
	}

	/* Character count */
	.char-count {
		position: absolute;
		bottom: 0.25rem;
		right: 0.5rem;
		font-size: 0.625rem;
		color: var(--color-text-muted, #6b7280);
		pointer-events: none;
	}

	.char-count.over-limit {
		color: var(--color-error, #ef4444);
	}

	.chat-input.over-limit {
		color: var(--color-error, #ef4444);
	}

	/* Action buttons */
	.input-actions {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		flex-shrink: 0;
	}

	.input-actions.leading {
		padding-right: 0.25rem;
	}

	.input-actions.trailing {
		padding-left: 0.25rem;
	}
</style>

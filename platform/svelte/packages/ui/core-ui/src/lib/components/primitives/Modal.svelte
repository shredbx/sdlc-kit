<script lang="ts">
	/**
	 * Modal Component
	 *
	 * Overlay modal dialog with backdrop, Escape-to-close, and animations.
	 * NOTE: renders in place (position:fixed) — NO portal and NO focus trap:
	 * Tab still walks the page behind the overlay, and focus may sit on
	 * document.body while open. Consumers with global keyboard shortcuts must
	 * guard on an open dialog themselves (e.g. query [role="dialog"]).
	 *
	 * @example
	 * <Modal open={showModal} onclose={() => showModal = false}>
	 *   {#snippet header()}Modal Title{/snippet}
	 *   <p>Modal content here</p>
	 * </Modal>
	 */
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'full';

	interface ModalProps {
		/** Whether modal is open */
		open?: boolean;
		/** Modal size preset */
		size?: ModalSize;
		/** Custom title (alternative to header snippet) */
		title?: string;
		/** Close on backdrop click */
		closeOnBackdrop?: boolean;
		/** Close on escape key */
		closeOnEscape?: boolean;
		/** Show close button */
		showClose?: boolean;
		/** Close handler */
		onclose?: () => void;
		/** Header snippet */
		header?: Snippet;
		/** Footer snippet */
		footer?: Snippet;
		/** Main content */
		children?: Snippet;
		/** Additional CSS class */
		class?: string;
	}

	let {
		open = false,
		size = 'md',
		title = '',
		closeOnBackdrop = true,
		closeOnEscape = true,
		showClose = true,
		onclose,
		header,
		footer,
		children,
		class: className = ''
	}: ModalProps = $props();

	function handleBackdropClick(e: MouseEvent) {
		if (closeOnBackdrop && e.target === e.currentTarget) {
			onclose?.();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (closeOnEscape && e.key === 'Escape') {
			onclose?.();
		}
	}

	// Body scroll lock + Escape listener (no focus trap — see header note).
	$effect(() => {
		if (open) {
			document.body.style.overflow = 'hidden';
			// Add keydown listener
			document.addEventListener('keydown', handleKeydown);
		}
		return () => {
			document.body.style.overflow = '';
			document.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="modal-backdrop"
		role="dialog"
		aria-modal="true"
		aria-labelledby={title ? 'modal-title' : undefined}
		onclick={handleBackdropClick}
		onkeydown={handleKeydown}
	>
		<div class="modal-container modal-{size} {className}">
			{#if header || title || showClose}
				<header class="modal-header">
					{#if header}
						{@render header()}
					{:else if title}
						<h2 id="modal-title" class="modal-title">{title}</h2>
					{/if}
					{#if showClose}
						<button
							class="modal-close"
							onclick={onclose}
							aria-label="Close modal"
							type="button"
						>
							<Icon name="x" size="md" />
						</button>
					{/if}
				</header>
			{/if}

			<div class="modal-body">
				{#if children}
					{@render children()}
				{/if}
			</div>

			{#if footer}
				<footer class="modal-footer">
					{@render footer()}
				</footer>
			{/if}
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		/* -webkit- twin: unprefixed backdrop-filter is Safari 18+ (SWC-6). */
		-webkit-backdrop-filter: blur(4px);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 1rem;
		animation: fadeIn 0.15s ease-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}

	.modal-container {
		background: var(--color-bg-secondary, #1a1a2e);
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: 1rem;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
		display: flex;
		flex-direction: column;
		max-height: calc(100vh - 2rem);
		overflow: hidden;
		animation: slideIn 0.2s ease-out;
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: scale(0.95) translateY(-10px);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}

	/* Size variants */
	.modal-sm {
		width: 100%;
		max-width: 400px;
	}

	.modal-md {
		width: 100%;
		max-width: 560px;
	}

	.modal-lg {
		width: 100%;
		max-width: 800px;
	}

	.modal-xl {
		width: 100%;
		max-width: 1140px;
	}

	.modal-full {
		width: calc(100vw - 2rem);
		height: calc(100vh - 2rem);
		max-width: none;
	}

	/* Header */
	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		border-bottom: 1px solid var(--color-border, #2a2a4a);
		flex-shrink: 0;
	}

	.modal-title {
		margin: 0;
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--color-text, #fff);
	}

	.modal-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		background: transparent;
		border: none;
		border-radius: 0.5rem;
		color: var(--color-text-muted, #9ca3af);
		cursor: pointer;
		transition: all 0.15s ease;
		margin-left: auto;
	}

	.modal-close:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--color-text, #fff);
	}

	/* Body */
	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 1.5rem;
	}

	/* Footer */
	.modal-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.75rem;
		padding: 1rem 1.5rem;
		border-top: 1px solid var(--color-border, #2a2a4a);
		flex-shrink: 0;
	}

	/* Responsive */
	@media (max-width: 640px) {
		.modal-backdrop {
			padding: 0;
			align-items: flex-end;
		}

		.modal-container {
			max-width: 100%;
			max-height: 90vh;
			border-radius: 1rem 1rem 0 0;
			animation: slideUp 0.25s ease-out;
		}

		@keyframes slideUp {
			from {
				opacity: 0;
				transform: translateY(100%);
			}
			to {
				opacity: 1;
				transform: translateY(0);
			}
		}

		.modal-full {
			height: 100vh;
			border-radius: 0;
		}
	}
</style>

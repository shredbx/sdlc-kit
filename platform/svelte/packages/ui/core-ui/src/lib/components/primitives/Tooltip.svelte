<script lang="ts">
	/**
	 * Tooltip Primitive Component
	 *
	 * A lightweight tooltip that appears on hover with customizable position.
	 * Uses CSS for positioning - no JavaScript measurement needed.
	 *
	 * @example
	 * <Tooltip text="Settings" position="right">
	 *   <IconButton icon="settings" />
	 * </Tooltip>
	 */

	import type { Snippet } from 'svelte';

	interface TooltipProps {
		/** Tooltip text */
		text: string;
		/** Position relative to trigger */
		position?: 'top' | 'bottom' | 'left' | 'right';
		/** Delay before showing (ms) */
		delay?: number;
		/** Additional CSS classes */
		class?: string;
		/** Trigger content */
		children: Snippet;
	}

	let {
		text,
		position = 'right',
		delay = 0,
		class: className = '',
		children
	}: TooltipProps = $props();

	let visible = $state(false);
	let timeoutId: ReturnType<typeof setTimeout> | null = null;

	function show() {
		if (delay > 0) {
			timeoutId = setTimeout(() => {
				visible = true;
			}, delay);
		} else {
			visible = true;
		}
	}

	function hide() {
		if (timeoutId) {
			clearTimeout(timeoutId);
			timeoutId = null;
		}
		visible = false;
	}
</script>

<div
	class="tooltip-wrapper {className}"
	role="group"
	onmouseenter={show}
	onmouseleave={hide}
	onfocus={show}
	onblur={hide}
>
	{@render children()}
	{#if visible && text}
		<div class="tooltip tooltip-{position}" role="tooltip">
			{text}
		</div>
	{/if}
</div>

<style>
	.tooltip-wrapper {
		position: relative;
		display: inline-flex;
	}

	.tooltip {
		position: absolute;
		z-index: 1000;
		padding: 0.5rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 500;
		white-space: nowrap;
		color: var(--color-text, #fff);
		background: var(--color-bg-elevated, #1f1f2e);
		border: 1px solid var(--color-border, rgba(255, 255, 255, 0.1));
		border-radius: 0.375rem;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		pointer-events: none;
		animation: tooltip-fade-in 150ms ease-out;
	}

	@keyframes tooltip-fade-in {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}

	/* Position variants */
	.tooltip-right {
		left: calc(100% + 8px);
		top: 50%;
		transform: translateY(-50%);
	}

	.tooltip-left {
		right: calc(100% + 8px);
		top: 50%;
		transform: translateY(-50%);
	}

	.tooltip-top {
		bottom: calc(100% + 8px);
		left: 50%;
		transform: translateX(-50%);
	}

	.tooltip-bottom {
		top: calc(100% + 8px);
		left: 50%;
		transform: translateX(-50%);
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.tooltip {
			animation: none;
		}
	}
</style>

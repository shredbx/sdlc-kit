<script lang="ts">
	/**
	 * Badge Primitive Component
	 *
	 * A compact label for counts, status, or categorization.
	 * Supports variants, sizes, animations, and IView conformance.
	 *
	 * @example
	 * <Badge count={5} variant="error" />
	 * <Badge label="New" variant="success" size="sm" />
	 * <Badge label="Beta" variant="warning" dot />
	 */

	import type { Snippet } from 'svelte';

	type BadgeVariant = 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'danger' | 'info' | 'muted';
	type BadgeSize = 'xs' | 'sm' | 'md' | 'lg';

	interface BadgeProps {
		/** Numeric count to display (auto-formats 99+) */
		count?: number;
		/** Text label (alternative to count) */
		label?: string;
		/** Visual variant */
		variant?: BadgeVariant;
		/** Size preset */
		size?: BadgeSize;
		/** Show as dot only (no content) */
		dot?: boolean;
		/** Maximum count before showing + */
		max?: number;
		/** Show zero count */
		showZero?: boolean;
		/** Pulse animation for attention */
		pulse?: boolean;
		/** Custom icon before label */
		icon?: string;
		/** Additional CSS classes */
		class?: string;
		/** Child content (alternative to label) */
		children?: Snippet;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		count,
		label,
		variant = 'default',
		size = 'sm',
		dot = false,
		max = 99,
		showZero = false,
		pulse = false,
		icon,
		class: className = '',
		children,
		'data-view-id': viewId,
		...restProps
	}: BadgeProps = $props();

	// Computed display value
	let displayValue = $derived.by(() => {
		if (dot) return '';
		if (label) return label;
		if (count !== undefined) {
			if (count === 0 && !showZero) return null;
			if (count > max) return `${max}+`;
			return String(count);
		}
		return null;
	});

	// Hide badge if no value and not dot
	let isVisible = $derived(dot || displayValue !== null || children);
</script>

{#if isVisible}
	<span
		class="badge badge-{variant} badge-{size} {className}"
		class:dot
		class:pulse
		class:with-icon={!!icon}
		class:has-children={!!children}
		data-view-id={viewId}
		{...restProps}
	>
		{#if icon}
			<span class="badge-icon">{icon}</span>
		{/if}
		{#if !dot}
			{#if children}
				{@render children()}
			{:else if displayValue}
				{displayValue}
			{/if}
		{/if}
	</span>
{/if}

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 600;
		white-space: nowrap;
		border-radius: 9999px;
		transition: all 0.15s ease;
	}

	/* Sizes */
	.badge-xs {
		font-size: 0.625rem;
		padding: 0 0.375rem;
		min-width: 1rem;
		height: 1rem;
	}

	.badge-sm {
		font-size: 0.75rem;
		padding: 0.125rem 0.5rem;
		min-width: 1.25rem;
		height: 1.25rem;
	}

	.badge-md {
		font-size: 0.875rem;
		padding: 0.25rem 0.625rem;
		min-width: 1.5rem;
		height: 1.5rem;
	}

	.badge-lg {
		font-size: 1rem;
		padding: 0.375rem 0.75rem;
		min-width: 1.75rem;
		height: 1.75rem;
	}

	/* Variants */
	.badge-default {
		background: var(--badge-default-bg, #374151);
		color: var(--badge-default-text, #fff);
	}

	.badge-primary {
		background: var(--badge-primary-bg, var(--color-accent, #6366f1));
		color: var(--badge-primary-text, #fff);
	}

	.badge-success {
		background: var(--badge-success-bg, #10b981);
		color: var(--badge-success-text, #fff);
	}

	.badge-warning {
		background: var(--badge-warning-bg, #f59e0b);
		color: var(--badge-warning-text, #000);
	}

	.badge-error {
		background: var(--badge-error-bg, #ef4444);
		color: var(--badge-error-text, #fff);
	}

	.badge-info {
		background: var(--badge-info-bg, #3b82f6);
		color: var(--badge-info-text, #fff);
	}

	/* Dot variant */
	.badge.dot {
		padding: 0;
		min-width: 0;
	}

	.badge.dot.badge-xs {
		width: 0.375rem;
		height: 0.375rem;
	}

	.badge.dot.badge-sm {
		width: 0.5rem;
		height: 0.5rem;
	}

	.badge.dot.badge-md {
		width: 0.625rem;
		height: 0.625rem;
	}

	.badge.dot.badge-lg {
		width: 0.75rem;
		height: 0.75rem;
	}

	/* Icon */
	.badge-icon {
		margin-right: 0.25rem;
		font-size: 0.875em;
	}

	.badge.with-icon {
		padding-left: 0.375rem;
	}

	/* Pulse animation */
	.badge.pulse {
		animation: badge-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes badge-pulse {
		0%, 100% {
			opacity: 1;
		}
		50% {
			opacity: 0.7;
		}
	}

	/* Has children (custom content) */
	.badge.has-children {
		padding: 0.25rem 0.5rem;
	}
</style>

<script lang="ts">
	/**
	 * IconButton Primitive Component
	 *
	 * A button optimized for icon-only interactions with proper accessibility.
	 * Supports variants, sizes, shapes, loading states, and IView conformance.
	 *
	 * Per architecture: Action primitive for icon-based interactions.
	 *
	 * @example
	 * <IconButton icon="bell" label="Notifications" />
	 * <IconButton icon="x" variant="ghost" size="sm" />
	 * <IconButton icon="message-circle" variant="primary" shape="circle" />
	 * <IconButton icon="settings" loading />
	 */

	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	type IconButtonVariant = 'default' | 'primary' | 'secondary' | 'ghost' | 'danger' | 'success';
	type IconButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';
	type IconButtonShape = 'square' | 'rounded' | 'circle';

	interface IconButtonProps {
		/** Icon to display (emoji, text, or use slot) */
		icon?: string;
		/** Accessible label (falls back to title if not provided) */
		label?: string;
		/** Visual variant */
		variant?: IconButtonVariant;
		/** Size preset */
		size?: IconButtonSize;
		/** Button shape */
		shape?: IconButtonShape;
		/** Disabled state */
		disabled?: boolean;
		/** Loading state (shows spinner, disables button) */
		loading?: boolean;
		/** Active/pressed state */
		active?: boolean;
		/** Show tooltip on hover */
		tooltip?: boolean;
		/** Tooltip position */
		tooltipPosition?: 'top' | 'bottom' | 'left' | 'right';
		/** Badge count (optional notification badge) */
		badge?: number;
		/** Badge variant */
		badgeVariant?: 'default' | 'error' | 'warning' | 'success';
		/** Button type */
		type?: 'button' | 'submit' | 'reset';
		/** HTML title attribute (tooltip on hover) */
		title?: string;
		/** Additional CSS classes */
		class?: string;
		/** Custom icon content */
		children?: Snippet;
		/** Click handler */
		onclick?: (e: MouseEvent) => void;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		icon,
		label,
		variant = 'default',
		size = 'md',
		shape = 'rounded',
		disabled = false,
		loading = false,
		active = false,
		tooltip = true,
		tooltipPosition = 'top',
		badge,
		badgeVariant = 'error',
		type = 'button',
		title,
		class: className = '',
		children,
		onclick,
		'data-view-id': viewId,
		...restProps
	}: IconButtonProps = $props();

	// Size mappings
	const sizeMap: Record<IconButtonSize, { size: string; iconSize: string; padding: string }> = {
		xs: { size: '1.5rem', iconSize: '0.75rem', padding: '0.25rem' },
		sm: { size: '2rem', iconSize: '1rem', padding: '0.375rem' },
		md: { size: '2.5rem', iconSize: '1.25rem', padding: '0.5rem' },
		lg: { size: '3rem', iconSize: '1.5rem', padding: '0.625rem' },
		xl: { size: '3.5rem', iconSize: '1.75rem', padding: '0.75rem' }
	};

	// Computed style
	let customStyle = $derived.by(() => {
		const s = sizeMap[size];
		return `--btn-size: ${s.size}; --btn-icon-size: ${s.iconSize}; --btn-padding: ${s.padding}`;
	});

	// Badge display value
	let badgeDisplay = $derived.by(() => {
		if (badge === undefined) return null;
		if (badge > 99) return '99+';
		return String(badge);
	});
</script>

<button
	{type}
	class="icon-button icon-button-{variant} icon-button-{size} icon-button-{shape} {className}"
	class:active
	class:loading
	style={customStyle}
	disabled={disabled || loading}
	aria-label={label ?? title}
	aria-pressed={active}
	aria-busy={loading}
	title={title ?? (tooltip ? (label ?? title) : undefined)}
	data-tooltip-position={tooltip ? tooltipPosition : undefined}
	data-view-id={viewId}
	onclick={onclick}
	{...restProps}
>
	<span class="icon-button-content" class:hidden={loading}>
		{#if icon}
			<Icon name={icon} size={size === 'xs' ? 'xs' : size === 'sm' ? 'sm' : size === 'md' ? 'md' : size === 'lg' ? 'lg' : 'xl'} />
		{:else if children}
			{@render children()}
		{/if}
	</span>

	{#if loading}
		<span class="icon-button-spinner" aria-hidden="true"></span>
	{/if}

	{#if badgeDisplay !== null}
		<span class="icon-button-badge icon-button-badge-{badgeVariant}">
			{badgeDisplay}
		</span>
	{/if}
</button>

<style>
	.icon-button {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--btn-size);
		height: var(--btn-size);
		padding: var(--btn-padding);
		border: none;
		cursor: pointer;
		font-family: inherit;
		font-size: var(--btn-icon-size);
		line-height: 1;
		transition: all 0.15s ease;
		flex-shrink: 0;
	}

	.icon-button:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	.icon-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Shapes */
	.icon-button-square {
		border-radius: 0;
	}

	.icon-button-rounded {
		border-radius: 0.375rem;
	}

	.icon-button-circle {
		border-radius: 50%;
	}

	/* Variants */
	.icon-button-default {
		background: var(--icon-btn-default-bg, var(--color-bg-tertiary, #252540));
		color: var(--icon-btn-default-text, var(--color-text-muted, #9ca3af));
	}

	.icon-button-default:hover:not(:disabled) {
		background: var(--icon-btn-default-bg-hover, var(--color-bg-hover, #2a2a4a));
		color: var(--icon-btn-default-text-hover, var(--color-text, #fff));
	}

	.icon-button-primary {
		background: var(--icon-btn-primary-bg, var(--color-accent, #6366f1));
		color: var(--icon-btn-primary-text, #fff);
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
	}

	.icon-button-primary:hover:not(:disabled) {
		background: var(--icon-btn-primary-bg-hover, #5855e0);
		box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
		transform: scale(1.05);
	}

	.icon-button-secondary {
		background: var(--icon-btn-secondary-bg, transparent);
		color: var(--icon-btn-secondary-text, var(--color-text, #fff));
		border: 1px solid var(--color-border, #2a2a4a);
	}

	.icon-button-secondary:hover:not(:disabled) {
		background: var(--icon-btn-secondary-bg-hover, rgba(255, 255, 255, 0.05));
		border-color: var(--color-border-hover, #3a3a5a);
	}

	.icon-button-ghost {
		background: transparent;
		color: var(--icon-btn-ghost-text, var(--color-text-muted, #9ca3af));
	}

	.icon-button-ghost:hover:not(:disabled) {
		background: var(--icon-btn-ghost-bg-hover, rgba(255, 255, 255, 0.1));
		color: var(--icon-btn-ghost-text-hover, var(--color-text, #fff));
	}

	.icon-button-danger {
		background: var(--color-error, #ef4444);
		color: #fff;
	}

	.icon-button-danger:hover:not(:disabled) {
		background: #dc2626;
		transform: scale(1.05);
	}

	.icon-button-success {
		background: var(--color-success, #10b981);
		color: #fff;
	}

	.icon-button-success:hover:not(:disabled) {
		background: #059669;
		transform: scale(1.05);
	}

	/* Active state */
	.icon-button.active {
		background: var(--icon-btn-active-bg, var(--color-accent, #6366f1));
		color: var(--icon-btn-active-text, #fff);
	}

	/* Content */
	.icon-button-content {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.icon-button-content.hidden {
		visibility: hidden;
	}

	.icon-button-icon {
		font-family: 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif;
	}

	/* Loading spinner */
	.icon-button-spinner {
		position: absolute;
		width: 1em;
		height: 1em;
		border: 2px solid transparent;
		border-top-color: currentColor;
		border-radius: 50%;
		animation: icon-button-spin 0.6s linear infinite;
	}

	@keyframes icon-button-spin {
		to { transform: rotate(360deg); }
	}

	/* Badge */
	.icon-button-badge {
		position: absolute;
		top: -4px;
		right: -4px;
		min-width: 1.125rem;
		height: 1.125rem;
		padding: 0 0.25rem;
		font-size: 0.625rem;
		font-weight: 600;
		line-height: 1.125rem;
		text-align: center;
		border-radius: 9999px;
		border: 2px solid var(--color-bg-secondary, #1a1a2e);
	}

	.icon-button-badge-default {
		background: var(--color-bg-tertiary, #374151);
		color: #fff;
	}

	.icon-button-badge-error {
		background: var(--color-error, #ef4444);
		color: #fff;
	}

	.icon-button-badge-warning {
		background: var(--color-warning, #f59e0b);
		color: #000;
	}

	.icon-button-badge-success {
		background: var(--color-success, #10b981);
		color: #fff;
	}

</style>

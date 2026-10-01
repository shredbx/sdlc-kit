<script lang="ts">
	/**
	 * NavItem Block Component
	 *
	 * Navigation link with icon, label, badge, and active state.
	 * Inspired by finndollimore.com floating nav with underline animation.
	 *
	 * @example
	 * <NavItem href="/dashboard" icon="home" label="Dashboard" active />
	 * <NavItem href="/settings" icon="settings" label="Settings" badge={3} />
	 */

	import Icon from '../primitives/Icon.svelte';
	import Badge from '../primitives/Badge.svelte';

	type NavItemVariant = 'default' | 'floating' | 'sidebar' | 'pill';
	type BadgeConfig = {
		count?: number;
		label?: string;
		variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
	};

	interface NavItemProps {
		/** Navigation URL */
		href: string;
		/** Display label */
		label: string;
		/** Icon name (from Icon component) */
		icon?: string;
		/** Active state */
		active?: boolean;
		/** Disabled state */
		disabled?: boolean;
		/** Badge configuration */
		badge?: number | BadgeConfig;
		/** Visual variant */
		variant?: NavItemVariant;
		/** External link (opens in new tab) */
		external?: boolean;
		/** Click handler (alternative to href) */
		onclick?: (e: Event) => void;
		/** Tooltip text */
		title?: string;
		/** Collapsed mode (icon only) */
		collapsed?: boolean;
		/** Data attribute for page transitions */
		'data-transition'?: string;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		href,
		label,
		icon,
		active = false,
		disabled = false,
		badge,
		variant = 'default',
		external = false,
		onclick,
		title,
		collapsed = false,
		'data-transition': dataTransition,
		class: className = '',
		'data-view-id': viewId,
		...restProps
	}: NavItemProps = $props();

	// Normalize badge prop
	let badgeConfig = $derived.by(() => {
		if (typeof badge === 'number') {
			return { count: badge, variant: 'warning' as const };
		}
		return badge;
	});

	function handleClick(e: Event) {
		if (disabled) {
			e.preventDefault();
			return;
		}
		onclick?.(e);
	}
</script>

{#if disabled}
	<span
		class="nav-item nav-item-{variant} {className}"
		class:active
		class:disabled
		class:collapsed
		class:has-icon={!!icon}
		class:has-badge={!!badge}
		data-view-id={viewId}
		{title}
		{...restProps}
	>
		{#if icon}
			<Icon name={icon} size={collapsed ? 'md' : 'sm'} />
		{/if}
		{#if !collapsed}
			<span class="nav-label">{label}</span>
		{/if}
		{#if badgeConfig && !collapsed}
			<Badge
				count={badgeConfig.count}
				label={badgeConfig.label}
				variant={badgeConfig.variant ?? 'warning'}
				size="xs"
			/>
		{/if}
	</span>
{:else}
	<a
		{href}
		class="nav-item nav-item-{variant} {className}"
		class:active
		class:collapsed
		class:has-icon={!!icon}
		class:has-badge={!!badge}
		data-view-id={viewId}
		data-transition={dataTransition}
		data-sveltekit-preload-data={external ? undefined : 'hover'}
		target={external ? '_blank' : undefined}
		rel={external ? 'noopener noreferrer' : undefined}
		{title}
		onclick={handleClick}
		{...restProps}
	>
		{#if icon}
			<Icon name={icon} size={collapsed ? 'md' : 'sm'} />
		{/if}
		{#if !collapsed}
			<span class="nav-label">{label}</span>
		{/if}
		{#if badgeConfig && !collapsed}
			<Badge
				count={badgeConfig.count}
				label={badgeConfig.label}
				variant={badgeConfig.variant ?? 'warning'}
				size="xs"
			/>
		{/if}
		{#if external && !collapsed}
			<Icon name="external-link" size="xs" class="external-icon" />
		{/if}
	</a>
{/if}

<style>
	/* Base nav item */
	.nav-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		color: var(--nav-item-color, rgba(255, 255, 255, 0.7));
		text-decoration: none;
		font-size: 0.9375rem;
		font-weight: 500;
		border-radius: 0.375rem;
		position: relative;
		transition: all 0.2s ease;
	}

	/* Underline animation (finndollimore.com style) */
	.nav-item::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0.75rem;
		right: 0.75rem;
		height: 2px;
		background: var(--nav-item-accent, #a78bfa);
		transform: scaleX(0);
		transform-origin: left;
		transition: transform 0.3s cubic-bezier(0.25, 1, 0.5, 1);
	}

	.nav-item:hover::after,
	.nav-item.active::after {
		transform: scaleX(1);
	}

	/* Hover state */
	.nav-item:hover {
		color: var(--nav-item-color-hover, #fff);
	}

	/* Active state */
	.nav-item.active {
		color: var(--nav-item-color-active, #fff);
	}

	/* Disabled state */
	.nav-item.disabled {
		color: var(--nav-item-color-disabled, rgba(255, 255, 255, 0.3));
		cursor: not-allowed;
		user-select: none;
	}

	.nav-item.disabled::after {
		display: none;
	}

	/* Collapsed state */
	.nav-item.collapsed {
		justify-content: center;
		padding: 0.75rem;
	}

	.nav-item.collapsed::after {
		left: 50%;
		right: 50%;
		bottom: 0.25rem;
		width: 0.25rem;
		height: 0.25rem;
		border-radius: 50%;
		transform: scale(0);
	}

	.nav-item.collapsed:hover::after,
	.nav-item.collapsed.active::after {
		transform: scale(1);
	}

	/* Label */
	.nav-label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* External icon */
	.nav-item :global(.external-icon) {
		opacity: 0.5;
		margin-left: auto;
	}

	/* ==========================================================================
	   VARIANT: Floating (glassmorphism nav bar style)
	   ========================================================================== */
	.nav-item-floating {
		padding: 0.5rem 0;
		border-radius: 0;
		background: transparent;
	}

	.nav-item-floating::after {
		left: 0;
		right: 0;
	}

	/* ==========================================================================
	   VARIANT: Sidebar (left/right sidebar style)
	   ========================================================================== */
	.nav-item-sidebar {
		width: 100%;
		padding: 0.625rem 0.875rem;
		border-radius: 0.5rem;
	}

	.nav-item-sidebar::after {
		display: none;
	}

	.nav-item-sidebar:hover {
		background: var(--nav-item-bg-hover, var(--color-bg-hover, #2a2a4a));
	}

	.nav-item-sidebar.active {
		background: var(--nav-item-bg-active, var(--color-bg-tertiary, #252540));
		color: var(--nav-item-color-active, #fff);
	}

	.nav-item-sidebar.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 3px;
		height: 1.25rem;
		background: var(--nav-item-accent, #a78bfa);
		border-radius: 0 2px 2px 0;
	}

	/* ==========================================================================
	   VARIANT: Pill (tab-style navigation)
	   ========================================================================== */
	.nav-item-pill {
		padding: 0.375rem 0.875rem;
		border-radius: 9999px;
	}

	.nav-item-pill::after {
		display: none;
	}

	.nav-item-pill:hover {
		background: var(--nav-item-bg-hover, rgba(255, 255, 255, 0.1));
	}

	.nav-item-pill.active {
		background: var(--nav-item-bg-active, var(--color-accent, #6366f1));
		color: var(--nav-item-color-active, #fff);
	}

	/* Focus state */
	.nav-item:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}
</style>

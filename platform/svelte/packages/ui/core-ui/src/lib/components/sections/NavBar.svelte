<script lang="ts">
	/**
	 * NavBar Component
	 *
	 * A floating glassmorphism navigation bar inspired by finndollimore.com.
	 * Supports horizontal and vertical layouts, badges, active states, and
	 * route-specific page transitions.
	 *
	 * Architecture:
	 * - Follows IView interface for decorator pattern support
	 * - Uses CSS custom properties for theming
	 * - Supports IPresenter for auto-registration in /components
	 * - Integrates with navigation.yml schema
	 *
	 * @example
	 * <NavBar
	 *   items={[{ id: 'home', label: 'Home', href: '/' }]}
	 *   position="top-center"
	 *   variant="floating"
	 * />
	 */

	import { page } from '$app/stores';
	import Badge from '../primitives/Badge.svelte';
	import type { Snippet } from 'svelte';

	// Types matching navigation.yml schema
	interface NavItem {
		id: string;
		label: string;
		href: string;
		icon?: string;
		badge?: { count?: number; label?: string; variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'danger' | 'info' | 'muted' };
		disabled?: boolean;
		external?: boolean;
		children?: NavItem[];
		transition?: string;
	}

	interface NavGroup {
		id: string;
		label?: string;
		items: NavItem[];
		collapsible?: boolean;
		collapsed?: boolean;
	}

	type NavBarVariant = 'floating' | 'fixed' | 'inline';
	type NavBarPosition = 'top-center' | 'top-left' | 'top-right' | 'bottom-center';
	type NavBarStyle = 'horizontal' | 'vertical';

	interface NavBarProps {
		/** Navigation items (flat list) */
		items?: NavItem[];
		/** Navigation groups (grouped items) */
		groups?: NavGroup[];
		/** Visual variant */
		variant?: NavBarVariant;
		/** Position on screen */
		position?: NavBarPosition;
		/** Layout direction */
		style?: NavBarStyle;
		/** Logo text or emoji */
		logo?: string;
		/** Logo href */
		logoHref?: string;
		/** Show/hide state (for animations) */
		visible?: boolean;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
		/** Slot for custom content (e.g., auth buttons) */
		children?: Snippet;
	}

	let {
		items = [],
		groups = [],
		variant = 'floating',
		position = 'top-center',
		style = 'horizontal',
		logo,
		logoHref = '/',
		visible = true,
		class: className = '',
		'data-view-id': viewId,
		children
	}: NavBarProps = $props();

	// Current path for active state
	let currentPath = $derived($page.url.pathname);

	// Compute flat items from groups if provided
	let allItems = $derived.by(() => {
		if (items.length > 0) return items;
		return groups.flatMap((g) => g.items);
	});

	// Check if item is active
	function isActive(href: string): boolean {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}

	// Handle navigation with transition hint
	function handleNavClick(item: NavItem) {
		if (item.transition) {
			// Store transition type for PageTransition component
			document.documentElement.dataset.navTransition = item.transition;
		}
	}
</script>

<nav
	class="navbar navbar-{variant} navbar-{position} navbar-{style} {className}"
	class:visible
	class:hidden={!visible}
	data-view-id={viewId}
	aria-label="Main navigation"
>
	<div class="navbar-container">
		{#if logo}
			<a href={logoHref} class="navbar-logo">
				{logo}
			</a>
		{/if}

		<div class="navbar-items">
			{#if groups.length > 0}
				{#each groups as group (group.id)}
					{#if group.label}
						<span class="navbar-group-label">{group.label}</span>
					{/if}
					{#each group.items as item (item.id)}
						{@render navItem(item)}
					{/each}
				{/each}
			{:else}
				{#each items as item (item.id)}
					{@render navItem(item)}
				{/each}
			{/if}
		</div>

		{#if children}
			<div class="navbar-extra">
				{@render children()}
			</div>
		{/if}
	</div>
</nav>

{#snippet navItem(item: NavItem)}
	{#if item.disabled}
		<span class="navbar-item disabled" title={item.label}>
			{#if item.icon}
				<span class="navbar-icon">{item.icon}</span>
			{/if}
			<span class="navbar-label">{item.label}</span>
		</span>
	{:else}
		<a
			href={item.href}
			class="navbar-item"
			class:active={isActive(item.href)}
			target={item.external ? '_blank' : undefined}
			rel={item.external ? 'noopener noreferrer' : undefined}
			onclick={() => handleNavClick(item)}
			data-sveltekit-preload-data="hover"
		>
			{#if item.icon}
				<span class="navbar-icon">{item.icon}</span>
			{/if}
			<span class="navbar-label">{item.label}</span>
			{#if item.badge}
				<Badge
					count={item.badge.count}
					label={item.badge.label}
					variant={item.badge.variant ?? 'error'}
					size="xs"
				/>
			{/if}
		</a>
	{/if}
{/snippet}

<style>
	/* Base navbar styles */
	.navbar {
		z-index: 1000;
		pointer-events: none;
		transition:
			opacity 0.4s cubic-bezier(0.25, 1, 0.5, 1),
			transform 0.4s cubic-bezier(0.25, 1, 0.5, 1);
	}

	.navbar.hidden {
		opacity: 0;
		transform: translateY(-20px);
	}

	.navbar.visible {
		opacity: 1;
		transform: translateY(0);
	}

	/* Container with glassmorphism */
	.navbar-container {
		display: flex;
		align-items: center;
		gap: var(--navbar-gap, 2rem);
		padding: var(--navbar-padding, 0.75rem 1.5rem);
		pointer-events: auto;
	}

	/* Variant: Floating (finndollimore.com style) */
	.navbar-floating {
		position: fixed;
	}

	.navbar-floating .navbar-container {
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		background: var(--navbar-bg, rgba(16, 13, 20, 0.85));
		border: 1px solid var(--navbar-border, rgba(255, 255, 255, 0.1));
		border-radius: var(--navbar-radius, 0.75rem);
		box-shadow: var(--navbar-shadow, 0 8px 32px 0 rgba(0, 0, 0, 0.37));
	}

	/* Variant: Fixed (solid bar) */
	.navbar-fixed {
		position: fixed;
		width: 100%;
	}

	.navbar-fixed .navbar-container {
		background: var(--navbar-bg, #1a1a2e);
		border-bottom: 1px solid var(--navbar-border, #2a2a4a);
		max-width: 100%;
		justify-content: center;
	}

	/* Variant: Inline (embedded) */
	.navbar-inline {
		position: relative;
	}

	.navbar-inline .navbar-container {
		background: transparent;
	}

	/* Position variants */
	.navbar-top-center {
		top: var(--navbar-offset, 1.5rem);
		left: 50%;
		transform: translateX(-50%);
	}

	.navbar-top-center.hidden {
		transform: translateX(-50%) translateY(-20px);
	}

	.navbar-top-left {
		top: var(--navbar-offset, 1.5rem);
		left: var(--navbar-offset, 1.5rem);
	}

	.navbar-top-right {
		top: var(--navbar-offset, 1.5rem);
		right: var(--navbar-offset, 1.5rem);
	}

	.navbar-bottom-center {
		bottom: var(--navbar-offset, 1.5rem);
		left: 50%;
		transform: translateX(-50%);
	}

	/* Style: Horizontal */
	.navbar-horizontal .navbar-items {
		display: flex;
		align-items: center;
		gap: var(--navbar-item-gap, 0.5rem);
	}

	/* Style: Vertical */
	.navbar-vertical .navbar-container {
		flex-direction: column;
		align-items: flex-start;
	}

	.navbar-vertical .navbar-items {
		display: flex;
		flex-direction: column;
		gap: var(--navbar-item-gap, 0.25rem);
		width: 100%;
	}

	/* Logo */
	.navbar-logo {
		font-weight: 700;
		font-size: 1.125rem;
		color: var(--navbar-logo-color, var(--color-accent, #a78bfa));
		text-decoration: none;
		transition: color 0.2s ease;
		letter-spacing: -0.02em;
	}

	.navbar-logo:hover {
		color: var(--navbar-logo-hover, #c4b5fd);
	}

	/* Group label */
	.navbar-group-label {
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--navbar-group-color, rgba(255, 255, 255, 0.4));
		padding: 0.5rem 0.75rem;
		letter-spacing: 0.05em;
	}

	/* Nav items */
	.navbar-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.75rem;
		color: var(--navbar-text, rgba(255, 255, 255, 0.7));
		text-decoration: none;
		font-size: 0.9375rem;
		font-weight: 500;
		border-radius: 0.375rem;
		transition: all 0.2s ease;
		position: relative;
	}

	.navbar-item:hover:not(.disabled) {
		color: var(--navbar-text-hover, #ffffff);
		background: var(--navbar-item-hover-bg, rgba(255, 255, 255, 0.1));
	}

	.navbar-item.active {
		color: var(--navbar-text-active, #ffffff);
	}

	/* Active underline for horizontal */
	.navbar-horizontal .navbar-item::after {
		content: '';
		position: absolute;
		bottom: -2px;
		left: 0.75rem;
		right: 0.75rem;
		height: 2px;
		background: var(--navbar-accent, var(--color-accent, #a78bfa));
		transform: scaleX(0);
		transition: transform 0.3s ease;
	}

	.navbar-horizontal .navbar-item:hover::after,
	.navbar-horizontal .navbar-item.active::after {
		transform: scaleX(1);
	}

	/* Active background for vertical */
	.navbar-vertical .navbar-item.active {
		background: var(--navbar-active-bg, var(--color-accent, #6366f1));
	}

	/* Disabled state */
	.navbar-item.disabled {
		color: var(--navbar-text-disabled, rgba(255, 255, 255, 0.3));
		cursor: not-allowed;
		user-select: none;
	}

	/* Icon */
	.navbar-icon {
		font-size: 1.125em;
		width: 1.25rem;
		text-align: center;
	}

	/* Label */
	.navbar-label {
		white-space: nowrap;
	}

	/* Extra slot (auth, actions) */
	.navbar-extra {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-left: auto;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.navbar-floating {
			left: 1rem;
			right: 1rem;
			transform: none;
		}

		.navbar-floating.navbar-top-center {
			top: 1rem;
		}

		.navbar-floating .navbar-container {
			width: 100%;
			padding: 0.625rem 1rem;
			gap: 1rem;
			justify-content: space-between;
		}

		.navbar-logo {
			font-size: 1rem;
		}

		.navbar-items {
			gap: 0.25rem;
		}

		.navbar-item {
			padding: 0.25rem 0.5rem;
			font-size: 0.875rem;
		}

		.navbar-label {
			display: none;
		}

		.navbar-icon {
			font-size: 1.25rem;
		}
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.navbar,
		.navbar-item,
		.navbar-item::after {
			transition-duration: 0.01ms !important;
		}
	}
</style>

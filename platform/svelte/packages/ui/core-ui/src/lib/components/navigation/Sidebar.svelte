<script lang="ts">
	/**
	 * Sidebar Component
	 *
	 * Generic sidebar that accepts navigation items from fixtures or direct props.
	 * Supports multiple content types: navigation items, sections, or custom content.
	 *
	 * Architecture:
	 * - Fixture-driven: Items come from navigation fixtures (hub-main-nav.yml)
	 * - Extensible: Custom content via snippet for filetree, etc.
	 * - IView conformance: data-view-id for dev mode
	 *
	 * Usage:
	 * <Sidebar items={navItems} position="left" />
	 * <Sidebar sections={sections} position="left" />
	 * <Sidebar position="left">{#snippet content()}<FileTree />{/snippet}</Sidebar>
	 *
	 * @see navigation.yml schema for item definitions
	 * @see IPresenter interface for conformance
	 */

	import { page } from '$app/stores';
	import type { Snippet } from 'svelte';
	import Badge from '../primitives/Badge.svelte';

	// =============================================================================
	// TYPE DEFINITIONS
	// =============================================================================

	/** Badge configuration for sidebar items */
	interface SidebarBadge {
		/** Text label or count */
		label?: string;
		count?: number;
		/** Badge variant for styling */
		variant?: 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info';
	}

	/** Individual sidebar navigation item */
	export interface SidebarItem {
		/** Unique identifier */
		id: string;
		/** Display text */
		label: string;
		/** Navigation href */
		href: string;
		/** Icon (emoji or component name) */
		icon?: string;
		/** Optional badge */
		badge?: SidebarBadge;
		/** Disabled state */
		disabled?: boolean;
		/** Transition hint for PageTransition */
		transition?: string;
	}

	/** Group of sidebar items (matches navigation fixture groups) */
	export interface SidebarSection {
		/** Section identifier */
		id: string;
		/** Optional section title */
		title?: string;
		/** Items in this section */
		items: SidebarItem[];
		/** Whether section can be collapsed (default: true if has title) */
		collapsible?: boolean;
		/** Initial collapsed state (default: false) */
		defaultCollapsed?: boolean;
	}

	/** Header configuration */
	interface SidebarHeader {
		/** Title text */
		title?: string;
		/** Logo image URL */
		logo?: string;
		/** Logo href (defaults to /) */
		href?: string;
	}

	// =============================================================================
	// PROPS
	// =============================================================================

	interface SidebarProps {
		/** Flat list of items (simple mode) */
		items?: SidebarItem[];
		/** Grouped sections (advanced mode, matches fixture format) */
		sections?: SidebarSection[];
		/** Sidebar position */
		position?: 'left' | 'right';
		/** Collapsed state */
		collapsed?: boolean;
		/** Header configuration */
		header?: SidebarHeader;
		/** Width when expanded (px) */
		width?: number;
		/** Width when collapsed (px) */
		collapsedWidth?: number;
		/** Show collapse button */
		showCollapseButton?: boolean;
		/** Custom content snippet (for filetree, etc.) */
		content?: Snippet;
		/** Footer snippet */
		footer?: Snippet;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
		/** Callback when collapse state changes */
		onCollapseChange?: (collapsed: boolean) => void;
	}

	let {
		items = [],
		sections = [],
		position = 'left',
		collapsed = false,
		header,
		width = 240,
		collapsedWidth = 60,
		showCollapseButton = true,
		content,
		footer,
		class: className = '',
		'data-view-id': viewId,
		onCollapseChange
	}: SidebarProps = $props();

	// =============================================================================
	// STATE
	// =============================================================================

	// svelte-ignore state_referenced_locally
	let internalCollapsed = $state(collapsed);

	// Track collapsed sections by id
	let collapsedSections = $state<Set<string>>(new Set());

	// Initialize collapsed sections from defaults
	$effect(() => {
		const initialCollapsed = new Set<string>();
		for (const section of sections) {
			if (section.defaultCollapsed && section.title) {
				initialCollapsed.add(section.id);
			}
		}
		collapsedSections = initialCollapsed;
	});

	// Sync external collapsed prop
	$effect(() => {
		internalCollapsed = collapsed;
	});

	// Current path for active state detection
	let currentPath = $derived($page.url.pathname);

	// Check if section is collapsed
	function isSectionCollapsed(sectionId: string): boolean {
		return collapsedSections.has(sectionId);
	}

	// Toggle section collapsed state
	function toggleSection(sectionId: string) {
		const newSet = new Set(collapsedSections);
		if (newSet.has(sectionId)) {
			newSet.delete(sectionId);
		} else {
			newSet.add(sectionId);
		}
		collapsedSections = newSet;
	}

	// Normalize items to sections format for unified rendering
	let normalizedSections = $derived.by(() => {
		if (sections.length > 0) {
			return sections;
		}
		if (items.length > 0) {
			return [{ id: 'main', items }] as SidebarSection[];
		}
		return [];
	});

	// =============================================================================
	// METHODS
	// =============================================================================

	function isActive(href: string): boolean {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}

	function toggleCollapse() {
		internalCollapsed = !internalCollapsed;
		onCollapseChange?.(internalCollapsed);
	}

	function handleNavClick(item: SidebarItem, event: MouseEvent) {
		if (item.disabled) {
			event.preventDefault();
			return;
		}
		// Set transition hint for PageTransition
		if (item.transition && typeof document !== 'undefined') {
			document.documentElement.dataset.navTransition = item.transition;
		}
	}

	// CSS custom properties for sizing
	let sidebarVars = $derived({
		'--sidebar-width': `${width}px`,
		'--sidebar-collapsed-width': `${collapsedWidth}px`
	});
</script>

<aside
	class="sidebar sidebar-{position} {className}"
	class:collapsed={internalCollapsed}
	style={Object.entries(sidebarVars)
		.map(([k, v]) => `${k}: ${v}`)
		.join('; ')}
	data-view-id={viewId}
>
	<!-- Header -->
	{#if header}
		<div class="sidebar-header">
			<a href={header.href ?? '/'} class="header-link">
				{#if header.logo}
					<img src={header.logo} alt={header.title ?? 'Logo'} class="header-logo" />
				{/if}
				{#if !internalCollapsed && header.title}
					<span class="header-title">{header.title}</span>
				{/if}
			</a>
		</div>
	{/if}

	<!-- Content Area -->
	<div class="sidebar-content">
		{#if content}
			<!-- Custom content mode (filetree, etc.) -->
			{@render content()}
		{:else}
			<!-- Navigation items mode -->
			<nav class="sidebar-nav" aria-label={position === 'left' ? 'Main navigation' : 'Secondary navigation'}>
				{#each normalizedSections as section (section.id)}
					{@const isCollapsible = section.collapsible !== false && section.title}
					{@const sectionIsCollapsed = isSectionCollapsed(section.id)}
					{#if section.title && !internalCollapsed}
						<button
							class="section-title"
							class:collapsible={isCollapsible}
							class:collapsed={sectionIsCollapsed}
							onclick={() => isCollapsible && toggleSection(section.id)}
							aria-expanded={isCollapsible ? !sectionIsCollapsed : undefined}
						>
							<span class="section-title-text">{section.title}</span>
							{#if isCollapsible}
								<span class="section-chevron">{sectionIsCollapsed ? '▸' : '▾'}</span>
							{/if}
						</button>
					{/if}
					{#if !sectionIsCollapsed || internalCollapsed || !section.title}
						<ul class="nav-items" class:section-collapsed={sectionIsCollapsed && !internalCollapsed}>
							{#each section.items as item (item.id)}
								<li class="nav-item">
									<a
										href={item.href}
										class="nav-link"
										class:active={isActive(item.href)}
										class:disabled={item.disabled}
										title={internalCollapsed ? item.label : undefined}
										aria-disabled={item.disabled}
										onclick={(e) => handleNavClick(item, e)}
									>
										{#if item.icon}
											<span class="nav-icon">{item.icon}</span>
										{/if}
										{#if !internalCollapsed}
											<span class="nav-label">{item.label}</span>
											{#if item.badge}
												<Badge
													variant={item.badge.variant ?? 'primary'}
													size="sm"
												>
													{item.badge.count ?? item.badge.label}
												</Badge>
											{/if}
										{/if}
									</a>
								</li>
							{/each}
						</ul>
					{/if}
				{/each}
			</nav>
		{/if}
	</div>

	<!-- Footer -->
	<div class="sidebar-footer">
		{#if footer}
			{@render footer()}
		{/if}
		{#if showCollapseButton}
			<button
				class="collapse-btn"
				onclick={toggleCollapse}
				title={internalCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
				aria-expanded={!internalCollapsed}
			>
				<span class="collapse-icon">
					{#if position === 'left'}
						{internalCollapsed ? '→' : '←'}
					{:else}
						{internalCollapsed ? '←' : '→'}
					{/if}
				</span>
			</button>
		{/if}
	</div>
</aside>

<style>
	/* ==========================================================================
	   BASE SIDEBAR
	   ========================================================================== */
	.sidebar {
		display: flex;
		flex-direction: column;
		width: var(--sidebar-width, 240px);
		height: 100vh;
		background: var(--color-bg-secondary, #1a1a2e);
		position: sticky;
		top: 0;
		transition: width 0.2s ease;
	}

	.sidebar.collapsed {
		width: var(--sidebar-collapsed-width, 60px);
	}

	/* Position variants */
	.sidebar-left {
		border-right: 1px solid var(--color-border, #2a2a4a);
	}

	.sidebar-right {
		border-left: 1px solid var(--color-border, #2a2a4a);
	}

	/* ==========================================================================
	   HEADER
	   ========================================================================== */
	.sidebar-header {
		padding: 1rem;
		border-bottom: 1px solid var(--color-border, #2a2a4a);
	}

	.header-link {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		text-decoration: none;
		color: var(--color-text, #fff);
	}

	.header-logo {
		width: 32px;
		height: 32px;
		flex-shrink: 0;
	}

	.header-title {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--color-accent, #6366f1);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.collapsed .sidebar-header {
		padding: 1rem 0.5rem;
		justify-content: center;
	}

	/* ==========================================================================
	   CONTENT AREA
	   ========================================================================== */
	.sidebar-content {
		flex: 1;
		padding: 0.5rem 0;
		overflow-y: auto;
		overflow-x: hidden;
	}

	/* ==========================================================================
	   NAVIGATION
	   ========================================================================== */
	.sidebar-nav {
		padding: 0.5rem 0;
	}

	.section-title {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0.5rem 1rem;
		margin: 0;
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-text-muted, #6b7280);
		background: transparent;
		border: none;
		text-align: left;
		cursor: default;
	}

	.section-title.collapsible {
		cursor: pointer;
		transition: color 0.15s ease;
	}

	.section-title.collapsible:hover {
		color: var(--color-text, #fff);
	}

	.section-title-text {
		flex: 1;
	}

	.section-chevron {
		font-size: 0.625rem;
		opacity: 0.6;
		transition: transform 0.2s ease;
	}

	.section-title.collapsed .section-chevron {
		transform: rotate(0deg);
	}

	.nav-items.section-collapsed {
		display: none;
	}

	.nav-items {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.nav-item {
		margin: 0.25rem 0.5rem;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
		color: var(--color-text-muted, #9ca3af);
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.nav-link:hover:not(.disabled) {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.nav-link.active {
		background: var(--color-accent, #6366f1);
		color: #fff;
	}

	.nav-link.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.nav-icon {
		font-size: 1.25rem;
		width: 1.5rem;
		text-align: center;
		flex-shrink: 0;
	}

	.nav-label {
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* Collapsed state for nav */
	.collapsed .nav-link {
		justify-content: center;
		padding: 0.75rem;
	}

	.collapsed .nav-item {
		margin: 0.25rem;
	}

	/* ==========================================================================
	   FOOTER
	   ========================================================================== */
	.sidebar-footer {
		padding: 0.5rem;
		border-top: 1px solid var(--color-border, #2a2a4a);
	}

	.collapse-btn {
		width: 100%;
		padding: 0.5rem;
		background: transparent;
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: 0.375rem;
		color: var(--color-text-muted, #9ca3af);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.collapse-btn:hover {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.collapse-icon {
		font-size: 1rem;
	}

	/* ==========================================================================
	   REDUCED MOTION
	   ========================================================================== */
	@media (prefers-reduced-motion: reduce) {
		.sidebar,
		.nav-link,
		.collapse-btn {
			transition-duration: 0.01ms !important;
		}
	}
</style>

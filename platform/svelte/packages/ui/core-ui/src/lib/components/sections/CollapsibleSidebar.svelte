<script lang="ts">
	/**
	 * CollapsibleSidebar Section Component
	 *
	 * Space-efficient sidebar with auto-collapsing NavRail. ContextPanel is primary,
	 * NavRail auto-collapses and expands on hover near edge for more screen space.
	 *
	 * Architecture:
	 * - NavRail (48-72px): Icon navigation, auto-collapses by default
	 * - ContextPanel (200-240px): Primary content panel, always visible unless collapsed
	 * - Rail expands on hover near context panel edge (hover zone)
	 * - Context panel can be collapsed via CTA
	 * - IView conformance: data-view-id attribute
	 *
	 * @example
	 * <CollapsibleSidebar
	 *   railItems={navItems}
	 *   contextSections={sections}
	 *   railAutoCollapse={true}
	 *   onrailselect={(item) => console.log(item)}
	 * />
	 */

	import NavRail from '../blocks/NavRail.svelte';
	import type { NavRailItem } from '../blocks/NavRail.svelte';
	import ContextPanel from './ContextPanel.svelte';
	import type { ContextSection, TreeItem } from './ContextPanel.svelte';
	import Icon from '../primitives/Icon.svelte';
	import IconButton from '../primitives/IconButton.svelte';

	// =============================================================================
	// TYPE DEFINITIONS
	// =============================================================================

	/** Expansion state for the sidebar */
	export type ExpansionState = 'collapsed' | 'expanded' | 'pinned';

	/** Rail visibility state */
	export type RailState = 'hidden' | 'visible' | 'pinned';

	export interface CollapsibleSidebarProps {
		/** Rail navigation items (icon-only) */
		railItems?: NavRailItem[];
		/** Context panel sections (shown when expanded) */
		contextSections?: ContextSection[];
		/** Currently active rail item ID */
		activeRailItem?: string;
		/** Currently active context section */
		activeSection?: string;
		/** Expanded state (shows context panel) - legacy prop */
		expanded?: boolean;
		/** Current expansion state (collapsed | expanded | pinned) */
		expansionState?: ExpansionState;
		/** Header configuration */
		header?: {
			logo?: string;
			title?: string;
			collapsed?: boolean;
		};
		/** Rail width in pixels (default: 48px per spec) */
		railWidth?: number;
		/** Context panel width in pixels (default: 200px per spec) */
		contextWidth?: number;
		/** Position (left or right) */
		position?: 'left' | 'right';
		/** Show labels under nav rail icons instead of tooltips */
		showRailLabels?: boolean;
		/** Show expand/collapse button for context panel */
		showToggle?: boolean;
		/** Allow pinning the rail open */
		allowPin?: boolean;
		/** Auto-collapse rail (expand on hover near context edge) */
		railAutoCollapse?: boolean;
		/** Delay before expanding rail on hover (ms) */
		hoverDelay?: number;
		/** Width of hover zone to trigger rail expansion (px) */
		hoverZoneWidth?: number;
		/** Callback when expanded state changes (legacy) */
		onexpand?: (expanded: boolean) => void;
		/** Callback when state changes */
		onstatechange?: (state: ExpansionState) => void;
		/** Callback when rail item is clicked */
		onrailselect?: (item: NavRailItem) => void;
		/** Callback when context item is clicked */
		oncontextselect?: (item: TreeItem, sectionId: string) => void;
		/** Callback when context action is triggered */
		oncontextaction?: (sectionId: string, actionId: string) => void;
		/** Additional CSS class */
		class?: string;
		/** IView data attribute */
		'data-view-id'?: string;
	}

	// =============================================================================
	// PROPS
	// =============================================================================

	let {
		railItems = [],
		contextSections = [],
		activeRailItem = '',
		activeSection = '',
		expanded = true,
		expansionState,
		header,
		railWidth = 48,
		contextWidth = 200,
		position = 'left',
		showRailLabels = false,
		showToggle = true,
		allowPin = true,
		railAutoCollapse = true,
		hoverDelay = 150,
		hoverZoneWidth = 12,
		onexpand,
		onstatechange,
		onrailselect,
		oncontextselect,
		oncontextaction,
		class: className = '',
		'data-view-id': viewId
	}: CollapsibleSidebarProps = $props();

	// Auto-widen rail when showing labels
	let effectiveRailWidth = $derived(showRailLabels ? 72 : railWidth);

	// =============================================================================
	// STATE
	// =============================================================================

	// Context panel collapse state (user-controlled via CTA)
	let contextCollapsed = $state(false);

	// Rail visibility state (auto-collapse behavior)
	// svelte-ignore state_referenced_locally
	let railVisible = $state(!railAutoCollapse);
	let railPinned = $state(false);

	// Hover tracking
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;
	let collapseTimeout: ReturnType<typeof setTimeout> | null = null;
	let isHoveringRail = $state(false);
	let isHoveringZone = $state(false);

	// Legacy expansion state for compatibility
	let currentExpansionState = $derived.by<ExpansionState>(() =>
		expansionState ?? (expanded ? 'expanded' : 'collapsed')
	);

	// Cleanup timeouts on unmount
	$effect(() => {
		return () => {
			if (hoverTimeout) clearTimeout(hoverTimeout);
			if (collapseTimeout) clearTimeout(collapseTimeout);
		};
	});

	// =============================================================================
	// DERIVED
	// =============================================================================

	/** Whether context panel is visible */
	let isContextVisible = $derived(!contextCollapsed);

	/** Whether rail is visible (pinned, manually shown, or hovering) */
	let isRailVisible = $derived(railPinned || railVisible || !railAutoCollapse);

	/** Effective active section - reactive computation for ContextPanel */
	let effectiveActiveSection = $derived(activeSection || activeRailItem);

	// Calculate total width based on what's visible
	let totalWidth = $derived.by(() => {
		let width = 0;
		if (isRailVisible) width += effectiveRailWidth;
		if (isContextVisible) width += contextWidth;
		// Add hover zone width when rail is hidden but context is visible
		if (!isRailVisible && isContextVisible && railAutoCollapse) width += hoverZoneWidth;
		return width;
	});

	// CSS custom properties
	let cssVars = $derived({
		'--rail-width': `${effectiveRailWidth}px`,
		'--context-width': `${contextWidth}px`,
		'--total-width': `${totalWidth}px`,
		'--hover-zone-width': `${hoverZoneWidth}px`
	});

	// =============================================================================
	// HANDLERS
	// =============================================================================

	function handleRailSelect(id: string) {
		const item = railItems.find((i) => i.id === id);
		if (item) {
			onrailselect?.(item);
		}
	}

	/** Toggle context panel collapse (via CTA button) */
	function handleContextToggle() {
		contextCollapsed = !contextCollapsed;
		// Notify parent for legacy compatibility
		onexpand?.(!contextCollapsed);
		onstatechange?.(contextCollapsed ? 'collapsed' : 'expanded');
	}

	/** Pin/unpin the rail */
	function handleRailPinToggle() {
		if (!allowPin) return;
		railPinned = !railPinned;
		if (railPinned) {
			railVisible = true;
		}
	}

	/** Hover zone mouse enter - start showing rail */
	function handleHoverZoneEnter() {
		if (!railAutoCollapse || railPinned) return;

		isHoveringZone = true;
		if (collapseTimeout) {
			clearTimeout(collapseTimeout);
			collapseTimeout = null;
		}

		hoverTimeout = setTimeout(() => {
			if (isHoveringZone) {
				railVisible = true;
			}
		}, hoverDelay);
	}

	/** Hover zone mouse leave */
	function handleHoverZoneLeave() {
		isHoveringZone = false;
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = null;
		}
	}

	/** Rail mouse enter */
	function handleRailMouseEnter() {
		isHoveringRail = true;
		if (collapseTimeout) {
			clearTimeout(collapseTimeout);
			collapseTimeout = null;
		}
	}

	/** Rail mouse leave - start collapse timer */
	function handleRailMouseLeave() {
		isHoveringRail = false;
		if (!railAutoCollapse || railPinned) return;

		collapseTimeout = setTimeout(() => {
			if (!isHoveringRail && !isHoveringZone) {
				railVisible = false;
			}
		}, 300);
	}

	/** Keyboard shortcuts */
	function handleKeydown(event: KeyboardEvent) {
		// Cmd/Ctrl + B to toggle context panel
		if ((event.metaKey || event.ctrlKey) && event.key === 'b') {
			event.preventDefault();
			handleContextToggle();
		}
		// Cmd/Ctrl + [ to toggle rail visibility
		if ((event.metaKey || event.ctrlKey) && event.key === '[') {
			event.preventDefault();
			railVisible = !railVisible;
		}
		// Cmd/Ctrl + Shift + P to toggle rail pin
		if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key === 'p') {
			event.preventDefault();
			handleRailPinToggle();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<aside
	class="collapsible-sidebar {position} {className}"
	class:rail-visible={isRailVisible}
	class:rail-pinned={railPinned}
	class:context-collapsed={contextCollapsed}
	class:rail-auto-collapse={railAutoCollapse}
	style={Object.entries(cssVars)
		.map(([k, v]) => `${k}: ${v}`)
		.join('; ')}
	data-view-id={viewId}
	aria-label="Collapsible sidebar"
>
	<!-- Header (optional) -->
	{#if header && !header.collapsed}
		<div class="sidebar-header">
			{#if header.logo}
				<img src={header.logo} alt={header.title || 'Logo'} class="header-logo" />
			{/if}
			{#if isContextVisible && header.title}
				<span class="header-title">{header.title}</span>
			{/if}
		</div>
	{/if}

	<div class="sidebar-body">
		<!-- Hover Zone (visible when rail is hidden, triggers rail expansion) -->
		{#if railAutoCollapse && !isRailVisible && isContextVisible}
			<div
				class="hover-zone"
				onmouseenter={handleHoverZoneEnter}
				onmouseleave={handleHoverZoneLeave}
				role="button"
				tabindex="0"
				aria-label="Show navigation rail"
				title="Hover to show navigation"
			>
				<div class="hover-zone-indicator"></div>
			</div>
		{/if}

		<!-- Nav Rail (conditionally visible) -->
		{#if isRailVisible}
			<div
				class="rail-container"
				onmouseenter={handleRailMouseEnter}
				onmouseleave={handleRailMouseLeave}
				role="navigation"
				aria-label="Mode navigation"
			>
				<NavRail
					items={railItems}
					activeId={activeRailItem}
					width={effectiveRailWidth}
					showLabels={showRailLabels}
					onSelect={handleRailSelect}
					data-view-id={viewId ? `${viewId}-rail` : undefined}
				/>

				<!-- Pin button at bottom of rail -->
				{#if allowPin && railAutoCollapse}
					<button
						class="pin-btn"
						onclick={handleRailPinToggle}
						title={railPinned ? 'Unpin rail (Cmd+Shift+P)' : 'Pin rail (Cmd+Shift+P)'}
						aria-pressed={railPinned}
						aria-label={railPinned ? 'Unpin navigation rail' : 'Pin navigation rail'}
					>
						<Icon name={railPinned ? 'pin-off' : 'pin'} size="sm" />
					</button>
				{/if}
			</div>
		{/if}

		<!-- Context Panel (primary, always visible unless collapsed) -->
		{#if isContextVisible}
			<div class="context-container">
				<!-- Collapse toggle in context panel header -->
				{#if showToggle}
					<div class="context-header-actions">
						<IconButton
							icon={position === 'left' ? 'panel-left-close' : 'panel-right-close'}
							label="Collapse panel (Cmd+B)"
							variant="ghost"
							size="sm"
							onclick={handleContextToggle}
						/>
					</div>
				{/if}
				<ContextPanel
					sections={contextSections}
					activeSection={activeRailItem}
					width={contextWidth}
					collapsible={false}
					collapsed={false}
					resizable={true}
					onItemSelect={oncontextselect}
					onAction={oncontextaction}
					data-view-id={viewId ? `${viewId}-context` : undefined}
				/>
			</div>
		{:else}
			<!-- Collapsed state: show expand button -->
			<div class="collapsed-indicator">
				<button
					class="expand-btn"
					onclick={handleContextToggle}
					title="Expand panel (Cmd+B)"
					aria-label="Expand panel"
				>
					<Icon name={position === 'left' ? 'panel-left-open' : 'panel-right-open'} size="sm" />
				</button>
			</div>
		{/if}
	</div>
</aside>

<style>
	/* ==========================================================================
	   BASE STYLES
	   ========================================================================== */
	.collapsible-sidebar {
		display: flex;
		flex-direction: column;
		width: var(--total-width, 200px);
		height: 100vh;
		background: var(--sidebar-bg, var(--color-bg-secondary, #0f0f1a));
		position: sticky;
		top: 0;
		transition: width 200ms ease-out;
		z-index: 100;
		overflow: visible;
	}

	.collapsible-sidebar.context-collapsed {
		width: 40px;
	}

	.collapsible-sidebar.left {
		border-right: 1px solid var(--sidebar-border, var(--color-border, rgba(255, 255, 255, 0.08)));
	}

	.collapsible-sidebar.right {
		border-left: 1px solid var(--sidebar-border, var(--color-border, rgba(255, 255, 255, 0.08)));
	}

	/* ==========================================================================
	   HEADER
	   ========================================================================== */
	.sidebar-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--color-border, rgba(255, 255, 255, 0.08));
		min-height: 48px;
	}

	.header-logo {
		width: 24px;
		height: 24px;
		flex-shrink: 0;
	}

	.header-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-text, #fff);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* ==========================================================================
	   BODY LAYOUT
	   ========================================================================== */
	.sidebar-body {
		display: flex;
		flex: 1;
		overflow: visible;
		position: relative;
	}

	/* ==========================================================================
	   HOVER ZONE (triggers rail expansion)
	   ========================================================================== */
	.hover-zone {
		position: absolute;
		left: 0;
		top: 0;
		width: var(--hover-zone-width, 12px);
		height: 100%;
		z-index: 50;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.hover-zone-indicator {
		width: 3px;
		height: 48px;
		background: var(--color-border, rgba(255, 255, 255, 0.1));
		border-radius: 2px;
		opacity: 0;
		transition: opacity 150ms ease, background 150ms ease;
	}

	.hover-zone:hover .hover-zone-indicator {
		opacity: 1;
		background: var(--color-accent, #6366f1);
	}

	.right .hover-zone {
		left: auto;
		right: 0;
	}

	/* ==========================================================================
	   RAIL CONTAINER (slides in from left)
	   ========================================================================== */
	.rail-container {
		display: flex;
		flex-direction: column;
		width: var(--rail-width, 48px);
		min-width: var(--rail-width, 48px);
		flex-shrink: 0;
		border-right: 1px solid var(--rail-border, var(--color-border, rgba(255, 255, 255, 0.08)));
		background: var(--rail-bg, var(--color-bg-tertiary, #0a0a12));
		animation: slideInLeft 200ms ease-out;
	}

	.right .rail-container {
		order: 2;
		border-right: none;
		border-left: 1px solid var(--rail-border, var(--color-border, rgba(255, 255, 255, 0.08)));
		animation: slideInRight 200ms ease-out;
	}

	@keyframes slideInLeft {
		from {
			transform: translateX(-100%);
			opacity: 0;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	@keyframes slideInRight {
		from {
			transform: translateX(100%);
			opacity: 0;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	/* ==========================================================================
	   PIN BUTTON (in rail)
	   ========================================================================== */
	.pin-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		margin: auto auto 0.5rem;
		background: transparent;
		border: 1px solid var(--color-border, rgba(255, 255, 255, 0.1));
		border-radius: 6px;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		transition: background 150ms ease, color 150ms ease, border-color 150ms ease;
	}

	.pin-btn:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--color-text, #fff);
		border-color: var(--color-border-hover, rgba(255, 255, 255, 0.15));
	}

	.pin-btn:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	.pin-btn[aria-pressed='true'] {
		background: var(--color-accent, #6366f1);
		border-color: var(--color-accent, #6366f1);
		color: white;
	}

	/* ==========================================================================
	   CONTEXT CONTAINER (primary panel)
	   ========================================================================== */
	.context-container {
		position: relative;
		flex: 1;
		overflow: hidden;
		background: var(--sidebar-bg, var(--color-bg-secondary, #0f0f1a));
	}

	.right .context-container {
		order: 1;
	}

	/* Context header actions (collapse button) */
	.context-header-actions {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	/* ==========================================================================
	   COLLAPSED INDICATOR (when context is hidden)
	   ========================================================================== */
	.collapsed-indicator {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-start;
		padding-top: 0.5rem;
		width: 40px;
		background: var(--sidebar-bg, var(--color-bg-secondary, #0f0f1a));
	}

	.expand-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		background: transparent;
		border: 1px solid var(--color-border, rgba(255, 255, 255, 0.1));
		border-radius: 6px;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		transition: background 150ms ease, color 150ms ease, border-color 150ms ease;
	}

	.expand-btn:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--color-text, #fff);
		border-color: var(--color-border-hover, rgba(255, 255, 255, 0.15));
	}

	.expand-btn:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	/* ==========================================================================
	   REDUCED MOTION
	   ========================================================================== */
	@media (prefers-reduced-motion: reduce) {
		.collapsible-sidebar,
		.context-container,
		.rail-container,
		.hover-zone-indicator,
		.pin-btn,
		.expand-btn {
			transition-duration: 0.01ms !important;
			animation-duration: 0.01ms !important;
		}
	}
</style>

<script lang="ts">
	/**
	 * SidebarLayout — Unified layout shell for all sidebar routes.
	 * Owns the flex container, sticky sidebar, mobile drawer, and toolbar row.
	 * Routes compose their content via Svelte 5 snippets.
	 *
	 * Desktop: sidebar collapse/expand persisted to localStorage (component-wide).
	 * Mobile: single contextual section toggle inline in the toolbar row —
	 *         avoids colliding with a global header hamburger.
	 *
	 * @example
	 * <SidebarLayout>
	 *   {#snippet sidebar()}
	 *     <ContentSidebar sections={navSections} {currentPath} />
	 *   {/snippet}
	 *   {#snippet toolbar()}
	 *     <Breadcrumb items={breadcrumb} />
	 *   {/snippet}
	 *   {#snippet actions()}
	 *     <button>Save</button><button>Publish</button>
	 *   {/snippet}
	 *   {@render children()}
	 * </SidebarLayout>
	 */
	import type { Snippet } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { layoutOffset } from '../../stores/layout';
	import { BREAKPOINTS, type BreakpointName } from '../../breakpoints';
	import DotNav from '../navigation/DotNav.svelte';

	let {
		children,
		sidebar,
		toolbar,
		actions,
		sidebarWidth = 240,
		maxWidth = 1200,
		storageKey = 'sidebar-collapsed',
		shell = false,
		hasSidebar = true,
		hasToolbar = true,
		stickyToolbar = false,
		floatingToolbar = false,
		dotNav = false,
		sidebarOverlayBreakpoint = 'desktop',
		collapsed = $bindable(false),
	}: {
		children: Snippet;
		sidebar?: Snippet;
		toolbar?: Snippet;
		/** Right-aligned region of the toolbar row — page-level CTAs like Save / Publish.
		 *  Renders only when the toolbar row is visible (`hasToolbar !== false`). */
		actions?: Snippet;
		sidebarWidth?: number;
		maxWidth?: number;
		/** localStorage key for collapse state — shared global default.
		 *  All sidebar pages use the same key for consistent cross-page UX.
		 *  Only override for truly independent sidebars (e.g. admin panels). */
		storageKey?: string;
		/** Shell mode — zero-pads layout-content AND removes its max-width
		 *  (sections inside become full-bleed for proper stripe backgrounds).
		 *  Used by PageShell to hand all geometry control to <Section>. */
		shell?: boolean;
		/** Whether sidebar content exists — hides mobile Sections toggle when false */
		hasSidebar?: boolean;
		/** When false, suppresses the toolbar slot entirely (used for context-based pages
		 *  that render breadcrumb/filter chrome inline inside the page content).
		 *  On desktop this also hides the toolbar row (no reserved space). Default true. */
		hasToolbar?: boolean;
		/** When true, the toolbar row docks sticky under the global nav using --sidebar-top.
		 *  Drops the 1.5rem margin-bottom so content sits flush beneath. Default false. */
		stickyToolbar?: boolean;
		/** When true, the toolbar floats as a position:fixed overlay on the top-right
		 *  on desktop, reserving no block in the layout flow. Mobile stays in-flow
		 *  (the drawer toggle needs the row). Default false. */
		floatingToolbar?: boolean;
		/** When true, renders a <DotNav /> that auto-discovers [data-dot-nav-section]
		 *  nodes inside the layout. Sections emit that attribute from their `id`
		 *  prop automatically — pages need no extra wiring. Default false. */
		dotNav?: boolean;
		/** Below this named breakpoint the docked sidebar overlays content as a
		 *  drawer instead of pushing it via margin-left. Default 'desktop' keeps
		 *  today's behavior (drawer below 769). 'wide' makes the sidebar overlay
		 *  below 1280 — for narrow-desktop admin shells where a pushing column
		 *  would crush the content (Wave 2 / B2). Pass the NAME, never a number. */
		sidebarOverlayBreakpoint?: BreakpointName;
		/** Desktop sidebar collapsed state. Bindable so a host toolbar (e.g. a
		 *  hamburger in the page chrome) can drive show/hide while the component
		 *  still persists + restores it via storageKey and its own collapse chrome.
		 *  Additive — consumers that omit it keep the default (expanded), identical
		 *  to before. Default false. */
		collapsed?: boolean;
	} = $props();

	let drawerOpen = $state(false);

	// Static opt-in marker for the overlay regime — derived from the prop (not from
	// matchMedia), so it's SSR-stable and never causes a hydration flash. The width
	// gating lives entirely in the CSS media band keyed off this class.
	const overlayWide = $derived(sidebarOverlayBreakpoint === 'wide');

	// Restore persisted collapse state AFTER mount (desktop only). Reading
	// localStorage during the initial render makes the client compute a collapsed
	// rail the server rendered expanded → hydration_mismatch, which drops the first
	// sidebar interactions until Svelte rebuilds the subtree. $effect runs
	// post-hydration in the browser only, so SSR + the first client render both
	// start expanded and agree; the persisted state applies one frame later (a
	// brief expand→collapse for collapse-preference users — accepted tradeoff). It
	// re-reads if storageKey changes. No-flash alternative for a future need: read
	// the state from a cookie server-side and pass it in as an initial prop.
	$effect(() => {
		try {
			collapsed = localStorage.getItem(storageKey) === 'true';
		} catch {}
	});

	function openDrawer() {
		drawerOpen = true;
	}

	function closeDrawer() {
		drawerOpen = false;
	}

	function toggleCollapsed() {
		collapsed = !collapsed;
		try { localStorage.setItem(storageKey, String(collapsed)); } catch {}
	}

	$effect(() => {
		if (drawerOpen) {
			const prev = document.body.style.overflow;
			document.body.style.overflow = 'hidden';
			return () => { document.body.style.overflow = prev; };
		}
	});

	// Content is only pushed (offset) when the sidebar is an in-flow column — i.e.
	// the viewport is at or above the overlay breakpoint AND the sidebar isn't
	// collapsed. Below it the sidebar overlays content, so the offset is 0. With
	// the default 'desktop' breakpoint this is `>= 769` === the old `> 768` check.
	function contentOffset(): number {
		const pushing = window.innerWidth >= BREAKPOINTS[sidebarOverlayBreakpoint];
		return (!collapsed && pushing) ? sidebarWidth : 0;
	}

	$effect(() => {
		if (typeof window !== 'undefined') {
			// Read reactive deps so the effect re-runs on collapse / prop change.
			void [collapsed, sidebarWidth, sidebarOverlayBreakpoint];
			layoutOffset.set(contentOffset());
			return () => { layoutOffset.set(0); };
		}
	});

	function handleResize() {
		layoutOffset.set(contentOffset());
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && drawerOpen) {
			closeDrawer();
		}
	}

	afterNavigate(() => {
		if (drawerOpen) drawerOpen = false;
	});
</script>

<svelte:window onkeydown={handleKeydown} onresize={handleResize} />

{#if drawerOpen}
	<button class="sidebar-backdrop" class:overlay-wide={overlayWide} aria-label="Close navigation" onclick={closeDrawer}></button>
{/if}

<!-- Desktop expand tab — shown only when sidebar is collapsed on desktop -->
{#if collapsed}
	<button
		class="sidebar-expand-tab"
		class:overlay-wide={overlayWide}
		aria-label="Expand navigation"
		title="Expand navigation"
		onclick={toggleCollapsed}
	>
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
			<polyline points="9 18 15 12 9 6"/>
		</svg>
	</button>
{/if}

<div
	class="sidebar-layout"
	class:collapsed
	class:shell
	class:sticky-toolbar={stickyToolbar}
	class:floating-toolbar={floatingToolbar}
	class:overlay-wide={overlayWide}
	style="--sidebar-width: {sidebarWidth}px; --layout-max-width: {maxWidth}px;"
>
	<div class="sidebar-column" class:open={drawerOpen}>
		<!-- Mobile close button inside the drawer -->
		<button
			class="sidebar-drawer-close"
			aria-label="Close navigation"
			onclick={closeDrawer}
		>
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
				<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
			</svg>
		</button>

		{#if sidebar}{@render sidebar()}{/if}

		<!-- Desktop collapse button — positioned to align with sidebar-hdr row -->
		<button
			class="sidebar-collapse-btn"
			aria-label="Collapse navigation"
			title="Collapse navigation"
			onclick={toggleCollapsed}
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
				<polyline points="15 18 9 12 15 6"/>
			</svg>
		</button>
	</div>

	<div class="layout-content">
		<div class="layout-toolbar" class:no-toolbar-slot={(!toolbar && !actions) || !hasToolbar}>
			<!-- Inline section toggle — contextual control, shown only on mobile.
			     Avoids conflict with a global header hamburger. -->
			{#if hasSidebar}
				<button
					class="sidebar-inline-toggle"
					aria-label="Open section navigation"
					aria-expanded={drawerOpen}
					onclick={openDrawer}
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
						<line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
					</svg>
					<span class="sidebar-inline-toggle-label">Sections</span>
				</button>
			{/if}
			{#if toolbar && hasToolbar}
				{@render toolbar()}
			{/if}
			{#if actions && hasToolbar}
				<div class="layout-toolbar-actions">
					{@render actions()}
				</div>
			{/if}
		</div>
		{@render children()}
	</div>

	{#if dotNav}
		<DotNav />
	{/if}
</div>

<style>
	.sidebar-layout {
		min-height: calc(100vh - 4rem);
	}

	.sidebar-column {
		scrollbar-width: thin;
		scrollbar-color: color-mix(in srgb, var(--color-text) 12%, transparent) transparent;
	}

	.sidebar-column::-webkit-scrollbar {
		width: 4px;
	}

	.sidebar-column::-webkit-scrollbar-track {
		background: transparent;
	}

	.sidebar-column::-webkit-scrollbar-thumb {
		background: color-mix(in srgb, var(--color-text) 12%, transparent);
		border-radius: 2px;
	}

	.sidebar-column::-webkit-scrollbar-thumb:hover {
		background: color-mix(in srgb, var(--color-text) 20%, transparent);
	}

	.layout-content {
		min-width: 0;
		/* BASE (mobile-first) padding — the <769px regime's real spacing (2607-110,
		   owner ruling: root-fix in core-ui). Before this, the base had NO padding and
		   only the ≥769px block set any, so below 769px every consumer's content sat
		   flush against the viewport edge. The desktop regime overrides with its own
		   padding + --layout-pad-x below; shell mode zeroes both (PageShell owns its
		   padding) — both already later in the cascade, so nothing desktop/shell moves. */
		padding: 1rem;
		--layout-pad-x: 1rem;
		/* Vertical rhythm tokens (consumed by the sticky/no-toolbar top-inset and the
		   sticky toolbar band below). Defined on the base so they resolve at every
		   breakpoint; shell mode zeroes --layout-pad-y for full-bleed hero pages.
		   Consumers can override either token. (2605-176) */
		--layout-pad-y: 1.5rem;
		--layout-toolbar-pad-y: 0.5rem;
	}

	.layout-toolbar {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	/* Right-aligned actions region. margin-left:auto pushes it to the row's
	   right edge regardless of how much horizontal space the toolbar snippet
	   takes (breadcrumb is short; filter bar may stretch wide). flex-shrink:0
	   prevents Save/Publish buttons from getting crushed when the breadcrumb
	   trail grows long on deep routes. */
	.layout-toolbar-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-left: auto;
		flex-shrink: 0;
	}

	/* Collapse/expand CTA chrome — defaults work both light/dark via tokens */
	.sidebar-collapse-btn,
	.sidebar-drawer-close,
	.sidebar-expand-tab,
	.sidebar-inline-toggle {
		display: none;
		align-items: center;
		justify-content: center;
		border: 1px solid var(--color-border, rgba(255, 255, 255, 0.08));
		background: var(--color-bg-secondary, rgba(255, 255, 255, 0.03));
		color: var(--color-text-muted, #8b8b90);
		cursor: pointer;
		padding: 0.375rem;
		border-radius: 0.375rem;
		transition: color 0.15s ease, border-color 0.15s ease, background 0.15s ease;
	}

	.sidebar-collapse-btn:hover,
	.sidebar-drawer-close:hover,
	.sidebar-expand-tab:hover,
	.sidebar-inline-toggle:hover {
		color: var(--color-text, #eeeff1);
		border-color: var(--color-accent, #e94560);
	}

	.sidebar-inline-toggle {
		gap: 0.4375rem;
		padding: 0.4375rem 0.75rem;
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		flex-shrink: 0;
	}

	/* Desktop: flush-fixed sidebar, content offset via margin-left. Smooth collapse. */
	@media (min-width: 769px) {
		.sidebar-column {
			position: fixed;
			top: var(--sidebar-top, 5rem);
			left: 0;
			bottom: 0;
			width: var(--sidebar-width, 240px);
			background: var(--color-bg-secondary);
			border-right: 1px solid var(--color-border);
			overflow-y: auto;
			padding-bottom: 2rem;
			z-index: 50;
			transition: top 0.3s ease, transform 0.25s ease, opacity 0.2s ease;
		}

		.layout-content {
			margin-left: var(--sidebar-width, 240px);
			max-width: var(--layout-max-width, 1200px);
			padding: 1.5rem 2rem;
			--layout-pad-x: 2rem;
			/* --layout-pad-y is defined on the base .layout-content (resolves at all
			   breakpoints); no need to re-state it here. */
			transition: margin-left 0.25s ease, max-width 0.25s ease;
		}

		/* Collapsed state — slide sidebar out, expand content */
		.sidebar-layout.collapsed .sidebar-column {
			transform: translateX(-100%);
			opacity: 0;
			pointer-events: none;
		}

		.sidebar-layout.collapsed .layout-content {
			margin-left: 0;
			max-width: none;
			padding-left: 3rem;
			padding-right: 3rem;
			--layout-pad-x: 3rem;
		}

		/* Desktop-only collapse button — aligned with sidebar-hdr-logo row */
		.sidebar-collapse-btn {
			display: inline-flex;
			position: absolute;
			/* Align with sidebar-hdr padding (0.625rem) — content-sidebar padding is now 0 via override */
			top: 0.625rem;
			right: 0.5rem;
			z-index: 2;
			width: 1.75rem;
			height: 1.75rem;
			padding: 0;
		}

		/* Desktop expand tab — floating pill on left edge when collapsed */
		.sidebar-expand-tab {
			display: inline-flex;
			position: fixed;
			top: 50%;
			left: 0;
			transform: translateY(-50%);
			z-index: 51;
			width: 1.75rem;
			height: 3.5rem;
			padding: 0;
			border-left: none;
			border-top-left-radius: 0;
			border-bottom-left-radius: 0;
			border-top-right-radius: 0.5rem;
			border-bottom-right-radius: 0.5rem;
			box-shadow: 0 0 18px rgba(0, 0, 0, 0.3);
		}

		/* Shell mode — PageShell owns ALL padding + hands width control to <Section>.
		   layout-content is full-bleed within the sidebar offset so sections with
		   stripe backgrounds extend to the viewport edge, and their inner
		   containers center via --section-max-width. --layout-pad-x is zeroed so
		   consumers that read it (e.g. PageHeader bleed) become no-ops. */
		.sidebar-layout.shell .layout-content {
			padding: 0;
			max-width: none;
			--layout-pad-x: 0;
			--layout-pad-y: 0;
		}

		.sidebar-layout.shell.collapsed .layout-content {
			padding: 0;
			max-width: none;
		}

		.sidebar-layout.shell .layout-toolbar {
			margin-bottom: 0;
		}

		/* Floating toolbar — overlays the top-right, reserves no row in the layout
		   flow. Used by PageShell to keep the hero flush under the global nav.
		   Desktop only; mobile falls back to the in-flow row (the drawer toggle
		   needs that space). Right offset is hardcoded (not --layout-pad-x) so
		   shell mode's zeroed --layout-pad-x doesn't collapse the toolbar onto
		   the viewport edge. */
		.sidebar-layout.floating-toolbar .layout-toolbar {
			position: fixed;
			top: var(--sidebar-top, 4rem);
			right: 2rem;
			left: auto;
			width: auto;
			margin: 0;
			padding: 0.5rem 0;
			background: transparent;
			z-index: 45;
			pointer-events: none;
		}
		.sidebar-layout.floating-toolbar .layout-toolbar > :global(*) {
			pointer-events: auto;
		}
		/* When toolbar floats, content starts flush under the global nav —
		   the toolbar lives outside the flow. */
		.sidebar-layout.floating-toolbar .layout-content {
			padding-top: 0;
		}

		/* Hide the toolbar row entirely when there is nothing to show on desktop —
		   either no snippet was passed, or the caller opted out via hasToolbar={false}.
		   Prevents a reserved empty row / gap between nav and page content. */
		.sidebar-layout .layout-toolbar.no-toolbar-slot {
			display: none;
		}
	}

	/* Sticky toolbar opt-in — docks the toolbar row under the global nav.
	   Applies at all breakpoints (narrow viewports benefit from a docked filter too).
	   Shell mode has its own sticky path on mobile, so rules layer without conflict. */
	/* Content top-inset when there is no in-flow chrome above the page content —
	 * either because the toolbar is sticky (and docks itself at the top edge) or
	 * because it's suppressed entirely. The inset is the --layout-pad-y token, so
	 * shell/hero consumers (which set --layout-pad-y: 0) stay flush with the global
	 * nav, while plain admin shells get a consistent top breathing-space instead of
	 * jamming content against the viewport top (task 2605-176). */
	.sidebar-layout.sticky-toolbar .layout-content,
	.sidebar-layout:has(.layout-toolbar.no-toolbar-slot) .layout-content {
		padding-top: var(--layout-pad-y);
	}

	.sidebar-layout.sticky-toolbar .layout-toolbar {
		position: sticky;
		top: var(--sidebar-top, 0);
		z-index: 40;
		margin-bottom: 0;
		/* Symmetric internal band padding — tokenized so consumers tune the docked
		   toolbar's height in one place rather than via ad-hoc overrides (2605-176). */
		padding: var(--layout-toolbar-pad-y, 0.5rem) 0;
		/* Row itself is transparent — page content scrolls behind freely.
		   Toolbar children (filter pills, breadcrumbs) own their own
		   visual boundary if they need a fill against scrolled content. */
		background: transparent;
	}

	/* Anchor scroll-margin — keep [id] targets clear of sticky chrome.
	   Single rule covers every consumer of this layout (packages, knowledge,
	   components, portfolio) without per-page workarounds. The offset auto-
	   adjusts when the toolbar docks sticky vs. only the global nav is sticky. */
	.sidebar-layout {
		--layout-anchor-offset: var(--sidebar-top, 4rem);
	}
	.sidebar-layout.sticky-toolbar {
		--layout-anchor-offset: calc(var(--sidebar-top, 4rem) + 3.5rem);
	}
	:global(.sidebar-layout .layout-content [id]) {
		scroll-margin-top: var(--layout-anchor-offset);
	}

	/* ContentSidebar override — prevent double-sticky and double-scroll when nested in SidebarLayout */
	:global(.sidebar-column .content-sidebar) {
		position: static !important;
		padding-top: 0 !important;
		max-height: none !important;
		overflow-y: visible !important;
	}

	.sidebar-backdrop {
		display: none;
	}

	/* Mobile: drawer + inline section toggle in toolbar row */
	@media (max-width: 768px) {
		.sidebar-layout {
			flex-direction: column;
			gap: 0;
		}

		/* Disable desktop collapse on mobile — drawer is the only mode */
		.sidebar-layout.collapsed .sidebar-column {
			transform: translateX(-100%);
			opacity: 1;
			pointer-events: auto;
		}
		.sidebar-layout.collapsed .layout-content {
			/* Drawer mode: no sidebar column to offset. The desktop-collapsed 3rem
			   padding never applies here (it lives in the ≥769px block), and the
			   mobile BASE padding must survive — a padding-left:0 here would jam
			   content back against the edge (the pre-2607-110 bug). */
			margin-left: 0;
		}

		.sidebar-collapse-btn,
		.sidebar-expand-tab {
			display: none !important;
		}

		.sidebar-column {
			position: fixed;
			top: 0;
			left: 0;
			bottom: 0;
			width: min(280px, 85vw);
			max-height: 100vh;
			background: var(--color-bg);
			border-right: 1px solid var(--color-border);
			padding: 3.5rem 1rem 1rem;
			z-index: 1001;
			transform: translateX(-100%);
			transition: transform 0.2s ease;
			overflow-y: auto;
		}

		.sidebar-column.open {
			transform: translateX(0);
		}

		.sidebar-drawer-close {
			display: inline-flex;
			position: absolute;
			top: 0.75rem;
			right: 0.75rem;
			z-index: 2;
			width: 2rem;
			height: 2rem;
			padding: 0;
		}

		.sidebar-backdrop {
			display: block;
			position: fixed;
			inset: 0;
			z-index: 1000;
			background: color-mix(in srgb, var(--color-text) 40%, transparent);
			border: 0;
			padding: 0;
		}

		/* Inline section toggle sits first in the toolbar row */
		.sidebar-inline-toggle {
			display: inline-flex;
		}

		/* Empty toolbar row still shows the inline toggle */
		.layout-toolbar.no-toolbar-slot {
			margin-bottom: 1rem;
		}

		/* Shell mode on mobile — keep toolbar for Sections toggle */
		.sidebar-layout.shell .layout-content {
			padding: 0;
			/* Mirror desktop: zero the inset token so the content-padding contract is
			   source-order-independent for shell consumers on mobile too (2605-176). */
			--layout-pad-y: 0;
		}

		.sidebar-layout.shell .layout-toolbar {
			margin-bottom: 0;
			padding: 0.5rem 1rem;
			background: var(--color-bg);
			border-bottom: 1px solid var(--color-border);
			position: sticky;
			top: var(--sidebar-top, 4rem);
			z-index: 40;
		}
	}

	/* ── Overlay sidebar regime — opt-in via sidebarOverlayBreakpoint="wide" ───────
	   In the narrow-desktop band the docked sidebar overlays content as a drawer
	   instead of pushing it via margin-left (Wave 2 / B2). Gated by BOTH the static
	   .overlay-wide class (derived from the prop → SSR-stable, no hydration flash)
	   AND this pure-CSS width band, so consumers that don't opt in match no rule
	   here and render exactly as before. The 769/1279 literals mirror
	   BREAKPOINTS.desktop … BREAKPOINTS.wide − 1 (see src/lib/breakpoints.ts) —
	   CSS @media can't read the TS constants (no postcss-custom-media). */
	@media (min-width: 769px) and (max-width: 1279px) {
		/* The column is already position:fixed at --sidebar-width (desktop block);
		   here it just rides off-canvas and slides in on drawerOpen. opacity +
		   pointer-events neutralize a stale persisted .collapsed state so the drawer
		   always opens; z-index lifts it above the scrim. */
		.sidebar-layout.overlay-wide .sidebar-column {
			transform: translateX(-100%);
			opacity: 1;
			pointer-events: auto;
			z-index: 1001;
		}
		.sidebar-layout.overlay-wide .sidebar-column.open {
			transform: translateX(0);
		}

		/* Content reclaims the full row — the drawer floats over it. */
		.sidebar-layout.overlay-wide .layout-content,
		.sidebar-layout.overlay-wide.collapsed .layout-content {
			margin-left: 0;
		}

		/* Keep the toolbar row (hence the Sections opener) reachable even when it
		   would otherwise auto-collapse — the drawer has no other open affordance. */
		.sidebar-layout.overlay-wide .layout-toolbar.no-toolbar-slot {
			display: flex;
		}

		/* Drawer controls: show the inline "Sections" opener + the in-drawer close;
		   hide the desktop collapse chrome (drawer mode has no collapsed column). */
		.sidebar-layout.overlay-wide .sidebar-inline-toggle {
			display: inline-flex;
		}
		.sidebar-layout.overlay-wide .sidebar-drawer-close {
			display: inline-flex;
			position: absolute;
			top: 0.75rem;
			right: 0.75rem;
			z-index: 2;
			width: 2rem;
			height: 2rem;
			padding: 0;
		}
		.sidebar-layout.overlay-wide .sidebar-collapse-btn {
			display: none !important;
		}

		/* Scrim + expand tab are siblings of .sidebar-layout, so they carry their
		   own .overlay-wide marker (set in markup). */
		.sidebar-backdrop.overlay-wide {
			display: block;
			position: fixed;
			inset: 0;
			z-index: 1000;
			background: color-mix(in srgb, var(--color-text) 40%, transparent);
			border: 0;
			padding: 0;
		}
		.sidebar-expand-tab.overlay-wide {
			display: none !important;
		}
	}

</style>

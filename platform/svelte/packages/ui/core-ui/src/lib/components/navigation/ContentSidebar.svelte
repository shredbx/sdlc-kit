<script lang="ts" module>
	// =============================================================================
	// TYPE DEFINITIONS (exported for consumers)
	// =============================================================================

	/** A section in the content sidebar tree */
	export interface ContentSidebarSection {
		/** Unique identifier */
		id: string;
		/** Display label */
		label: string;
		/** Optional navigation href */
		href?: string;
		/** Optional icon (emoji or symbol) */
		icon?: string;
		/** Optional badge text (e.g. "Soon", "Kotlin") */
		badge?: string;
		/** Nested child sections (L2+) */
		children?: ContentSidebarSection[];
		/** Leaf items within this section */
		items?: ContentSidebarItem[];
		/** Whether this section starts expanded (default: false) */
		defaultExpanded?: boolean;
		/** Section type: 'section-header' renders as flat group header without toggle */
		type?: 'section-header' | string;
		/** When true, href matches only the exact path — no prefix matching on child routes */
		exactMatch?: boolean;
		/** Optional role gate — section visible only when current user role is in this list.
		 *  Undefined or empty = visible to all. Consumers should pre-filter with `filterTreeByRole`
		 *  before passing to ContentSidebar; the component itself does not consult this field.
		 *  Server-side handlers MUST still enforce role-based access on protected routes —
		 *  this is UX filtering only, not a security perimeter. */
		roles?: string[];
	}

	/** A leaf item within a content sidebar section */
	export interface ContentSidebarItem {
		/** Unique identifier */
		id: string;
		/** Display label */
		label: string;
		/** Navigation href */
		href: string;
		/** Optional badge text */
		badge?: string;
	}
</script>

<script lang="ts">
	/**
	 * ContentSidebar Component
	 *
	 * Data-driven sidebar for all content/section pages.
	 * Pass sections data + optional header config; all rendering is internal.
	 * Consumers should never compose SidebarHeader/SidebarSection/ContentNavGroup
	 * manually — configure this component with props and snippets instead.
	 *
	 * Features:
	 * - 2-level depth rule with back navigation
	 * - Optional header with title and search trigger
	 * - Optional headerExtra snippet for route-specific content (e.g. filters)
	 * - Collapsed horizontal bar mode
	 */

	import type { Snippet } from 'svelte';

	// =============================================================================
	// PROPS
	// =============================================================================

	interface Props {
		/** The full section tree */
		sections: ContentSidebarSection[];
		/** Current page path for active highlighting */
		currentPath: string;
		/** Whether the sidebar is collapsed into horizontal mode */
		collapsed?: boolean;
		/** Callback when collapse state changes */
		onCollapseChange?: (collapsed: boolean) => void;
		/** Width in pixels (default 240) */
		width?: number;
		/** Optional header title (e.g. "Knowledge", "Components", "Packages") */
		title?: string;
		/** Show a search trigger input in the header */
		showSearch?: boolean;
		/** Placeholder text for the search input */
		searchPlaceholder?: string;
		/** Keyboard shortcut label displayed in search input (e.g. "⌘K") */
		searchShortcut?: string;
		/** Callback when search input receives focus (use to open search modal) */
		onsearchfocus?: () => void;
		/** Optional snippet rendered between header and nav tree (e.g. platform filters) */
		headerExtra?: Snippet;
		/** Active anchor ID from scroll-spy (overrides path-based active state for hash links) */
		activeAnchor?: string | null;
	}

	let {
		sections,
		currentPath,
		collapsed = false,
		onCollapseChange,
		width = 240,
		title,
		showSearch = false,
		searchPlaceholder = 'Search...',
		searchShortcut,
		onsearchfocus,
		headerExtra,
		activeAnchor = null
	}: Props = $props();

	// =============================================================================
	// SCROLL-SPY (internal) — observes every <section id=...> on the current page
	// whose hash is targeted by a sidebar href. Consumer prop `activeAnchor` (if
	// set) overrides. Pages compose Section primitives; sidebar auto-tracks.
	// =============================================================================

	function anchorIdFor(href: string | undefined, pagePath: string): string | null {
		if (!href) return null;
		if (href.startsWith('#')) return href.slice(1);
		const [path, hash] = href.split('#');
		if (!hash) return null;
		const [pathOnly] = path.split('?');
		return pathOnly === pagePath ? hash : null;
	}

	function collectAnchorIds(tree: ContentSidebarSection[], pagePath: string): string[] {
		const out: string[] = [];
		const visit = (nodes: ContentSidebarSection[]) => {
			for (const node of nodes) {
				const id = anchorIdFor(node.href, pagePath);
				if (id) out.push(id);
				if (node.items) {
					for (const item of node.items) {
						const id2 = anchorIdFor(item.href, pagePath);
						if (id2) out.push(id2);
					}
				}
				if (node.children) visit(node.children);
			}
		};
		visit(tree);
		return [...new Set(out)];
	}

	const pagePathOnly = $derived(currentPath.split('#')[0].split('?')[0]);
	const anchorIds = $derived(collectAnchorIds(sections, pagePathOnly));

	let activeAnchorInternal = $state<string | null>(null);
	const effectiveAnchor = $derived(activeAnchor ?? activeAnchorInternal);

	$effect(() => {
		if (typeof window === 'undefined') return;
		const ids = anchorIds;
		if (ids.length === 0) {
			activeAnchorInternal = null;
			return;
		}

		let observer: IntersectionObserver | null = null;
		let retryTimer: ReturnType<typeof setTimeout> | null = null;

		function resolveAndObserve() {
			const targets: HTMLElement[] = [];
			for (const id of ids) {
				const el = document.getElementById(id);
				if (el) targets.push(el);
			}
			if (targets.length === 0) {
				retryTimer = setTimeout(resolveAndObserve, 350);
				return;
			}
			const ratios = new Map<string, number>();
			observer = new IntersectionObserver(
				(entries) => {
					for (const e of entries) {
						const id = (e.target as HTMLElement).id;
						if (!id) continue;
						ratios.set(id, e.isIntersecting ? e.intersectionRatio : 0);
					}
					let best: string | null = null;
					let bestR = 0;
					for (const id of ids) {
						const r = ratios.get(id) ?? 0;
						if (r > bestR) {
							bestR = r;
							best = id;
						}
					}
					if (best !== null) activeAnchorInternal = best;
				},
				{ threshold: [0, 0.25, 0.5, 0.75, 1], rootMargin: '-80px 0px 0px 0px' }
			);
			for (const t of targets) observer.observe(t);
		}

		resolveAndObserve();

		return () => {
			if (retryTimer) clearTimeout(retryTimer);
			if (observer) observer.disconnect();
		};
	});

	// =============================================================================
	// STATE
	// =============================================================================

	/** Tracks which section IDs are manually expanded/collapsed by the user */
	let manualExpansion = $state<Map<string, boolean>>(new Map());

	// =============================================================================
	// ACTIVE PATH DETECTION
	// =============================================================================

	/** Find the path of section IDs from root to the active item */
	let activePath = $derived.by(() => {
		return findActivePath(sections, currentPath);
	});

	// Expand sections on the active path only when the URL changes (not on manual toggle)
	let lastExpandedPath = $state('');
	$effect(() => {
		if (currentPath !== lastExpandedPath) {
			lastExpandedPath = currentPath;
			if (activePath.length > 0) {
				const next = new Map(manualExpansion);
				let changed = false;
				for (const id of activePath) {
					if (!next.has(id) || next.get(id) === false) {
						next.set(id, true);
						changed = true;
					}
				}
				if (changed) {
					manualExpansion = next;
				}
			}
		}
	});

	function findActivePath(
		tree: ContentSidebarSection[],
		target: string,
		path: string[] = []
	): string[] {
		for (const node of tree) {
			const currentNodePath = [...path, node.id];

			// Check items for a match
			if (node.items) {
				for (const item of node.items) {
					if (isPathMatch(item.href, target)) {
						return currentNodePath;
					}
				}
			}

			// Recurse into children FIRST — prefer deeper matches
			if (node.children) {
				const result = findActivePath(node.children, target, currentNodePath);
				if (result.length > 0) return result;
			}

			// Check if the node href itself matches (only if no deeper match found)
			if (node.href && isPathMatch(node.href, target)) {
				return currentNodePath;
			}
		}
		return [];
	}

	function isPathMatch(href: string, target: string): boolean {
		const [hrefNoHash, hrefHash] = href.split('#');
		const [targetNoHash, targetHash] = target.split('#');

		if (hrefHash) {
			// href has an anchor. Active when:
			//  (a) same path + same hash (anchor is current scroll target), OR
			//  (b) target drills deeper into a segment named after the hash
			//      e.g. href=/A#X is active when target=/A/X or /A/X/...
			//      This makes L2 sidebar anchors light up on L3 detail routes.
			if (hrefNoHash === targetNoHash && hrefHash === targetHash) return true;
			const drilldown = `${hrefNoHash}/${hrefHash}`;
			if (targetNoHash === drilldown || targetNoHash.startsWith(drilldown + '/')) return true;
			return false;
		}

		if (!hrefNoHash.includes('?')) {
			if (hrefNoHash === targetNoHash) return true;
			if (hrefNoHash !== '/' && targetNoHash.startsWith(hrefNoHash + '/')) return true;
			return false;
		}

		const [hrefPath, hrefQuery] = hrefNoHash.split('?');
		const [targetPath] = targetNoHash.split('?');
		if (hrefPath !== targetPath && !targetPath.startsWith(hrefPath + '/')) return false;
		const hrefParams = new URLSearchParams(hrefQuery);
		const targetParams = new URLSearchParams(targetNoHash.includes('?') ? targetNoHash.split('?')[1] : '');
		for (const [key, val] of hrefParams) {
			if (targetParams.get(key) !== val) return false;
		}
		return true;
	}

	// =============================================================================
	// SECTION EXPANSION
	// =============================================================================

	function isSectionExpanded(sectionId: string, section: ContentSidebarSection): boolean {
		// Manual override takes priority
		const manual = manualExpansion.get(sectionId);
		if (manual !== undefined) return manual;

		// Auto-expand if on the active path
		if (activePath.includes(sectionId)) return true;

		// Default from prop
		return section.defaultExpanded ?? false;
	}

	function toggleSection(sectionId: string, section: ContentSidebarSection) {
		const current = isSectionExpanded(sectionId, section);
		const next = new Map(manualExpansion);
		next.set(sectionId, !current);
		manualExpansion = next;
	}

	// =============================================================================
	// ITEM ACTIVE CHECK
	// =============================================================================

	function isItemActive(href: string, exact = false): boolean {
		if (exact) {
			const [hrefNoHash] = href.split('#');
			const [targetNoHash] = currentPath.split('#');
			return hrefNoHash === targetNoHash;
		}
		if (effectiveAnchor && href.includes('#')) {
			const [base, hash] = href.split('#');
			const [basePath] = base.split('?');
			if ((basePath === '' || basePath === pagePathOnly) && hash === effectiveAnchor) {
				return true;
			}
		}
		return isPathMatch(href, currentPath);
	}

	function isSectionActive(section: ContentSidebarSection): boolean {
		if (section.href) {
			if (section.exactMatch) {
				const [hrefNoHash] = section.href.split('#');
				const [targetNoHash] = currentPath.split('#');
				if (hrefNoHash !== targetNoHash) return activePath.includes(section.id);
				return true;
			}
			if (isPathMatch(section.href, currentPath)) return true;
		}
		return activePath.includes(section.id);
	}

	// =============================================================================
	// COLLAPSE TOGGLE
	// =============================================================================

	function toggleCollapse() {
		onCollapseChange?.(!collapsed);
	}
</script>

{#if collapsed}
	<!-- ======================================================================
	     HORIZONTAL BAR MODE
	     ====================================================================== -->
	<nav class="content-sidebar-bar" aria-label="Content navigation">
		{#each sections as section (section.id)}
			{#if section.href}
				<a
					href={section.href}
					class="bar-tab"
					class:active={isSectionActive(section)}
				>
					{#if section.icon}<span class="bar-icon">{section.icon}</span>{/if}
					<span class="bar-label">{section.label}</span>
				</a>
			{:else}
				<span
					class="bar-tab"
					class:active={isSectionActive(section)}
				>
					{#if section.icon}<span class="bar-icon">{section.icon}</span>{/if}
					<span class="bar-label">{section.label}</span>
				</span>
			{/if}
		{/each}
		<button
			class="bar-expand-btn"
			onclick={toggleCollapse}
			title="Expand sidebar"
			aria-label="Expand sidebar"
		>
			<span class="bar-expand-icon">&#9776;</span>
		</button>
	</nav>
{:else}
	<!-- ======================================================================
	     EXPANDED SIDEBAR MODE
	     ====================================================================== -->
	<nav
		class="content-sidebar"
		style="--content-sidebar-width: {width}px"
		aria-label="Content navigation"
	>
		<!-- Optional header with title and search -->
		{#if title}
			<div class="sidebar-hdr" class:has-search={showSearch}>
				<div class="sidebar-hdr-logo">
					<svg
						class="sidebar-hdr-svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						aria-hidden="true"
					>
						<path
							d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"
						/>
					</svg>
					<span class="sidebar-hdr-title">{title}</span>
				</div>

				{#if showSearch}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div class="sidebar-search-wrap" onfocusin={onsearchfocus}>
						<svg
							class="sidebar-search-icon"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							aria-hidden="true"
						>
							<circle cx="11" cy="11" r="8" />
							<line x1="21" y1="21" x2="16.65" y2="16.65" />
						</svg>
						<span class="sidebar-search-placeholder" aria-hidden="true">
							{searchPlaceholder}
						</span>
						<input
							class="sidebar-search-input"
							type="search"
							readonly
							aria-label="Search {title}"
						/>
						{#if searchShortcut}
							<kbd class="sidebar-search-kbd">{searchShortcut}</kbd>
						{/if}
					</div>
				{/if}
			</div>
		{/if}

		<!-- Optional extra content between header and nav (e.g. platform filters) -->
		{#if headerExtra}
			{@render headerExtra()}
		{/if}

		<!-- Section tree -->
		<ul class="section-list">
			{#each sections as section (section.id)}
				{@const isSectionHeader = section.type === 'section-header'}
				{@const expanded = isSectionHeader ? true : isSectionExpanded(section.id, section)}
				{@const hasChildren = (section.children && section.children.length > 0) || (section.items && section.items.length > 0)}
				<li class="section-node" class:section-node--group={isSectionHeader}>
					<!-- Section header -->
					{#if isSectionHeader}
						<div class="section-group-header" class:active={isSectionActive(section)}>
							{#if section.href}
								<a href={section.href} class="section-group-label" class:active={section.href ? isItemActive(section.href, section.exactMatch) : false}>
									{section.label}
								</a>
							{:else}
								<span class="section-group-label">{section.label}</span>
							{/if}
						</div>
					{:else}
						<div class="section-header" class:active={isSectionActive(section)}>
							{#if section.href}
								<a href={section.href} class="section-label" class:active={section.href ? isItemActive(section.href, section.exactMatch) : false}
									onclick={() => { if (hasChildren) toggleSection(section.id, section); }}>
									{#if section.icon}<span class="section-icon">{section.icon}</span>{/if}
									{section.label}
								</a>
							{:else}
								<button
									class="section-label section-label-btn"
									onclick={() => hasChildren && toggleSection(section.id, section)}
								>
									{#if section.icon}<span class="section-icon">{section.icon}</span>{/if}
									{section.label}
								</button>
							{/if}
							{#if hasChildren}
								<button
									class="section-toggle"
									onclick={() => toggleSection(section.id, section)}
									aria-expanded={expanded}
								>
									<span class="chevron" class:expanded>{expanded ? '\u25BC' : '\u25B6'}</span>
								</button>
							{/if}
						</div>
					{/if}

					<!-- L2: Children & items (shown when expanded) -->
					{#if expanded && hasChildren}
						<ul class="children-list">
							{#if section.children}
								{#each section.children as child (child.id)}
									{@const childHasNested = (child.children && child.children.length > 0) || (child.items && child.items.length > 0)}
									{@const childExpanded = isSectionExpanded(child.id, child)}
									<li class="child-node">
										<div class="child-header">
											{#if childHasNested}
												<button
													class="child-toggle"
													onclick={() => toggleSection(child.id, child)}
													aria-expanded={childExpanded}
												>
													<span class="chevron-sm">{childExpanded ? '\u25BC' : '\u25B6'}</span>
												</button>
											{/if}
											{#if child.href}
												<a href={child.href} class="child-link" class:active={isItemActive(child.href)} class:has-toggle={childHasNested}
													onclick={() => { if (childHasNested) toggleSection(child.id, child); }}>
													{#if child.icon}<span class="child-icon">{child.icon}</span>{/if}
													<span class="child-text">{child.label}</span>
													{#if child.badge}
														<span class="child-badge">{child.badge}</span>
													{/if}
												</a>
											{:else}
												<button
													class="child-link child-link-btn"
													class:has-toggle={childHasNested}
													onclick={() => childHasNested && toggleSection(child.id, child)}
												>
													{#if child.icon}<span class="child-icon">{child.icon}</span>{/if}
													<span class="child-text">{child.label}</span>
													{#if child.badge}
														<span class="child-badge">{child.badge}</span>
													{/if}
												</button>
											{/if}
										</div>
										{#if childExpanded && childHasNested}
											<ul class="nested-list">
												{#if child.children}
													{#each child.children as nested (nested.id)}
														<li class="nested-node">
															{#if nested.href}
																<a href={nested.href} class="nested-link" class:active={isItemActive(nested.href)}>
																	{nested.label}
																</a>
															{:else}
																<span class="nested-link">{nested.label}</span>
															{/if}
														</li>
													{/each}
												{/if}
												{#if child.items}
													{#each child.items as item (item.id)}
														<li class="nested-node">
															<a href={item.href} class="nested-link" class:active={isItemActive(item.href)}>
																{item.label}
															</a>
														</li>
													{/each}
												{/if}
											</ul>
										{/if}
									</li>
								{/each}
							{/if}
							{#if section.items}
								{#each section.items as item (item.id)}
									<li class="child-node">
										<a href={item.href} class="child-link" class:active={isItemActive(item.href)}>
											<span class="child-text">{item.label}</span>
											{#if item.badge}
												<span class="child-badge">{item.badge}</span>
											{/if}
										</a>
									</li>
								{/each}
							{/if}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>
	</nav>
{/if}

<style>
	/* ==========================================================================
	   EXPANDED SIDEBAR
	   ========================================================================== */
	.content-sidebar {
		width: var(--content-sidebar-width, 240px);
		position: sticky;
		top: var(--sidebar-top, 5rem);
		align-self: flex-start;
		padding: 0.5rem 0;
		overflow-y: auto;
		max-height: calc(100vh - var(--sidebar-top, 5rem) - 1rem);
		transition: top 0.3s ease;
	}

	/* ==========================================================================
	   HEADER (title + search)
	   ========================================================================== */
	.sidebar-hdr {
		padding: 0.625rem 0.5rem 0.625rem;
		border-bottom: 1px solid var(--color-border, #2a2a2d);
		flex-shrink: 0;
	}

	.sidebar-hdr-logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 10px;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--color-text-muted, #8b8b90);
		line-height: 1;
		min-height: 1.75rem;
	}

	.sidebar-hdr.has-search .sidebar-hdr-logo {
		margin-bottom: 0.5rem;
	}

	.sidebar-hdr-svg {
		flex-shrink: 0;
		display: block;
		width: 14px;
		height: 14px;
		color: var(--color-text-muted, #8b8b90);
	}

	.sidebar-hdr-title {
		display: inline-flex;
		align-items: center;
		line-height: 1;
		color: var(--color-text-muted, #8b8b90);
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.sidebar-search-wrap {
		position: relative;
	}

	.sidebar-search-icon {
		position: absolute;
		left: 0.625rem;
		top: 50%;
		transform: translateY(-50%);
		width: 0.9375rem;
		height: 0.9375rem;
		color: var(--color-text-dim, #55555a);
		pointer-events: none;
	}

	.sidebar-search-input {
		width: 100%;
		background: var(--color-bg, #0a0a0b);
		border: 1px solid var(--color-border, #2a2a2d);
		border-radius: 0.375rem;
		padding: 0.4375rem 2.25rem 0.4375rem 2rem;
		color: var(--color-text, #eeeff1);
		font-family: inherit;
		font-size: 0.8rem;
		outline: none;
		cursor: pointer;
		transition: border-color 0.2s ease;
	}

	.sidebar-search-placeholder {
		position: absolute;
		left: 2rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.8rem;
		pointer-events: none;
		z-index: 1;
	}

	.sidebar-search-placeholder :global(.tr-wrap) {
		color: var(--color-text-dim, #55555a);
	}

	.sidebar-search-placeholder :global(.tr-target) {
		color: var(--color-text-dim, #55555a);
	}

	.sidebar-search-placeholder :global(.tr-cursor) {
		color: var(--color-text-dim, #55555a);
		animation: none;
	}

	.sidebar-search-input::placeholder {
		color: var(--color-text-dim, #55555a);
	}

	.sidebar-search-input:focus {
		border-color: var(--color-accent, #e94560);
	}

	.sidebar-search-input::-webkit-search-decoration,
	.sidebar-search-input::-webkit-search-cancel-button,
	.sidebar-search-input::-webkit-search-results-button,
	.sidebar-search-input::-webkit-search-results-decoration {
		-webkit-appearance: none;
	}

	.sidebar-search-kbd {
		position: absolute;
		right: 0.5rem;
		top: 50%;
		transform: translateY(-50%);
		display: inline-flex;
		align-items: center;
		padding: 0.0625rem 0.3rem;
		border: 1px solid color-mix(in srgb, var(--color-text) 10%, transparent);
		border-radius: 0.1875rem;
		font-family: inherit;
		font-size: 0.625rem;
		color: var(--color-text-muted, #8b8b90);
		background: color-mix(in srgb, var(--color-text) 4%, transparent);
		opacity: 0.5;
		pointer-events: none;
	}

	/* ==========================================================================
	   SECTION LIST
	   ========================================================================== */
	.section-list {
		list-style: none;
		margin: 0;
		padding: 0 0.75rem;
	}

	.section-node {
		margin-bottom: 0.125rem;
	}

	/* ==========================================================================
	   SECTION HEADER
	   ========================================================================== */
	.section-header {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.125rem 0;
	}

	.section-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		flex-shrink: 0;
		margin-left: auto;
		background: transparent;
		border: none;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		padding: 0;
		border-radius: var(--radius-sm, 0.25rem);
		transition: color 0.15s ease, background-color 0.15s ease;
	}

	.section-toggle:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.chevron {
		font-size: 0.5rem;
		line-height: 1;
		transition: transform 0.2s ease;
	}

	.section-label {
		flex: 1;
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--color-text, #fff);
		text-decoration: none;
		padding: 0.375rem 0.5rem;
		border-radius: var(--radius-sm, 0.25rem);
		transition: color 0.15s ease, background-color 0.15s ease;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.section-label:hover {
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.section-label.active {
		color: var(--color-accent-label, var(--color-accent, #e94560));
	}

	.section-label-btn {
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		font-family: inherit;
	}

	.section-icon {
		margin-right: 0.375rem;
	}

	/* ==========================================================================
	   SECTION GROUP HEADER (type: section-header)
	   ========================================================================== */
	.section-node--group {
		margin-top: 1rem;
	}

	.section-node--group:first-child {
		margin-top: 0;
	}

	.section-group-header {
		/* Inactive: no bottom padding — next L2 item absorbs the space where the
		   underline would render, keeping active/inactive vertical rhythm identical. */
		padding: 0.4rem 0.2rem 0;
		box-shadow: inset 0 -2px 0 transparent;
		margin-bottom: 0;
		transition: box-shadow 0.15s ease;
	}

	.section-group-header.active {
		/* Active: add bottom padding to hold the 2px underline in place without
		   pushing L2 items down — matches inactive total height. */
		padding: 0.4rem 0.2rem 0.3rem;
		box-shadow: inset 0 -2px 0 var(--color-accent, #e94560);
	}

	.section-group-label {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--color-text-muted, #6b7280);
		text-decoration: none;
		transition: color 0.15s ease;
	}

	a.section-group-label:hover {
		color: var(--color-text, #fff);
	}

	a.section-group-label.active {
		color: var(--color-accent-label, var(--color-accent, #e94560));
	}

	/* ==========================================================================
	   CHILDREN LIST (L2)
	   ========================================================================== */
	.children-list {
		list-style: none;
		margin: 0;
		padding: 0 0 0.25rem 0;
	}

	.child-node {
		margin: 0;
	}

	.child-link {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.2375rem 0.625rem 0.2375rem 0.5rem;
		font-size: 0.8125rem;
		color: var(--color-text-muted, #6b7280);
		text-decoration: none;
		border-radius: var(--radius-sm, 0.25rem);
		transition: color 0.15s ease, background-color 0.15s ease;
	}

	.child-link:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.child-link.active {
		color: var(--color-accent-label, var(--color-accent, #e94560));
		background: var(--color-accent-subtle, rgba(233, 69, 96, 0.12));
	}

	.child-icon {
		flex-shrink: 0;
	}

	.child-text {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.child-badge {
		font-size: 0.75rem;
		padding: 0.0625rem 0.375rem;
		border-radius: var(--radius-sm, 0.25rem);
		background: var(--color-bg-secondary, #1a1a2e);
		color: var(--color-text-muted, #6b7280);
		flex-shrink: 0;
		font-weight: 500;
	}

	/* ==========================================================================
	   L2 CHILD HEADER WITH TOGGLE
	   ========================================================================== */
	.child-header {
		display: flex;
		align-items: center;
		gap: 0.125rem;
	}

	.child-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1rem;
		height: 1rem;
		flex-shrink: 0;
		background: transparent;
		border: none;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		padding: 0;
		border-radius: var(--radius-sm, 0.25rem);
		margin-left: 1rem;
	}

	.child-toggle:hover {
		color: var(--color-text, #fff);
	}

	.chevron-sm {
		font-size: 0.4375rem;
		line-height: 1;
	}

	.child-link.has-toggle {
		padding-left: 0.25rem;
	}

	.child-link-btn {
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		font-family: inherit;
	}

	/* ==========================================================================
	   L3 NESTED LIST
	   ========================================================================== */
	.nested-list {
		list-style: none;
		margin: 0;
		padding: 0 0 0.125rem 0;
	}

	.nested-node {
		margin: 0;
	}

	.nested-link {
		display: block;
		padding: 0.25rem 0.625rem 0.25rem 1.5rem;
		font-size: 0.75rem;
		color: var(--color-text-muted, #6b7280);
		text-decoration: none;
		border-radius: var(--radius-sm, 0.25rem);
		transition: color 0.15s ease, background-color 0.15s ease;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.nested-link:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.nested-link.active {
		color: var(--color-accent-label, var(--color-accent, #e94560));
		background: var(--color-accent-subtle, rgba(233, 69, 96, 0.12));
	}

	/* ==========================================================================
	   HORIZONTAL BAR MODE (collapsed)
	   ========================================================================== */
	.content-sidebar-bar {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		width: 100%;
		padding: 0.375rem 0.5rem;
		border-bottom: 1px solid var(--color-border, #2a2a4a);
		background: var(--color-bg, #0f0f1a);
		overflow-x: auto;
	}

	.bar-tab {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.375rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--color-text-muted, #6b7280);
		text-decoration: none;
		border-radius: var(--radius-md, 0.375rem);
		white-space: nowrap;
		cursor: pointer;
		transition: color 0.15s ease, background-color 0.15s ease;
	}

	.bar-tab:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.bar-tab.active {
		color: var(--color-accent-label, var(--color-accent, #e94560));
		background: var(--color-accent-subtle, rgba(233, 69, 96, 0.12));
	}

	.bar-icon {
		flex-shrink: 0;
	}

	.bar-label {
		white-space: nowrap;
	}

	.bar-expand-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		margin-left: auto;
		padding: 0.375rem 0.5rem;
		background: transparent;
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: var(--radius-sm, 0.25rem);
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		flex-shrink: 0;
		transition: color 0.15s ease, background-color 0.15s ease;
	}

	.bar-expand-btn:hover {
		color: var(--color-text, #fff);
		background: var(--color-bg-hover, color-mix(in srgb, var(--color-text) 5%, transparent));
	}

	.bar-expand-icon {
		font-size: 0.875rem;
		line-height: 1;
	}

	/* ==========================================================================
	   REDUCED MOTION
	   ========================================================================== */
	@media (prefers-reduced-motion: reduce) {
		.content-sidebar,
		.section-toggle,
		.section-label,
		.child-link,
		.bar-tab,
		.bar-expand-btn,
		.chevron {
			transition-duration: 0.01ms !important;
		}
	}
</style>

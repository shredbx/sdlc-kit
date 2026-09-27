<script lang="ts" module>
	import type { Snippet } from 'svelte';

	// One tabbed page surface — a label + the route it links to. `badge` is an
	// optional unread/pending count rendered as a small pill (only when > 0). This is
	// THE single tab type for EVERY shell mode (#0290 F1b): the unification of
	// AdminPage / AdminPageShell (both re-exported this shape), AdminScaffoldPage (which
	// had no tabs) AND — folded in at F1b-ii — EntityDetailShell's facet tabs (which used
	// the same {id,label,href} shape, badge-free; the optional badge here is a superset).
	export type TabEntry = {
		id: string;
		label: string;
		href: string;
		badge?: number;
	};

	// A small cover affordance for the detail-mode compact toolbar (folded in from
	// EntityDetailShell, 2606-001). `url` is an ALREADY-RESOLVED image src (the consumer
	// runs its own image-resize helper — the shell stays storage-agnostic); empty/absent
	// renders a neutral placeholder tile (never a broken img). `href` is where "Change"
	// navigates — the entity's media surface. Composable: any entity passes its own
	// resolved url + media href.
	export type CoverRef = { url?: string; href?: string };

	/**
	 * The active tab for a given pathname. Generalizes the property layout's
	 * "last path segment" rule into one robust prefix match so nested sub-routes
	 * work (e.g. /manage/inquiries [Listing] vs /manage/inquiries/page [Page]):
	 *
	 *   1. an EXACT pathname match wins;
	 *   2. otherwise the tab whose href is the LONGEST path-prefix of the
	 *      pathname, where the prefix must end on a `/` boundary so
	 *      `/manage/inquiries` never spuriously matches `/manage/inquiriesX`.
	 *
	 * Argument order is the standardized (tabs, pathname) — carried verbatim from
	 * AdminPageShell. This is now THE single order for ALL modes: EntityDetailShell's
	 * detail-mode copy used the opposite (pathname, tabs); when it folded in at F1b-ii
	 * every facet call switched to this order, so there is exactly one. Exported so the
	 * unit test asserts the rule directly, not just the DOM.
	 */
	export function activeTabId(tabs: TabEntry[], pathname: string): string | undefined {
		let bestId: string | undefined;
		let bestLen = -1;
		for (const tab of tabs) {
			if (tab.href === pathname) return tab.id; // exact match wins outright
			// Prefix must end at a segment boundary: the char after the href in the
			// pathname is a `/`. Guard the bare-slash href so it doesn't match all.
			const boundary = tab.href.endsWith('/') ? tab.href : `${tab.href}/`;
			if (pathname.startsWith(boundary) && tab.href.length > bestLen) {
				bestId = tab.id;
				bestLen = tab.href.length;
			}
		}
		return bestId;
	}
</script>

<script lang="ts">
	// THE ONE tabbed page shell (#0290 F1b) — the single composable surface for EVERY
	// admin page so each consumer carries ZERO layout CSS. Two modes share this shell:
	//
	//   LISTING / SCAFFOLD (default — F1b-i) — unifies the three old listing shells:
	//     - AdminPage      → composed back/title/subtitle + topbar-forwarded actions
	//     - AdminPageShell → config-driven title + route-driven tab strip (layout level)
	//     - AdminScaffoldPage → landing title (4xl/accent) + subtitle
	//   DETAIL (F1b-ii — `surfaceRoot` set) — folds in the old EntityDetailShell:
	//     sr-only <h1> (identity rides the breadcrumb + compact toolbar, not a visual
	//     band), facet tab strip, and the topbar cover/status-chips/actions plumbing that
	//     PERSISTS across facet sub-routes within the entity.
	//
	// Render order (top → bottom): heading band [back? → <PageTitle> + subtitle?] when
	// heading='visible'; an sr-only <h1> when heading='sr-only'; nothing when 'none' →
	// then [tabs? (only ≥2 — "no fake tabs")] → children.
	//
	// `actions` does NOT render its own region — it FORWARDS into the shell layout's
	// persistent topbar `actions` slot (useTopbarSlots), so page CTAs sit in the shell
	// topbar exactly like the properties-list New Property button. In detail mode the
	// `cover` + `status` snippet ride the topbar `toolbar` slot beside the breadcrumb.
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { afterNavigate, beforeNavigate } from '$app/navigation';
	import { useTopbarSlots } from '@sbx/core-ui/stores/topbarSlots';
	import PageTitle from '../primitives/PageTitle.svelte';

	interface Props {
		title: string;
		subtitle?: string;
		/** Optional back link rendered above the title (label defaults to "Back"). */
		back?: { href: string; label?: string };
		tabs?: TabEntry[];
		/** Heading rendering:
		    - 'visible' (default) = the full title band (back? → <PageTitle> → subtitle?);
		    - 'sr-only' = a visually-hidden <h1> only — for DETAIL pages where the breadcrumb
		      (≥2 crumbs) carries identity, so the visual band is redundant but the page must
		      still keep exactly one accessible <h1> (a11y/SEO);
		    - 'none' = render NO heading at all (the page owns its own <h1>). */
		heading?: 'visible' | 'sr-only' | 'none';
		/** Title scale: 'default' = 3xl/strong (PageTitle lock); 'landing' = 4xl/accent
		    (the scaffold landing look — Analytics/Backup/Social/Transactions/Featured). */
		titleScale?: 'default' | 'landing';
		/**
		 * DETAIL mode (folded in from EntityDetailShell). The route prefix that scopes
		 * this entity's surface (e.g. `/manage/properties/<id>/`). Its presence switches
		 * the shell into detail mode: the registered topbar `toolbar` + `actions` are
		 * CLEARED only when navigating OUTSIDE this prefix, so they PERSIST across facet
		 * sub-routes (Information ↔ Media) but never leak onto a sibling surface.
		 */
		surfaceRoot?: string;
		/**
		 * DETAIL mode cover affordance for the compact toolbar (2606-001). A small rounded
		 * thumbnail + a "Change" link, rendered in the toolbar slot leading the status
		 * chips. The thumbnail shows `cover.url` (already resolved by the consumer) or a
		 * neutral placeholder tile when empty; "Change" links to `cover.href` (the entity's
		 * media surface). Omit it and no cover renders — additive.
		 */
		cover?: CoverRef;
		/** DETAIL mode chip row rendered in the topbar toolbar slot (publish/lifecycle/
		    offering chips). Registered beside the breadcrumb, persists across facet tabs. */
		status?: Snippet;
		/** Right-aligned page CTAs — forwarded to the shell topbar actions slot. */
		actions?: Snippet;
		children: Snippet;
	}

	let {
		title,
		subtitle,
		back,
		tabs,
		heading = 'visible',
		titleScale = 'default',
		surfaceRoot,
		cover,
		status,
		actions,
		children
	}: Props = $props();

	// Detail mode is keyed off `surfaceRoot` — its presence means "this is a per-entity
	// detail surface" and selects the EntityDetailShell-style topbar persistence below.
	const detailMode = $derived(Boolean(surfaceRoot));

	// Tabs are route-driven anchors, shown only with ≥2 real tabs (the "no fake tabs"
	// rule) — same gate + active-tab logic carried verbatim from AdminPageShell,
	// reading the live path.
	const showTabs = $derived(Boolean(tabs && tabs.length > 1));
	const activeId = $derived(tabs ? activeTabId(tabs, page.url.pathname) : undefined);

	// Forward `actions` (and, in detail mode, the compact `toolbar`) into the shell
	// topbar. Clearing is ALWAYS via beforeNavigate, NEVER $effect cleanup — Svelte's
	// destroy ordering races the layout re-render and leaks stale snippets across
	// navigation (the Topbar context rule). The two modes differ only in WHEN they
	// register and WHEN they clear:
	//
	//   LISTING (no surfaceRoot, carried verbatim from F1b-i, 2605-175): afterNavigate
	//   (re)registers `{toolbar: undefined, actions}` on mount and after each navigation;
	//   beforeNavigate clears ONLY when the pathname actually changes, so same-route query
	//   updates keep the slot populated.
	//
	//   DETAIL (surfaceRoot set, folded in from EntityDetailShell): onMount registers the
	//   compact toolbar (cover/status) + actions ONCE so they survive facet child swaps
	//   (afterNavigate would re-fire and flicker; onMount won't re-fire across facets);
	//   beforeNavigate clears ONLY when leaving surfaceRoot, so the slots persist across
	//   facet sub-routes (Information ↔ Media) but never leak onto a sibling surface.
	const topbar = useTopbarSlots();

	afterNavigate(() => {
		// Listing mode only — detail mode registers once on mount (below) so it doesn't
		// re-register on every facet navigation.
		if (detailMode) return;
		topbar.set({ toolbar: undefined, actions });
	});

	onMount(() => {
		// Detail mode registration (verbatim from EntityDetailShell). Fires ONCE — the
		// shell layout doesn't remount between facets, so this single registration
		// survives every facet child swap (afterNavigate would re-fire and flicker the
		// bar). Identity rides the breadcrumb; the compact toolbar (cover + status chips)
		// rides the toolbar slot beside it; `actions` ride the actions slot. With no
		// status/cover/actions it sets both slots undefined (a visual no-op, call fires).
		if (!detailMode) return;
		topbar.set({ toolbar: status || cover ? compactToolbar : undefined, actions });
	});

	beforeNavigate(({ to }) => {
		if (detailMode) {
			// Clear only when navigating OUTSIDE the entity surface — persist across facets.
			if (!surfaceRoot || !to || !to.url.pathname.startsWith(surfaceRoot)) {
				topbar.set({ toolbar: undefined, actions: undefined });
			}
			return;
		}
		// Listing mode — clear only when the pathname actually changes.
		if (to && to.url.pathname !== page.url.pathname) {
			topbar.set({ toolbar: undefined, actions: undefined });
		}
	});
</script>

<!-- DETAIL-mode compact toolbar (folded in verbatim from EntityDetailShell, 2606-001):
     an optional cover thumbnail + the status chips, registered into the topbar slot
     onMount and rendered by the shell layout beside the breadcrumb — never inline.
     Wrapped so the cover + chips keep a dense gap; the cover leads the chips so it sits
     closest to the entity title crumb. -->
{#snippet compactToolbar()}
	<span class="page-shell-toolbar-lead">
		{#if cover}
			<!-- Cover affordance: rounded thumbnail (or neutral placeholder when no url)
			     + a "Change" link to the entity's media surface (2606-001, D3). The whole
			     tile is the link when href is set, so the image and the change hint share
			     one target; inline-modal upload is a later refinement. -->
			{#if cover.href}
				<a
					class="page-shell-cover"
					href={cover.href}
					title="Change cover image"
					aria-label="Change cover image"
				>
					{#if cover.url}
						<img class="page-shell-cover-img" src={cover.url} alt="" />
					{:else}
						<span class="page-shell-cover-placeholder" aria-hidden="true"></span>
					{/if}
					<span class="page-shell-cover-change">Change</span>
				</a>
			{:else}
				<span class="page-shell-cover page-shell-cover--static">
					{#if cover.url}
						<img class="page-shell-cover-img" src={cover.url} alt="" />
					{:else}
						<span class="page-shell-cover-placeholder" aria-hidden="true"></span>
					{/if}
				</span>
			{/if}
		{/if}
		{#if status}
			<span class="page-shell-toolbar-chips">{@render status()}</span>
		{/if}
	</span>
{/snippet}

<div class="page-shell" class:page-shell--detail={detailMode}>
	{#if heading === 'visible'}
		<header class="page-shell-header">
			{#if back}
				<a class="page-shell-back" href={back.href}>
					<span class="page-shell-back-arrow" aria-hidden="true">←</span>
					{back.label ?? 'Back'}
				</a>
			{/if}
			<PageTitle {title} scale={titleScale} />
			{#if subtitle}
				<p class="page-shell-subtitle">{subtitle}</p>
			{/if}
		</header>
	{:else if heading === 'sr-only'}
		<!-- DETAIL pages: identity shows visually in the breadcrumb (≥2 crumbs) + the
		     compact-toolbar chips; this sr-only <h1> keeps the page's single accessible
		     heading (a11y/SEO) without re-introducing the visual title band. Class +
		     tokens carried verbatim from EntityDetailShell's .entity-shell__sr-title. -->
		<h1 class="page-shell-sr-title">{title}</h1>
	{/if}

	{#if showTabs && tabs}
		<nav class="page-shell-tabs" aria-label="{title} sections">
			{#each tabs as tab (tab.id)}
				{@const isActive = activeId === tab.id}
				{@const hasBadge = tab.badge !== undefined && tab.badge > 0}
				<a
					class="page-shell-tab"
					class:page-shell-tab--active={isActive}
					href={tab.href}
					aria-current={isActive ? 'page' : undefined}
					aria-label={hasBadge ? `${tab.label} (${tab.badge})` : undefined}
				>
					{tab.label}
					{#if hasBadge}
						<span class="page-shell-badge" aria-hidden="true">{tab.badge}</span>
					{/if}
				</a>
			{/each}
		</nav>
	{/if}

	<div class="page-shell-body">
		{@render children()}
	</div>
</div>

<style>
	/* This component OWNS the page's vertical rhythm — consumers add no layout CSS.
	   Matches the old shells' column gap so composed and config-driven surfaces read
	   identically. Top/bottom insets are the shell layout's job (--admin-content-inset),
	   so this sets only internal spacing (2605-210, carried from AdminPage). The outer
	   column gap is --spacing-xl in BOTH modes (it matched EntityDetailShell's
	   .entity-shell when that folded in at F1b-ii). */
	.page-shell {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-xl);
	}

	/* DETAIL-mode body (folded in from EntityDetailShell's .entity-shell__body): stacks
	   the consumer's section cards with --spacing-lg so a multi-card detail (e.g.
	   Contact: Categories · identity · Notes) never touches. Listing/scaffold bodies keep
	   the F1b-i behavior — a plain wrapper with no internal gap (a single root child, so
	   nothing to space) — preserved by gating this on the detail modifier. */
	.page-shell--detail .page-shell-body {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-lg);
	}

	/* sr-only detail heading — the visual title band is gone for detail pages (#0290 F1b);
	   identity shows in the breadcrumb + compact-toolbar chips. Keeps exactly one
	   accessible <h1> per detail page (a11y/SEO). Standard visually-hidden pattern,
	   carried verbatim from EntityDetailShell's .entity-shell__sr-title. */
	.page-shell-sr-title {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	.page-shell-header {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-xs);
	}

	/* Back link — sits above the title, neutral until hover. */
	.page-shell-back {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-xs);
		align-self: flex-start;
		font-family: var(--font-body);
		font-size: var(--text-sm);
		font-weight: 600;
		color: var(--color-text-muted);
		text-decoration: none;
		transition: color var(--duration-fast) var(--easing-default);
	}

	.page-shell-back:hover {
		color: var(--color-accent);
	}

	.page-shell-back-arrow {
		font-size: var(--text-base);
		line-height: 1;
	}

	/* Subtitle — folded in from AdminScaffoldPage; the supporting line under the
	   title. Full-width supporting copy (no max-width) per the narrative-width rule. */
	.page-shell-subtitle {
		font-family: var(--font-body);
		font-size: var(--text-sm);
		line-height: var(--leading-normal, 1.5);
		color: var(--color-text-muted);
		margin: 0;
	}

	/* Top tab bar — one real sub-route link per tab, active tab carries the accent
	   underline, the row sits on a hairline so it reads as a segmented control
	   attached to the header. Generalized verbatim from the property layout's
	   .property-tabs (2606-005), carried from AdminPage. */
	.page-shell-tabs {
		display: flex;
		gap: var(--spacing-lg);
		border-bottom: var(--border-width) solid var(--color-border);
	}

	.page-shell-tab {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-xs);
		padding: var(--spacing-sm) var(--spacing-xs);
		font-family: var(--font-body);
		font-size: var(--text-base);
		font-weight: 600;
		color: var(--color-text-muted);
		text-decoration: none;
		border-bottom: 2px solid transparent;
		margin-bottom: -1px;
		transition: color var(--duration-fast) var(--easing-default),
			border-color var(--duration-fast) var(--easing-default);
	}

	.page-shell-tab:hover {
		color: var(--color-accent);
	}

	.page-shell-tab--active {
		color: var(--color-accent);
		border-bottom-color: var(--color-accent);
	}

	/* Count pill — small unread/pending badge trailing the tab label. The link
	   carries an aria-label with the count, so this is aria-hidden chrome. */
	.page-shell-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.25rem;
		height: 1.25rem;
		padding: 0 0.375rem;
		font-family: var(--font-body);
		font-size: var(--text-xs);
		font-weight: 700;
		line-height: 1;
		color: var(--color-text-on-accent);
		background: var(--color-accent);
		border-radius: var(--radius-full);
	}

	/* ── DETAIL-mode compact toolbar (folded in verbatim from EntityDetailShell) ──────
	   The cover thumbnail + status chips render in the shell layout's topbar toolbar
	   slot beside the breadcrumb — NOT inline in this component's DOM. They still carry
	   THIS component's scope hash (the compactToolbar snippet is defined here), so these
	   plain scoped selectors reach them in the core-ui-rendered subtree — the same
	   mechanism the listing-mode `actions` snippet relies on. */

	/* Compact-mode leading cluster: the cover thumbnail + the chips, kept on one line
	   beside the breadcrumb with a small gap so the cover reads as part of the entity
	   identity, not floating chrome. */
	.page-shell-toolbar-lead {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-sm);
	}

	/* The consumer's status chips render in the toolbar slot beside the breadcrumb.
	   Inline-flex keeps them on one line with the same dense gap; align-items centers
	   them to the breadcrumb text. */
	.page-shell-toolbar-chips {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-xs);
	}

	/* Cover thumbnail — a small rounded tile. As a link it carries a "Change" hint that
	   reveals on hover/focus; as a static tile it's just the image. Fixed square so the
	   toolbar height never jumps between entities with/without a cover. The image fills
	   via object-fit:cover (never distorts, never clips text — this is chrome, not
	   content copy). */
	.page-shell-cover {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: var(--radius-sm);
		overflow: hidden;
		border: var(--border-width) solid var(--color-border);
		background: var(--color-bg-secondary);
		text-decoration: none;
		flex: none;
		transition: border-color var(--duration-fast) var(--easing-default);
	}

	.page-shell-cover:hover,
	.page-shell-cover:focus-visible {
		border-color: var(--color-accent);
	}

	.page-shell-cover-img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}

	/* Neutral placeholder tile when there is no cover url — a soft brand-tinted fill,
	   NEVER a broken <img>. */
	.page-shell-cover-placeholder {
		width: 100%;
		height: 100%;
		background: linear-gradient(
			135deg,
			var(--color-bg-secondary),
			var(--color-bg-muted)
		);
	}

	/* "Change" hint — overlaid label on the cover link, shown on hover/focus so the tile
	   reads as an actionable affordance without permanently covering the image. */
	.page-shell-cover-change {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		font-family: var(--font-body);
		font-size: 0.5625rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-on-accent);
		background: color-mix(in srgb, var(--color-accent) 78%, transparent);
		opacity: 0;
		transition: opacity var(--duration-fast) var(--easing-default);
	}

	.page-shell-cover:hover .page-shell-cover-change,
	.page-shell-cover:focus-visible .page-shell-cover-change {
		opacity: 1;
	}
</style>

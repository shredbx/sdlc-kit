<script lang="ts">
	/**
	 * RailsContentLayout — content column flanked by optional left/right rails,
	 * with an optional full-width header band above. THE reusable building block for
	 * the "1-2-1" family of admin surfaces (1-2-1, 1-3, 3-1, or plain 1) — every
	 * arrangement is the same component, configured by props, never a rewrite.
	 *
	 * Owns ONLY structure, responsive behaviour, and scroll mode. It knows nothing
	 * about any domain or brand — consumers attach their own building blocks into the
	 * `header` / `left` / `right` / `children` slots and pass brand tokens via props
	 * (e.g. `gap="var(--br-space-xl)"`).
	 *
	 *   ┌────────────────────────────────────────────────────────┐
	 *   │  header   (optional · full-bleed opt · size = content)  │
	 *   ├────────┬─────────────────────────────────┬─────────────┤
	 *   │ left   │            children              │   right     │
	 *   │ rail   │           (content)             │   rail      │
	 *   └────────┴─────────────────────────────────┴─────────────┘
	 *
	 * Per-rail scroll mode (independent):
	 *   'sticky'   — the rail sticks below `stickyTop` and gets its OWN scrollbar
	 *                when its content overflows the viewport (like a side panel).
	 *   'embedded' — the rail flows in the page's single scroll, taking its natural
	 *                height (the content column drives one general scroll).
	 *   Mixing is fine: left='embedded' + right='sticky'.
	 *
	 * Responsive: each rail auto-hides below its `leftAt` / `rightAt` width via
	 * matchMedia (SSR renders both rails; the client refines on mount). The tradeoff
	 * for SSR safety is a possible one-frame flash on a narrow first paint (rails
	 * render server-side, then hide on hydration) — fine for a desktop-first admin
	 * surface; weigh it for a mobile-first public one.
	 *
	 * `headerBleed` assumes this layout sits in a container padded by `--layout-pad-x`
	 * (the core-ui shell content column) — it negates exactly that inset to go edge-
	 * to-edge, so the bleed only aligns where the real inset equals `--layout-pad-x`.
	 *
	 * @example
	 * <RailsContentLayout
	 *   headerBleed
	 *   showRight={false}
	 *   leftWidth="160px"
	 *   gap="var(--br-space-xl)"
	 * >
	 *   {#snippet header()}<PropertyHero ... />{/snippet}
	 *   {#snippet left()}<SectionRail ... />{/snippet}
	 *   {@render children()}
	 * </RailsContentLayout>
	 *
	 * @layer layouts (level 4 in atomic design)
	 */
	import type { Snippet } from 'svelte';

	type RailScroll = 'sticky' | 'embedded';

	let {
		header,
		left,
		right,
		children,
		showLeft = !!left,
		showRight = !!right,
		leftWidth = '160px',
		rightWidth = '220px',
		leftScroll = 'sticky',
		rightScroll = 'sticky',
		leftAt = 900,
		rightAt = 1200,
		headerBleed = false,
		headerGap = '0',
		gap = '1.5rem',
		stickyTop = 'var(--layout-anchor-offset, 4rem)'
	}: {
		/** Optional full-width band above the columns (e.g. an entity hero). */
		header?: Snippet;
		/** Optional left rail (e.g. section nav). */
		left?: Snippet;
		/** Optional right rail (e.g. stats / assistance / media list). */
		right?: Snippet;
		/** Center content column. Required. */
		children: Snippet;
		/** Attach/detach the left rail (default: shown when `left` is provided). */
		showLeft?: boolean;
		/** Attach/detach the right rail (default: shown when `right` is provided). */
		showRight?: boolean;
		/** Left column track — any CSS width (px · fr · clamp). 1-3 = small rail + 1fr. */
		leftWidth?: string;
		/** Right column track — any CSS width. */
		rightWidth?: string;
		/** Left rail scroll mode — own scrollbar (sticky) or page flow (embedded). */
		leftScroll?: RailScroll;
		/** Right rail scroll mode — own scrollbar (sticky) or page flow (embedded). */
		rightScroll?: RailScroll;
		/** Hide the left rail below this viewport width (px). */
		leftAt?: number;
		/** Hide the right rail below this viewport width (px). */
		rightAt?: number;
		/** Break the header out of the layout's horizontal padding (full-bleed). */
		headerBleed?: boolean;
		/** Vertical space between the header band and the columns row. */
		headerGap?: string;
		/** Gap between columns. Pass a brand token from the consumer. */
		gap?: string;
		/** Sticky offset for sticky-mode rails (where they dock below on scroll). */
		stickyTop?: string;
	} = $props();

	// Responsive gates — matchMedia keyed off the per-rail breakpoints. $effect runs
	// client-only, so SSR keeps both true (renders the full layout) and the client
	// refines on mount. Custom breakpoints work because the query string is built
	// from the props.
	let wideForLeft = $state(true);
	let wideForRight = $state(true);
	$effect(() => {
		const mqL = window.matchMedia(`(min-width: ${leftAt}px)`);
		const mqR = window.matchMedia(`(min-width: ${rightAt}px)`);
		const update = () => {
			wideForLeft = mqL.matches;
			wideForRight = mqR.matches;
		};
		update();
		mqL.addEventListener('change', update);
		mqR.addEventListener('change', update);
		return () => {
			mqL.removeEventListener('change', update);
			mqR.removeEventListener('change', update);
		};
	});

	const renderLeft = $derived(!!left && showLeft && wideForLeft);
	const renderRight = $derived(!!right && showRight && wideForRight);

	// Grid track template — only the present columns, so the content always fills the
	// remaining space (minmax(0,1fr) lets it shrink past min-content for nested grids).
	const columns = $derived(
		renderLeft && renderRight
			? `${leftWidth} minmax(0, 1fr) ${rightWidth}`
			: renderLeft
				? `${leftWidth} minmax(0, 1fr)`
				: renderRight
					? `minmax(0, 1fr) ${rightWidth}`
					: 'minmax(0, 1fr)'
	);
</script>

<div class="rails-layout">
	{#if header}
		<div class="rails-layout__header" class:rails-layout__header--bleed={headerBleed}>
			{@render header()}
		</div>
	{/if}
	<div
		class="rails-layout__cols"
		style:grid-template-columns={columns}
		style:gap
		style:padding-top={header ? headerGap : undefined}
	>
		{#if renderLeft}
			<aside
				class="rails-layout__rail"
				class:rails-layout__rail--sticky={leftScroll === 'sticky'}
				style:--rails-sticky-top={stickyTop}
			>
				{@render left?.()}
			</aside>
		{/if}
		<div class="rails-layout__content">
			{@render children()}
		</div>
		{#if renderRight}
			<aside
				class="rails-layout__rail"
				class:rails-layout__rail--sticky={rightScroll === 'sticky'}
				style:--rails-sticky-top={stickyTop}
			>
				{@render right?.()}
			</aside>
		{/if}
	</div>
</div>

<style>
	.rails-layout {
		display: flex;
		flex-direction: column;
	}

	/* Header band — full-width above the columns. `--bleed` breaks out of the
	   layout's horizontal padding so a hero can paint edge-to-edge; its content
	   re-pads itself (the consumer's hero owns its own inner padding + background). */
	.rails-layout__header--bleed {
		margin-inline: calc(-1 * var(--layout-pad-x, 2rem));
	}

	/* Column row. align-items:start so a sticky rail can dock at its own top rather
	   than stretch to the content height. */
	.rails-layout__cols {
		display: grid;
		align-items: start;
	}

	.rails-layout__content {
		min-width: 0;
	}

	.rails-layout__rail {
		min-width: 0;
	}

	/* Sticky mode — the rail stays in view and gets its OWN scroll when it overflows
	   the viewport (max-height bounds it to the space below the dock line). Embedded
	   mode is the default block flow (no rule needed): it scrolls with the page. */
	.rails-layout__rail--sticky {
		position: sticky;
		top: var(--rails-sticky-top, 4rem);
		max-height: calc(100dvh - var(--rails-sticky-top, 4rem));
		overflow-y: auto;
	}
</style>

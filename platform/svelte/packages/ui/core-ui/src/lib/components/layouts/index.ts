/**
 * Layouts — Page-level composition shells (Templates in Atomic Design)
 *
 * Layout components own the page grid: sidebar zones, hero zones, content zones.
 * Consumer pages pass content via snippets — they never write layout CSS.
 *
 * @layer layouts (level 4 in atomic design)
 */

// Page shells
export { default as CollapsibleSidebarLayout } from './CollapsibleSidebarLayout.svelte';
export { default as PageShell } from './PageShell.svelte';

// TabbedPageShell — THE one composable admin page shell (#0290): listing/scaffold +
// detail modes, route-driven tab strip, topbar actions/toolbar forwarding. Promoted
// from BR-local (F2). The active-tab rule + tab/cover types are exported for consumers.
export { default as TabbedPageShell } from './TabbedPageShell.svelte';
export { activeTabId, type TabEntry, type CoverRef } from './TabbedPageShell.svelte';

// RailsContentLayout — content column flanked by optional left/right rails + an
// optional header band. The reusable "1-2-1" family primitive (1-2-1 / 1-3 / 3-1 /
// plain 1), configured by props: per-rail attach/detach, width, scroll mode
// (sticky | embedded), and responsive breakpoints. Consumers attach their own rail
// building blocks into the slots; the layout owns only structure + scroll.
export { default as RailsContentLayout } from './RailsContentLayout.svelte';

// Backward-compat alias
export { default as SidebarLayout } from './CollapsibleSidebarLayout.svelte';

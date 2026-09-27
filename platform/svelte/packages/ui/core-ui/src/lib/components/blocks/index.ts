/**
 * Blocks Module - Composite UI elements (Molecules in Atomic Design)
 *
 * Blocks are composed of primitives and represent meaningful UI units
 * like list items, cards, and form groups. They have semantic meaning
 * within the application context.
 *
 * @layer blocks (layer 2 in modified atomic design)
 *
 * @module blocks
 */

// =============================================================================
// COMPONENT EXPORTS
// =============================================================================

export { default as NavItem } from './NavItem.svelte';
export { default as Dock } from './Dock.svelte';
export { default as NavRail } from './NavRail.svelte';

// Knowledge Base Navigation
export { default as KnowledgeTabs } from './KnowledgeTabs.svelte';
export { default as KnowledgeSidebar } from './KnowledgeSidebar.svelte';

// Entity Page Tabs
export { default as Tabs } from './Tabs.svelte';
export type { Tab } from './Tabs.svelte';

// News feed — aggregated RSS news section (NewsFeed orchestrates NewsItem rows).
export { default as NewsItem } from './NewsItem.svelte';
export type { NewsFeedItem } from './NewsItem.svelte';
export { default as NewsFeed } from './NewsFeed.svelte';
export type { NewsFilterOption, NewsFilterState, NewsView } from './NewsFeed.svelte';

// =============================================================================
// TYPE EXPORTS
// =============================================================================

export type NavItemVariant = 'default' | 'floating' | 'sidebar' | 'pill';

export interface NavItemBadge {
	count?: number;
	label?: string;
	variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
}

export type { NavRailItem } from './NavRail.svelte';

// =============================================================================
// PRESENTER REGISTRY
// =============================================================================

import NavItemPresenter from './NavItem.presenter';
import NavRailPresenter from './NavRail.presenter';
import DockPresenter from './Dock.presenter';

export { default as NavItemPresenter } from './NavItem.presenter';
export { default as NavRailPresenter } from './NavRail.presenter';

export const blockPresenters = [
	NavItemPresenter,
	NavRailPresenter,
	DockPresenter
];

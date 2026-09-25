/**
 * Navigation — Wayfinding and sidebar navigation components
 *
 * @layer sections (level 3 in atomic design)
 */

// Breadcrumb
export { default as Breadcrumb } from './Breadcrumb.svelte';
export { default as PageBreadcrumb } from './PageBreadcrumb.svelte';
export type { BreadcrumbItem } from './Breadcrumb.svelte';

// DotNav
export { default as DotNav } from './DotNav.svelte';

// Sidebar navigation
export { default as Sidebar } from './Sidebar.svelte';
export { default as ContentSidebar } from './ContentSidebar.svelte';
export { default as SidebarHeader } from './SidebarHeader.svelte';
export { default as SidebarSection } from './SidebarSection.svelte';
export { default as ContentNavGroup } from './ContentNavGroup.svelte';
export { default as ContentNavItem } from './ContentNavItem.svelte';

// Types
export type { ContentSidebarSection, ContentSidebarItem } from './ContentSidebar.svelte';
export type { SidebarItem, SidebarSection as SidebarSectionType } from './Sidebar.svelte';

// Nav helpers (pure functions — drill-down resolution, breadcrumb derivation, role filtering)
export { resolveDrillDown } from './resolve-drilldown';
export type { DrillDownResult, ResolveDrillDownOptions } from './resolve-drilldown';
export { buildBreadcrumb } from './build-breadcrumb';
export type { BreadcrumbConfig } from './build-breadcrumb';
export { filterTreeByRole } from './filter-tree-by-role';

// Presenters
export { default as SidebarPresenter } from './Sidebar.presenter';

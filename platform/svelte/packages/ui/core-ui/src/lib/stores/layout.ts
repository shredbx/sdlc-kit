import { writable } from 'svelte/store';

/**
 * Layout content offset from left edge (px).
 * Written by SidebarLayout when sidebar expands/collapses.
 * Read by Footer and any root-level component that needs alignment.
 * Defaults to 0 (no offset — full-width content).
 */
export const layoutOffset = writable(0);

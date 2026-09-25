/**
 * topbarSlots — context-driven toolbar content for shell layouts.
 *
 * A shell layout (e.g. an admin `(admin)/+layout.svelte`) calls
 * `createTopbarSlots()` once. It renders the layout's toolbar/actions
 * snippets so they read from the returned reactive store. Descendant
 * pages call `useTopbarSlots()` and assign their own snippets to the
 * `toolbar` / `actions` fields to override the defaults for the
 * duration of that page.
 *
 * Why context-based: SvelteKit snippet props are wired statically at
 * the parent (layout) level — a child page cannot pass a snippet to
 * its grand-parent layout via prop drilling. Svelte 5 `getContext` /
 * `setContext` with a writable store lets a page register a snippet
 * that the layout renders.
 *
 * Per-instance scope: each `createTopbarSlots()` call sets its own
 * context frame, so nested shells don't bleed into one another.
 * Tests can mount a shell and write to the returned store directly.
 */

import { getContext, setContext, type Snippet } from 'svelte';
import { writable, type Writable } from 'svelte/store';

const KEY = Symbol('@sbx/core-ui/topbar-slots');

export interface TopbarSnippets {
	/** Left/center region of the topbar (breadcrumb default, filter bar override, …). */
	toolbar: Snippet | undefined;
	/** Right-aligned region of the topbar (Save / Publish / ⋯ / page CTAs). */
	actions: Snippet | undefined;
}

export type TopbarSlotsContext = Writable<TopbarSnippets>;

/**
 * Create a topbar-slots context. Called once by the shell layout that
 * owns the topbar. Returns a writable store whose `toolbar` and
 * `actions` fields pages can update via `useTopbarSlots()`.
 */
export function createTopbarSlots(): TopbarSlotsContext {
	const store = writable<TopbarSnippets>({ toolbar: undefined, actions: undefined });
	setContext(KEY, store);
	return store;
}

/**
 * Read the topbar-slots context. Must be called inside a shell layout
 * that called `createTopbarSlots()`. Throws if no provider is found.
 */
export function useTopbarSlots(): TopbarSlotsContext {
	const store = getContext<TopbarSlotsContext | undefined>(KEY);
	if (!store) {
		throw new Error(
			'useTopbarSlots: no provider. Wrap the page tree with a shell layout that calls createTopbarSlots().'
		);
	}
	return store;
}

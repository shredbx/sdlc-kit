// Ambient types for @event-calendar/core v5 (vkurko).
//
// The published package ships its Svelte component as source under the `svelte`
// export condition and carries NO `.d.ts`. svelte-check resolves the JS `default`
// condition for type info and falls back to an implicit `any`, which trips the
// no-`any` guardrail. This declaration types the SMALL surface @sbx/ui-calendar
// consumes — the Calendar component, the four render plugins, and the CSS side-
// effect import — so the wrapper stays fully typed without an `any` escape hatch.
//
// We intentionally do NOT re-derive EC's full options/callback shapes here: the
// wrapper (Calendar.svelte) defines its own typed `info` interfaces and a typed
// `EcOptions`, and passes `options` to the component below. Keeping this surface
// minimal means a library upgrade can't silently drift a hand-mirrored type.

declare module '@event-calendar/core' {
	import type { Component } from 'svelte';

	/**
	 * The imperative instance surface exposed via `bind:this` — the methods the
	 * wrapper calls (e.g. `unselect()` after a create). Typed as a focused subset;
	 * the full EC instance has more, but the wrapper only needs these.
	 */
	export interface EventCalendarInstance {
		unselect(): void;
		addEvent(event: Record<string, unknown>): unknown;
		removeEventById(id: string): void;
		refetchEvents(): void;
		getView(): { type: string };
		setOption(name: string, value: unknown): void;
	}

	/**
	 * The unified EC calendar component. Props: `plugins` (the active render/feature
	 * plugins) and `options` (the reactive config object the wrapper owns + types).
	 * `options` is `unknown` so a consumer's own strongly-typed options object fits
	 * without forcing an index signature. `bind:this` yields the instance above.
	 */
	export const Calendar: Component<
		{ plugins?: unknown[]; options?: unknown },
		EventCalendarInstance
	>;

	// Render + feature plugins (opaque tokens passed to `plugins`).
	export const TimeGrid: unknown;
	export const DayGrid: unknown;
	export const List: unknown;
	export const Interaction: unknown;
	export const ResourceTimeGrid: unknown;
	export const ResourceTimeline: unknown;

	// Imperative API (used when constructing without the Svelte component).
	export function createCalendar(
		target: Element,
		plugins: unknown[],
		options: Record<string, unknown>
	): unknown;
	export function destroyCalendar(instance: unknown): void;
}

// CSS side-effect import — no exported members, just makes the import type-clean.
declare module '@event-calendar/core/index.css';

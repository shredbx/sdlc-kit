<script lang="ts">
	// Reusable chevron-collapsible section (D17) used by every contextual panel.
	// Header = chevron + title (+ optional count) + optional right-aligned actions;
	// clicking the header toggles open/closed. Accessible: a real <button> with
	// aria-expanded, and the body region is hidden when collapsed. Open state is
	// local + self-managed (seeded from the `open` prop) — panels declare which
	// sections start collapsed (e.g. Template, Document info) via `open={false}`.
	import { untrack } from 'svelte';
	import { Icon } from '@sbx/core-ui/components/primitives';

	interface Props {
		title: string;
		/** Initial open state (default true). */
		open?: boolean;
		/** Optional count badge after the title (e.g. layer count). */
		count?: number;
		/** Compact spacing variant — tighter header/body padding + gap. Used by the
		 *  Inspector's Xcode-style category stack (left-rail panels keep the default). */
		dense?: boolean;
		/** Right-aligned header actions (e.g. a ＋ button). */
		actions?: import('svelte').Snippet;
		/** Section body. */
		children?: import('svelte').Snippet;
	}

	let { title, open = true, count, dense = false, actions, children }: Props = $props();

	// Self-managed open state, seeded ONCE from the prop (untrack makes the one-time
	// seed explicit — the section owns its open/closed state after mount). After
	// mount the prop is mostly a one-time declaration, with ONE reactive rule below.
	let isOpen = $state(untrack(() => open));

	// Auto-expand when `open` transitions to truthy (e.g. a Sources section gaining
	// its first attached record) so a just-added item is never hidden behind a
	// collapsed header. One-directional: it never force-collapses, so the user's own
	// collapse of a still-populated section sticks (open stays true → effect no-ops).
	$effect(() => {
		if (open) isOpen = true;
	});
</script>

<section class="section" class:section--open={isOpen} class:section--dense={dense}>
	<div class="section__head">
		<button
			class="section__toggle"
			type="button"
			aria-expanded={isOpen}
			onclick={() => (isOpen = !isOpen)}
		>
			<span class="section__chevron" aria-hidden="true">
				<Icon name={isOpen ? 'chevron-down' : 'chevron-right'} size="sm" />
			</span>
			<span class="section__title">{title}</span>
			{#if count !== undefined}
				<span class="section__count">{count}</span>
			{/if}
		</button>
		{#if actions}
			<span class="section__actions">{@render actions()}</span>
		{/if}
	</div>

	{#if isOpen}
		<div class="section__body">
			{@render children?.()}
		</div>
	{/if}
</section>

<style>
	.section {
		display: flex;
		flex-direction: column;
		border-bottom: 1px solid var(--cv-color-neutral-100, #f5f5f5);
	}

	.section__head {
		display: flex;
		align-items: center;
		gap: var(--cv-space-xs, 0.25rem);
	}

	.section__toggle {
		flex: 1;
		display: inline-flex;
		align-items: center;
		gap: var(--cv-space-xs, 0.25rem);
		min-width: 0;
		padding: 0.5rem 0;
		border: none;
		background: transparent;
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
		text-align: left;
	}

	.section__chevron {
		display: inline-flex;
		color: var(--cv-color-neutral-500, #707070);
	}

	.section__title {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.section__count {
		font-size: 0.6875rem;
		font-weight: 600;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	/* Header actions right-aligned (hard rule). */
	.section__actions {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
	}

	.section__body {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
		padding: 0 0 var(--cv-space-md, 1rem);
	}

	/* Dense variant (Inspector category stack) — tighter header + body so the
	   Xcode-style property list packs more rows per screen. */
	.section--dense .section__toggle {
		padding: 0.3125rem 0;
	}

	.section--dense .section__body {
		gap: var(--cv-space-xs, 0.25rem);
		padding: 0 0 var(--cv-space-sm, 0.5rem);
	}
</style>

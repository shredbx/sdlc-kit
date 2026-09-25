<script lang="ts">
	// Full-width editor toolbar (IA refactor R2) — the top-most row of the full-bleed
	// editor (D16), spanning the whole viewport above the rail + canvas + inspector.
	// It ABSORBS the dissolved BrandCard's chrome on the LEFT (← Back · Bestie RE serif
	// wordmark + MEDIA CANVAS label · document title + Template badge) and carries the
	// global actions on the RIGHT (the Animated toggle, Export, and the debounced
	// auto-save status pill — CTAs right-aligned, hard rule). There is no Save button:
	// persistence is automatic (the shell debounces on every document mutation;
	// Cmd/Ctrl-S forces it). The Animated toggle (R3) flips the doc between static and
	// motion render; Preview lands in S-PREVIEW (full-screen presentation). Light theme,
	// --cv-* tokens (consumer-mapped); the title never clips (wraps).
	import { Icon } from '@sbx/core-ui/components/primitives';

	interface Props {
		/** Active document title (allowed to wrap — never clipped). */
		title: string;
		/** Marks the doc as a reusable template (renders a badge). */
		isTemplate?: boolean;
		/** Where Back navigates (full-bleed editor escape, D16). Defaults to the Media
		 *  Canvas hub so the editor returns to the canvas list, not the generic /manage.
		 *  TODO(neutrality): a BR route as the default is a smell — the only consumer is
		 *  the BR editor (Bestie RE wordmark is also hardcoded below); when the package
		 *  gains a third consumer, drive this purely via the EditorShell prop. */
		backHref?: string;
		/** The Back link's text. Default targets the Media Canvas; the watermark surface
		 *  passes "Back to Watermarks" (Decision #0298). */
		backLabel?: string;
		/** Whether the Back link is shown. Default true. The watermark surface hides it
		 *  (its list is not a destination here) and routes home via the clickable wordmark. */
		showBack?: boolean;
		/** When set, the "Bestie RE" wordmark becomes a link to this href (the admin
		 *  dashboard). The watermark surface passes /manage so the wordmark is the way home. */
		homeHref?: string;
		/** The eyebrow under the wordmark — the surface name. Default 'MEDIA CANVAS'; the
		 *  watermark surface passes 'WATERMARK' (Decision #0298). */
		eyebrow?: string;
		/** Debounced auto-save status (drives the right-side pill):
		 *  'idle' resting clean — NOTHING renders (no permanent "Saved" clutter) ·
		 *  'saved' a transient post-save confirmation flash · 'pending' edits awaiting the
		 *  debounce · 'saving' a write is in flight · 'error' the last write failed. */
		status?: 'idle' | 'saved' | 'pending' | 'saving' | 'error';
		/** R3 animated mode — drives the Animated toggle's on/off state. */
		animated?: boolean;
		/** Whether the Animated toggle is shown at all (Decision #0298). Default true =
		 *  the existing Media Canvas; false hides it (watermark mode is static — no motion
		 *  authoring control). */
		showAnimatedToggle?: boolean;
		/** Toggle animated mode (the shell flips the doc flag → shows/hides the timeline
		 *  and switches the canvas between motion and static render). */
		onToggleAnimated?: (value: boolean) => void;
		/** Export the active page (S-EXPORT; R4 swaps this for the export dialog). */
		onexport?: () => void;
		/** Publish the design into the watermark registry (watermark mode · Decision
		 *  #0298). Rendered as a LABELLED CTA (not an icon — an upload glyph was
		 *  indistinguishable from Export) when the doc is NOT yet published. */
		onpublish?: () => void;
		/** Whether the current doc is already published/Active — flips the Publish CTA to
		 *  Unpublish (publishing an already-live doc is a no-op; the action you want is the
		 *  inverse). Drives which of onpublish/onunpublish the single CTA invokes. */
		published?: boolean;
		/** Unpublish the current doc (watermark mode) — the flipped state of the Publish CTA,
		 *  shown only when `published` is true. The consumer owns the API call + the refresh. */
		onunpublish?: () => void;
		/** Whether Publish is currently BLOCKED (e.g. a watermark must be square to publish).
		 *  The CTA still renders so the action is discoverable, but it's disabled with a reason
		 *  (`publishDisabledReason` as the hover title). Only affects the not-yet-published state. */
		publishDisabled?: boolean;
		/** Why Publish is disabled — the button's hover title (e.g. "Only square watermarks can
		 *  be published."). Shown only when `publishDisabled` is true. */
		publishDisabledReason?: string;
		/** Force a save NOW — the pill calls this when 'pending' (save now) or 'error'
		 *  (retry); the shell also binds it to Cmd/Ctrl-S. */
		onforcesave?: () => void;
	}

	let {
		title,
		isTemplate = false,
		backHref = '/manage/tools/media-canvas',
		backLabel = 'Back to Media Canvas',
		showBack = true,
		homeHref,
		eyebrow = 'MEDIA CANVAS',
		status = 'idle',
		animated = false,
		showAnimatedToggle = true,
		onToggleAnimated,
		onexport,
		onpublish,
		published = false,
		onunpublish,
		publishDisabled = false,
		publishDisabledReason,
		onforcesave
	}: Props = $props();
</script>

<header class="toolbar">
	<div class="toolbar__left">
		{#if showBack}
			<a class="toolbar__back" href={backHref}>
				<Icon name="arrow-left" size="sm" />
				<span>{backLabel}</span>
			</a>

			<span class="toolbar__divider" aria-hidden="true"></span>
		{/if}

		<!-- The wordmark doubles as the way home when `homeHref` is set (the watermark surface
		     has no Back link — the brand IS the exit to the admin dashboard). -->
		{#if homeHref}
			<a class="toolbar__mark toolbar__mark--link" href={homeHref}>
				<span class="toolbar__name">Bestie RE</span>
				<span class="toolbar__eyebrow">{eyebrow}</span>
			</a>
		{:else}
			<div class="toolbar__mark">
				<span class="toolbar__name">Bestie RE</span>
				<span class="toolbar__eyebrow">{eyebrow}</span>
			</div>
		{/if}

		<span class="toolbar__divider" aria-hidden="true"></span>

		<div class="toolbar__doc">
			<span class="toolbar__title">{title}</span>
			{#if isTemplate}
				<span class="toolbar__badge">Template</span>
			{/if}
		</div>
	</div>

	<div class="toolbar__right">
		<!-- Auto-save status (task 2607-033: FIRST in the right cluster, before the CTAs). A
		     reserved-width slot: at rest ('idle') NOTHING renders, but the slot keeps its
		     min-width so idle↔Saving↔Saved transitions never reflow the Export/Publish CTAs to
		     its right (they used to jump when the pill mounted/unmounted). Clickable when there's
		     an action to take (pending → save now, error → retry); a transient confirmation
		     otherwise (user directive 2026-06-26 — no permanent "Saved" pill). aria-live announces
		     each transition (Saving… → Saved → silence). -->
		<span class="toolbar__save" aria-live="polite">
			{#if status === 'error'}
				<button
					class="pill pill--error"
					type="button"
					onclick={() => onforcesave?.()}
					aria-label="Couldn’t save — retry"
				>
					<span class="pill__dot"></span>Retry save
				</button>
			{:else if status === 'pending'}
				<button
					class="pill pill--pending"
					type="button"
					onclick={() => onforcesave?.()}
					aria-label="Unsaved changes — save now"
				>
					<span class="pill__dot"></span>Unsaved
				</button>
			{:else if status === 'saving'}
				<span class="pill pill--saving"><span class="pill__dot pill__dot--pulse"></span>Saving…</span>
			{:else if status === 'saved'}
				<span class="pill pill--saved"><Icon name="check" size="sm" />Saved</span>
			{/if}
		</span>

		<!-- Animated mode toggle (R3): flips the doc between static and motion. role=switch
		     so AT announces on/off; the visual track/thumb is decorative. Hidden in a
		     static mode (watermark · #0298) — there is no motion to author. -->
		{#if showAnimatedToggle}
			<button
				class="anim-toggle"
				type="button"
				role="switch"
				aria-checked={animated}
				onclick={() => onToggleAnimated?.(!animated)}
			>
				<span class="anim-toggle__label">Animated</span>
				<span class="anim-toggle__track" class:anim-toggle__track--on={animated}>
					<span class="anim-toggle__thumb"></span>
				</span>
			</button>
		{/if}

		<!-- Export CTA — compact icon button (#7). Icon-forward keeps the bar light; the
		     label rides as title/aria so it stays discoverable + accessible. -->
		<button
			class="export-btn"
			type="button"
			onclick={() => onexport?.()}
			title="Export"
			aria-label="Export"
		>
			<Icon name="download" size="sm" />
		</button>

		<!-- Publish / Unpublish CTA (watermark mode · #0298) — a LABELLED action, NOT an
		     icon: an upload glyph read as a near-twin of the Export download icon, and was a
		     dead no-op once the doc was published (locked). When the doc is live it flips to
		     Unpublish (the inverse of what you can do). Primary fill = the commit; the
		     Unpublish state is a quiet outline (a retreat, not a CTA). -->
		{#if published && onunpublish}
			<button class="text-btn" type="button" onclick={() => onunpublish?.()}>Unpublish</button>
		{:else if !published && onpublish}
			<!-- Publish stays VISIBLE when blocked (discoverable) but disabled, with the reason as
			     its hover title — paired with the note next to the artboard size in Document settings. -->
			<button
				class="text-btn text-btn--primary"
				type="button"
				disabled={publishDisabled}
				title={publishDisabled ? publishDisabledReason : undefined}
				onclick={() => onpublish?.()}
			>Publish</button>
		{/if}
	</div>
</header>

<style>
	/* Chrome theme contract (design pass A, 2026-06-07): every chrome surface reads
	   --cv-chrome-* FIRST and falls back to the original light chain — a consumer
	   that sets the chrome tokens (BR: deep teal + gold) recolors the toolbar and
	   rail together; one that doesn't keeps the exact pre-pass look. */
	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--cv-space-sm, 0.5rem);
		padding: 0.375rem var(--cv-space-md, 1rem);
		background: var(--cv-chrome-bg, var(--cv-color-surface, #fff));
		border-bottom: 1px solid var(--cv-chrome-border, var(--cv-border-color, #e0e0e0));
	}

	/* Left chrome cluster. Wraps (never clips) so a long title flows to a second line
	   rather than truncating or pushing the right actions off-screen. */
	.toolbar__left {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--cv-space-md, 1rem);
		min-width: 0;
	}

	.toolbar__back {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--cv-chrome-fg, var(--cv-color-neutral-500, #707070));
		text-decoration: none;
	}

	.toolbar__back:hover {
		color: var(--cv-chrome-fg-strong, var(--cv-color-primary, #333333));
	}

	.toolbar__divider {
		width: 1px;
		height: 1.25rem;
		background: var(--cv-chrome-border, var(--cv-border-color, #e0e0e0));
	}

	.toolbar__mark {
		display: inline-flex;
		align-items: baseline;
		gap: var(--cv-space-sm, 0.5rem);
	}

	/* When the wordmark is the way home (watermark surface) it's an <a> — strip link
	   chrome so it reads as the brand, with a subtle hover/focus cue that it's clickable. */
	.toolbar__mark--link {
		text-decoration: none;
		border-radius: var(--cv-radius-sm, 6px);
		cursor: pointer;
	}

	.toolbar__mark--link:hover .toolbar__name {
		text-decoration: underline;
		text-underline-offset: 3px;
	}

	.toolbar__mark--link:focus-visible {
		outline: 2px solid var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		outline-offset: 2px;
	}

	.toolbar__name {
		font-family: var(--cv-font-heading, Georgia, serif);
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
	}

	.toolbar__eyebrow {
		font-family: var(--cv-font-body, system-ui, sans-serif);
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		color: var(--cv-chrome-fg, var(--cv-color-neutral-400, #a0a0a0));
	}

	.toolbar__doc {
		display: inline-flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--cv-space-sm, 0.5rem);
		min-width: 0;
	}

	.toolbar__title {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--cv-chrome-fg-strong, var(--cv-color-neutral-800, #202020));
		/* Never clip — long titles wrap. */
		overflow-wrap: anywhere;
	}

	.toolbar__badge {
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		padding: 0.125rem 0.5rem;
		border-radius: 999px;
		color: var(--cv-chrome-fg-strong, var(--cv-color-accent-dark, #555555));
		background: var(--cv-chrome-active-bg, var(--cv-color-accent-soft, rgba(0, 0, 0, 0.06)));
	}

	/* Right action cluster — CTAs right-aligned (hard rule); never shrinks. */
	.toolbar__right {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		flex-shrink: 0;
	}

	/* Export CTA — compact icon button (#7). Gold-accented outline that fills on hover,
	   so it stays an unmistakable CTA while taking a fraction of the old button's width. */
	.export-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		padding: 0;
		border: 1px solid var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		border-radius: var(--cv-radius-sm, 6px);
		background: transparent;
		color: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		cursor: pointer;
		transition:
			background 0.15s ease,
			color 0.15s ease;
	}

	.export-btn:hover {
		background: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		color: var(--cv-chrome-bg, var(--cv-color-surface, #fff));
	}

	@media (prefers-reduced-motion: reduce) {
		.export-btn {
			transition: none;
		}
	}

	/* Publish / Unpublish — a LABELLED CTA (text, not an icon) so it never reads as a
	   second Export. Same chrome-accent shell as the icon buttons but with room for the
	   word. Primary = filled (the commit, Publish); the bare outline is the Unpublish
	   retreat. */
	.text-btn {
		display: inline-flex;
		align-items: center;
		height: 1.75rem;
		padding: 0 0.75rem;
		font: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		border: 1px solid var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		border-radius: var(--cv-radius-sm, 6px);
		background: transparent;
		color: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		cursor: pointer;
		transition:
			background 0.15s ease,
			color 0.15s ease;
	}

	.text-btn:hover {
		background: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		color: var(--cv-chrome-bg, var(--cv-color-surface, #fff));
	}

	.text-btn--primary {
		background: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		color: var(--cv-chrome-bg, var(--cv-color-surface, #fff));
	}

	.text-btn--primary:hover {
		filter: brightness(0.92);
	}

	/* Blocked CTA (e.g. a non-square watermark can't publish) — visibly inert, no hover lift,
	   the hover title carries the reason. Discoverable but clearly not actionable. */
	.text-btn:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}

	.text-btn:disabled:hover {
		background: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
		filter: none;
	}

	@media (prefers-reduced-motion: reduce) {
		.text-btn {
			transition: none;
		}
	}

	/* Animated-mode switch (R3) — a labelled on/off toggle. */
	.anim-toggle {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0;
		border: none;
		background: transparent;
		font: inherit;
		color: var(--cv-chrome-fg, var(--cv-color-neutral-600, #505050));
		cursor: pointer;
	}

	.anim-toggle__label {
		font-size: 0.75rem;
		font-weight: 500;
	}

	.anim-toggle__track {
		position: relative;
		width: 2rem;
		height: 1.125rem;
		border-radius: 999px;
		background: var(--cv-color-neutral-300, #d9d9d9);
		transition: background 0.15s ease;
	}

	.anim-toggle__track--on {
		/* Chrome-accented ON state — gold on the BR green chrome, primary by default. */
		background: var(--cv-chrome-accent, var(--cv-color-primary, #333333));
	}

	.anim-toggle__thumb {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 0.875rem;
		height: 0.875rem;
		border-radius: 50%;
		background: #fff;
		transition: transform 0.15s ease;
	}

	.anim-toggle__track--on .anim-toggle__thumb {
		transform: translateX(0.875rem);
	}

	@media (prefers-reduced-motion: reduce) {
		.anim-toggle__track,
		.anim-toggle__thumb {
			transition: none;
		}
	}

	.toolbar__save {
		/* Reserved slot (task 2607-033): a fixed min-width, right-aligned content. The pill
		   renders nothing at 'idle' but the slot holds its width, so the Export/Publish CTAs to
		   its right never reflow as the status flips idle↔Saving↔Saved. Sits FIRST in the right
		   cluster (before the CTAs) so any residual width change pushes into the flexible gap,
		   never the CTAs. */
		display: inline-flex;
		align-items: center;
		justify-content: flex-end;
		min-width: 6.5rem;
	}

	/* Save status pill — shared shape; colour + interactivity vary by state. */
	.pill {
		display: inline-flex;
		align-items: center;
		gap: 0.3125rem;
		padding: 0.1875rem 0.5rem;
		border: 1px solid transparent;
		border-radius: 999px;
		font-size: 0.6875rem;
		font-weight: 600;
		white-space: nowrap;
	}

	button.pill {
		cursor: pointer;
	}

	.pill__dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: currentColor;
	}

	/* Resting "Saved" is the editor's DEFAULT state — keep it quiet (muted text, no filled
	   box) so it confirms without clutter. The actionable/transient states (Saving…,
	   Unsaved, Retry) keep their pill fills so they stand out against this calm baseline. */
	.pill--saved {
		color: var(--cv-chrome-fg, var(--cv-color-neutral-500, #707070));
		background: transparent;
		font-weight: 500;
	}

	.pill--saving {
		color: var(--cv-chrome-fg, var(--cv-color-neutral-500, #707070));
		background: var(--cv-chrome-hover-bg, var(--cv-color-neutral-100, #f5f5f5));
	}

	.pill--pending {
		color: var(--cv-chrome-fg-strong, var(--cv-color-accent-dark, #555555));
		background: var(--cv-chrome-active-bg, var(--cv-color-accent-soft, rgba(0, 0, 0, 0.06)));
	}

	.pill--error {
		color: var(--cv-color-error, #c0392b);
		background: var(--cv-color-error-soft, rgba(192, 57, 43, 0.1));
		border-color: var(--cv-color-error, #c0392b);
	}

	.pill__dot--pulse {
		animation: pill-pulse 1s ease-in-out infinite;
	}

	@keyframes pill-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.3;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.pill__dot--pulse {
			animation: none;
		}
	}
</style>

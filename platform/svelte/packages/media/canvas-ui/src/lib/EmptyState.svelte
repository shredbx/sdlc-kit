<script lang="ts">
	// Shared empty-state block — a small, descriptive placeholder used inside panel
	// sections that have no content yet (Media uploads, Texts fields, Map regions…).
	// These describe WHAT will fill the section ("No uploads yet", "Attach a source
	// in Settings to bind fields") — never "coming soon" (banned). One source for the
	// empty-state chrome so the structural panels never copy-paste this markup/CSS.
	interface Props {
		/** The descriptive message. */
		text: string;
		/** Optional CTA row under the message (e.g. an attach button) — rendered
		 *  right-aligned (hard rule). One definition here so the panels never
		 *  copy-paste the empty-state action chrome. */
		actions?: import('svelte').Snippet;
	}

	let { text, actions }: Props = $props();
</script>

<p class="empty">{text}</p>
{#if actions}
	<div class="empty__actions">{@render actions()}</div>
{/if}

<style>
	.empty {
		margin: 0;
		padding: var(--cv-space-md, 1rem);
		border: 1px dashed var(--cv-color-neutral-200, #e8e8e8);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		font-size: 0.75rem;
		line-height: 1.5;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* CTA row — right-aligned (hard rule), below the message box. */
	.empty__actions {
		display: flex;
		justify-content: flex-end;
		margin-top: var(--cv-space-sm, 0.5rem);
	}
</style>

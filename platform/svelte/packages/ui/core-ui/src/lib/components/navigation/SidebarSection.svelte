<script lang="ts">
	/**
	 * SidebarSection Component
	 *
	 * Non-clickable section divider/heading within a sidebar.
	 * Renders as an uppercase label that groups related sidebar items below it.
	 *
	 * Used in component browser and knowledge sidebars to label groups such as
	 * "Browse", "Categories", "Status" etc.
	 *
	 * @example
	 * <SidebarSection label="Browse" />
	 */

	// =============================================================================
	// PROPS
	// =============================================================================

	interface Props {
		/** The display label for this section heading */
		label: string;
	}

	let { label }: Props = $props();
</script>

<div class="sidebar-section" aria-label={label} role="heading" aria-level={3}>
	<span class="sidebar-section-label">{label}</span>
</div>

<style>
	.sidebar-section {
		padding: 0.25rem 0.5rem 0.25rem;
		margin-top: 1rem;
	}

	.sidebar-section:first-child {
		margin-top: 0.25rem;
	}

	.sidebar-section-label {
		display: block;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--color-text, #eeeff1);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		user-select: none;
	}
</style>

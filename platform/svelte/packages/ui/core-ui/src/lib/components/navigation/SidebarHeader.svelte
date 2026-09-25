<script lang="ts">
	/**
	 * SidebarHeader Component
	 *
	 * Top header block for a sidebar. Displays the "shred" + "bx" brand logo
	 * in accent red with a dynamic title on the right, plus an optional search input.
	 *
	 * Matches the `.sidebar-header` + `.sidebar-logo` pattern from:
	 * - packages-detail-a.html: icon + "ShredBX Packages" with search
	 * - components-browser-c-mixed.html: logo + search input
	 *
	 * Brand rule: "shred" and "bx" both render in var(--color-accent) (red).
	 *
	 * @example
	 * <SidebarHeader title="Packages" showSearch />
	 * <SidebarHeader title="Components" icon="⬛" showSearch />
	 */

	// =============================================================================
	// PROPS
	// =============================================================================

	interface Props {
		/** Page/section title displayed next to the logo (e.g. "Packages", "Components") */
		title: string;
		/** Optional emoji or unicode icon displayed before the logo */
		icon?: string;
		/** Whether to show the search input (default: false) */
		showSearch?: boolean;
		/** Placeholder text for the search input */
		searchPlaceholder?: string;
		/** Current search value (bindable) */
		searchValue?: string;
		/** Callback fired when search input changes */
		onSearch?: (value: string) => void;
		/** Optional keyboard shortcut label to display in the search input (e.g. "\u2318K") */
		searchShortcut?: string;
	}

	let {
		title,
		icon,
		showSearch = false,
		searchPlaceholder = 'Search...',
		searchValue = $bindable(''),
		onSearch,
		searchShortcut
	}: Props = $props();

	function handleInput(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		searchValue = input.value;
		onSearch?.(input.value);
	}
</script>

<div class="sidebar-header">
	<div class="sidebar-logo">
		{#if icon}
			<span class="sidebar-logo-icon" aria-hidden="true">{icon}</span>
		{:else}
			<!-- Default box/package SVG icon -->
			<svg
				class="sidebar-logo-svg"
				width="16"
				height="16"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				aria-hidden="true"
			>
				<path
					d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"
				/>
			</svg>
		{/if}
		{#if title}
			<span class="sidebar-logo-title">{title}</span>
		{/if}
	</div>

	{#if showSearch}
		<div class="sidebar-search-wrap">
			<svg
				class="sidebar-search-icon"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				aria-hidden="true"
			>
				<circle cx="11" cy="11" r="8" />
				<line x1="21" y1="21" x2="16.65" y2="16.65" />
			</svg>
			<input
				class="sidebar-search-input"
				type="search"
				placeholder={searchPlaceholder}
				value={searchValue}
				oninput={handleInput}
				aria-label="Search {title}"
			/>
			{#if searchShortcut}
				<kbd class="sidebar-search-kbd">{searchShortcut}</kbd>
			{/if}
		</div>
	{/if}
</div>

<style>
	.sidebar-header {
		padding: 0.625rem 0.5rem 0.625rem;
		border-bottom: 1px solid var(--color-border, #2a2a2d);
		flex-shrink: 0;
	}

	/* ==========================================================================
	   LOGO ROW
	   ========================================================================== */
	.sidebar-logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--color-text-muted, #8b8b90);
		margin-bottom: 0.5rem;
	}

	.sidebar-logo-icon {
		font-size: 1rem;
		line-height: 1;
		flex-shrink: 0;
	}

	.sidebar-logo-svg {
		flex-shrink: 0;
		color: var(--color-text-muted, #8b8b90);
	}

	.sidebar-logo-title {
		color: var(--color-text-muted, #8b8b90);
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	/* ==========================================================================
	   SEARCH INPUT
	   ========================================================================== */
	.sidebar-search-wrap {
		position: relative;
	}

	.sidebar-search-icon {
		position: absolute;
		left: 0.625rem;
		top: 50%;
		transform: translateY(-50%);
		width: 0.9375rem;
		height: 0.9375rem;
		color: var(--color-text-dim, #55555a);
		pointer-events: none;
	}

	.sidebar-search-input {
		width: 100%;
		background: var(--color-bg, #0a0a0b);
		border: 1px solid var(--color-border, #2a2a2d);
		border-radius: 0.375rem;
		padding: 0.4375rem 2.25rem 0.4375rem 2rem;
		color: var(--color-text, #eeeff1);
		font-family: inherit;
		font-size: 0.8rem;
		outline: none;
		transition: border-color 0.2s ease;
	}

	.sidebar-search-input::placeholder {
		color: var(--color-text-dim, #55555a);
	}

	.sidebar-search-input:focus {
		border-color: var(--color-accent, #e94560);
	}

	/* Remove default search input appearance */
	.sidebar-search-input::-webkit-search-decoration,
	.sidebar-search-input::-webkit-search-cancel-button,
	.sidebar-search-input::-webkit-search-results-button,
	.sidebar-search-input::-webkit-search-results-decoration {
		-webkit-appearance: none;
	}

	/* Keyboard shortcut badge */
	.sidebar-search-kbd {
		position: absolute;
		right: 0.5rem;
		top: 50%;
		transform: translateY(-50%);
		display: inline-flex;
		align-items: center;
		padding: 0.0625rem 0.3rem;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 0.1875rem;
		font-family: inherit;
		font-size: 0.625rem;
		color: var(--color-text-muted, #8b8b90);
		background: rgba(255, 255, 255, 0.04);
		opacity: 0.5;
		pointer-events: none;
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.sidebar-search-input {
			transition-duration: 0.01ms !important;
		}
	}
</style>

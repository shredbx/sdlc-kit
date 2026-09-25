<script lang="ts">
	/**
	 * NavRail Block Component
	 *
	 * A vertical navigation rail for editor layouts.
	 * Supports icon-only (with tooltips) or icon+label modes.
	 * Follows Material Design 3 nav rail pattern.
	 *
	 * Per architecture: Block (molecule) composed of IconButton primitives.
	 *
	 * @example
	 * <NavRail
	 *   items={[
	 *     { id: 'workspace', icon: 'home', tooltip: 'Workspace', label: 'Work' },
	 *     { id: 'hub', icon: 'hexagon', tooltip: 'Hub', label: 'Hub' }
	 *   ]}
	 *   activeId="workspace"
	 *   showLabels={true}
	 *   onSelect={(id) => console.log('Selected:', id)}
	 * />
	 */

	import IconButton from '../primitives/IconButton.svelte';
	import Tooltip from '../primitives/Tooltip.svelte';
	import Icon from '../primitives/Icon.svelte';

	export interface NavRailItem {
		/** Unique identifier */
		id: string;
		/** Lucide icon name */
		icon: string;
		/** Tooltip text (shown on hover when showLabels=false) */
		tooltip: string;
		/** Short label (shown under icon when showLabels=true) */
		label?: string;
		/** Keyboard shortcut hint */
		shortcut?: string;
		/** Show divider after this item */
		divider?: boolean;
		/** Position in rail */
		position?: 'top' | 'bottom';
		/** Disabled state */
		disabled?: boolean;
		/** Badge count */
		badge?: number;
	}

	interface NavRailProps {
		/** Navigation items */
		items: NavRailItem[];
		/** Currently active item ID */
		activeId?: string;
		/** Rail width in pixels (auto-calculated when showLabels=true) */
		width?: number;
		/** Show labels under icons instead of tooltips */
		showLabels?: boolean;
		/** Selection handler */
		onSelect?: (id: string) => void;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		items = [],
		activeId = '',
		width = 48,
		showLabels = false,
		onSelect = () => {},
		class: className = '',
		'data-view-id': viewId
	}: NavRailProps = $props();

	// Auto-widen when showing labels
	let effectiveWidth = $derived(showLabels ? 72 : width);

	// Split items by position
	let topItems = $derived(items.filter((item) => item.position !== 'bottom'));
	let bottomItems = $derived(items.filter((item) => item.position === 'bottom'));

	function handleSelect(id: string) {
		onSelect(id);
	}

	function handleKeyDown(e: KeyboardEvent, item: NavRailItem) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleSelect(item.id);
		}
	}
</script>

<nav
	class="nav-rail {className}"
	class:with-labels={showLabels}
	style="--rail-width: {effectiveWidth}px"
	aria-label="Mode navigation"
	data-view-id={viewId}
>
	<div class="nav-rail-top">
		{#each topItems as item (item.id)}
			<div class="nav-rail-item" class:has-divider={item.divider}>
				{#if showLabels}
					<!-- Label mode: icon + text below -->
					<button
						class="nav-item-labeled"
						class:active={activeId === item.id}
						disabled={item.disabled}
						onclick={() => handleSelect(item.id)}
						data-view-id={viewId ? `${viewId}-item-${item.id}` : undefined}
					>
						<Icon name={item.icon} size="sm" />
						<span class="nav-item-label">{item.label || item.tooltip}</span>
					</button>
				{:else}
					<!-- Icon-only mode with tooltip -->
					<Tooltip text={item.tooltip} position="right">
						<IconButton
							icon={item.icon}
							label={item.tooltip}
							variant={activeId === item.id ? 'primary' : 'ghost'}
							size="md"
							shape="rounded"
							active={activeId === item.id}
							disabled={item.disabled}
							badge={item.badge}
							tooltip={false}
							onclick={() => handleSelect(item.id)}
							data-view-id={viewId ? `${viewId}-item-${item.id}` : undefined}
						/>
					</Tooltip>
				{/if}
			</div>
		{/each}
	</div>

	{#if bottomItems.length > 0}
		<div class="nav-rail-bottom">
			{#each bottomItems as item (item.id)}
				<div class="nav-rail-item" class:has-divider={item.divider}>
					{#if showLabels}
						<button
							class="nav-item-labeled"
							class:active={activeId === item.id}
							disabled={item.disabled}
							onclick={() => handleSelect(item.id)}
							data-view-id={viewId ? `${viewId}-item-${item.id}` : undefined}
						>
							<Icon name={item.icon} size="sm" />
							<span class="nav-item-label">{item.label || item.tooltip}</span>
						</button>
					{:else}
						<Tooltip text={item.tooltip} position="right">
							<IconButton
								icon={item.icon}
								label={item.tooltip}
								variant={activeId === item.id ? 'primary' : 'ghost'}
								size="md"
								shape="rounded"
								active={activeId === item.id}
								disabled={item.disabled}
								badge={item.badge}
								tooltip={false}
								onclick={() => handleSelect(item.id)}
								data-view-id={viewId ? `${viewId}-item-${item.id}` : undefined}
							/>
						</Tooltip>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</nav>

<style>
	.nav-rail {
		display: flex;
		flex-direction: column;
		width: var(--rail-width, 48px);
		min-width: var(--rail-width, 48px);
		height: 100%;
		background: var(--nav-rail-bg, var(--color-bg-primary, #0f0f1a));
		border-right: 1px solid var(--nav-rail-border, var(--color-border, #2a2a4a));
		padding: 0.5rem 0;
	}

	.nav-rail-top {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem;
	}

	.nav-rail-bottom {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem;
		margin-top: auto;
		border-top: 1px solid var(--nav-rail-divider, var(--color-border, #2a2a4a));
		padding-top: 0.5rem;
	}

	.nav-rail-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		position: relative;
	}

	.nav-rail-item.has-divider::after {
		content: '';
		position: absolute;
		bottom: -0.375rem;
		left: 50%;
		transform: translateX(-50%);
		width: 24px;
		height: 1px;
		background: var(--nav-rail-divider, var(--color-border, #2a2a4a));
	}

	/* Active indicator */
	.nav-rail-item :global(.icon-button.active)::before {
		content: '';
		position: absolute;
		left: -4px;
		top: 50%;
		transform: translateY(-50%);
		width: 3px;
		height: 16px;
		background: var(--color-accent, #6366f1);
		border-radius: 0 2px 2px 0;
	}

	/* Hover state for the whole rail area */
	.nav-rail:hover {
		background: var(--nav-rail-bg-hover, var(--color-bg-primary, #0f0f1a));
	}

	/* Labeled mode styles */
	.nav-rail.with-labels {
		padding: 0.75rem 0;
	}

	.nav-rail.with-labels .nav-rail-top,
	.nav-rail.with-labels .nav-rail-bottom {
		gap: 0.125rem;
	}

	.nav-item-labeled {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.25rem;
		width: 100%;
		padding: 0.5rem 0.25rem;
		background: transparent;
		border: none;
		border-radius: 0.5rem;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.nav-item-labeled:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--color-text, #fff);
	}

	.nav-item-labeled.active {
		background: var(--color-accent-subtle, rgba(99, 102, 241, 0.15));
		color: var(--color-accent, #6366f1);
	}

	.nav-item-labeled:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.nav-item-label {
		font-size: 0.625rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.02em;
		line-height: 1;
		white-space: nowrap;
	}

	/* Responsive - slightly larger touch targets on mobile */
	@media (max-width: 768px) {
		.nav-rail {
			--rail-width: 56px;
		}

		.nav-rail.with-labels {
			--rail-width: 80px;
		}
	}
</style>

<script module lang="ts">
	export type { TreeItem, ContextSection } from './ContextPanel.types';
</script>

<script lang="ts">
	/**
	 * ContextPanel Section Component
	 *
	 * Left panel in the Interface Builder that shows contextual content
	 * based on the active nav rail selection. Supports tree, list, and grid views.
	 *
	 * Per architecture: Section (organism) composing blocks for editor context.
	 *
	 * @example
	 * <ContextPanel
	 *   activeSection="components"
	 *   sections={[
	 *     { id: 'components', type: 'tree', header: 'Components', items: [...] }
	 *   ]}
	 *   onItemSelect={(item) => console.log('Selected:', item)}
	 * />
	 */

	import Icon from '../primitives/Icon.svelte';
	import IconButton from '../primitives/IconButton.svelte';
	import Input from '../primitives/Input.svelte';
	import type { TreeItem, ContextSection } from './ContextPanel.types';

	interface ContextPanelProps {
		/** Currently active section ID (from nav rail) */
		activeSection: string;
		/** Section configurations */
		sections: ContextSection[];
		/** Panel width in pixels */
		width?: number;
		/** Minimum width when resizable */
		minWidth?: number;
		/** Allow collapse */
		collapsible?: boolean;
		/** Allow resize */
		resizable?: boolean;
		/** Currently collapsed */
		collapsed?: boolean;
		/** Item selection handler */
		onItemSelect?: (item: TreeItem, section: string) => void;
		/** Item expand/collapse handler */
		onItemToggle?: (item: TreeItem, section: string) => void;
		/** Action handler */
		onAction?: (actionId: string, section: string) => void;
		/** Collapse toggle handler */
		onCollapseToggle?: () => void;
		/** Search handler */
		onSearch?: (query: string, section: string) => void;
		/** Additional CSS classes */
		class?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		activeSection = '',
		sections = [],
		width = 240,
		minWidth = 160,
		collapsible = true,
		resizable = true,
		collapsed = false,
		onItemSelect = () => {},
		onItemToggle = () => {},
		onAction = () => {},
		onCollapseToggle = () => {},
		onSearch = () => {},
		class: className = '',
		'data-view-id': viewId
	}: ContextPanelProps = $props();

	// =============================================================================
	// STATE
	// =============================================================================

	let searchQuery = $state('');
	// svelte-ignore state_referenced_locally
	let panelWidth = $state(width);
	let isResizing = $state(false);

	// =============================================================================
	// DERIVED
	// =============================================================================

	// Use prop directly in $derived - props from $props() ARE reactive
	let currentSection = $derived(sections.find((s) => s.id === activeSection));

	let filteredItems = $derived.by(() => {
		if (!currentSection || !searchQuery) return currentSection?.items ?? [];

		const query = searchQuery.toLowerCase();

		function filterTree(items: TreeItem[]): TreeItem[] {
			return items
				.map((item) => {
					const matchesLabel = item.label.toLowerCase().includes(query);
					const filteredChildren = item.children ? filterTree(item.children) : [];
					const hasMatchingChildren = filteredChildren.length > 0;

					if (matchesLabel || hasMatchingChildren) {
						return {
							...item,
							children: filteredChildren,
							expanded: hasMatchingChildren ? true : item.expanded
						} as TreeItem;
					}
					return null;
				})
				.filter((item): item is TreeItem => item !== null);
		}

		return filterTree(currentSection.items);
	});

	// =============================================================================
	// HANDLERS
	// =============================================================================

	function handleItemClick(item: TreeItem) {
		if (item.disabled) return;

		if (item.children && item.children.length > 0) {
			onItemToggle(item, activeSection);
		} else {
			onItemSelect(item, activeSection);
		}
	}

	function handleItemSelect(item: TreeItem) {
		if (item.disabled) return;
		onItemSelect(item, activeSection);
	}

	function handleSearchInput(value: string) {
		searchQuery = value;
		onSearch(searchQuery, activeSection);
	}

	function handleActionClick(actionId: string) {
		onAction(actionId, activeSection);
	}

	// Resize handling
	function handleResizeStart(e: MouseEvent) {
		if (!resizable) return;
		e.preventDefault();
		isResizing = true;

		const startX = e.clientX;
		const startWidth = panelWidth;

		function onMouseMove(e: MouseEvent) {
			const delta = e.clientX - startX;
			panelWidth = Math.max(minWidth, Math.min(400, startWidth + delta));
		}

		function onMouseUp() {
			isResizing = false;
			window.removeEventListener('mousemove', onMouseMove);
			window.removeEventListener('mouseup', onMouseUp);
		}

		window.addEventListener('mousemove', onMouseMove);
		window.addEventListener('mouseup', onMouseUp);
	}

	// Keyboard navigation
	function handleKeyDown(e: KeyboardEvent, item: TreeItem) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleItemClick(item);
		} else if (e.key === 'ArrowRight' && item.children) {
			e.preventDefault();
			if (!item.expanded) {
				onItemToggle(item, activeSection);
			}
		} else if (e.key === 'ArrowLeft' && item.children) {
			e.preventDefault();
			if (item.expanded) {
				onItemToggle(item, activeSection);
			}
		}
	}
</script>

{#if !collapsed}
	<aside
		class="context-panel {className}"
		class:resizing={isResizing}
		style="--panel-width: {panelWidth}px; --min-width: {minWidth}px"
		aria-label="Context panel"
		data-view-id={viewId}
	>
		<!-- Header -->
		{#if currentSection}
			<header class="context-panel-header">
				<div class="header-title">
					{#if currentSection.header}
						<h2>{currentSection.header}</h2>
					{/if}
				</div>

				<div class="header-actions">
					{#if currentSection.actions}
						{#each currentSection.actions as action (action.id)}
							<IconButton
								icon={action.icon}
								label={action.tooltip}
								variant="ghost"
								size="sm"
								onclick={() => handleActionClick(action.id)}
							/>
						{/each}
					{/if}

					{#if collapsible}
						<IconButton
							icon="panel-left-close"
							label="Collapse panel"
							variant="ghost"
							size="sm"
							onclick={onCollapseToggle}
						/>
					{/if}
				</div>
			</header>

			<!-- Search -->
			{#if currentSection.searchable}
				<div class="context-panel-search">
					<Input
						type="search"
						placeholder="Search..."
						size="sm"
						value={searchQuery}
						oninput={handleSearchInput}
					/>
				</div>
			{/if}

			<!-- Content -->
			<div class="context-panel-content">
				{#if currentSection.type === 'tree'}
					<div class="tree-view" role="tree">
						{#each filteredItems as item (item.id)}
							{@render treeItem(item, 0)}
						{/each}
					</div>
				{:else if currentSection.type === 'list'}
					<ul class="list-view" role="listbox">
						{#each filteredItems as item (item.id)}
							{#if item.href}
								<li class="list-item-wrapper" class:selected={item.selected} class:disabled={item.disabled}>
									<a
										href={item.href}
										class="list-item list-item-link"
										role="option"
										aria-selected={item.selected}
									>
										{#if item.icon}
											<Icon name={item.icon} size="sm" />
										{/if}
										<span class="item-label">{item.label}</span>
										{#if item.badge !== undefined}
											<span class="item-badge">{item.badge}</span>
										{/if}
									</a>
								</li>
							{:else}
								<li
									class="list-item"
									class:selected={item.selected}
									class:disabled={item.disabled}
									role="option"
									aria-selected={item.selected}
									tabindex={item.disabled ? -1 : 0}
									onclick={() => handleItemSelect(item)}
									onkeydown={(e) => handleKeyDown(e, item)}
								>
									{#if item.icon}
										<Icon name={item.icon} size="sm" />
									{/if}
									<span class="item-label">{item.label}</span>
									{#if item.badge !== undefined}
										<span class="item-badge">{item.badge}</span>
									{/if}
								</li>
							{/if}
						{/each}
					</ul>
				{:else if currentSection.type === 'grid'}
					<div class="grid-view">
						{#each filteredItems as item (item.id)}
							<button
								class="grid-item"
								class:selected={item.selected}
								class:disabled={item.disabled}
								disabled={item.disabled}
								onclick={() => handleItemSelect(item)}
							>
								{#if item.icon}
									<Icon name={item.icon} size="lg" />
								{/if}
								<span class="item-label">{item.label}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{:else}
			<div class="context-panel-empty">
				<p>Select a mode from the nav rail</p>
			</div>
		{/if}

		<!-- Resize handle -->
		{#if resizable}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="resize-handle" onmousedown={handleResizeStart}></div>
		{/if}
	</aside>
{:else}
	<!-- Collapsed state - just a button to expand -->
	<button class="context-panel-collapsed" onclick={onCollapseToggle} aria-label="Expand panel">
		<Icon name="panel-left-open" size="sm" />
	</button>
{/if}

{#snippet treeItem(item: TreeItem, depth: number)}
	<div
		class="tree-item"
		class:expanded={item.expanded}
		class:selected={item.selected}
		class:disabled={item.disabled}
		class:has-children={item.children && item.children.length > 0}
		style="--depth: {depth}"
		role="treeitem"
		aria-expanded={item.children ? item.expanded : undefined}
		aria-selected={item.selected}
		tabindex={item.disabled ? -1 : 0}
		onclick={() => handleItemClick(item)}
		onkeydown={(e) => handleKeyDown(e, item)}
	>
		<div class="tree-item-content">
			{#if item.children && item.children.length > 0}
				<span class="expand-icon">
					<Icon name={item.expanded ? 'chevron-down' : 'chevron-right'} size="xs" />
				</span>
			{:else}
				<span class="expand-icon spacer"></span>
			{/if}

			{#if item.icon}
				<Icon name={item.icon} size="sm" />
			{/if}

			<span class="item-label">{item.label}</span>

			{#if item.badge !== undefined}
				<span class="item-badge">{item.badge}</span>
			{/if}
		</div>
	</div>

	{#if item.expanded && item.children}
		<div class="tree-children" role="group">
			{#each item.children as child (child.id)}
				{@render treeItem(child, depth + 1)}
			{/each}
		</div>
	{/if}
{/snippet}

<style>
	.context-panel {
		display: flex;
		flex-direction: column;
		width: var(--panel-width, 240px);
		min-width: var(--min-width, 160px);
		max-width: 400px;
		height: 100%;
		background: var(--context-panel-bg, var(--color-bg-secondary, #1a1a2e));
		border-right: 1px solid var(--context-panel-border, var(--color-border, #2a2a4a));
		position: relative;
		overflow: hidden;
	}

	.context-panel.resizing {
		user-select: none;
	}

	/* Header */
	.context-panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 0.75rem 0.5rem;
		border-bottom: 1px solid var(--color-border, #2a2a4a);
	}

	.header-title h2 {
		margin: 0;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-text-muted, #9ca3af);
	}

	.header-actions {
		display: flex;
		gap: 0.25rem;
	}

	/* Search */
	.context-panel-search {
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid var(--color-border, #2a2a4a);
	}

	/* Content */
	.context-panel-content {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
	}

	/* Tree View */
	.tree-view {
		padding: 0.5rem 0;
	}

	.tree-item {
		cursor: pointer;
		outline: none;
	}

	.tree-item-content {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.75rem;
		padding-left: calc(0.75rem + var(--depth, 0) * 1rem);
		color: var(--color-text-muted, #9ca3af);
		transition: background 0.1s ease, color 0.1s ease;
	}

	.tree-item:hover .tree-item-content {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.tree-item:focus-visible .tree-item-content {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: -2px;
	}

	.tree-item.selected .tree-item-content {
		background: var(--color-accent-muted, rgba(99, 102, 241, 0.2));
		color: var(--color-accent, #6366f1);
	}

	.tree-item.disabled .tree-item-content {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.expand-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1rem;
		height: 1rem;
		flex-shrink: 0;
	}

	.expand-icon.spacer {
		visibility: hidden;
	}

	/* List View */
	.list-view {
		list-style: none;
		margin: 0;
		padding: 0.5rem 0;
	}

	.list-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		cursor: pointer;
		color: var(--color-text-muted, #9ca3af);
		transition: background 0.1s ease, color 0.1s ease;
	}

	.list-item:hover {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.list-item.selected {
		background: var(--color-accent-muted, rgba(99, 102, 241, 0.2));
		color: var(--color-accent, #6366f1);
	}

	.list-item.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* List item wrapper for links */
	.list-item-wrapper {
		list-style: none;
	}

	.list-item-wrapper.selected .list-item-link {
		background: var(--color-accent-muted, rgba(99, 102, 241, 0.2));
		color: var(--color-accent, #6366f1);
	}

	.list-item-wrapper.disabled .list-item-link {
		opacity: 0.5;
		pointer-events: none;
	}

	.list-item-link {
		text-decoration: none;
	}

	/* Grid View */
	.grid-view {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
		gap: 0.5rem;
		padding: 0.75rem;
	}

	.grid-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.375rem;
		padding: 0.75rem 0.5rem;
		background: var(--color-bg-tertiary, #252540);
		border: 1px solid transparent;
		border-radius: 0.5rem;
		cursor: pointer;
		color: var(--color-text-muted, #9ca3af);
		transition: all 0.15s ease;
	}

	.grid-item:hover {
		background: var(--color-bg-hover, #2a2a4a);
		border-color: var(--color-border, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.grid-item.selected {
		background: var(--color-accent-muted, rgba(99, 102, 241, 0.2));
		border-color: var(--color-accent, #6366f1);
		color: var(--color-accent, #6366f1);
	}

	.grid-item.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.grid-item .item-label {
		font-size: 0.75rem;
		text-align: center;
		word-break: break-word;
	}

	/* Shared item styles */
	.item-label {
		flex: 1;
		font-size: 0.8125rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.item-badge {
		padding: 0.125rem 0.375rem;
		font-size: 0.625rem;
		font-weight: 600;
		background: var(--color-bg-tertiary, #374151);
		color: var(--color-text-muted, #9ca3af);
		border-radius: 9999px;
	}

	/* Empty state */
	.context-panel-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		padding: 2rem;
		color: var(--color-text-muted, #9ca3af);
		text-align: center;
	}

	/* Resize handle */
	.resize-handle {
		position: absolute;
		top: 0;
		right: 0;
		width: 4px;
		height: 100%;
		cursor: col-resize;
		background: transparent;
		transition: background 0.15s ease;
	}

	.resize-handle:hover,
	.context-panel.resizing .resize-handle {
		background: var(--color-accent, #6366f1);
	}

	/* Collapsed button */
	.context-panel-collapsed {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 100%;
		background: var(--context-panel-bg, var(--color-bg-secondary, #1a1a2e));
		border: none;
		border-right: 1px solid var(--context-panel-border, var(--color-border, #2a2a4a));
		color: var(--color-text-muted, #9ca3af);
		cursor: pointer;
		transition: background 0.15s ease, color 0.15s ease;
	}

	.context-panel-collapsed:hover {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}
</style>

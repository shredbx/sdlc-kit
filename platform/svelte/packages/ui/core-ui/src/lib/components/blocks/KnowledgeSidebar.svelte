<script lang="ts">
	/**
	 * KnowledgeSidebar - Collapsible sidebar for knowledge sub-categories
	 *
	 * Phase 21: Knowledge Base UI Navigation
	 *
	 * Chakra UI-inspired collapsible sections with:
	 * - Section headers that expand/collapse
	 * - Item counts per section
	 * - Active item highlighting
	 * - Search within current section
	 */

	import Icon from '../primitives/Icon.svelte';
	import Badge from '../primitives/Badge.svelte';

	interface SidebarSection {
		id: string;
		label: string;
		icon?: string;
		count?: number;
		items: SidebarItem[];
	}

	interface SidebarItem {
		id: string;
		label: string;
		count?: number;
		badge?: string;
		badgeVariant?: 'default' | 'primary' | 'success' | 'warning' | 'danger';
	}

	interface Props {
		sections?: SidebarSection[];
		activeSection?: string;
		activeItem?: string | null;
		collapsed?: boolean;
		onItemClick?: (sectionId: string, itemId: string) => void;
		onToggleCollapse?: () => void;
	}

	// Default sections for patterns tab
	const defaultSections: SidebarSection[] = [
		{
			id: 'structural',
			label: 'Structural',
			icon: 'building',
			count: 12,
			items: [
				{ id: 'adapter', label: 'Adapter', count: 3 },
				{ id: 'bridge', label: 'Bridge', count: 2 },
				{ id: 'composite', label: 'Composite', count: 4 },
				{ id: 'decorator', label: 'Decorator', count: 2 },
				{ id: 'facade', label: 'Facade', count: 1 }
			]
		},
		{
			id: 'behavioral',
			label: 'Behavioral',
			icon: 'activity',
			count: 15,
			items: [
				{ id: 'observer', label: 'Observer', count: 5 },
				{ id: 'strategy', label: 'Strategy', count: 3 },
				{ id: 'command', label: 'Command', count: 4 },
				{ id: 'state', label: 'State', count: 2 },
				{ id: 'template-method', label: 'Template Method', count: 1 }
			]
		},
		{
			id: 'creational',
			label: 'Creational',
			icon: 'plus-circle',
			count: 8,
			items: [
				{ id: 'factory', label: 'Factory', count: 3 },
				{ id: 'singleton', label: 'Singleton', count: 2 },
				{ id: 'builder', label: 'Builder', count: 2 },
				{ id: 'prototype', label: 'Prototype', count: 1 }
			]
		}
	];

	let {
		sections = defaultSections,
		activeSection = '',
		activeItem = null,
		collapsed = false,
		onItemClick = () => {},
		onToggleCollapse = () => {}
	}: Props = $props();

	// Track which sections are expanded
	// svelte-ignore state_referenced_locally
	let expandedSections = $state<Set<string>>(new Set(sections.map(s => s.id)));

	function toggleSection(sectionId: string) {
		const newSet = new Set(expandedSections);
		if (newSet.has(sectionId)) {
			newSet.delete(sectionId);
		} else {
			newSet.add(sectionId);
		}
		expandedSections = newSet;
	}

	function handleItemClick(sectionId: string, itemId: string) {
		onItemClick(sectionId, itemId);
	}
</script>

<aside class="knowledge-sidebar" class:collapsed>
	<div class="sidebar-header">
		<span class="title">Categories</span>
		<button class="collapse-btn" onclick={onToggleCollapse} title={collapsed ? 'Expand' : 'Collapse'}>
			<Icon name={collapsed ? 'chevron-right' : 'chevron-left'} size="sm" />
		</button>
	</div>

	{#if !collapsed}
		<nav class="sidebar-content">
			{#each sections as section (section.id)}
				<div class="section" class:expanded={expandedSections.has(section.id)}>
					<button
						class="section-header"
						class:active={activeSection === section.id && !activeItem}
						onclick={() => toggleSection(section.id)}
					>
						{#if section.icon}
							<Icon name={section.icon} size="sm" />
						{/if}
						<span class="section-label">{section.label}</span>
						{#if section.count !== undefined}
							<span class="section-count">{section.count}</span>
						{/if}
						<span class="chevron" class:rotated={expandedSections.has(section.id)}>
							<Icon name="chevron-down" size="xs" />
						</span>
					</button>

					{#if expandedSections.has(section.id)}
						<ul class="section-items">
							{#each section.items as item, i (`${section.id}-${item.id}-${i}`)}
								<li>
									<button
										class="item"
										class:active={activeSection === section.id && activeItem === item.id}
										onclick={() => handleItemClick(section.id, item.id)}
									>
										<span class="item-label">{item.label}</span>
										{#if item.badge}
											<Badge variant={item.badgeVariant || 'default'} size="sm">
												{item.badge}
											</Badge>
										{:else if item.count !== undefined}
											<span class="item-count">{item.count}</span>
										{/if}
									</button>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			{/each}
		</nav>
	{:else}
		<nav class="sidebar-collapsed">
			{#each sections as section (section.id)}
				<button
					class="collapsed-item"
					class:active={activeSection === section.id}
					onclick={() => {
						onToggleCollapse();
						toggleSection(section.id);
					}}
					title={section.label}
				>
					{#if section.icon}
						<Icon name={section.icon} size="sm" />
					{:else}
						<span class="initial">{section.label[0]}</span>
					{/if}
				</button>
			{/each}
		</nav>
	{/if}
</aside>

<style>
	.knowledge-sidebar {
		width: 240px;
		min-width: 240px;
		background: var(--color-bg-secondary);
		border-right: 1px solid var(--color-border);
		display: flex;
		flex-direction: column;
		transition: width 0.2s ease, min-width 0.2s ease;
	}

	.knowledge-sidebar.collapsed {
		width: 48px;
		min-width: 48px;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--spacing-sm) var(--spacing-md);
		border-bottom: 1px solid var(--color-border);
	}

	.collapsed .sidebar-header {
		padding: var(--spacing-sm);
		justify-content: center;
	}

	.title {
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--color-text-muted);
		letter-spacing: 0.5px;
	}

	.collapsed .title {
		display: none;
	}

	.collapse-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: all 0.15s ease;
	}

	.collapse-btn:hover {
		background: var(--color-bg-tertiary);
		color: var(--color-text);
	}

	.sidebar-content {
		flex: 1;
		overflow-y: auto;
		padding: var(--spacing-sm) 0;
	}

	/* Sections */
	.section {
		margin-bottom: var(--spacing-xs);
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: var(--spacing-sm);
		width: 100%;
		padding: var(--spacing-sm) var(--spacing-md);
		background: transparent;
		border: none;
		color: var(--color-text);
		cursor: pointer;
		font-size: 0.8125rem;
		font-weight: 500;
		text-align: left;
		transition: all 0.15s ease;
	}

	.section-header:hover {
		background: var(--color-bg-tertiary);
	}

	.section-header.active {
		color: var(--color-accent);
		background: var(--color-bg-tertiary);
	}

	.section-label {
		flex: 1;
	}

	.section-count {
		font-size: 0.7rem;
		color: var(--color-text-muted);
		background: var(--color-bg);
		padding: 2px 6px;
		border-radius: var(--radius-full);
	}

	.chevron {
		display: inline-flex;
		transition: transform 0.2s ease;
	}

	.chevron.rotated {
		transform: rotate(180deg);
	}

	/* Items */
	.section-items {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: var(--spacing-xs) var(--spacing-md);
		padding-left: calc(var(--spacing-md) + var(--spacing-lg));
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		font-size: 0.8125rem;
		text-align: left;
		transition: all 0.15s ease;
	}

	.item:hover {
		background: var(--color-bg-tertiary);
		color: var(--color-text);
	}

	.item.active {
		color: var(--color-accent);
		background: var(--color-bg-tertiary);
		font-weight: 500;
	}

	.item-label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.item-count {
		font-size: 0.7rem;
		color: var(--color-text-muted);
	}

	/* Collapsed mode */
	.sidebar-collapsed {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--spacing-xs);
		padding: var(--spacing-sm);
	}

	.collapsed-item {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: all 0.15s ease;
	}

	.collapsed-item:hover {
		background: var(--color-bg-tertiary);
		color: var(--color-text);
	}

	.collapsed-item.active {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.initial {
		font-size: 0.75rem;
		font-weight: 600;
	}

	/* Scrollbar */
	.sidebar-content::-webkit-scrollbar {
		width: 4px;
	}

	.sidebar-content::-webkit-scrollbar-track {
		background: transparent;
	}

	.sidebar-content::-webkit-scrollbar-thumb {
		background: var(--color-border);
		border-radius: var(--radius-full);
	}

	/* Responsive */
	@media (max-width: 768px) {
		.knowledge-sidebar {
			position: fixed;
			left: 0;
			top: 0;
			bottom: 0;
			z-index: 100;
			transform: translateX(-100%);
			transition: transform 0.2s ease;
		}

		.knowledge-sidebar:not(.collapsed) {
			transform: translateX(0);
		}
	}
</style>

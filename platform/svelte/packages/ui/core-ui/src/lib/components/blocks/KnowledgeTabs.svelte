<script lang="ts">
	/**
	 * KnowledgeTabs - Chakra UI-inspired horizontal tabs for knowledge sections
	 *
	 * Phase 21: Knowledge Base UI Navigation
	 *
	 * Sections:
	 * - Patterns (design patterns from GoF, POSA, DDD)
	 * - Standards (guidelines, best practices, checklists)
	 * - Toolkit (domain expert tools: 1password, git, docker, etc.)
	 * - Articles (blog, devlog, sci-fi)
	 * - Components (UI component library)
	 */

	import Icon from '../primitives/Icon.svelte';

	interface Tab {
		id: string;
		label: string;
		icon: string;
		count?: number;
		description?: string;
	}

	type SizeVariant = 'sm' | 'md' | 'lg';

	interface Props {
		tabs?: Tab[];
		activeTab?: string;
		onTabChange?: (tabId: string) => void;
		variant?: 'default' | 'underline' | 'pills';
		size?: SizeVariant;
	}

	const defaultTabs: Tab[] = [
		{ id: 'patterns', label: 'Patterns', icon: 'puzzle', description: 'GoF, POSA, DDD patterns' },
		{ id: 'standards', label: 'Standards', icon: 'shield-check', description: 'Guidelines & best practices' },
		{ id: 'toolkit', label: 'Toolkit', icon: 'wrench', description: 'Tools & domain experts' },
		{ id: 'articles', label: 'Articles', icon: 'file-text', description: 'Blog, devlog, tutorials' },
		{ id: 'components', label: 'Components', icon: 'layout', description: 'UI component library' }
	];

	let {
		tabs = defaultTabs,
		activeTab = 'patterns',
		onTabChange = () => {},
		variant = 'underline',
		size = 'md'
	}: Props = $props();

	function handleTabClick(tabId: string) {
		onTabChange(tabId);
	}
</script>

<div class="knowledge-tabs {variant} {size}" role="tablist">
	{#each tabs as tab (tab.id)}
		<button
			class="tab"
			class:active={activeTab === tab.id}
			role="tab"
			aria-selected={activeTab === tab.id}
			onclick={() => handleTabClick(tab.id)}
			title={tab.description}
		>
			<Icon name={tab.icon} size={size} />
			<span class="label">{tab.label}</span>
			{#if tab.count !== undefined && tab.count > 0}
				<span class="count">{tab.count}</span>
			{/if}
		</button>
	{/each}
</div>

<style>
	.knowledge-tabs {
		display: flex;
		gap: var(--spacing-xs);
		padding: var(--spacing-sm) 0;
		border-bottom: 1px solid var(--color-border);
	}

	.knowledge-tabs.pills {
		border-bottom: none;
		background: var(--color-bg-secondary);
		padding: var(--spacing-xs);
		border-radius: var(--radius-md);
	}

	.tab {
		display: flex;
		align-items: center;
		gap: var(--spacing-xs);
		padding: var(--spacing-sm) var(--spacing-md);
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		font-size: 0.875rem;
		font-weight: 500;
		border-radius: var(--radius-sm);
		transition: all 0.15s ease;
		position: relative;
	}

	/* Size variants */
	.sm .tab {
		padding: var(--spacing-xs) var(--spacing-sm);
		font-size: 0.75rem;
	}

	.lg .tab {
		padding: var(--spacing-md) var(--spacing-lg);
		font-size: 1rem;
	}

	.tab:hover {
		color: var(--color-text);
		background: var(--color-bg-secondary);
	}

	.tab.active {
		color: var(--color-accent);
	}

	/* Underline variant */
	.underline .tab.active::after {
		content: '';
		position: absolute;
		bottom: -1px;
		left: 0;
		right: 0;
		height: 2px;
		background: var(--color-accent);
		border-radius: var(--radius-full);
	}

	/* Pills variant */
	.pills .tab.active {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.pills .tab:hover:not(.active) {
		background: var(--color-bg-tertiary);
	}

	/* Default variant */
	.default .tab.active {
		background: var(--color-bg-tertiary);
	}

	.label {
		white-space: nowrap;
	}

	.count {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.5rem;
		height: 1.25rem;
		padding: 0 var(--spacing-xs);
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-full);
		font-size: 0.7rem;
		font-weight: 600;
	}

	.tab.active .count {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.pills .tab.active .count {
		background: var(--color-bg);
		color: var(--color-accent);
	}

	/* Responsive */
	@media (max-width: 768px) {
		.knowledge-tabs {
			overflow-x: auto;
			-webkit-overflow-scrolling: touch;
			scrollbar-width: none;
		}

		.knowledge-tabs::-webkit-scrollbar {
			display: none;
		}

		.tab .label {
			display: none;
		}

		.sm .tab .label,
		.md .tab .label {
			display: inline;
		}
	}
</style>

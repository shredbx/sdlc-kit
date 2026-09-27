<script lang="ts">
	interface DockItem {
		id: string;
		label: string;
		icon?: string;
		tooltip?: string;
	}

	let {
		items = [],
		activeId = '',
		showLabels = true,
		position = 'bottom',
		onSelect = (id: string) => {}
	}: {
		items: DockItem[];
		activeId?: string;
		showLabels?: boolean;
		position?: 'bottom' | 'top';
		onSelect?: (id: string) => void;
	} = $props();
</script>

<div class="dock-container" class:top={position === 'top'}>
	<nav class="dock" aria-label="Page sections">
		<ul class="dock-items">
			{#each items as item (item.id)}
				<li class="dock-item">
					<button
						class="dock-btn"
						class:active={activeId === item.id}
						title={item.tooltip ?? item.label}
						onclick={() => onSelect(item.id)}
					>
						{#if item.icon}
							<span class="dock-icon">{item.icon}</span>
						{/if}
						{#if showLabels}
							<span class="dock-label">{item.label}</span>
						{/if}
					</button>
				</li>
			{/each}
		</ul>
	</nav>
</div>

<style>
	.dock-container {
		display: flex;
		justify-content: center;
		padding: 1rem;
		margin-top: auto;
	}

	.dock-container.top {
		margin-top: 0;
		margin-bottom: auto;
	}

	.dock {
		display: inline-flex;
		background: var(--color-bg-secondary, #1a1a2e);
		border: 1px solid var(--color-border, #2a2a4a);
		border-radius: 1rem;
		padding: 0.5rem;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
	}

	.dock-items {
		display: flex;
		gap: 0.25rem;
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.dock-item {
		position: relative;
	}

	.dock-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		background: transparent;
		border: none;
		border-radius: 0.75rem;
		color: var(--color-text-muted, #9ca3af);
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.dock-btn:hover {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.dock-btn.active {
		background: var(--color-accent, #6366f1);
		color: #fff;
	}

	.dock-icon {
		font-size: 1.125rem;
	}

	.dock-label {
		font-weight: 500;
	}

	/* Indicator dot for active */
	.dock-btn.active::after {
		content: '';
		position: absolute;
		bottom: -4px;
		left: 50%;
		transform: translateX(-50%);
		width: 4px;
		height: 4px;
		background: var(--color-accent, #6366f1);
		border-radius: 50%;
	}

	/* Responsive */
	@media (max-width: 640px) {
		.dock {
			padding: 0.375rem;
		}

		.dock-btn {
			padding: 0.5rem 0.75rem;
			font-size: 0.75rem;
		}

		.dock-icon {
			font-size: 1rem;
		}
	}
</style>

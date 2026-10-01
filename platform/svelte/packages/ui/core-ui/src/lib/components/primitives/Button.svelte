<script lang="ts">
	import type { Snippet } from 'svelte';

	type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'outline' | 'destructive';
	type ButtonSize = 'sm' | 'md' | 'lg';

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		icon = '',
		children,
		onclick
	}: {
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		loading?: boolean;
		icon?: string;
		children?: Snippet;
		onclick?: () => void;
	} = $props();
</script>

<button
	class="btn btn-{variant} btn-{size}"
	{disabled}
	class:loading
	{onclick}
>
	{#if loading}
		<span class="spinner">⟳</span>
	{:else if icon}
		<span class="icon">{icon}</span>
	{/if}
	{#if children}
		<span class="label">
			{@render children()}
		</span>
	{/if}
</button>

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		font-weight: 500;
		border-radius: 0.5rem;
		border: 1px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Sizes */
	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	.btn-md {
		padding: 0.5rem 1rem;
		font-size: 0.875rem;
	}

	.btn-lg {
		padding: 0.75rem 1.5rem;
		font-size: 1rem;
	}

	/* Variants */
	.btn-primary {
		background: var(--color-accent, #6366f1);
		color: #fff;
		border-color: var(--color-accent, #6366f1);
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--color-accent-hover, #4f46e5);
		border-color: var(--color-accent-hover, #4f46e5);
	}

	.btn-secondary {
		background: var(--color-bg-tertiary, #252540);
		color: var(--color-text, #fff);
		border-color: var(--color-border, #2a2a4a);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--color-bg-hover, #2a2a4a);
	}

	.btn-ghost {
		background: transparent;
		color: var(--color-text-muted, #9ca3af);
		border-color: transparent;
	}

	.btn-ghost:hover:not(:disabled) {
		background: var(--color-bg-hover, #2a2a4a);
		color: var(--color-text, #fff);
	}

	.btn-outline {
		background: transparent;
		color: var(--color-accent, #6366f1);
		border-color: var(--color-accent, #6366f1);
	}

	.btn-outline:hover:not(:disabled) {
		background: var(--color-accent, #6366f1);
		color: #fff;
	}

	.btn-destructive {
		background: var(--color-error, #ef4444);
		color: #fff;
		border-color: var(--color-error, #ef4444);
	}

	.btn-destructive:hover:not(:disabled) {
		background: #dc2626;
		border-color: #dc2626;
	}

	/* Loading state */
	.loading {
		pointer-events: none;
	}

	.spinner {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.icon {
		font-size: 1.125em;
	}
</style>

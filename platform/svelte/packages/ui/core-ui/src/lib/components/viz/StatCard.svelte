<script lang="ts">
	/**
	 * StatCard — a single headline metric: optional icon, big value, label.
	 *
	 * A generic, token-driven mirror of a dashboard stat card. The consumer maps
	 * brand tokens onto the neutral `--va-*` custom properties below; nothing is
	 * hardcoded to any one project's palette.
	 *
	 * Token contract (all have sensible fallbacks):
	 *   --va-card-bg        card surface           (fallback: #fff)
	 *   --va-card-radius    card corner radius     (fallback: 0.75rem)
	 *   --va-card-shadow    card elevation         (fallback: subtle shadow)
	 *   --va-card-pad       card padding           (fallback: 1.25rem)
	 *   --va-value-color    big-value text color   (fallback: currentColor)
	 *   --va-label-color    label text color       (fallback: #6b7280)
	 *   --va-accent         icon tint / accent     (fallback: currentColor)
	 *   --va-accent-soft    icon chip background    (fallback: rgba(0,0,0,0.06))
	 *
	 * @example
	 * <StatCard value={1280} label="Views">
	 *   {#snippet icon()}<EyeIcon />{/snippet}
	 * </StatCard>
	 */
	import type { Snippet } from 'svelte';

	interface StatCardProps {
		/** The headline number/text. Strings are rendered verbatim (e.g. "—"). */
		value: number | string;
		/** What the value measures, e.g. "Views" / "Daily visits". */
		label: string;
		/** Optional icon snippet rendered in a tinted chip beside the value. */
		icon?: Snippet;
		/** Additional CSS classes for layout composition by the consumer. */
		class?: string;
	}

	let { value, label, icon, class: className = '' }: StatCardProps = $props();
</script>

<div class="va-stat-card {className}">
	{#if icon}
		<div class="va-stat-card__icon" aria-hidden="true">{@render icon()}</div>
	{/if}
	<div class="va-stat-card__body">
		<span class="va-stat-card__value">{value}</span>
		<span class="va-stat-card__label">{label}</span>
	</div>
</div>

<style>
	.va-stat-card {
		display: flex;
		align-items: center;
		gap: var(--va-card-gap, 1rem);
		padding: var(--va-card-pad, 1.25rem);
		background: var(--va-card-bg, #fff);
		border-radius: var(--va-card-radius, 0.75rem);
		box-shadow: var(--va-card-shadow, 0 1px 3px rgba(0, 0, 0, 0.08));
	}

	.va-stat-card__icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 48px;
		height: 48px;
		flex-shrink: 0;
		border-radius: var(--va-icon-radius, 0.5rem);
		background: var(--va-accent-soft, rgba(0, 0, 0, 0.06));
		color: var(--va-accent, currentColor);
	}

	.va-stat-card__body {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.va-stat-card__value {
		font-family: var(--va-value-font, inherit);
		font-size: var(--va-value-size, 1.875rem);
		font-weight: 700;
		line-height: 1.1;
		color: var(--va-value-color, currentColor);
	}

	.va-stat-card__label {
		margin-top: 0.25rem;
		font-size: var(--va-label-size, 0.8125rem);
		color: var(--va-label-color, #6b7280);
	}
</style>

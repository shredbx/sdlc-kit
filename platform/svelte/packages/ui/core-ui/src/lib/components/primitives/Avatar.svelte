<script lang="ts">
	/**
	 * Avatar Primitive Component
	 *
	 * A user representation with image, initials, or icon fallback.
	 * Supports sizes, status indicators, groups, and IView conformance.
	 *
	 * @example
	 * <Avatar src="/user.jpg" alt="John Doe" />
	 * <Avatar name="John Doe" />
	 * <Avatar icon="👤" size="lg" />
	 * <Avatar src="/user.jpg" status="online" />
	 */

	import type { Snippet } from 'svelte';

	type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
	type AvatarShape = 'circle' | 'square' | 'rounded';
	type AvatarStatus = 'online' | 'offline' | 'busy' | 'away' | 'invisible';

	interface AvatarProps {
		/** Image source URL */
		src?: string;
		/** Alt text for image (required for accessibility when src provided) */
		alt?: string;
		/** User name for initials fallback */
		name?: string;
		/** Explicit initials text (overrides name-derived initials) */
		initials?: string;
		/** Icon fallback (emoji or icon name) */
		icon?: string;
		/** Size preset */
		size?: AvatarSize;
		/** Shape variant */
		shape?: AvatarShape;
		/** Status indicator */
		status?: AvatarStatus;
		/** Border/ring color */
		ring?: string;
		/** Background color for initials/icon mode */
		bgColor?: string;
		/** Text color for initials */
		textColor?: string;
		/** Clickable (adds button semantics) */
		clickable?: boolean;
		/** Loading state */
		loading?: boolean;
		/** Additional CSS classes */
		class?: string;
		/** Custom fallback content */
		fallback?: Snippet;
		/** Click handler */
		onclick?: () => void;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		src,
		alt,
		name,
		icon,
		size = 'md',
		shape = 'circle',
		status,
		ring,
		bgColor,
		textColor,
		clickable = false,
		loading = false,
		class: className = '',
		fallback,
		onclick,
		'data-view-id': viewId,
		...restProps
	}: AvatarProps = $props();

	// Compute initials from name
	let initials = $derived.by(() => {
		if (!name) return '';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
		return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
	});

	// Image loading state
	let imageLoaded = $state(false);
	let imageError = $state(false);

	function handleImageLoad() {
		imageLoaded = true;
		imageError = false;
	}

	function handleImageError() {
		imageError = true;
		imageLoaded = false;
	}

	// Show image only if src provided and loaded without error
	let showImage = $derived(src && !imageError && (imageLoaded || !loading));

	// Determine what to show
	let showMode = $derived.by(() => {
		if (showImage) return 'image';
		if (initials) return 'initials';
		if (icon) return 'icon';
		if (fallback) return 'fallback';
		return 'placeholder';
	});

	// Size mappings
	const sizeMap: Record<AvatarSize, string> = {
		xs: '1.5rem',
		sm: '2rem',
		md: '2.5rem',
		lg: '3rem',
		xl: '4rem',
		'2xl': '5rem'
	};

	const fontSizeMap: Record<AvatarSize, string> = {
		xs: '0.625rem',
		sm: '0.75rem',
		md: '0.875rem',
		lg: '1rem',
		xl: '1.25rem',
		'2xl': '1.5rem'
	};

	// Custom style
	let customStyle = $derived.by(() => {
		const styles: string[] = [];
		styles.push(`--avatar-size: ${sizeMap[size]}`);
		styles.push(`--avatar-font-size: ${fontSizeMap[size]}`);
		if (bgColor) styles.push(`--avatar-bg: ${bgColor}`);
		if (textColor) styles.push(`--avatar-text: ${textColor}`);
		if (ring) styles.push(`--avatar-ring: ${ring}`);
		return styles.join('; ');
	});

	// Status color mapping
	const statusColors: Record<AvatarStatus, string> = {
		online: '#10b981',
		offline: '#6b7280',
		busy: '#ef4444',
		away: '#f59e0b',
		invisible: 'transparent'
	};
</script>

<svelte:element
	this={clickable ? 'button' : 'div'}
	class="avatar avatar-{size} avatar-{shape} {className}"
	class:clickable
	class:loading
	class:has-ring={!!ring}
	style={customStyle}
	type={clickable ? 'button' : undefined}
	onclick={clickable ? onclick : undefined}
	data-view-id={viewId}
	role={clickable ? undefined : 'img'}
	aria-label={alt ?? name ?? 'User avatar'}
	{...restProps}
>
	{#if loading}
		<div class="avatar-skeleton"></div>
	{:else if showMode === 'image'}
		<img
			class="avatar-image"
			{src}
			alt={alt ?? name ?? ''}
			onload={handleImageLoad}
			onerror={handleImageError}
		/>
	{:else if showMode === 'initials'}
		<span class="avatar-initials">{initials}</span>
	{:else if showMode === 'icon'}
		<span class="avatar-icon">{icon}</span>
	{:else if showMode === 'fallback' && fallback}
		{@render fallback()}
	{:else}
		<span class="avatar-placeholder">👤</span>
	{/if}

	{#if status && status !== 'invisible'}
		<span
			class="avatar-status"
			style="--status-color: {statusColors[status]}"
			aria-label="{status} status"
		></span>
	{/if}
</svelte:element>

<style>
	.avatar {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--avatar-size);
		height: var(--avatar-size);
		background: var(--avatar-bg, var(--color-bg-tertiary, #252540));
		color: var(--avatar-text, var(--color-text, #fff));
		font-size: var(--avatar-font-size);
		font-weight: 600;
		overflow: hidden;
		flex-shrink: 0;
		user-select: none;
	}

	/* Shapes */
	.avatar-circle {
		border-radius: 50%;
	}

	.avatar-square {
		border-radius: 0;
	}

	.avatar-rounded {
		border-radius: 0.375rem;
	}

	/* Ring */
	.has-ring {
		box-shadow: 0 0 0 2px var(--avatar-ring, var(--color-accent, #6366f1));
	}

	/* Image */
	.avatar-image {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	/* Initials & Icon */
	.avatar-initials,
	.avatar-icon,
	.avatar-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
	}

	.avatar-icon {
		font-family: 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif;
	}

	/* Status indicator */
	.avatar-status {
		position: absolute;
		bottom: 0;
		right: 0;
		width: 25%;
		height: 25%;
		min-width: 8px;
		min-height: 8px;
		max-width: 14px;
		max-height: 14px;
		background: var(--status-color);
		border: 2px solid var(--color-bg-secondary, #1a1a2e);
		border-radius: 50%;
	}

	/* Clickable */
	.clickable {
		cursor: pointer;
		border: none;
		padding: 0;
		transition: all 0.15s ease;
	}

	.clickable:hover {
		transform: scale(1.05);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	}

	.clickable:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	/* Loading skeleton */
	.avatar-skeleton {
		width: 100%;
		height: 100%;
		background: linear-gradient(
			90deg,
			var(--color-bg-tertiary, #252540) 25%,
			var(--color-bg-secondary, #1a1a2e) 50%,
			var(--color-bg-tertiary, #252540) 75%
		);
		background-size: 200% 100%;
		animation: skeleton-shimmer 1.5s ease-in-out infinite;
	}

	@keyframes skeleton-shimmer {
		0% { background-position: 200% 0; }
		100% { background-position: -200% 0; }
	}

	.loading {
		pointer-events: none;
	}
</style>

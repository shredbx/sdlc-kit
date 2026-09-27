<script lang="ts">
	/**
	 * Icon Primitive Component
	 *
	 * A flexible icon display supporting Lucide icons, emoji, SVG symbols, and icon fonts.
	 * Supports size variants, colors, animations, and IView conformance.
	 *
	 * Icon Types:
	 * - lucide: Lucide icons via registry (default, recommended)
	 * - emoji: Native emoji characters (zero-dependency)
	 * - symbol: SVG symbol sprites via <use> element
	 * - font: Icon font classes (e.g., Material Icons, FontAwesome)
	 *
	 * @example
	 * <Icon name="bell" />
	 * <Icon name="settings" size="lg" />
	 * <Icon name="🔔" type="emoji" />
	 * <Icon name="loader" spin />
	 */

	import type { Snippet, Component } from 'svelte';
	import { getIcon } from './IconRegistry';

	type IconType = 'lucide' | 'emoji' | 'symbol' | 'font';
	type IconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';

	interface IconProps {
		/** Icon name (lucide name, emoji, symbol id, or font icon name) */
		name: string;
		/** Icon type */
		type?: IconType;
		/** Size preset or pixel value */
		size?: IconSize | number;
		/** Custom color (CSS color value) */
		color?: string;
		/** Spin animation for loading states */
		spin?: boolean;
		/** Pulse animation for attention */
		pulse?: boolean;
		/** Flip icon horizontally */
		flipX?: boolean;
		/** Flip icon vertically */
		flipY?: boolean;
		/** Rotation in degrees */
		rotate?: number;
		/** Accessible label for screen readers */
		label?: string;
		/** Icon font family (for type="font") */
		fontFamily?: 'material' | 'fontawesome' | 'lucide' | string;
		/** SVG sprite path (for type="symbol") */
		spritePath?: string;
		/** Additional CSS classes */
		class?: string;
		/** Fallback content if icon fails to load */
		fallback?: Snippet;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		name,
		type = 'lucide',
		size = 'md',
		color,
		spin = false,
		pulse = false,
		flipX = false,
		flipY = false,
		rotate = 0,
		label,
		fontFamily = 'material',
		spritePath = '/icons.svg',
		class: className = '',
		fallback,
		'data-view-id': viewId,
		...restProps
	}: IconProps = $props();

	// Size mappings for Lucide icons (in pixels)
	const lucideSizeMap: Record<IconSize, number> = {
		xs: 12,
		sm: 16,
		md: 20,
		lg: 24,
		xl: 32,
		'2xl': 40
	};

	// Get Lucide icon component
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let LucideIcon: Component<any> | undefined = $derived(
		type === 'lucide' ? getIcon(name) : undefined
	);

	// Size mappings (rem)
	const sizeMap: Record<IconSize, string> = {
		xs: '0.75rem',
		sm: '1rem',
		md: '1.25rem',
		lg: '1.5rem',
		xl: '2rem',
		'2xl': '2.5rem'
	};

	// Resolve size: number passes through as px, string preset uses map
	let resolvedFontSize = $derived(typeof size === 'number' ? `${size}px` : sizeMap[size]);
	let resolvedLucideSize = $derived(typeof size === 'number' ? size : lucideSizeMap[size]);

	// Computed styles
	let computedStyle = $derived.by(() => {
		const styles: string[] = [];
		styles.push(`font-size: ${resolvedFontSize}`);
		if (color) styles.push(`color: ${color}`);
		if (rotate !== 0) styles.push(`--icon-rotate: ${rotate}deg`);
		return styles.join('; ');
	});

	// Transform classes
	let transformClasses = $derived.by(() => {
		const classes: string[] = [];
		if (spin) classes.push('spin');
		if (pulse) classes.push('pulse');
		if (flipX) classes.push('flip-x');
		if (flipY) classes.push('flip-y');
		if (rotate !== 0) classes.push('rotated');
		return classes.join(' ');
	});

	// Font icon class mapping
	let fontIconClass = $derived.by(() => {
		if (type !== 'font') return '';
		switch (fontFamily) {
			case 'material':
				return 'material-icons';
			case 'fontawesome':
				return `fa fa-${name}`;
			case 'lucide':
				return `lucide lucide-${name}`;
			default:
				return fontFamily;
		}
	});
</script>

<span
	class="icon icon-{size} {transformClasses} {className}"
	style={computedStyle}
	role={label ? 'img' : 'presentation'}
	aria-label={label}
	aria-hidden={!label}
	data-view-id={viewId}
	data-icon-type={type}
	{...restProps}
>
	{#if type === 'lucide' && LucideIcon}
		<LucideIcon
			size={resolvedLucideSize}
			{color}
			strokeWidth={2}
			class="icon-lucide"
		/>
	{:else if type === 'lucide' && !LucideIcon}
		<!-- Fallback for unknown lucide icon names -->
		<span class="icon-fallback" title="Unknown icon: {name}">{name}</span>
	{:else if type === 'emoji'}
		<span class="icon-emoji">{name}</span>
	{:else if type === 'symbol'}
		<svg class="icon-svg" aria-hidden="true">
			<use href="{spritePath}#{name}" />
		</svg>
	{:else if type === 'font'}
		<i class="icon-font {fontIconClass}" aria-hidden="true">
			{#if fontFamily === 'material'}
				{name}
			{/if}
		</i>
	{:else if fallback}
		{@render fallback()}
	{/if}
</span>

<style>
	.icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		line-height: 1;
		vertical-align: middle;
		flex-shrink: 0;
	}

	/* Sizes */
	.icon-xs { width: 0.75rem; height: 0.75rem; }
	.icon-sm { width: 1rem; height: 1rem; }
	.icon-md { width: 1.25rem; height: 1.25rem; }
	.icon-lg { width: 1.5rem; height: 1.5rem; }
	.icon-xl { width: 2rem; height: 2rem; }
	.icon-2xl { width: 2.5rem; height: 2.5rem; }

	/* Content types */
	.icon :global(.icon-lucide) {
		width: 100%;
		height: 100%;
		flex-shrink: 0;
	}

	.icon-fallback {
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.65em;
		color: var(--color-text-muted, #9ca3af);
		opacity: 0.7;
		text-overflow: ellipsis;
		overflow: hidden;
		white-space: nowrap;
	}

	.icon-emoji {
		display: flex;
		align-items: center;
		justify-content: center;
		font-family: 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif;
	}

	.icon-svg {
		width: 100%;
		height: 100%;
		fill: currentColor;
	}

	.icon-font {
		font-size: inherit;
		line-height: 1;
	}

	/* Transforms */
	.flip-x {
		transform: scaleX(-1);
	}

	.flip-y {
		transform: scaleY(-1);
	}

	.rotated {
		transform: rotate(var(--icon-rotate, 0deg));
	}

	/* Animations */
	.spin {
		animation: icon-spin 1s linear infinite;
	}

	.pulse {
		animation: icon-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes icon-spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	@keyframes icon-pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	/* Combined transforms with animations */
	.flip-x.spin,
	.flip-y.spin,
	.rotated.spin {
		animation: none;
	}

	.flip-x.pulse,
	.flip-y.pulse {
		animation-name: icon-pulse;
	}
</style>

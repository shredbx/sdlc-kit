<script lang="ts">
	/**
	 * SocialIcon Primitive
	 *
	 * Maps a platform name to its brand SVG icon via socialIconPath() — a pure,
	 * unit-tested resolver (alias layer + globe fallback) in social-icon-paths.ts.
	 * Separate from IconRegistry (which is Lucide UI icons only).
	 *
	 * @layer primitives
	 * @decision D2 — SocialIcon separate from IconRegistry
	 */
	import { socialIconPath } from './social-icon-paths';

	let {
		platform,
		size = 14,
		class: className = ''
	}: {
		platform: string;
		size?: number;
		class?: string;
	} = $props();

	let path = $derived(socialIconPath(platform));
</script>

<svg
	class="social-icon {className}"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="currentColor"
	aria-label="{platform} icon"
>
	<path d={path} />
</svg>

<style>
	.social-icon {
		display: inline-block;
		vertical-align: middle;
		flex-shrink: 0;
	}
</style>

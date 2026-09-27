<script lang="ts">
	import { mediaSrcset, imgSizes, type ImgFit } from './media';

	interface Props {
		src: string;
		alt: string;
		fit?: ImgFit;
		sizes?: string;
		class?: string;
		loading?: 'lazy' | 'eager';
		decoding?: 'async' | 'auto' | 'sync';
		[key: string]: unknown;
	}

	let {
		src,
		alt,
		fit = 'card',
		sizes,
		class: className = '',
		loading = 'lazy',
		decoding = 'async',
		...rest
	}: Props = $props();

	const srcset = $derived(mediaSrcset(src));
	const resolvedSizes = $derived(sizes ?? imgSizes(fit));
</script>

<img
	{src}
	{alt}
	srcset={srcset || undefined}
	sizes={srcset ? resolvedSizes : undefined}
	class={className}
	{loading}
	{decoding}
	{...rest}
/>

<script lang="ts">
	// JsonLdScript — the SINGLE audited sink for emitting JSON-LD structured data
	// into the document head. Wraps the one at-html that serializes a schema node
	// (or array) into a JSON-LD script block. Owns its own svelte:head, so it can
	// be dropped anywhere in a page's markup (a sibling of the page's own
	// svelte:head), exactly like SeoHead.
	//
	// Use this instead of hand-writing the at-html serialize line per page: one
	// place to audit the XSS boundary. serializeJsonLd escapes every "<" to its
	// unicode escape, so user/CMS-derived text cannot break out of the script
	// element (a stray closing-script sequence in a title becomes inert JSON text).
	// SeoHead emits via the same serializer for the page's primary node; this
	// component covers pages that emit a secondary/standalone node (ItemList,
	// BreadcrumbList, Organization).
	import { serializeJsonLd } from './schema';
	import type { JsonLd } from './types';

	interface Props {
		/** schema.org node (or list) — build via the ./schema builders. */
		nodes: JsonLd | JsonLd[];
	}

	let { nodes }: Props = $props();
</script>

<svelte:head>
	{#if nodes}
		<!-- eslint-disable-next-line svelte/no-at-html-tags — serializeJsonLd escapes every "<" (XSS-safe) -->
		{@html '<script type="application/ld+json">' + serializeJsonLd(nodes) + '</' + 'script>'}
	{/if}
</svelte:head>

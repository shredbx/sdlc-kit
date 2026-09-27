<!--
RevealGroup — section-level preset override via Svelte context.

Wraps a group of <Reveal> children and provides a preset override via context.
Children read via getContext('reveal-group') as a fallback when no brand CSS vars
are set directly on the element.

Usage:
  <RevealGroup preset="expressive">
    <Reveal>Card 1</Reveal>
    <Reveal>Card 2</Reveal>
  </RevealGroup>

Decision #0220: RevealGroup provides the `preset` prop pattern for section-level
overrides without polluting individual component props.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { setContext } from 'svelte';
	import { transitionPresets, type TransitionPresetName, type TransitionPreset } from '@sbx/animations';

	interface Props {
		/** Override preset for all child Reveal components. */
		preset?: TransitionPresetName;
		as?: 'div' | 'section' | 'article';
		class?: string;
		children?: Snippet;
	}

	let { preset = 'minimal', as = 'div', class: className = '', children }: Props = $props();

	// RevealGroup is section-static: preset is chosen at composition time, not
	// changed at runtime. Capturing the initial value is the intended behavior.
	// svelte-ignore state_referenced_locally
	const resolvedPreset: TransitionPreset = transitionPresets[preset];
	setContext('reveal-group', resolvedPreset);
</script>

<svelte:element this={as} class="sbx-reveal-group {className}">
	{@render children?.()}
</svelte:element>

<style>
	.sbx-reveal-group {
		/* No visual styles — purely a context provider wrapper. */
		display: contents;
	}
</style>

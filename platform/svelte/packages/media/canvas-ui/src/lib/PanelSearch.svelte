<script lang="ts">
	// The shared search-field molecule — a small Input with a leading search glyph.
	// One source for the affordance so the panel-level search (Panel) and any
	// scoped per-section search (Templates) never re-declare the same markup
	// (never-copy-paste-CSS). Consumers own only their container/padding.
	import { Input, Icon } from '@sbx/core-ui/components/primitives';

	interface Props {
		/** Two-way bound query string. */
		value: string;
		placeholder?: string;
		/** Accessible label (the field is icon-only otherwise). */
		label: string;
	}

	let { value = $bindable(), placeholder = 'Search…', label }: Props = $props();
</script>

<Input bind:value placeholder={placeholder} size="sm" aria-label={label}>
	{#snippet left()}<Icon name="search" size="sm" />{/snippet}
</Input>

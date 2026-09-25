// @sbx/ui-source-picker — reusable popup record picker driven by the kit SourceProvider
// contract (property · guide · service · news · later contact). Composes @sbx/core-ui
// primitives; depends on @sbx/canvas-kit ONLY for the Source contract TYPES (erased at
// build). The Modal shell is supplied by the caller. Selection bubbles up — the caller
// maps the verb (add to a document · assign an unlinked slot · replace a linked source).
// See docs/plans/2026-06-04-ui-source-picker-design.md (task 2606-012).

export { default as SourcePicker } from './SourcePicker.svelte';

// Typed props for consumers — the contract lives in the component's `module` block.
export type {
	SourcePickerProps,
	SourcePickerSelect,
	SourcePickerView,
	SourcePickerCategoryDisplay
} from './SourcePicker.svelte';

// SourceBrowser — two-step wrapper that lets one dialog span multiple source kinds
// (kind chooser → the single-provider SourcePicker for the chosen kind).
export { default as SourceBrowser } from './SourceBrowser.svelte';
export type { SourceBrowserProps } from './SourceBrowser.svelte';

// The underlying data contract is the kit Source contract — import those directly from
// '@sbx/canvas-kit':
//   SourceProvider, SourceRecord, SourcePage, SourceCategory, SourceFacet, SourceFilters

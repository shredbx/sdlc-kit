/**
 * Primitives - Atomic UI Components
 *
 * Per Modified Atomic Design pattern:
 * Primitives are the smallest, indivisible UI elements.
 * They handle display, input, and action semantics.
 *
 * Categories:
 * - Display: Badge, Icon, Avatar (show data)
 * - Input: Input (accept user data)
 * - Action: Button, IconButton (trigger events)
 */

// Display primitives
export { default as Badge } from './Badge.svelte';
export { default as Icon } from './Icon.svelte';
export { default as Avatar } from './Avatar.svelte';

// Input primitives
export { default as Input } from './Input.svelte';
export { default as Select } from './Select.svelte';
// Checkbox — branded boolean control: a real (sr-only) native <input type=checkbox>
// inside a clickable .checkbox row with a teal-fill + white-tick custom box. Label
// wraps, never clipped. CheckboxGroup clusters related Checkboxes under an uppercase
// subhead in N sub-columns (composes the amenity quadrant).
export { default as Checkbox } from './Checkbox.svelte';
export { default as CheckboxGroup } from './CheckboxGroup.svelte';
// FontPicker — searchable typeahead over the Google Fonts catalogue (brand +
// recent + windowed all), replacing a brand-only <Select>. Lazily previews each
// row in its own face via the @sbx/core-ui/fonts loader.
export { default as FontPicker } from './FontPicker.svelte';
export type { BrandFont, CatalogueFont } from './FontPicker.svelte';
export { default as Textarea } from './Textarea.svelte';
export { default as SizeInput } from './SizeInput.svelte';
export { default as MoneyInput } from './MoneyInput.svelte';

// Form layout primitives — label/help/error wrapper + responsive grid.
// Every form control composes inside <Field>; <FieldGrid> lays Fields out.
// All share the --field-* token contract (see Field.svelte / Textarea.svelte
// header docs) so Input/Select/Textarea/MoneyInput/SizeInput render with one
// consistent visual language, themeable per consumer.
export { default as Field } from './Field.svelte';
export { default as FieldGrid } from './FieldGrid.svelte';
// AddressFields — the one true postal-address editor (FieldGrid+Field+Input over the 7 postal fields); Svelte mirror of pkg/address postal subset.
export { default as AddressFields } from './AddressFields.svelte';
export type { AddressValue, AddressLabels, AddressErrors } from './AddressFields.svelte';

// Social primitives
export { default as SocialIcon } from './SocialIcon.svelte';
export { default as SocialNetworkList, resolveSocialUrl } from './SocialNetworkList.svelte';
export type { SocialNetworkEntry, SocialNetworkPlatform } from './SocialNetworkList.svelte';
export { default as SocialNetworkListEditor } from './SocialNetworkListEditor.svelte';

// Action primitives
export { default as Button } from './Button.svelte';
export { default as IconButton } from './IconButton.svelte';

// Page heading — the single locked page-title scale (default 3xl + landing 4xl),
// promoted from BR-local AdminTitle (#0290 F2). Composed by layouts/TabbedPageShell.
export { default as PageTitle } from './PageTitle.svelte';

// ProgressRing — compact circular completeness indicator (role=progressbar,
// aria-valuenow = value*100 clamped). A teal arc over a neutral track; sized for a
// rail item. The consumer computes the fraction.
export { default as ProgressRing } from './ProgressRing.svelte';

// Money Types (re-exported from foundation)
export type { MoneyValue, CurrencyConfig } from '../../types/foundation';
export { CURRENCIES, formatMoney, moneyFromFloat } from '../../types/foundation';

// Canonical, options-aware display formatter for stored prices (minor units).
export { formatPrice, type PriceFormatOptions } from './money';

// Google Fonts loader (#3) — re-exported here so consumers (canvas-ui) import it from
// the same specifier as FontPicker, without relying on the bare `./*` package export
// wildcard (which cross-package TS module resolution doesn't reliably honour).
export { ensureGoogleFont, markFontLoaded, isFontRequested } from '../../fonts';

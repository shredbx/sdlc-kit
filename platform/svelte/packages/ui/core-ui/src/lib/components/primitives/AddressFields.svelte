<script lang="ts" module>
	/**
	 * AddressValue — the seven postal fields of a person/postal address.
	 *
	 * Svelte mirror of the postal subset of Go `pkg/address.Address`
	 * (street/unit/sub_district/city/province/postal_code/country). The geo
	 * fields (latitude/longitude/polygon/map views) are DELIBERATELY excluded —
	 * this is a postal editor, never a map/location picker. Consumers that need
	 * a pin use LocationPicker; this primitive never touches coordinates.
	 */
	export interface AddressValue {
		street: string;
		unit: string;
		subDistrict: string;
		city: string;
		province: string;
		postalCode: string;
		country: string;
	}

	/** Per-field label overrides (partial — unset keys keep the defaults). */
	export type AddressLabels = Partial<Record<keyof AddressValue, string>>;

	/** Per-field error strings (partial). Address is optional, so usually empty. */
	export type AddressErrors = Partial<Record<keyof AddressValue, string>>;
</script>

<script lang="ts">
	/**
	 * AddressFields — the one true postal-address editor primitive.
	 *
	 * Composes FieldGrid + Field + Input for the seven postal fields so no
	 * consumer ever hand-rolls the address block again (it was duplicated raw in
	 * three BR forms before this). Layout: street spans the full row, then unit /
	 * postal code / sub-district / city / province / country flow into the grid.
	 *
	 * Theme-neutral: all box/label/error styling comes from the shared `--field-*`
	 * token contract via Field + Input. The primitive sets NO `--br-*` or any
	 * consumer-specific token — a consumer themes it once by mapping `--field-*`.
	 *
	 * Every input carries the correct WHATWG autocomplete token, a `name`, and an
	 * explicit `type` (governance: input-attributes-full). Country stays a TEXT
	 * input (ISO 3166-1 alpha-2, e.g. "TH") for now — a country <select> backed by
	 * the ISO dataset is a separate future task, not built here.
	 *
	 * Error display: each `<Field>` owns its error message (single `role="alert"`
	 * node). The inner `<Input>` is intentionally NOT given `error` — that is the
	 * established Field+control contract (Field owns the chrome, the control owns
	 * its box) and avoids a duplicate alert. This matches every other Field
	 * consumer in the codebase.
	 *
	 * @example
	 *   <AddressFields bind:value={address} labels={{ city: 'District' }} idPrefix="contact" />
	 */

	import FieldGrid from './FieldGrid.svelte';
	import Field from './Field.svelte';
	import Input from './Input.svelte';

	interface Props {
		/** The seven postal fields. Bindable; inner inputs bind member-wise. */
		value?: AddressValue;
		/** Partial label overrides (e.g. BR maps `city` → "District"). */
		labels?: AddressLabels;
		/** id namespace so multiple instances on one page get unique ids. */
		idPrefix?: string;
		/** Disable every input (forwarded to each control). */
		disabled?: boolean;
		/** Optional per-field error strings, forwarded to the matching Field. */
		errors?: AddressErrors;
		/** Number of grid columns at full width (street always spans all). */
		columns?: 1 | 2 | 3 | 4;
		/**
		 * Help text under the Country field. Defaults to the ISO-3166 hint for
		 * formal address surfaces (property listings). Informal/optional surfaces
		 * (e.g. a contact address book) can soften it or pass '' to hide it.
		 */
		countryHelp?: string;
	}

	let {
		value = $bindable({
			street: '',
			unit: '',
			subDistrict: '',
			city: '',
			province: '',
			postalCode: '',
			country: ''
		}),
		labels,
		idPrefix = 'address',
		disabled = false,
		errors,
		columns = 2,
		countryHelp = '2-letter code (ISO 3166)'
	}: Props = $props();

	// Defaults merged with consumer overrides. Derived so a reactive `labels`
	// prop (or none) always resolves to a complete label set.
	const resolvedLabels = $derived<Record<keyof AddressValue, string>>({
		street: 'Street',
		unit: 'Unit · Apt',
		postalCode: 'Postal code',
		subDistrict: 'Sub-district',
		city: 'City',
		province: 'Province',
		country: 'Country',
		...labels
	});
</script>

<FieldGrid {columns}>
	<Field label={resolvedLabels.street} for="{idPrefix}-street" error={errors?.street} full>
		<Input
			id="{idPrefix}-street"
			name="street-address"
			type="text"
			autocomplete="street-address"
			{disabled}
			bind:value={value.street}
		/>
	</Field>

	<Field label={resolvedLabels.unit} for="{idPrefix}-unit" error={errors?.unit}>
		<Input
			id="{idPrefix}-unit"
			name="address-line2"
			type="text"
			autocomplete="address-line2"
			{disabled}
			bind:value={value.unit}
		/>
	</Field>

	<Field label={resolvedLabels.postalCode} for="{idPrefix}-postal-code" error={errors?.postalCode}>
		<Input
			id="{idPrefix}-postal-code"
			name="postal-code"
			type="text"
			inputmode="numeric"
			autocomplete="postal-code"
			{disabled}
			bind:value={value.postalCode}
		/>
	</Field>

	<Field label={resolvedLabels.subDistrict} for="{idPrefix}-sub-district" error={errors?.subDistrict}>
		<Input
			id="{idPrefix}-sub-district"
			name="address-level3"
			type="text"
			autocomplete="address-level3"
			{disabled}
			bind:value={value.subDistrict}
		/>
	</Field>

	<Field label={resolvedLabels.city} for="{idPrefix}-city" error={errors?.city}>
		<Input
			id="{idPrefix}-city"
			name="address-level2"
			type="text"
			autocomplete="address-level2"
			{disabled}
			bind:value={value.city}
		/>
	</Field>

	<Field label={resolvedLabels.province} for="{idPrefix}-province" error={errors?.province}>
		<Input
			id="{idPrefix}-province"
			name="address-level1"
			type="text"
			autocomplete="address-level1"
			{disabled}
			bind:value={value.province}
		/>
	</Field>

	<Field
		label={resolvedLabels.country}
		for="{idPrefix}-country"
		help={countryHelp}
		error={errors?.country}
	>
		<Input
			id="{idPrefix}-country"
			name="country"
			type="text"
			autocomplete="country"
			placeholder="TH"
			{disabled}
			bind:value={value.country}
		/>
	</Field>
</FieldGrid>

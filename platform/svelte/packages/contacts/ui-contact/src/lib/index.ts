// @sbx/ui-contact — reusable contact picker (modal). Modeled on @sbx/ui-source-picker's
// provider pattern but DECOUPLED from canvas/media: depends only on @sbx/core-ui. Consumers
// (BR transactions/inquiries, calendar EventEditor) supply a ContactProvider + an inline
// create-form snippet. See docs/plans/2026-06-07-br-contact-picker-package-design.md.

export { default as ContactPickerModal } from './ContactPickerModal.svelte';

export type {
	ContactPickerRecord,
	ContactCategory,
	ContactPage,
	ContactProvider,
	ContactPickerMode,
	ContactPickerProps
} from './types';

export { categoryColor } from './palette';
export { mockContactProvider } from './adapters/mock';

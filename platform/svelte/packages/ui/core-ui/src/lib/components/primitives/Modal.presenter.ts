/**
 * Modal Presenter
 *
 * Defines presentation metadata for Modal component.
 * Enables auto-registration in /components page.
 */

import Modal from './Modal.svelte';
import type { IPresenter } from '../../presenter/IPresenter';

const presenter: IPresenter = {
	name: 'Modal',
	slug: 'modal',
	category: 'primitives',
	presenterType: 'component',
	component: Modal,
	description:
		'Overlay modal dialog with backdrop, keyboard support, and animations. Supports different sizes and customizable header/footer.',
	status: 'stable',

	variants: [
		{
			name: 'Small',
			description: 'Compact modal for confirmations',
			props: { open: true, size: 'sm', title: 'Confirm Action' }
		},
		{
			name: 'Medium (Default)',
			description: 'Standard modal size',
			props: { open: true, size: 'md', title: 'Modal Title' }
		},
		{
			name: 'Large',
			description: 'For complex content',
			props: { open: true, size: 'lg', title: 'Large Modal' }
		},
		{
			name: 'Extra Large',
			description: 'Near-fullscreen modal',
			props: { open: true, size: 'xl', title: 'Extra Large Modal' }
		},
		{
			name: 'Full Screen',
			description: 'Takes entire viewport',
			props: { open: true, size: 'full', title: 'Full Screen Modal' }
		},
		{
			name: 'No Close Button',
			description: 'Modal without X button',
			props: { open: true, title: 'Required Action', showClose: false }
		}
	],

	presets: [
		{
			name: 'Confirmation Dialog',
			description: 'Simple yes/no confirmation',
			props: { size: 'sm', title: 'Confirm', closeOnBackdrop: false }
		},
		{
			name: 'Form Modal',
			description: 'Modal for form content',
			props: { size: 'md', title: 'Edit Details', closeOnEscape: true }
		},
		{
			name: 'Content Viewer',
			description: 'Large content display',
			props: { size: 'lg', title: 'Preview', showClose: true }
		}
	],

	props: [
		{
			name: 'open',
			type: 'boolean',
			description: 'Whether modal is visible',
			default: false
		},
		{
			name: 'size',
			type: 'select',
			description: 'Modal size preset',
			options: ['sm', 'md', 'lg', 'xl', 'full'],
			default: 'md'
		},
		{
			name: 'title',
			type: 'string',
			description: 'Modal title text',
			default: ''
		},
		{
			name: 'closeOnBackdrop',
			type: 'boolean',
			description: 'Close when clicking backdrop',
			default: true
		},
		{
			name: 'closeOnEscape',
			type: 'boolean',
			description: 'Close when pressing Escape',
			default: true
		},
		{
			name: 'showClose',
			type: 'boolean',
			description: 'Show close button',
			default: true
		}
	],

	// Preview with closed modal (prevents portal rendering issues)
	preview: {
		defaultProps: {
			open: false,
			title: 'Modal Preview'
		}
	},

	source: {
		path: 'components/primitives/Modal.svelte'
	}
};

export default presenter;

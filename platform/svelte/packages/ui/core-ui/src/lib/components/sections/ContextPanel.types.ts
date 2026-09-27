/**
 * ContextPanel types — extracted for cross-file import compatibility.
 * Svelte 5 instance-script exports can't be re-exported through barrel files.
 */

export interface TreeItem {
	id: string;
	label: string;
	icon?: string;
	/** Navigation href - when set, renders as <a> link for native navigation */
	href?: string;
	children?: TreeItem[];
	expanded?: boolean;
	selected?: boolean;
	disabled?: boolean;
	badge?: number;
	/** Custom data attached to item */
	data?: unknown;
}

export interface ContextSection {
	/** Unique identifier matching nav rail item */
	id: string;
	/** Section display type */
	type: 'tree' | 'list' | 'grid';
	/** Header text */
	header?: string;
	/** Enable search/filter */
	searchable?: boolean;
	/** Section items */
	items: TreeItem[];
	/** Header actions */
	actions?: Array<{
		id: string;
		icon: string;
		tooltip: string;
	}>;
}

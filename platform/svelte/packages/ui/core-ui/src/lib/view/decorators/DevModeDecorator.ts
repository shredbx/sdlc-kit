/**
 * DevModeDecorator - Developer tools for component inspection
 *
 * Provides visual overlays and interaction handlers for inspecting,
 * selecting, and debugging components at runtime.
 *
 * Features:
 * - Highlight on hover (configurable color)
 * - Click to select with outline
 * - Props panel integration
 * - Component tree navigation
 * - Bounds visualization
 *
 * @example
 * const view: IView = {
 *   id: 'button-1',
 *   name: 'Submit Button',
 *   component: 'Button',
 *   category: 'primitives',
 *   decorators: [createDevModeDecorator()]
 * };
 */

import type { IDecorator, IDecoratorContext, IDecoratorHooks, IView } from '../IView';

// =============================================================================
// CONFIGURATION
// =============================================================================

/**
 * DevMode decorator configuration.
 */
export interface DevModeConfig {
	/** Highlight color on hover. Default: '#6366f1' (indigo) */
	highlightColor?: string;

	/** Outline color when selected. Default: '#22c55e' (green) */
	selectColor?: string;

	/** Outline width in pixels. Default: 2 */
	outlineWidth?: number;

	/** Show component name tooltip. Default: true */
	showTooltip?: boolean;

	/** Show bounds overlay (padding, margin). Default: false */
	showBounds?: boolean;

	/** Enable keyboard shortcuts. Default: true */
	enableShortcuts?: boolean;

	/** Z-index for overlays. Default: 9999 */
	zIndex?: number;
}

const DEFAULT_CONFIG: DevModeConfig = {
	highlightColor: '#6366f1',
	selectColor: '#22c55e',
	outlineWidth: 2,
	showTooltip: true,
	showBounds: false,
	enableShortcuts: true,
	zIndex: 9999
};

// =============================================================================
// STATE
// =============================================================================

/** Currently selected view ID */
let selectedViewId: string | null = null;

/** Currently hovered element */
let hoveredElement: HTMLElement | null = null;

/** Tooltip element */
let tooltipElement: HTMLElement | null = null;

/** Overlay elements */
const overlays = new Map<string, HTMLElement>();

// =============================================================================
// STYLES
// =============================================================================

/**
 * Inject CSS styles for dev mode overlays.
 */
function injectStyles(config: DevModeConfig): void {
	const styleId = 'dev-mode-decorator-styles';
	if (document.getElementById(styleId)) return;

	const style = document.createElement('style');
	style.id = styleId;
	style.textContent = `
		.dev-mode-highlight {
			outline: ${config.outlineWidth}px dashed ${config.highlightColor} !important;
			outline-offset: -${config.outlineWidth}px;
		}

		.dev-mode-selected {
			outline: ${config.outlineWidth}px solid ${config.selectColor} !important;
			outline-offset: -${config.outlineWidth}px;
		}

		.dev-mode-tooltip {
			position: fixed;
			background: ${config.highlightColor};
			color: white;
			font-family: system-ui, -apple-system, sans-serif;
			font-size: 11px;
			font-weight: 500;
			padding: 4px 8px;
			border-radius: 4px;
			z-index: ${config.zIndex};
			pointer-events: none;
			white-space: nowrap;
			box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
		}

		.dev-mode-tooltip::after {
			content: '';
			position: absolute;
			bottom: -4px;
			left: 50%;
			transform: translateX(-50%);
			border-left: 4px solid transparent;
			border-right: 4px solid transparent;
			border-top: 4px solid ${config.highlightColor};
		}

		.dev-mode-bounds {
			position: absolute;
			pointer-events: none;
			z-index: ${(config.zIndex ?? 9999) - 1};
		}

		.dev-mode-bounds-margin {
			background: rgba(255, 166, 0, 0.2);
		}

		.dev-mode-bounds-padding {
			background: rgba(0, 255, 0, 0.2);
		}

		.dev-mode-bounds-content {
			background: rgba(100, 149, 237, 0.2);
		}

		[data-view-id] {
			cursor: pointer;
		}
	`;
	document.head.appendChild(style);
}

// =============================================================================
// TOOLTIP
// =============================================================================

/**
 * Show tooltip with component info.
 */
function showTooltip(view: IView, element: HTMLElement, config: DevModeConfig): void {
	if (!config.showTooltip) return;

	if (!tooltipElement) {
		tooltipElement = document.createElement('div');
		tooltipElement.className = 'dev-mode-tooltip';
		document.body.appendChild(tooltipElement);
	}

	const rect = element.getBoundingClientRect();
	tooltipElement.textContent = `${view.name} (${view.category})`;
	tooltipElement.style.left = `${rect.left + rect.width / 2 - tooltipElement.offsetWidth / 2}px`;
	tooltipElement.style.top = `${rect.top - tooltipElement.offsetHeight - 8}px`;
	tooltipElement.style.display = 'block';
}

/**
 * Hide tooltip.
 */
function hideTooltip(): void {
	if (tooltipElement) {
		tooltipElement.style.display = 'none';
	}
}

// =============================================================================
// BOUNDS OVERLAY
// =============================================================================

/**
 * Show bounds overlay (margin, padding, content).
 */
function showBounds(view: IView, element: HTMLElement, config: DevModeConfig): void {
	if (!config.showBounds) return;

	const rect = element.getBoundingClientRect();
	const computed = getComputedStyle(element);

	// Parse margins
	const marginTop = parseFloat(computed.marginTop);
	const marginRight = parseFloat(computed.marginRight);
	const marginBottom = parseFloat(computed.marginBottom);
	const marginLeft = parseFloat(computed.marginLeft);

	// Parse padding
	const paddingTop = parseFloat(computed.paddingTop);
	const paddingRight = parseFloat(computed.paddingRight);
	const paddingBottom = parseFloat(computed.paddingBottom);
	const paddingLeft = parseFloat(computed.paddingLeft);

	// Create margin overlay
	const marginOverlay = document.createElement('div');
	marginOverlay.className = 'dev-mode-bounds dev-mode-bounds-margin';
	marginOverlay.style.left = `${rect.left - marginLeft}px`;
	marginOverlay.style.top = `${rect.top - marginTop}px`;
	marginOverlay.style.width = `${rect.width + marginLeft + marginRight}px`;
	marginOverlay.style.height = `${rect.height + marginTop + marginBottom}px`;
	document.body.appendChild(marginOverlay);
	overlays.set(`${view.id}-margin`, marginOverlay);

	// Create padding overlay
	const paddingOverlay = document.createElement('div');
	paddingOverlay.className = 'dev-mode-bounds dev-mode-bounds-padding';
	paddingOverlay.style.left = `${rect.left + paddingLeft}px`;
	paddingOverlay.style.top = `${rect.top + paddingTop}px`;
	paddingOverlay.style.width = `${rect.width - paddingLeft - paddingRight}px`;
	paddingOverlay.style.height = `${rect.height - paddingTop - paddingBottom}px`;
	document.body.appendChild(paddingOverlay);
	overlays.set(`${view.id}-padding`, paddingOverlay);
}

/**
 * Hide bounds overlay.
 */
function hideBounds(viewId: string): void {
	const marginOverlay = overlays.get(`${viewId}-margin`);
	const paddingOverlay = overlays.get(`${viewId}-padding`);

	if (marginOverlay) {
		marginOverlay.remove();
		overlays.delete(`${viewId}-margin`);
	}

	if (paddingOverlay) {
		paddingOverlay.remove();
		overlays.delete(`${viewId}-padding`);
	}
}

// =============================================================================
// SELECTION
// =============================================================================

/**
 * Select a view component.
 */
function selectView(view: IView, element: HTMLElement, context: IDecoratorContext): void {
	// Deselect previous
	if (selectedViewId) {
		const prevElement = document.querySelector(`[data-view-id="${selectedViewId}"]`);
		if (prevElement) {
			prevElement.classList.remove('dev-mode-selected');
		}
	}

	// Select new
	selectedViewId = view.id;
	element.classList.add('dev-mode-selected');

	// Emit selection event
	context.emit('view:selected', { viewId: view.id, view });
}

/**
 * Deselect current view.
 */
function deselectView(): void {
	if (selectedViewId) {
		const element = document.querySelector(`[data-view-id="${selectedViewId}"]`);
		if (element) {
			element.classList.remove('dev-mode-selected');
		}
		selectedViewId = null;
	}
}

// =============================================================================
// HOOKS
// =============================================================================

/**
 * Create DevMode decorator hooks.
 */
function createHooks(config: DevModeConfig): IDecoratorHooks {
	return {
		onAfterMount(view: IView, context: IDecoratorContext, element: HTMLElement): void {
			// Only enable in dev mode
			if (!context.devMode) return;

			// Inject styles
			injectStyles(config);

			// Set view ID on element
			element.dataset.viewId = view.id;

			// Mouse enter - highlight
			element.addEventListener('mouseenter', () => {
				if (!context.devMode) return;
				hoveredElement = element;
				element.classList.add('dev-mode-highlight');
				showTooltip(view, element, config);
				showBounds(view, element, config);
			});

			// Mouse leave - remove highlight
			element.addEventListener('mouseleave', () => {
				hoveredElement = null;
				element.classList.remove('dev-mode-highlight');
				hideTooltip();
				hideBounds(view.id);
			});

			// Click - select
			element.addEventListener('click', (e) => {
				if (!context.devMode) return;

				// Alt+Click to select without triggering action
				if (e.altKey) {
					e.preventDefault();
					e.stopPropagation();
					selectView(view, element, context);
				}
			});

			// Keyboard shortcuts
			if (config.enableShortcuts) {
				const handleKeydown = (e: KeyboardEvent) => {
					if (!context.devMode) return;

					// Escape - deselect
					if (e.key === 'Escape' && selectedViewId === view.id) {
						deselectView();
					}

					// i - inspect (when selected)
					if (e.key === 'i' && selectedViewId === view.id) {
						context.emit('view:inspect', { viewId: view.id, view });
					}
				};

				document.addEventListener('keydown', handleKeydown);
			}
		},

		onBeforeDestroy(view: IView): void {
			// Clean up overlays
			hideBounds(view.id);

			// Deselect if this view was selected
			if (selectedViewId === view.id) {
				selectedViewId = null;
			}
		},

		onEvent(view: IView, context: IDecoratorContext, event: Event): boolean | void {
			// Block click events when Alt is held (selection mode)
			if (event.type === 'click' && (event as MouseEvent).altKey && context.devMode) {
				return false;
			}
		}
	};
}

// =============================================================================
// FACTORY
// =============================================================================

/**
 * Create DevMode decorator with configuration.
 *
 * @example
 * const decorator = createDevModeDecorator({
 *   highlightColor: '#ff6b6b',
 *   showBounds: true
 * });
 */
export function createDevModeDecorator(config?: DevModeConfig): IDecorator<DevModeConfig> {
	const mergedConfig = { ...DEFAULT_CONFIG, ...config };

	return {
		id: 'dev-mode',
		name: 'DevMode',
		priority: 100, // Run early
		enabled: true,
		config: mergedConfig,
		hooks: createHooks(mergedConfig)
	};
}

// =============================================================================
// UTILITIES
// =============================================================================

/**
 * Get currently selected view ID.
 */
export function getSelectedViewId(): string | null {
	return selectedViewId;
}

/**
 * Check if a view is selected.
 */
export function isViewSelected(viewId: string): boolean {
	return selectedViewId === viewId;
}

/**
 * Programmatically select a view by ID.
 */
export function selectViewById(viewId: string, context: IDecoratorContext): void {
	const element = document.querySelector(`[data-view-id="${viewId}"]`) as HTMLElement;
	if (element) {
		const view = context.decorators.get('view-registry')?.config as unknown as IView;
		if (view) {
			selectView(view, element, context);
		}
	}
}

/**
 * Clear selection.
 */
export function clearSelection(): void {
	deselectView();
}

// =============================================================================
// DEFAULT EXPORT
// =============================================================================

export default createDevModeDecorator;

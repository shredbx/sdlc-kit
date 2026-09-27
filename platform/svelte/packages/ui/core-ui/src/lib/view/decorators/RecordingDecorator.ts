/**
 * RecordingDecorator - Record and persist component changes
 *
 * Tracks prop changes, interactions, and state mutations for:
 * - Saving modified component configurations
 * - Generating test fixtures from interactions
 * - Creating presets from real usage
 * - Undo/redo functionality
 *
 * Features:
 * - Change history with timestamps
 * - Prop diff tracking
 * - Session recording
 * - Export to YAML/JSON
 * - API integration for persistence
 *
 * @example
 * const view: IView = {
 *   id: 'form-1',
 *   name: 'Contact Form',
 *   component: 'Form',
 *   category: 'sections',
 *   decorators: [createRecordingDecorator({ autoSave: true })]
 * };
 */

import type { IDecorator, IDecoratorContext, IDecoratorHooks, IView } from '../IView';

// =============================================================================
// CONFIGURATION
// =============================================================================

/**
 * Recording decorator configuration.
 */
export interface RecordingConfig {
	/** Auto-save changes after delay. Default: false */
	autoSave?: boolean;

	/** Delay before auto-save in ms. Default: 2000 */
	autoSaveDelay?: number;

	/** Maximum history entries. Default: 50 */
	maxHistory?: number;

	/** Record user interactions (clicks, inputs). Default: false */
	recordInteractions?: boolean;

	/** Record state changes. Default: true */
	recordProps?: boolean;

	/** API endpoint for saving changes. Default: '/api/fixtures' */
	saveEndpoint?: string;

	/** Session identifier for grouping changes. Default: auto-generated */
	sessionId?: string;

	/** Callback when change is recorded */
	onRecordChange?: (change: ChangeRecord) => void;

	/** Callback when changes are saved */
	onSave?: (changes: ChangeRecord[]) => void;
}

const DEFAULT_CONFIG: RecordingConfig = {
	autoSave: false,
	autoSaveDelay: 2000,
	maxHistory: 50,
	recordInteractions: false,
	recordProps: true,
	saveEndpoint: '/api/fixtures'
};

// =============================================================================
// TYPES
// =============================================================================

/**
 * Types of recordable changes.
 */
export type ChangeType = 'prop' | 'state' | 'interaction' | 'mount' | 'unmount';

/**
 * Single change record.
 */
export interface ChangeRecord {
	/** Unique change ID */
	id: string;

	/** View that changed */
	viewId: string;

	/** Type of change */
	type: ChangeType;

	/** Property path that changed (for prop/state changes) */
	path?: string;

	/** Previous value */
	oldValue?: unknown;

	/** New value */
	newValue?: unknown;

	/** Interaction details (for interaction changes) */
	interaction?: {
		type: string;
		target?: string;
		data?: unknown;
	};

	/** Timestamp */
	timestamp: number;

	/** Session ID for grouping */
	sessionId: string;
}

/**
 * Recording session state.
 */
export interface RecordingSession {
	/** Session ID */
	id: string;

	/** Session start time */
	startTime: number;

	/** Changes in this session */
	changes: ChangeRecord[];

	/** Is session active */
	active: boolean;

	/** View IDs being recorded */
	views: Set<string>;
}

// =============================================================================
// STATE
// =============================================================================

/** Active sessions by ID */
const sessions = new Map<string, RecordingSession>();

/** Current session ID */
let currentSessionId: string | null = null;

/** Auto-save timers */
const saveTimers = new Map<string, NodeJS.Timeout>();

/** Change counter for unique IDs */
let changeCounter = 0;

// =============================================================================
// SESSION MANAGEMENT
// =============================================================================

/**
 * Generate a unique session ID.
 */
function generateSessionId(): string {
	return `session-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Generate a unique change ID.
 */
function generateChangeId(): string {
	return `change-${++changeCounter}`;
}

/**
 * Start a new recording session.
 */
export function startSession(sessionId?: string): RecordingSession {
	const id = sessionId ?? generateSessionId();

	const session: RecordingSession = {
		id,
		startTime: Date.now(),
		changes: [],
		active: true,
		views: new Set()
	};

	sessions.set(id, session);
	currentSessionId = id;

	return session;
}

/**
 * End a recording session.
 */
export function endSession(sessionId?: string): RecordingSession | undefined {
	const id = sessionId ?? currentSessionId;
	if (!id) return undefined;

	const session = sessions.get(id);
	if (session) {
		session.active = false;
	}

	if (currentSessionId === id) {
		currentSessionId = null;
	}

	return session;
}

/**
 * Get current session.
 */
export function getCurrentSession(): RecordingSession | undefined {
	return currentSessionId ? sessions.get(currentSessionId) : undefined;
}

/**
 * Get session by ID.
 */
export function getSession(sessionId: string): RecordingSession | undefined {
	return sessions.get(sessionId);
}

// =============================================================================
// RECORDING
// =============================================================================

/**
 * Record a change.
 */
function recordChange(
	viewId: string,
	type: ChangeType,
	config: RecordingConfig,
	details: Partial<ChangeRecord>
): ChangeRecord {
	const sessionId = config.sessionId ?? currentSessionId ?? generateSessionId();

	// Get or create session
	let session = sessions.get(sessionId);
	if (!session) {
		session = startSession(sessionId);
	}

	const change: ChangeRecord = {
		id: generateChangeId(),
		viewId,
		type,
		timestamp: Date.now(),
		sessionId,
		...details
	};

	// Add to session
	session.changes.push(change);
	session.views.add(viewId);

	// Trim history if needed
	if (config.maxHistory && session.changes.length > config.maxHistory) {
		session.changes = session.changes.slice(-config.maxHistory);
	}

	// Callback
	config.onRecordChange?.(change);

	// Schedule auto-save
	if (config.autoSave) {
		scheduleAutoSave(sessionId, config);
	}

	return change;
}

/**
 * Record a prop change.
 */
export function recordPropChange(
	viewId: string,
	path: string,
	oldValue: unknown,
	newValue: unknown,
	config: RecordingConfig
): ChangeRecord {
	return recordChange(viewId, 'prop', config, { path, oldValue, newValue });
}

/**
 * Record an interaction.
 */
export function recordInteraction(
	viewId: string,
	interactionType: string,
	target?: string,
	data?: unknown,
	config?: RecordingConfig
): ChangeRecord {
	return recordChange(viewId, 'interaction', config ?? DEFAULT_CONFIG, {
		interaction: { type: interactionType, target, data }
	});
}

// =============================================================================
// AUTO-SAVE
// =============================================================================

/**
 * Schedule auto-save for a session.
 */
function scheduleAutoSave(sessionId: string, config: RecordingConfig): void {
	// Clear existing timer
	const existingTimer = saveTimers.get(sessionId);
	if (existingTimer) {
		clearTimeout(existingTimer);
	}

	// Schedule new save
	const timer = setTimeout(async () => {
		await saveSession(sessionId, config);
		saveTimers.delete(sessionId);
	}, config.autoSaveDelay ?? 2000);

	saveTimers.set(sessionId, timer);
}

/**
 * Save session changes to server.
 */
export async function saveSession(
	sessionId: string,
	config: RecordingConfig
): Promise<boolean> {
	const session = sessions.get(sessionId);
	if (!session || session.changes.length === 0) return false;

	try {
		const response = await fetch(config.saveEndpoint ?? '/api/fixtures', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				sessionId,
				changes: session.changes,
				views: Array.from(session.views)
			})
		});

		if (response.ok) {
			config.onSave?.(session.changes);
			return true;
		}

		console.error('Failed to save session:', response.statusText);
		return false;
	} catch (error) {
		console.error('Failed to save session:', error);
		return false;
	}
}

// =============================================================================
// EXPORT
// =============================================================================

/**
 * Export session changes to JSON.
 */
export function exportToJSON(sessionId?: string): string {
	const session = sessionId ? sessions.get(sessionId) : getCurrentSession();
	if (!session) return '[]';

	return JSON.stringify(session.changes, null, 2);
}

/**
 * Export session changes to YAML format.
 */
export function exportToYAML(sessionId?: string): string {
	const session = sessionId ? sessions.get(sessionId) : getCurrentSession();
	if (!session) return '';

	const lines: string[] = ['# Recording Session Export', `session_id: ${session.id}`, 'changes:'];

	for (const change of session.changes) {
		lines.push(`  - id: ${change.id}`);
		lines.push(`    view_id: ${change.viewId}`);
		lines.push(`    type: ${change.type}`);
		if (change.path) lines.push(`    path: ${change.path}`);
		if (change.oldValue !== undefined) lines.push(`    old_value: ${JSON.stringify(change.oldValue)}`);
		if (change.newValue !== undefined) lines.push(`    new_value: ${JSON.stringify(change.newValue)}`);
		if (change.interaction) {
			lines.push(`    interaction:`);
			lines.push(`      type: ${change.interaction.type}`);
			if (change.interaction.target) lines.push(`      target: ${change.interaction.target}`);
			if (change.interaction.data) lines.push(`      data: ${JSON.stringify(change.interaction.data)}`);
		}
		lines.push(`    timestamp: ${change.timestamp}`);
		lines.push('');
	}

	return lines.join('\n');
}

// =============================================================================
// UNDO/REDO
// =============================================================================

/** Redo stack per session */
const redoStacks = new Map<string, ChangeRecord[]>();

/**
 * Undo the last change in a session.
 */
export function undo(sessionId?: string): ChangeRecord | undefined {
	const id = sessionId ?? currentSessionId;
	if (!id) return undefined;

	const session = sessions.get(id);
	if (!session || session.changes.length === 0) return undefined;

	const change = session.changes.pop();
	if (change) {
		// Add to redo stack
		let redoStack = redoStacks.get(id);
		if (!redoStack) {
			redoStack = [];
			redoStacks.set(id, redoStack);
		}
		redoStack.push(change);
	}

	return change;
}

/**
 * Redo the last undone change.
 */
export function redo(sessionId?: string): ChangeRecord | undefined {
	const id = sessionId ?? currentSessionId;
	if (!id) return undefined;

	const redoStack = redoStacks.get(id);
	if (!redoStack || redoStack.length === 0) return undefined;

	const change = redoStack.pop();
	if (change) {
		const session = sessions.get(id);
		session?.changes.push(change);
	}

	return change;
}

/**
 * Clear history for a session.
 */
export function clearHistory(sessionId?: string): void {
	const id = sessionId ?? currentSessionId;
	if (!id) return;

	const session = sessions.get(id);
	if (session) {
		session.changes = [];
	}

	redoStacks.delete(id);
}

// =============================================================================
// HOOKS
// =============================================================================

/**
 * Create Recording decorator hooks.
 */
function createHooks(config: RecordingConfig): IDecoratorHooks {
	return {
		onAfterMount(view: IView, context: IDecoratorContext, element: HTMLElement): void {
			// Record mount
			recordChange(view.id, 'mount', config, {
				newValue: view.props
			});

			// Set up interaction recording
			if (config.recordInteractions) {
				// Click tracking
				element.addEventListener('click', (e) => {
					const target = e.target as HTMLElement;
					recordInteraction(
						view.id,
						'click',
						target.tagName.toLowerCase(),
						{
							classList: Array.from(target.classList),
							id: target.id || undefined
						},
						config
					);
				});

				// Input tracking
				element.addEventListener('input', (e) => {
					const target = e.target as HTMLInputElement;
					if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
						recordInteraction(
							view.id,
							'input',
							target.name || target.id || 'unknown',
							{ value: target.value },
							config
						);
					}
				});
			}

			// Set up prop change recording via context
			if (config.recordProps && context.recordChange) {
				const originalRecordChange = context.recordChange;
				context.recordChange = (path: string, value: unknown) => {
					// Record in our history
					recordPropChange(view.id, path, undefined, value, config);
					// Call original
					originalRecordChange(path, value);
				};
			}
		},

		onBeforeUpdate(view: IView, context: IDecoratorContext, nextProps: unknown): void {
			if (!config.recordProps) return;

			// Record prop changes
			const currentProps = view.props ?? {};
			const next = nextProps as Record<string, unknown>;

			for (const key of Object.keys(next)) {
				const oldValue = (currentProps as Record<string, unknown>)[key];
				const newValue = next[key];

				if (oldValue !== newValue) {
					recordPropChange(view.id, key, oldValue, newValue, config);
				}
			}
		},

		onBeforeDestroy(view: IView): void {
			// Record unmount
			recordChange(view.id, 'unmount', config, {
				oldValue: view.props
			});
		}
	};
}

// =============================================================================
// FACTORY
// =============================================================================

/**
 * Create Recording decorator with configuration.
 *
 * @example
 * // Basic recording
 * const decorator = createRecordingDecorator();
 *
 * @example
 * // Auto-save with custom endpoint
 * const decorator = createRecordingDecorator({
 *   autoSave: true,
 *   saveEndpoint: '/api/components/save'
 * });
 */
export function createRecordingDecorator(config?: RecordingConfig): IDecorator<RecordingConfig> {
	const mergedConfig = { ...DEFAULT_CONFIG, ...config };

	// Start session if specified
	if (mergedConfig.sessionId) {
		startSession(mergedConfig.sessionId);
	}

	return {
		id: 'recording',
		name: 'Recording',
		priority: 300, // Run after DevMode and Branding
		enabled: true,
		config: mergedConfig,
		hooks: createHooks(mergedConfig)
	};
}

// =============================================================================
// DEFAULT EXPORT
// =============================================================================

export default createRecordingDecorator;

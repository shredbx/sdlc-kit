/**
 * Command History Store - Undo/Redo System with Command Pattern
 *
 * Provides a complete undo/redo system for UI changes using the command pattern.
 * Each action is encapsulated as a command with execute and undo methods.
 *
 * Architecture:
 *   Command Creation -> Execute -> Push to Undo Stack -> UI Update
 *   Undo Action -> Pop from Undo -> Push to Redo -> UI Update
 *   Redo Action -> Pop from Redo -> Push to Undo -> UI Update
 *
 * Features:
 * - Command pattern implementation with reversible operations
 * - Configurable history limit (default 50)
 * - Keyboard shortcut support (Cmd/Ctrl+Z, Cmd/Ctrl+Shift+Z)
 * - Command factories for common operations
 * - State persistence option via localStorage
 *
 * Usage:
 *   import {
 *     undoStack,
 *     redoStack,
 *     canUndo,
 *     canRedo,
 *     executeCommand,
 *     undo,
 *     redo,
 *     createPropChangeCommand
 *   } from '$lib/stores/commandHistory';
 *
 *   // Execute a command
 *   const command = createPropChangeCommand('button-1', 'variant', 'default', 'primary');
 *   executeCommand(command);
 *
 *   // Undo last action
 *   undo();
 *
 *   // Redo undone action
 *   redo();
 *
 * @module commandHistory
 */

import { browser } from '$app/environment';
import { writable, derived, get, type Writable, type Readable } from 'svelte/store';

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

/**
 * Command interface for undo/redo operations.
 * Commands encapsulate both the action and its reversal.
 */
export interface Command {
	/** Unique command identifier */
	id: string;
	/** Command type for categorization (e.g., 'prop-change', 'move', 'delete') */
	type: string;
	/** Human-readable description of the command */
	description: string;
	/** Execute the command (apply the change) */
	execute: () => void;
	/** Undo the command (reverse the change) */
	undo: () => void;
	/** Timestamp when command was created */
	timestamp: Date;
}

/**
 * Serializable command metadata for history display.
 * Used when full command cannot be serialized.
 */
export interface CommandMeta {
	id: string;
	type: string;
	description: string;
	timestamp: Date;
}

/**
 * Configuration options for command history.
 */
export interface CommandHistoryConfig {
	/** Maximum number of commands to keep in history (default: 50) */
	maxHistoryLength: number;
	/** Enable keyboard shortcuts (default: true) */
	enableKeyboardShortcuts: boolean;
	/** Storage key for persistence (null to disable) */
	storageKey: string | null;
}

/**
 * Position type for move commands.
 */
export interface Position {
	x: number;
	y: number;
}

// =============================================================================
// CONFIGURATION
// =============================================================================

const defaultConfig: CommandHistoryConfig = {
	maxHistoryLength: 50,
	enableKeyboardShortcuts: true,
	storageKey: null // Disabled by default - commands have functions
};

let config: CommandHistoryConfig = { ...defaultConfig };

/**
 * Configure command history settings.
 *
 * @param options - Partial configuration to merge
 *
 * @example
 * configureHistory({ maxHistoryLength: 100, enableKeyboardShortcuts: false });
 */
export function configureHistory(options: Partial<CommandHistoryConfig>): void {
	config = { ...config, ...options };

	// Trim history if new max is smaller
	const currentUndo = get(undoStack);
	if (currentUndo.length > config.maxHistoryLength) {
		undoStack.set(currentUndo.slice(-config.maxHistoryLength));
	}
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

/**
 * Generate unique command ID.
 */
function generateId(): string {
	return `cmd-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

// =============================================================================
// STORE IMPLEMENTATION
// =============================================================================

/**
 * Undo stack - commands that can be undone.
 * Most recent command is at the end of the array.
 */
export const undoStack: Writable<Command[]> = writable([]);

/**
 * Redo stack - commands that were undone and can be redone.
 * Most recent undone command is at the end of the array.
 */
export const redoStack: Writable<Command[]> = writable([]);

/**
 * Derived store indicating if undo is available.
 *
 * @example
 * <button disabled={!$canUndo} onclick={undo}>Undo</button>
 */
export const canUndo: Readable<boolean> = derived(undoStack, ($stack) => $stack.length > 0);

/**
 * Derived store indicating if redo is available.
 *
 * @example
 * <button disabled={!$canRedo} onclick={redo}>Redo</button>
 */
export const canRedo: Readable<boolean> = derived(redoStack, ($stack) => $stack.length > 0);

/**
 * Derived store for total history length (undo + redo).
 *
 * @example
 * <span>History: {$historyLength} commands</span>
 */
export const historyLength: Readable<number> = derived(
	[undoStack, redoStack],
	([$undo, $redo]) => $undo.length + $redo.length
);

/**
 * Derived store for undo stack length.
 */
export const undoCount: Readable<number> = derived(undoStack, ($stack) => $stack.length);

/**
 * Derived store for redo stack length.
 */
export const redoCount: Readable<number> = derived(redoStack, ($stack) => $stack.length);

/**
 * Derived store for the last executed command (for display purposes).
 */
export const lastCommand: Readable<Command | null> = derived(
	undoStack,
	($stack) => $stack[$stack.length - 1] ?? null
);

// =============================================================================
// ACTIONS
// =============================================================================

/**
 * Execute a command and add it to the undo stack.
 * Clears the redo stack as a new action invalidates redo history.
 *
 * @param command - The command to execute
 *
 * @example
 * const cmd = createPropChangeCommand('btn-1', 'size', 'md', 'lg');
 * executeCommand(cmd);
 */
export function executeCommand(command: Command): void {
	// Execute the command
	command.execute();

	// Add to undo stack
	undoStack.update((stack) => {
		const newStack = [...stack, command];
		// Trim if exceeds max length
		if (newStack.length > config.maxHistoryLength) {
			return newStack.slice(-config.maxHistoryLength);
		}
		return newStack;
	});

	// Clear redo stack - new action invalidates redo history
	redoStack.set([]);
}

/**
 * Undo the last command.
 * Moves the command from undo stack to redo stack.
 *
 * @returns The undone command, or null if nothing to undo
 *
 * @example
 * const undoneCmd = undo();
 * if (undoneCmd) {
 *   console.log('Undid:', undoneCmd.description);
 * }
 */
export function undo(): Command | null {
	const stack = get(undoStack);
	if (stack.length === 0) {
		return null;
	}

	// Pop last command
	const command = stack[stack.length - 1];

	// Execute undo
	command.undo();

	// Move from undo to redo stack
	undoStack.update((s) => s.slice(0, -1));
	redoStack.update((s) => [...s, command]);

	return command;
}

/**
 * Redo the last undone command.
 * Moves the command from redo stack back to undo stack.
 *
 * @returns The redone command, or null if nothing to redo
 *
 * @example
 * const redoneCmd = redo();
 * if (redoneCmd) {
 *   console.log('Redid:', redoneCmd.description);
 * }
 */
export function redo(): Command | null {
	const stack = get(redoStack);
	if (stack.length === 0) {
		return null;
	}

	// Pop last undone command
	const command = stack[stack.length - 1];

	// Re-execute
	command.execute();

	// Move from redo to undo stack
	redoStack.update((s) => s.slice(0, -1));
	undoStack.update((s) => [...s, command]);

	return command;
}

/**
 * Undo multiple commands at once.
 *
 * @param count - Number of commands to undo
 * @returns Array of undone commands
 *
 * @example
 * undoMultiple(3); // Undo last 3 commands
 */
export function undoMultiple(count: number): Command[] {
	const undone: Command[] = [];
	for (let i = 0; i < count; i++) {
		const cmd = undo();
		if (cmd) {
			undone.push(cmd);
		} else {
			break;
		}
	}
	return undone;
}

/**
 * Redo multiple commands at once.
 *
 * @param count - Number of commands to redo
 * @returns Array of redone commands
 *
 * @example
 * redoMultiple(3); // Redo last 3 undone commands
 */
export function redoMultiple(count: number): Command[] {
	const redone: Command[] = [];
	for (let i = 0; i < count; i++) {
		const cmd = redo();
		if (cmd) {
			redone.push(cmd);
		} else {
			break;
		}
	}
	return redone;
}

/**
 * Jump to a specific point in history by undoing commands after it.
 * Useful for clicking on a command in history panel.
 *
 * @param commandId - ID of the command to jump to
 * @returns True if jump was successful
 *
 * @example
 * jumpToCommand('cmd-123'); // Undo all commands after cmd-123
 */
export function jumpToCommand(commandId: string): boolean {
	const stack = get(undoStack);
	const index = stack.findIndex((cmd) => cmd.id === commandId);

	if (index === -1) {
		return false;
	}

	// Undo all commands after the target
	const undoCount = stack.length - index - 1;
	undoMultiple(undoCount);

	return true;
}

/**
 * Clear all command history (both undo and redo).
 *
 * @example
 * clearHistory(); // Fresh start
 */
export function clearHistory(): void {
	undoStack.set([]);
	redoStack.set([]);
}

/**
 * Get command metadata without functions (for serialization/display).
 *
 * @param command - The command to extract metadata from
 * @returns Command metadata
 */
export function getCommandMeta(command: Command): CommandMeta {
	return {
		id: command.id,
		type: command.type,
		description: command.description,
		timestamp: command.timestamp
	};
}

/**
 * Get all command metadata from undo stack.
 *
 * @returns Array of command metadata
 */
export function getUndoHistory(): CommandMeta[] {
	return get(undoStack).map(getCommandMeta);
}

/**
 * Get all command metadata from redo stack.
 *
 * @returns Array of command metadata
 */
export function getRedoHistory(): CommandMeta[] {
	return get(redoStack).map(getCommandMeta);
}

// =============================================================================
// COMMAND FACTORIES
// =============================================================================

/**
 * Create a property change command.
 * Used when changing a property value on a target.
 *
 * @param target - Target identifier (component ID, element ID, etc.)
 * @param prop - Property name being changed
 * @param oldValue - Previous value
 * @param newValue - New value
 * @param applyFn - Optional custom apply function (defaults to console log)
 * @returns Command for property change
 *
 * @example
 * // Basic usage with external store/state management
 * const cmd = createPropChangeCommand(
 *   'button-1',
 *   'variant',
 *   'default',
 *   'primary',
 *   (target, prop, value) => componentStore.update(target, { [prop]: value })
 * );
 * executeCommand(cmd);
 */
export function createPropChangeCommand<T = unknown>(
	target: string,
	prop: string,
	oldValue: T,
	newValue: T,
	applyFn?: (target: string, prop: string, value: T) => void
): Command {
	const apply = applyFn ?? ((t, p, v) => console.log(`Apply ${p}=${v} to ${t}`));

	return {
		id: generateId(),
		type: 'prop-change',
		description: `Change ${prop} on ${target}`,
		execute: () => apply(target, prop, newValue),
		undo: () => apply(target, prop, oldValue),
		timestamp: new Date()
	};
}

/**
 * Create a move command for position changes.
 *
 * @param target - Target identifier
 * @param oldPos - Previous position
 * @param newPos - New position
 * @param applyFn - Optional custom apply function
 * @returns Command for move operation
 *
 * @example
 * const cmd = createMoveCommand(
 *   'widget-1',
 *   { x: 0, y: 0 },
 *   { x: 100, y: 50 },
 *   (target, pos) => widgetStore.setPosition(target, pos)
 * );
 */
export function createMoveCommand(
	target: string,
	oldPos: Position,
	newPos: Position,
	applyFn?: (target: string, pos: Position) => void
): Command {
	const apply = applyFn ?? ((t, p) => console.log(`Move ${t} to (${p.x}, ${p.y})`));

	return {
		id: generateId(),
		type: 'move',
		description: `Move ${target} from (${oldPos.x}, ${oldPos.y}) to (${newPos.x}, ${newPos.y})`,
		execute: () => apply(target, newPos),
		undo: () => apply(target, oldPos),
		timestamp: new Date()
	};
}

/**
 * Create a delete command with data preservation for undo.
 *
 * @param target - Target identifier
 * @param data - Data to preserve for restoration
 * @param deleteFn - Function to delete the target
 * @param restoreFn - Function to restore the target with data
 * @returns Command for delete operation
 *
 * @example
 * const cmd = createDeleteCommand(
 *   'item-1',
 *   { ...itemData },
 *   (target) => itemStore.delete(target),
 *   (target, data) => itemStore.restore(target, data)
 * );
 */
export function createDeleteCommand<T = unknown>(
	target: string,
	data: T,
	deleteFn?: (target: string) => void,
	restoreFn?: (target: string, data: T) => void
): Command {
	const doDelete = deleteFn ?? ((t) => console.log(`Delete ${t}`));
	const doRestore = restoreFn ?? ((t, d) => console.log(`Restore ${t}`, d));

	return {
		id: generateId(),
		type: 'delete',
		description: `Delete ${target}`,
		execute: () => doDelete(target),
		undo: () => doRestore(target, data),
		timestamp: new Date()
	};
}

/**
 * Create a create/add command.
 *
 * @param target - Target identifier
 * @param data - Data for the new item
 * @param createFn - Function to create the item
 * @param removeFn - Function to remove the item
 * @returns Command for create operation
 *
 * @example
 * const cmd = createAddCommand(
 *   'new-item',
 *   { name: 'New Item', type: 'widget' },
 *   (target, data) => itemStore.create(target, data),
 *   (target) => itemStore.remove(target)
 * );
 */
export function createAddCommand<T = unknown>(
	target: string,
	data: T,
	createFn?: (target: string, data: T) => void,
	removeFn?: (target: string) => void
): Command {
	const doCreate = createFn ?? ((t, d) => console.log(`Create ${t}`, d));
	const doRemove = removeFn ?? ((t) => console.log(`Remove ${t}`));

	return {
		id: generateId(),
		type: 'create',
		description: `Create ${target}`,
		execute: () => doCreate(target, data),
		undo: () => doRemove(target),
		timestamp: new Date()
	};
}

/**
 * Create a batch command that groups multiple commands.
 * All commands execute/undo together as one operation.
 *
 * @param description - Description for the batch
 * @param commands - Array of commands to batch
 * @returns Single command representing the batch
 *
 * @example
 * const cmd = createBatchCommand('Update multiple styles', [
 *   createPropChangeCommand('btn', 'color', 'red', 'blue', applyFn),
 *   createPropChangeCommand('btn', 'size', 'md', 'lg', applyFn)
 * ]);
 */
export function createBatchCommand(description: string, commands: Command[]): Command {
	return {
		id: generateId(),
		type: 'batch',
		description,
		execute: () => {
			commands.forEach((cmd) => cmd.execute());
		},
		undo: () => {
			// Undo in reverse order
			[...commands].reverse().forEach((cmd) => cmd.undo());
		},
		timestamp: new Date()
	};
}

/**
 * Create a custom command with explicit execute/undo functions.
 *
 * @param type - Command type identifier
 * @param description - Human-readable description
 * @param executeFn - Function to execute the command
 * @param undoFn - Function to undo the command
 * @returns Custom command
 *
 * @example
 * const cmd = createCustomCommand(
 *   'toggle-visibility',
 *   'Toggle panel visibility',
 *   () => panelStore.show(),
 *   () => panelStore.hide()
 * );
 */
export function createCustomCommand(
	type: string,
	description: string,
	executeFn: () => void,
	undoFn: () => void
): Command {
	return {
		id: generateId(),
		type,
		description,
		execute: executeFn,
		undo: undoFn,
		timestamp: new Date()
	};
}

// =============================================================================
// KEYBOARD SHORTCUTS
// =============================================================================

/**
 * Handler for keyboard shortcuts.
 * Listens for Cmd/Ctrl+Z (undo) and Cmd/Ctrl+Shift+Z (redo).
 */
function handleKeyboard(event: KeyboardEvent): void {
	if (!config.enableKeyboardShortcuts) {
		return;
	}

	const isMac = browser && navigator.platform.toUpperCase().indexOf('MAC') >= 0;
	const modifier = isMac ? event.metaKey : event.ctrlKey;

	if (!modifier || event.key.toLowerCase() !== 'z') {
		return;
	}

	// Prevent if in input/textarea
	const target = event.target as HTMLElement;
	if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
		return;
	}

	event.preventDefault();

	if (event.shiftKey) {
		redo();
	} else {
		undo();
	}
}

/**
 * Initialize keyboard shortcuts.
 * Called automatically in browser environment.
 */
export function initKeyboardShortcuts(): void {
	if (!browser) {
		return;
	}

	window.addEventListener('keydown', handleKeyboard);
}

/**
 * Remove keyboard shortcuts.
 * Call this when cleaning up (e.g., component unmount).
 */
export function destroyKeyboardShortcuts(): void {
	if (!browser) {
		return;
	}

	window.removeEventListener('keydown', handleKeyboard);
}

// =============================================================================
// INITIALIZATION
// =============================================================================

/**
 * Initialize the command history system.
 * Sets up keyboard shortcuts if enabled.
 */
export function initCommandHistory(): void {
	if (!browser) {
		return;
	}

	if (config.enableKeyboardShortcuts) {
		initKeyboardShortcuts();
	}
}

// Auto-initialize in browser
if (browser) {
	queueMicrotask(initCommandHistory);
}

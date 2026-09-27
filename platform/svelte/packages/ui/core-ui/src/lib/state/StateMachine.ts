/**
 * StateMachine - State Machine Integration for Complex Components
 *
 * Provides a lightweight state machine implementation for managing complex
 * component behaviors. Inspired by XState but simplified for Svelte 5 integration.
 *
 * Architecture:
 *   Data (Events) → Transform (Machine) → Result (State transitions)
 *
 * Standards:
 * - SCXML (State Chart XML): State machine specification
 * - XState: API conventions and patterns
 * - Actor Model: Side effects handled via actions
 *
 * Use Cases:
 * - Multi-step forms
 * - Modal flows
 * - Component lifecycle states
 * - Async operation states (idle → loading → success/error)
 *
 * @example
 * // Define a simple fetch machine
 * const fetchMachine = createMachine({
 *   id: 'fetch',
 *   initial: 'idle',
 *   states: {
 *     idle: {
 *       on: { FETCH: 'loading' }
 *     },
 *     loading: {
 *       on: {
 *         RESOLVE: 'success',
 *         REJECT: 'error'
 *       }
 *     },
 *     success: { type: 'final' },
 *     error: {
 *       on: { RETRY: 'loading' }
 *     }
 *   }
 * });
 *
 * @example
 * // Use in Svelte component
 * const [state, send] = useMachine(fetchMachine);
 * // state.value === 'idle'
 * send('FETCH');
 * // state.value === 'loading'
 */

// =============================================================================
// TYPES
// =============================================================================

/**
 * Event sent to state machine.
 */
export interface MachineEvent<TType extends string = string, TPayload = unknown> {
	type: TType;
	payload?: TPayload;
}

/**
 * Guard function - returns true if transition should proceed.
 */
export type GuardFn<TContext = unknown, TEvent extends MachineEvent = MachineEvent> = (
	context: TContext,
	event: TEvent
) => boolean;

/**
 * Action function - side effect executed on transition.
 */
export type ActionFn<TContext = unknown, TEvent extends MachineEvent = MachineEvent> = (
	context: TContext,
	event: TEvent
) => void | TContext | Promise<void | TContext>;

/**
 * Assign function - returns new context.
 */
export type AssignFn<TContext = unknown, TEvent extends MachineEvent = MachineEvent> = (
	context: TContext,
	event: TEvent
) => Partial<TContext>;

/**
 * Transition definition.
 */
export interface TransitionConfig<TContext = unknown> {
	/** Target state */
	target?: string;
	/** Guard condition */
	guard?: GuardFn<TContext> | string;
	/** Actions to execute */
	actions?: ActionFn<TContext>[] | string[];
	/** Context assignments */
	assign?: AssignFn<TContext>;
	/** Internal transition (no state change notification) */
	internal?: boolean;
}

/**
 * State node type.
 */
export type StateNodeType = 'atomic' | 'compound' | 'parallel' | 'final' | 'history';

/**
 * State node definition.
 */
export interface StateNodeConfig<TContext = unknown, TEvent extends string = string> {
	/** State type */
	type?: StateNodeType;
	/** Entry actions */
	entry?: ActionFn<TContext>[] | string[];
	/** Exit actions */
	exit?: ActionFn<TContext>[] | string[];
	/** Event transitions */
	on?: Record<TEvent, string | TransitionConfig<TContext> | TransitionConfig<TContext>[]>;
	/** Nested states (for compound states) */
	states?: Record<string, StateNodeConfig<TContext, TEvent>>;
	/** Initial nested state (for compound states) */
	initial?: string;
	/** After delay transitions */
	after?: Record<number | string, string | TransitionConfig<TContext>>;
	/** Always transitions (transient) */
	always?: TransitionConfig<TContext>[];
	/** Invoke async service */
	invoke?: InvokeConfig<TContext>;
	/** Description for debugging */
	description?: string;
	/** Tags for filtering */
	tags?: string[];
}

/**
 * Invoke service configuration.
 */
export interface InvokeConfig<TContext = unknown> {
	/** Service ID */
	id?: string;
	/** Promise or async function */
	src: string | ((context: TContext) => Promise<unknown>);
	/** Success transition event */
	onDone?: string | TransitionConfig<TContext>;
	/** Error transition event */
	onError?: string | TransitionConfig<TContext>;
}

/**
 * Machine configuration.
 */
export interface MachineConfig<
	TContext = unknown,
	TEvent extends string = string
> {
	/** Machine ID */
	id: string;
	/** Initial state */
	initial: string;
	/** Initial context data */
	context?: TContext;
	/** State definitions */
	states: Record<string, StateNodeConfig<TContext, TEvent>>;
	/** Named guards */
	guards?: Record<string, GuardFn<TContext>>;
	/** Named actions */
	actions?: Record<string, ActionFn<TContext>>;
	/** Named services */
	services?: Record<string, (context: TContext) => Promise<unknown>>;
	/** Description */
	description?: string;
}

/**
 * Current state value (can be nested: 'parent.child').
 */
export type StateValue = string | Record<string, StateValue>;

/**
 * Machine state snapshot.
 */
export interface MachineState<TContext = unknown> {
	/** Current state value */
	value: StateValue;
	/** Current context */
	context: TContext;
	/** History of state values */
	history: StateValue[];
	/** Whether in final state */
	done: boolean;
	/** Tags from current state */
	tags: Set<string>;
	/** Whether transition matches event */
	matches: (value: StateValue) => boolean;
	/** Check if state has tag */
	hasTag: (tag: string) => boolean;
	/** Whether can transition on event */
	can: (event: string) => boolean;
	/** Timestamp of last transition */
	timestamp: number;
}

/**
 * State machine instance.
 */
export interface Machine<TContext = unknown, TEvent extends string = string> {
	/** Machine ID */
	id: string;
	/** Machine configuration */
	config: MachineConfig<TContext, TEvent>;
	/** Get initial state */
	initialState: MachineState<TContext>;
	/** Transition to next state */
	transition: (state: MachineState<TContext>, event: MachineEvent<TEvent>) => MachineState<TContext>;
	/** Get all state names */
	stateNames: string[];
	/** Get all event names */
	eventNames: string[];
}

/**
 * Interpreter (actor) for running machine.
 */
export interface Interpreter<TContext = unknown, TEvent extends string = string> {
	/** Current state */
	state: MachineState<TContext>;
	/** Send event to machine */
	send: (event: TEvent | MachineEvent<TEvent>) => void;
	/** Start interpreter */
	start: () => void;
	/** Stop interpreter */
	stop: () => void;
	/** Subscribe to state changes */
	subscribe: (listener: (state: MachineState<TContext>) => void) => () => void;
	/** Whether interpreter is running */
	running: boolean;
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Normalize event to MachineEvent object.
 */
function normalizeEvent<TEvent extends string>(
	event: TEvent | MachineEvent<TEvent>
): MachineEvent<TEvent> {
	if (typeof event === 'string') {
		return { type: event };
	}
	return event;
}

/**
 * Normalize transition config to array.
 */
function normalizeTransitions<TContext>(
	config: string | TransitionConfig<TContext> | TransitionConfig<TContext>[]
): TransitionConfig<TContext>[] {
	if (typeof config === 'string') {
		return [{ target: config }];
	}
	if (Array.isArray(config)) {
		return config;
	}
	return [config];
}

/**
 * Get state tags recursively.
 */
function getStateTags<TContext>(
	stateValue: StateValue,
	states: Record<string, StateNodeConfig<TContext>>
): Set<string> {
	const tags = new Set<string>();
	const path = typeof stateValue === 'string' ? stateValue.split('.') : [];

	let current: StateNodeConfig<TContext> | undefined = states[path[0]];
	if (current?.tags) {
		current.tags.forEach((t) => tags.add(t));
	}

	for (let i = 1; i < path.length && current?.states; i++) {
		current = current.states[path[i]];
		if (current?.tags) {
			current.tags.forEach((t) => tags.add(t));
		}
	}

	return tags;
}

/**
 * Check if state value matches pattern.
 */
function matchesState(current: StateValue, pattern: StateValue): boolean {
	if (typeof current === 'string' && typeof pattern === 'string') {
		return current === pattern || current.startsWith(`${pattern}.`);
	}
	if (typeof current === 'object' && typeof pattern === 'object') {
		for (const key of Object.keys(pattern)) {
			if (!current[key] || !matchesState(current[key], pattern[key])) {
				return false;
			}
		}
		return true;
	}
	return false;
}

/**
 * Get available events from current state.
 */
function getAvailableEvents<TContext>(
	stateValue: StateValue,
	states: Record<string, StateNodeConfig<TContext>>
): string[] {
	const events: string[] = [];
	const path = typeof stateValue === 'string' ? stateValue.split('.') : [];

	let current: StateNodeConfig<TContext> | undefined = states[path[0]];
	if (current?.on) {
		events.push(...Object.keys(current.on));
	}

	for (let i = 1; i < path.length && current?.states; i++) {
		current = current.states[path[i]];
		if (current?.on) {
			events.push(...Object.keys(current.on));
		}
	}

	return [...new Set(events)];
}

// =============================================================================
// MACHINE CREATION
// =============================================================================

/**
 * Create a state machine.
 *
 * @example
 * const toggleMachine = createMachine({
 *   id: 'toggle',
 *   initial: 'inactive',
 *   context: { count: 0 },
 *   states: {
 *     inactive: {
 *       on: { TOGGLE: { target: 'active', assign: (ctx) => ({ count: ctx.count + 1 }) } }
 *     },
 *     active: {
 *       on: { TOGGLE: 'inactive' }
 *     }
 *   }
 * });
 */
export function createMachine<
	TContext = unknown,
	TEvent extends string = string
>(config: MachineConfig<TContext, TEvent>): Machine<TContext, TEvent> {
	const { id, initial, context, states, guards = {}, actions = {} } = config;

	// Create initial state
	const createInitialState = (): MachineState<TContext> => ({
		value: initial,
		context: (context ?? {}) as TContext,
		history: [],
		done: states[initial]?.type === 'final',
		tags: getStateTags(initial, states),
		matches: (pattern) => matchesState(initial, pattern),
		hasTag: function (tag) {
			return this.tags.has(tag);
		},
		can: (event) => getAvailableEvents(initial, states).includes(event),
		timestamp: Date.now()
	});

	// Transition function
	const transition = (
		currentState: MachineState<TContext>,
		event: MachineEvent<TEvent>
	): MachineState<TContext> => {
		const { value, context: ctx, history } = currentState;
		const stateKey = typeof value === 'string' ? value.split('.')[0] : Object.keys(value)[0];
		const stateConfig = states[stateKey];

		if (!stateConfig || !stateConfig.on) {
			return currentState;
		}

		const transitionConfig = stateConfig.on[event.type as TEvent];
		if (!transitionConfig) {
			return currentState;
		}

		const transitions = normalizeTransitions(transitionConfig);

		// Find first matching transition
		for (const trans of transitions) {
			// Check guard
			if (trans.guard) {
				const guardFn =
					typeof trans.guard === 'string' ? guards[trans.guard] : trans.guard;
				if (guardFn && !guardFn(ctx, event)) {
					continue;
				}
			}

			// Execute actions
			let newContext = ctx;
			if (trans.actions) {
				for (const action of trans.actions) {
					const actionFn = typeof action === 'string' ? actions[action] : action;
					if (actionFn) {
						const result = actionFn(newContext, event);
						if (result && typeof result === 'object' && !('then' in result)) {
							newContext = result as TContext;
						}
					}
				}
			}

			// Apply assign
			if (trans.assign) {
				const assigned = trans.assign(newContext, event);
				newContext = { ...newContext, ...assigned };
			}

			// Execute exit actions
			if (stateConfig.exit) {
				for (const action of stateConfig.exit) {
					const actionFn = typeof action === 'string' ? actions[action] : action;
					if (actionFn) {
						actionFn(newContext, event);
					}
				}
			}

			// Get target state
			const target = trans.target ?? stateKey;
			const targetConfig = states[target];

			// Execute entry actions
			if (targetConfig?.entry) {
				for (const action of targetConfig.entry) {
					const actionFn = typeof action === 'string' ? actions[action] : action;
					if (actionFn) {
						actionFn(newContext, event);
					}
				}
			}

			// Create new state
			const newState: MachineState<TContext> = {
				value: target,
				context: newContext,
				history: trans.internal ? history : [...history, value],
				done: targetConfig?.type === 'final',
				tags: getStateTags(target, states),
				matches: (pattern) => matchesState(target, pattern),
				hasTag: function (tag) {
					return this.tags.has(tag);
				},
				can: (evt) => getAvailableEvents(target, states).includes(evt),
				timestamp: Date.now()
			};

			return newState;
		}

		return currentState;
	};

	return {
		id,
		config,
		initialState: createInitialState(),
		transition,
		stateNames: Object.keys(states),
		eventNames: Object.keys(states).flatMap((s) => Object.keys(states[s].on ?? {}))
	};
}

// =============================================================================
// INTERPRETER
// =============================================================================

/**
 * Create an interpreter (actor) for a machine.
 *
 * @example
 * const interpreter = interpret(toggleMachine);
 * interpreter.subscribe((state) => console.log(state.value));
 * interpreter.start();
 * interpreter.send('TOGGLE');
 */
export function interpret<TContext = unknown, TEvent extends string = string>(
	machine: Machine<TContext, TEvent>
): Interpreter<TContext, TEvent> {
	let currentState = machine.initialState;
	let running = false;
	const listeners = new Set<(state: MachineState<TContext>) => void>();

	const notifyListeners = () => {
		listeners.forEach((listener) => listener(currentState));
	};

	return {
		get state() {
			return currentState;
		},

		send(event) {
			if (!running) {
				console.warn(`[StateMachine] Cannot send event to stopped machine: ${machine.id}`);
				return;
			}

			const normalizedEvent = normalizeEvent(event);
			const nextState = machine.transition(currentState, normalizedEvent);

			if (nextState !== currentState) {
				currentState = nextState;
				notifyListeners();
			}
		},

		start() {
			running = true;
			notifyListeners();
		},

		stop() {
			running = false;
			listeners.clear();
		},

		subscribe(listener) {
			listeners.add(listener);
			listener(currentState);
			return () => listeners.delete(listener);
		},

		get running() {
			return running;
		}
	};
}

// =============================================================================
// SVELTE 5 INTEGRATION
// =============================================================================

/**
 * Create a reactive machine state for Svelte 5.
 *
 * Uses Svelte 5 runes ($state) for reactivity.
 *
 * @example
 * // In Svelte component
 * import { useMachine } from './StateMachine';
 *
 * const { state, send, matches, can } = useMachine(fetchMachine);
 *
 * // Reactive access
 * {#if state.value === 'loading'}
 *   <Spinner />
 * {/if}
 *
 * <button onclick={() => send('FETCH')} disabled={!can('FETCH')}>
 *   Fetch Data
 * </button>
 */
export function useMachine<TContext = unknown, TEvent extends string = string>(
	machine: Machine<TContext, TEvent>
): {
	state: MachineState<TContext>;
	send: (event: TEvent | MachineEvent<TEvent>) => void;
	matches: (pattern: StateValue) => boolean;
	can: (event: string) => boolean;
	context: TContext;
} {
	// Use Svelte 5 $state rune for reactivity
	let state = $state(machine.initialState);
	const interpreter = interpret(machine);

	// Subscribe to state changes
	interpreter.subscribe((newState) => {
		state = newState;
	});

	interpreter.start();

	return {
		get state() {
			return state;
		},
		send: (event) => interpreter.send(event),
		matches: (pattern) => state.matches(pattern),
		can: (event) => state.can(event),
		get context() {
			return state.context;
		}
	};
}

// =============================================================================
// COMMON MACHINE PRESETS
// =============================================================================

/**
 * Create a fetch/async machine.
 *
 * States: idle → loading → success/error
 *
 * @example
 * const machine = createFetchMachine<UserData>('fetchUser');
 */
export function createFetchMachine<TData = unknown>(
	id: string,
	options?: {
		retryable?: boolean;
		maxRetries?: number;
	}
): Machine<{ data: TData | null; error: Error | null; retries: number }, 'FETCH' | 'RESOLVE' | 'REJECT' | 'RETRY' | 'RESET'> {
	const { retryable = true, maxRetries = 3 } = options ?? {};

	return createMachine({
		id,
		initial: 'idle',
		context: { data: null, error: null, retries: 0 },
		states: {
			idle: {
				on: { FETCH: 'loading' },
				tags: ['idle']
			},
			loading: {
				on: {
					RESOLVE: {
						target: 'success',
						assign: (_ctx, event) => ({
							data: event.payload as TData,
							error: null
						})
					},
					REJECT: {
						target: 'error',
						assign: (_ctx, event) => ({
							error: event.payload as Error
						})
					}
				},
				tags: ['loading', 'busy']
			},
			success: {
				type: 'final' as const,
				on: { RESET: 'idle' },
				tags: ['success', 'done']
			},
			error: {
				on: retryable
					? {
							RETRY: [
								{
									target: 'loading',
									guard: (ctx) => ctx.retries < maxRetries,
									assign: (ctx) => ({ retries: ctx.retries + 1 })
								},
								{ target: 'error' }
							],
							RESET: {
								target: 'idle',
								assign: () => ({ data: null, error: null, retries: 0 })
							}
						}
					: {
							RESET: {
								target: 'idle',
								assign: () => ({ data: null, error: null, retries: 0 })
							}
						},
				tags: ['error', 'done']
			}
		}
	});
}

/**
 * Create a toggle machine.
 *
 * States: off ↔ on
 *
 * @example
 * const machine = createToggleMachine('darkMode', true);
 */
export function createToggleMachine(
	id: string,
	initiallyOn = false
): Machine<{ toggleCount: number }, 'TOGGLE' | 'ON' | 'OFF'> {
	return createMachine({
		id,
		initial: initiallyOn ? 'on' : 'off',
		context: { toggleCount: 0 },
		states: {
			off: {
				on: {
					TOGGLE: {
						target: 'on',
						assign: (ctx) => ({ toggleCount: ctx.toggleCount + 1 })
					},
					ON: {
						target: 'on',
						assign: (ctx) => ({ toggleCount: ctx.toggleCount + 1 })
					}
				},
				tags: ['inactive']
			},
			on: {
				on: {
					TOGGLE: {
						target: 'off',
						assign: (ctx) => ({ toggleCount: ctx.toggleCount + 1 })
					},
					OFF: {
						target: 'off',
						assign: (ctx) => ({ toggleCount: ctx.toggleCount + 1 })
					}
				},
				tags: ['active']
			}
		}
	});
}

/**
 * Create a multi-step wizard machine.
 *
 * States: step1 → step2 → ... → complete
 *
 * @example
 * const machine = createWizardMachine('signup', ['profile', 'preferences', 'confirm']);
 */
export function createWizardMachine<TData extends Record<string, unknown> = Record<string, unknown>>(
	id: string,
	steps: string[],
	options?: {
		allowSkip?: boolean;
		validateStep?: (step: string, data: TData) => boolean;
	}
): Machine<
	{ currentStep: number; data: TData; errors: Record<string, string> },
	'NEXT' | 'PREV' | 'SKIP' | 'GOTO' | 'UPDATE' | 'RESET'
> {
	const { allowSkip = false } = options ?? {};

	const states: Record<string, StateNodeConfig<{ currentStep: number; data: TData; errors: Record<string, string> }>> = {};

	steps.forEach((step, index) => {
		const isFirst = index === 0;
		const isLast = index === steps.length - 1;

		states[step] = {
			on: {
				...(isLast
					? { NEXT: 'complete' }
					: {
							NEXT: {
								target: steps[index + 1],
								assign: () => ({ currentStep: index + 1 })
							}
						}),
				...(!isFirst && {
					PREV: {
						target: steps[index - 1],
						assign: () => ({ currentStep: index - 1 })
					}
				}),
				...(allowSkip &&
					!isLast && {
						SKIP: {
							target: steps[index + 1],
							assign: () => ({ currentStep: index + 1 })
						}
					}),
				UPDATE: {
					assign: (ctx, event) => ({
						data: { ...ctx.data, ...(event.payload as Partial<TData>) }
					}),
					internal: true
				},
				GOTO: steps.reduce(
					(acc, targetStep, targetIndex) => {
						acc[targetStep] = {
							target: targetStep,
							assign: () => ({ currentStep: targetIndex })
						};
						return acc;
					},
					{} as Record<string, TransitionConfig<{ currentStep: number; data: TData; errors: Record<string, string> }>>
				),
				RESET: {
					target: steps[0],
					assign: () => ({
						currentStep: 0,
						data: {} as TData,
						errors: {}
					})
				}
			},
			tags: [`step-${index + 1}`, step]
		};
	});

	states.complete = {
		type: 'final',
		on: {
			RESET: {
				target: steps[0],
				assign: () => ({
					currentStep: 0,
					data: {} as TData,
					errors: {}
				})
			}
		},
		tags: ['complete', 'done']
	};

	return createMachine({
		id,
		initial: steps[0],
		context: { currentStep: 0, data: {} as TData, errors: {} },
		states
	});
}

/**
 * Create a modal/dialog machine.
 *
 * States: closed ↔ opening → open ↔ closing → closed
 *
 * @example
 * const machine = createModalMachine('confirmDialog');
 */
export function createModalMachine(
	id: string,
	options?: {
		hasAnimation?: boolean;
	}
): Machine<{ openedAt: number | null }, 'OPEN' | 'CLOSE' | 'ANIMATION_END'> {
	const { hasAnimation = true } = options ?? {};

	if (!hasAnimation) {
		return createMachine({
			id,
			initial: 'closed',
			context: { openedAt: null },
			states: {
				closed: {
					on: {
						OPEN: {
							target: 'open',
							assign: () => ({ openedAt: Date.now() })
						}
					},
					tags: ['hidden']
				},
				open: {
					on: {
						CLOSE: {
							target: 'closed',
							assign: () => ({ openedAt: null })
						}
					},
					tags: ['visible']
				}
			}
		});
	}

	return createMachine({
		id,
		initial: 'closed',
		context: { openedAt: null },
		states: {
			closed: {
				on: {
					OPEN: {
						target: 'opening',
						assign: () => ({ openedAt: Date.now() })
					}
				},
				tags: ['hidden']
			},
			opening: {
				on: {
					ANIMATION_END: 'open',
					CLOSE: 'closing'
				},
				tags: ['animating', 'visible']
			},
			open: {
				on: { CLOSE: 'closing' },
				tags: ['visible']
			},
			closing: {
				on: {
					ANIMATION_END: {
						target: 'closed',
						assign: () => ({ openedAt: null })
					},
					OPEN: 'opening'
				},
				tags: ['animating', 'visible']
			}
		}
	});
}

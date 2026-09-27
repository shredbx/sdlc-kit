/**
 * Flow - Multi-Step Workflow Abstraction
 *
 * Provides higher-level abstractions for complex UI workflows that go beyond
 * simple state machines. Flows handle sequential steps, branching, progress
 * tracking, and recovery.
 *
 * Architecture:
 *   Data (Steps + Context) → Transform (Flow Engine) → Result (UI State)
 *
 * Use Cases:
 * - Onboarding flows (multiple screens with branching)
 * - Checkout flows (cart → shipping → payment → confirmation)
 * - Form wizards with conditional steps
 * - Setup wizards with parallel tracks
 * - Recovery flows (resume where user left off)
 *
 * Key Differences from StateMachine:
 * - StateMachine: Low-level state transitions, guards, actions
 * - Flow: High-level workflow orchestration, progress, persistence
 *
 * @example
 * // Define a checkout flow
 * const checkoutFlow = createFlow({
 *   id: 'checkout',
 *   steps: [
 *     { id: 'cart', component: CartStep },
 *     { id: 'shipping', component: ShippingStep, skip: (ctx) => ctx.isDigital },
 *     { id: 'payment', component: PaymentStep },
 *     { id: 'review', component: ReviewStep },
 *     { id: 'confirmation', component: ConfirmationStep, final: true }
 *   ]
 * });
 *
 * @example
 * // Use in Svelte
 * const { step, context, next, prev, progress } = useFlow(checkoutFlow);
 */

import type { Component } from 'svelte';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Step validation result.
 */
export interface StepValidation {
	valid: boolean;
	errors: Record<string, string>;
	warnings?: Record<string, string>;
}

/**
 * Step execution result.
 */
export interface StepResult<T = unknown> {
	success: boolean;
	data?: T;
	error?: Error;
	nextStep?: string;
}

/**
 * Step definition in a flow.
 */
export interface FlowStep<TContext = unknown> {
	/** Unique step identifier */
	id: string;
	/** Step title for progress display */
	title?: string;
	/** Step description */
	description?: string;
	/** Svelte component to render */
	component?: Component;
	/** Skip this step if condition is true */
	skip?: (context: TContext) => boolean;
	/** Guard - prevent entering step if returns false */
	canEnter?: (context: TContext) => boolean | Promise<boolean>;
	/** Guard - prevent leaving step if returns false */
	canLeave?: (context: TContext) => boolean | Promise<boolean>;
	/** Validate step data before proceeding */
	validate?: (context: TContext) => StepValidation | Promise<StepValidation>;
	/** Execute when entering step */
	onEnter?: (context: TContext) => void | Promise<void>;
	/** Execute when leaving step */
	onLeave?: (context: TContext) => void | Promise<void>;
	/** Execute step logic (for non-UI steps) */
	execute?: (context: TContext) => StepResult | Promise<StepResult>;
	/** Mark as final step */
	final?: boolean;
	/** Metadata for UI */
	meta?: Record<string, unknown>;
	/** Tags for filtering/grouping */
	tags?: string[];
}

/**
 * Branch definition for conditional flows.
 */
export interface FlowBranch<TContext = unknown> {
	/** Branch identifier */
	id: string;
	/** Condition to take this branch */
	condition: (context: TContext) => boolean;
	/** Steps in this branch */
	steps: FlowStep<TContext>[];
	/** Priority (higher = checked first) */
	priority?: number;
}

/**
 * Flow configuration.
 */
export interface FlowConfig<TContext = unknown> {
	/** Flow identifier */
	id: string;
	/** Flow title */
	title?: string;
	/** Flow description */
	description?: string;
	/** Sequential steps */
	steps: FlowStep<TContext>[];
	/** Optional branches */
	branches?: FlowBranch<TContext>[];
	/** Initial context */
	context?: TContext;
	/** Enable persistence */
	persist?: {
		/** Storage key */
		key: string;
		/** Storage type */
		storage?: 'local' | 'session' | 'memory';
		/** Fields to persist */
		include?: (keyof TContext)[];
		/** Fields to exclude */
		exclude?: (keyof TContext)[];
	};
	/** Callbacks */
	on?: {
		/** Called when flow starts */
		start?: (context: TContext) => void;
		/** Called when flow completes */
		complete?: (context: TContext) => void;
		/** Called on step change */
		stepChange?: (from: string, to: string, context: TContext) => void;
		/** Called on error */
		error?: (error: Error, stepId: string, context: TContext) => void;
	};
}

/**
 * Flow progress information.
 */
export interface FlowProgress {
	/** Current step index (0-based) */
	currentIndex: number;
	/** Total steps (excluding skipped) */
	totalSteps: number;
	/** Percentage complete (0-100) */
	percentage: number;
	/** Completed step IDs */
	completedSteps: string[];
	/** Skipped step IDs */
	skippedSteps: string[];
	/** Remaining step IDs */
	remainingSteps: string[];
	/** Whether flow is complete */
	isComplete: boolean;
	/** Whether flow just started */
	isStart: boolean;
}

/**
 * Navigation result.
 */
export interface NavigationResult {
	success: boolean;
	stepId: string;
	validation?: StepValidation;
	error?: Error;
}

/**
 * Flow state snapshot.
 */
export interface FlowState<TContext = unknown> {
	/** Flow ID */
	flowId: string;
	/** Current step */
	currentStep: FlowStep<TContext>;
	/** Current step ID */
	currentStepId: string;
	/** Flow context */
	context: TContext;
	/** Progress info */
	progress: FlowProgress;
	/** Step history */
	history: string[];
	/** Whether flow is loading */
	loading: boolean;
	/** Current validation errors */
	errors: Record<string, string>;
	/** Whether at first step */
	isFirst: boolean;
	/** Whether at last step */
	isLast: boolean;
	/** Available steps (accounting for skips) */
	availableSteps: FlowStep<TContext>[];
}

/**
 * Flow instance.
 */
export interface Flow<TContext = unknown> {
	/** Flow ID */
	id: string;
	/** Flow configuration */
	config: FlowConfig<TContext>;
	/** Get initial state */
	initialState: FlowState<TContext>;
	/** Get step by ID */
	getStep: (id: string) => FlowStep<TContext> | undefined;
	/** Get all steps (flat, including branches) */
	getAllSteps: () => FlowStep<TContext>[];
	/** Calculate available steps for context */
	getAvailableSteps: (context: TContext) => FlowStep<TContext>[];
	/** Calculate progress for state */
	calculateProgress: (stepId: string, context: TContext) => FlowProgress;
}

/**
 * Flow controller for runtime.
 */
export interface FlowController<TContext = unknown> {
	/** Current state */
	state: FlowState<TContext>;
	/** Navigate to next step */
	next: () => Promise<NavigationResult>;
	/** Navigate to previous step */
	prev: () => Promise<NavigationResult>;
	/** Go to specific step */
	goto: (stepId: string) => Promise<NavigationResult>;
	/** Update context */
	updateContext: (partial: Partial<TContext>) => void;
	/** Validate current step */
	validate: () => Promise<StepValidation>;
	/** Reset flow */
	reset: () => void;
	/** Complete flow */
	complete: () => Promise<void>;
	/** Check if can go next */
	canNext: () => boolean;
	/** Check if can go prev */
	canPrev: () => boolean;
	/** Check if step is accessible */
	canAccess: (stepId: string) => boolean;
	/** Subscribe to state changes */
	subscribe: (listener: (state: FlowState<TContext>) => void) => () => void;
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Filter steps that should be skipped.
 */
function filterAvailableSteps<TContext>(
	steps: FlowStep<TContext>[],
	context: TContext
): FlowStep<TContext>[] {
	return steps.filter((step) => !step.skip?.(context));
}

/**
 * Calculate progress from current step.
 */
function calculateProgress<TContext>(
	currentStepId: string,
	steps: FlowStep<TContext>[],
	context: TContext,
	history: string[]
): FlowProgress {
	const available = filterAvailableSteps(steps, context);
	const currentIndex = available.findIndex((s) => s.id === currentStepId);
	const totalSteps = available.length;

	const completedSteps = history.filter((id) => id !== currentStepId);
	const skippedSteps = steps.filter((s) => s.skip?.(context)).map((s) => s.id);
	const remainingSteps = available.slice(currentIndex + 1).map((s) => s.id);

	return {
		currentIndex,
		totalSteps,
		percentage: totalSteps > 0 ? Math.round(((currentIndex + 1) / totalSteps) * 100) : 0,
		completedSteps,
		skippedSteps,
		remainingSteps,
		isComplete: available[currentIndex]?.final === true,
		isStart: currentIndex === 0
	};
}

/**
 * Get storage adapter.
 */
function getStorage(type: 'local' | 'session' | 'memory'): Storage | Map<string, string> {
	if (typeof window === 'undefined') {
		return new Map<string, string>();
	}

	switch (type) {
		case 'local':
			return localStorage;
		case 'session':
			return sessionStorage;
		case 'memory':
		default:
			return new Map<string, string>();
	}
}

/**
 * Persist flow state.
 */
function persistState<TContext>(
	config: FlowConfig<TContext>['persist'],
	state: { stepId: string; context: TContext; history: string[] }
): void {
	if (!config) return;

	const storage = getStorage(config.storage ?? 'local');
	let contextToSave = state.context;

	// Filter fields
	if (config.include) {
		contextToSave = {} as TContext;
		for (const key of config.include) {
			(contextToSave as Record<string, unknown>)[key as string] = (state.context as Record<string, unknown>)[key as string];
		}
	} else if (config.exclude) {
		contextToSave = { ...state.context };
		for (const key of config.exclude) {
			delete (contextToSave as Record<string, unknown>)[key as string];
		}
	}

	const data = JSON.stringify({
		stepId: state.stepId,
		context: contextToSave,
		history: state.history,
		timestamp: Date.now()
	});

	if (storage instanceof Map) {
		storage.set(config.key, data);
	} else {
		storage.setItem(config.key, data);
	}
}

/**
 * Restore flow state.
 */
function restoreState<TContext>(
	config: FlowConfig<TContext>['persist']
): { stepId: string; context: Partial<TContext>; history: string[] } | null {
	if (!config) return null;

	const storage = getStorage(config.storage ?? 'local');
	const data = storage instanceof Map ? storage.get(config.key) : storage.getItem(config.key);

	if (!data) return null;

	try {
		return JSON.parse(data);
	} catch {
		return null;
	}
}

// =============================================================================
// FLOW CREATION
// =============================================================================

/**
 * Create a flow.
 *
 * @example
 * const onboardingFlow = createFlow({
 *   id: 'onboarding',
 *   steps: [
 *     { id: 'welcome', title: 'Welcome' },
 *     { id: 'profile', title: 'Your Profile' },
 *     { id: 'preferences', title: 'Preferences', skip: (ctx) => ctx.skipPrefs },
 *     { id: 'complete', title: 'All Done!', final: true }
 *   ],
 *   context: { name: '', email: '', skipPrefs: false }
 * });
 */
export function createFlow<TContext = Record<string, unknown>>(
	config: FlowConfig<TContext>
): Flow<TContext> {
	const { id, steps, branches = [], context = {} as TContext } = config;

	const getStep = (stepId: string): FlowStep<TContext> | undefined => {
		const mainStep = steps.find((s) => s.id === stepId);
		if (mainStep) return mainStep;

		for (const branch of branches) {
			const branchStep = branch.steps.find((s) => s.id === stepId);
			if (branchStep) return branchStep;
		}

		return undefined;
	};

	const getAllSteps = (): FlowStep<TContext>[] => {
		const allSteps = [...steps];
		for (const branch of branches) {
			allSteps.push(...branch.steps);
		}
		return allSteps;
	};

	const getAvailableSteps = (ctx: TContext): FlowStep<TContext>[] => {
		// Check branches first (sorted by priority)
		const sortedBranches = [...branches].sort((a, b) => (b.priority ?? 0) - (a.priority ?? 0));

		for (const branch of sortedBranches) {
			if (branch.condition(ctx)) {
				return filterAvailableSteps(branch.steps, ctx);
			}
		}

		return filterAvailableSteps(steps, ctx);
	};

	const calculateProgressFn = (stepId: string, ctx: TContext): FlowProgress => {
		const available = getAvailableSteps(ctx);
		return calculateProgress(stepId, available, ctx, []);
	};

	const initialStep = filterAvailableSteps(steps, context)[0] ?? steps[0];
	const initialProgress = calculateProgressFn(initialStep.id, context);

	const initialState: FlowState<TContext> = {
		flowId: id,
		currentStep: initialStep,
		currentStepId: initialStep.id,
		context,
		progress: initialProgress,
		history: [],
		loading: false,
		errors: {},
		isFirst: true,
		isLast: initialProgress.remainingSteps.length === 0,
		availableSteps: getAvailableSteps(context)
	};

	return {
		id,
		config,
		initialState,
		getStep,
		getAllSteps,
		getAvailableSteps,
		calculateProgress: calculateProgressFn
	};
}

// =============================================================================
// FLOW CONTROLLER
// =============================================================================

/**
 * Create a flow controller.
 *
 * @example
 * const controller = createFlowController(checkoutFlow);
 * controller.subscribe((state) => console.log(state.currentStepId));
 * await controller.next();
 */
export function createFlowController<TContext = Record<string, unknown>>(
	flow: Flow<TContext>
): FlowController<TContext> {
	let state: FlowState<TContext>;
	const listeners = new Set<(state: FlowState<TContext>) => void>();

	// Try to restore state
	const restored = restoreState(flow.config.persist);
	if (restored) {
		const step = flow.getStep(restored.stepId);
		const ctx = { ...flow.config.context, ...restored.context } as TContext;
		const available = flow.getAvailableSteps(ctx);
		const progress = flow.calculateProgress(restored.stepId, ctx);

		state = {
			flowId: flow.id,
			currentStep: step ?? available[0],
			currentStepId: step?.id ?? available[0].id,
			context: ctx,
			progress,
			history: restored.history,
			loading: false,
			errors: {},
			isFirst: progress.currentIndex === 0,
			isLast: progress.remainingSteps.length === 0,
			availableSteps: available
		};
	} else {
		state = flow.initialState;
		flow.config.on?.start?.(state.context);
	}

	const notify = () => {
		listeners.forEach((l) => l(state));

		// Persist
		if (flow.config.persist) {
			persistState(flow.config.persist, {
				stepId: state.currentStepId,
				context: state.context,
				history: state.history
			});
		}
	};

	const updateState = (partial: Partial<FlowState<TContext>>) => {
		state = { ...state, ...partial };
		notify();
	};

	const navigateTo = async (stepId: string): Promise<NavigationResult> => {
		const targetStep = flow.getStep(stepId);
		if (!targetStep) {
			return { success: false, stepId, error: new Error(`Step not found: ${stepId}`) };
		}

		updateState({ loading: true });

		try {
			// Check canLeave on current step
			if (state.currentStep.canLeave) {
				const canLeave = await state.currentStep.canLeave(state.context);
				if (!canLeave) {
					updateState({ loading: false });
					return { success: false, stepId: state.currentStepId };
				}
			}

			// Check canEnter on target step
			if (targetStep.canEnter) {
				const canEnter = await targetStep.canEnter(state.context);
				if (!canEnter) {
					updateState({ loading: false });
					return { success: false, stepId };
				}
			}

			// Execute onLeave
			await state.currentStep.onLeave?.(state.context);

			// Update history
			const newHistory = [...state.history, state.currentStepId];

			// Execute onEnter
			await targetStep.onEnter?.(state.context);

			// Calculate new progress
			const available = flow.getAvailableSteps(state.context);
			const progress = flow.calculateProgress(stepId, state.context);

			// Notify step change
			flow.config.on?.stepChange?.(state.currentStepId, stepId, state.context);

			updateState({
				currentStep: targetStep,
				currentStepId: stepId,
				history: newHistory,
				progress,
				loading: false,
				errors: {},
				isFirst: progress.currentIndex === 0,
				isLast: progress.remainingSteps.length === 0,
				availableSteps: available
			});

			// Check if complete
			if (targetStep.final) {
				flow.config.on?.complete?.(state.context);
			}

			return { success: true, stepId };
		} catch (error) {
			const err = error instanceof Error ? error : new Error(String(error));
			flow.config.on?.error?.(err, stepId, state.context);
			updateState({ loading: false, errors: { _flow: err.message } });
			return { success: false, stepId, error: err };
		}
	};

	const validate = async (): Promise<StepValidation> => {
		if (!state.currentStep.validate) {
			return { valid: true, errors: {} };
		}

		try {
			const result = await state.currentStep.validate(state.context);
			updateState({ errors: result.errors });
			return result;
		} catch (error) {
			const err = error instanceof Error ? error : new Error(String(error));
			const result = { valid: false, errors: { _validation: err.message } };
			updateState({ errors: result.errors });
			return result;
		}
	};

	return {
		get state() {
			return state;
		},

		async next(): Promise<NavigationResult> {
			// Validate first
			const validation = await validate();
			if (!validation.valid) {
				return { success: false, stepId: state.currentStepId, validation };
			}

			// Find next available step
			const available = flow.getAvailableSteps(state.context);
			const currentIndex = available.findIndex((s) => s.id === state.currentStepId);
			const nextStep = available[currentIndex + 1];

			if (!nextStep) {
				return { success: false, stepId: state.currentStepId, error: new Error('No next step') };
			}

			return navigateTo(nextStep.id);
		},

		async prev(): Promise<NavigationResult> {
			// Find previous step from history
			const prevStepId = state.history[state.history.length - 1];
			if (!prevStepId) {
				return { success: false, stepId: state.currentStepId, error: new Error('No previous step') };
			}

			// Remove from history before navigating
			const newHistory = state.history.slice(0, -1);
			state = { ...state, history: newHistory };

			return navigateTo(prevStepId);
		},

		async goto(stepId: string): Promise<NavigationResult> {
			return navigateTo(stepId);
		},

		updateContext(partial: Partial<TContext>) {
			const newContext = { ...state.context, ...partial };
			const available = flow.getAvailableSteps(newContext);
			const progress = flow.calculateProgress(state.currentStepId, newContext);

			updateState({
				context: newContext,
				availableSteps: available,
				progress,
				isFirst: progress.currentIndex === 0,
				isLast: progress.remainingSteps.length === 0
			});
		},

		validate,

		reset() {
			state = flow.initialState;

			// Clear persistence
			if (flow.config.persist) {
				const storage = getStorage(flow.config.persist.storage ?? 'local');
				if (storage instanceof Map) {
					storage.delete(flow.config.persist.key);
				} else {
					storage.removeItem(flow.config.persist.key);
				}
			}

			flow.config.on?.start?.(state.context);
			notify();
		},

		async complete() {
			if (state.currentStep.final) {
				flow.config.on?.complete?.(state.context);
			}
		},

		canNext() {
			const available = flow.getAvailableSteps(state.context);
			const currentIndex = available.findIndex((s) => s.id === state.currentStepId);
			return currentIndex < available.length - 1;
		},

		canPrev() {
			return state.history.length > 0;
		},

		canAccess(stepId: string) {
			const available = flow.getAvailableSteps(state.context);
			return available.some((s) => s.id === stepId);
		},

		subscribe(listener) {
			listeners.add(listener);
			listener(state);
			return () => listeners.delete(listener);
		}
	};
}

// =============================================================================
// SVELTE 5 INTEGRATION
// =============================================================================

/**
 * Use a flow in a Svelte 5 component.
 *
 * @example
 * // In Svelte component
 * import { useFlow } from './Flow';
 *
 * const {
 *   state,
 *   next,
 *   prev,
 *   updateContext,
 *   progress
 * } = useFlow(checkoutFlow);
 *
 * // Render current step
 * <svelte:component this={state.currentStep.component} bind:data={state.context} />
 *
 * // Navigation buttons
 * <button onclick={prev} disabled={!state.isFirst}>Back</button>
 * <button onclick={next} disabled={state.isLast}>Next</button>
 *
 * // Progress bar
 * <progress value={progress.percentage} max="100" />
 */
export function useFlow<TContext = Record<string, unknown>>(
	flow: Flow<TContext>
): {
	state: FlowState<TContext>;
	next: () => Promise<NavigationResult>;
	prev: () => Promise<NavigationResult>;
	goto: (stepId: string) => Promise<NavigationResult>;
	updateContext: (partial: Partial<TContext>) => void;
	validate: () => Promise<StepValidation>;
	reset: () => void;
	canNext: boolean;
	canPrev: boolean;
	progress: FlowProgress;
	currentStep: FlowStep<TContext>;
	context: TContext;
} {
	const controller = createFlowController(flow);

	// Use Svelte 5 $state rune for reactivity
	let flowState = $state(controller.state);

	// Subscribe to state changes
	controller.subscribe((newState) => {
		flowState = newState;
	});

	return {
		get state() {
			return flowState;
		},
		next: () => controller.next(),
		prev: () => controller.prev(),
		goto: (stepId) => controller.goto(stepId),
		updateContext: (partial) => controller.updateContext(partial),
		validate: () => controller.validate(),
		reset: () => controller.reset(),
		get canNext() {
			return controller.canNext();
		},
		get canPrev() {
			return controller.canPrev();
		},
		get progress() {
			return flowState.progress;
		},
		get currentStep() {
			return flowState.currentStep;
		},
		get context() {
			return flowState.context;
		}
	};
}

// =============================================================================
// FLOW PRESETS
// =============================================================================

/**
 * Create a linear wizard flow.
 *
 * @example
 * const signupFlow = createLinearFlow('signup', [
 *   { id: 'email', title: 'Enter Email' },
 *   { id: 'password', title: 'Create Password' },
 *   { id: 'profile', title: 'Your Profile' },
 *   { id: 'confirm', title: 'Confirm', final: true }
 * ]);
 */
export function createLinearFlow<TContext = Record<string, unknown>>(
	id: string,
	steps: Array<{
		id: string;
		title?: string;
		description?: string;
		skip?: (ctx: TContext) => boolean;
		validate?: (ctx: TContext) => StepValidation | Promise<StepValidation>;
		final?: boolean;
	}>,
	options?: {
		context?: TContext;
		persist?: FlowConfig<TContext>['persist'];
		on?: FlowConfig<TContext>['on'];
	}
): Flow<TContext> {
	return createFlow({
		id,
		steps: steps.map((s) => ({
			...s,
			final: s.final ?? false
		})),
		context: options?.context,
		persist: options?.persist,
		on: options?.on
	});
}

/**
 * Create a branching flow with conditions.
 *
 * @example
 * const checkoutFlow = createBranchingFlow('checkout', {
 *   main: [
 *     { id: 'cart', title: 'Cart' },
 *     { id: 'shipping', title: 'Shipping' },
 *     { id: 'payment', title: 'Payment' },
 *     { id: 'confirm', title: 'Confirm', final: true }
 *   ],
 *   branches: [
 *     {
 *       id: 'digital',
 *       condition: (ctx) => ctx.isDigital,
 *       steps: [
 *         { id: 'cart', title: 'Cart' },
 *         { id: 'payment', title: 'Payment' },
 *         { id: 'download', title: 'Download', final: true }
 *       ]
 *     }
 *   ]
 * });
 */
export function createBranchingFlow<TContext = Record<string, unknown>>(
	id: string,
	config: {
		main: FlowStep<TContext>[];
		branches: FlowBranch<TContext>[];
		context?: TContext;
		persist?: FlowConfig<TContext>['persist'];
		on?: FlowConfig<TContext>['on'];
	}
): Flow<TContext> {
	return createFlow({
		id,
		steps: config.main,
		branches: config.branches,
		context: config.context,
		persist: config.persist,
		on: config.on
	});
}

/**
 * Create a decision tree flow.
 *
 * Each step can have multiple next options based on user choice.
 *
 * @example
 * const surveyFlow = createDecisionFlow('survey', {
 *   start: { id: 'q1', title: 'Do you like X?', next: { yes: 'q2a', no: 'q2b' } },
 *   steps: {
 *     q2a: { id: 'q2a', title: 'How much?', next: { lots: 'end_fan', some: 'end_casual' } },
 *     q2b: { id: 'q2b', title: 'Why not?', next: { end: 'end_critic' } },
 *     end_fan: { id: 'end_fan', title: 'You are a fan!', final: true },
 *     end_casual: { id: 'end_casual', title: 'Casual user', final: true },
 *     end_critic: { id: 'end_critic', title: 'Not for you', final: true }
 *   }
 * });
 */
export function createDecisionFlow<TContext extends { _decision?: string } = { _decision?: string }>(
	id: string,
	config: {
		start: {
			id: string;
			title?: string;
			next: Record<string, string>;
		};
		steps: Record<
			string,
			{
				id: string;
				title?: string;
				next?: Record<string, string>;
				final?: boolean;
			}
		>;
		context?: TContext;
	}
): Flow<TContext> {
	const allSteps: FlowStep<TContext>[] = [];

	// Add start step
	allSteps.push({
		id: config.start.id,
		title: config.start.title,
		onLeave: (ctx) => {
			const decision = ctx._decision;
			if (decision && config.start.next[decision]) {
				// Decision is stored in context._decision
			}
		}
	});

	// Add all other steps
	for (const [, step] of Object.entries(config.steps)) {
		allSteps.push({
			id: step.id,
			title: step.title,
			final: step.final
		});
	}

	return createFlow({
		id,
		steps: allSteps,
		context: config.context
	});
}

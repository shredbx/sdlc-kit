/**
 * State Module Index
 *
 * Exports state management utilities including state machines
 * and flow abstractions for complex UI behaviors.
 *
 * @example
 * import { createMachine, useMachine, createFetchMachine } from '.';
 * import { createFlow, useFlow, createLinearFlow } from '.';
 *
 * // State machine for component state
 * const machine = createFetchMachine('userData');
 * const { state, send } = useMachine(machine);
 *
 * // Flow for multi-step workflows
 * const flow = createLinearFlow('onboarding', [...steps]);
 * const { next, prev, progress } = useFlow(flow);
 */

// =============================================================================
// State Machine
// =============================================================================
export {
	// Types
	type MachineEvent,
	type GuardFn,
	type ActionFn,
	type AssignFn,
	type TransitionConfig,
	type StateNodeType,
	type StateNodeConfig,
	type InvokeConfig,
	type MachineConfig,
	type StateValue,
	type MachineState,
	type Machine,
	type Interpreter,
	// Functions
	createMachine,
	interpret,
	useMachine,
	// Presets
	createFetchMachine,
	createToggleMachine,
	createWizardMachine,
	createModalMachine
} from './StateMachine';

// =============================================================================
// Flow - Multi-Step Workflow Abstraction
// =============================================================================
export {
	// Types
	type StepValidation,
	type StepResult,
	type FlowStep,
	type FlowBranch,
	type FlowConfig,
	type FlowProgress,
	type NavigationResult,
	type FlowState,
	type Flow,
	type FlowController,
	// Functions
	createFlow,
	createFlowController,
	useFlow,
	// Presets
	createLinearFlow,
	createBranchingFlow,
	createDecisionFlow
} from './Flow';

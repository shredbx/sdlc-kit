/**
 * Sections — Content-level UI regions (Organisms in Atomic Design)
 *
 * Section components are self-contained content regions: timelines,
 * process steps, entity details, catalogues, editor panels.
 *
 * @layer sections (level 3 in atomic design)
 */

// =============================================================================
// BACKWARD COMPAT — layout shells moved to layouts/
// Existing imports still work. New code: import from '@sbx/core-ui/components/layouts'
// =============================================================================
export {
	CollapsibleSidebarLayout,
	SidebarLayout
} from '../layouts';

// =============================================================================
// COMPONENT EXPORTS
// =============================================================================

// Section primitives — PageShell + Section composition model
export { default as Section } from './Section.svelte';
export { default as SectionHeader } from './SectionHeader.svelte';
export { default as StripedStack } from './StripedStack.svelte';

// Backward compat �� sidebar/nav components moved to navigation/
export {
	Sidebar, ContentSidebar, SidebarHeader, SidebarSection,
	ContentNavGroup, ContentNavItem, SidebarPresenter,
	type ContentSidebarSection, type ContentSidebarItem, type SidebarItem, type SidebarSectionType
} from '../navigation';

// Navigation sections
export { default as NavBar } from './NavBar.svelte';

// Editor sections
export { default as ContextPanel } from './ContextPanel.svelte';
export { default as CollapsibleSidebar } from './CollapsibleSidebar.svelte';

// Chat sections — ChatPanel composes ChatMessage + ChatInput; ChatWidget is the
// floating FAB shell around ChatPanel. Co-located so their relative imports resolve.
export { default as ChatPanel } from './ChatPanel.svelte';
export { default as ChatMessage } from './ChatMessage.svelte';
export { default as ChatInput } from './ChatInput.svelte';
export { default as ChatWidget } from './ChatWidget.svelte';

// Prompt-document editor — a compact, flat (title + text) editor for a prompt
// "document", plus the pure renderer (xml | markdown) the editor previews. Lean v1:
// no presets, no nesting, no variables.
export { default as PromptDocumentEditor } from './PromptDocumentEditor.svelte';

// =============================================================================
// TYPE EXPORTS
// =============================================================================

export type { TreeItem, ContextSection } from './ContextPanel.types';
export type { CollapsibleSidebarProps, ExpansionState } from './CollapsibleSidebar.svelte';
export type {
	Message,
	ChatSuggestion,
	ChatButton,
	ChatQuickReply,
	ChatCard
} from './ChatPanel.svelte';

// Prompt-document model + renderer (pure; no Svelte) — importable server-side.
export {
	renderPromptDocument,
	escapeXml,
	type PromptBlock,
	type PromptDocumentFormat
} from './PromptDocument';

// =============================================================================
// PRESENTER EXPORTS
// =============================================================================

export { default as NavBarPresenter } from './NavBar.presenter';
export { default as ContextPanelPresenter } from './ContextPanel.presenter';
export { default as CollapsibleSidebarPresenter } from './CollapsibleSidebar.presenter';
export { default as ChatMessagePresenter } from './ChatMessage.presenter';
export { default as ChatInputPresenter } from './ChatInput.presenter';
export { default as ChatWidgetPresenter } from './ChatWidget.presenter';

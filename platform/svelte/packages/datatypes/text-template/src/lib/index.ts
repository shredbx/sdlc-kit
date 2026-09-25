// @sbx/text-template — headless token-interpolating text templates. Framework-agnostic + no
// consumer deps: the package owns parse/serialise/resolve; consumers implement TokenResolver /
// TokenCatalog. The Svelte composer that drives it lives in the UI layer (canvas-ui today).
export type { TemplateNode, Template, TemplateToken, TokenResolver, TokenCatalog } from './template.js';
export { parseTemplate, serializeTemplate, resolveTemplate, templateTokens } from './template.js';

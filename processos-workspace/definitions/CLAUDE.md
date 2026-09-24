# processos-workspace/definitions/sbx-sdlc-kit — filesystem governance

This scope's top-level namespace is locked to the 13 capabilities recorded as data under
`architecture/capability/` (`process-cli list records sbx-sdlc-kit/architecture/capability`;
`process-cli show record sbx-sdlc-kit/architecture/capability/<name>` for what each one covers).

**Never add a 14th top-level folder here.** A new schema/type/action/process/template nests
under whichever of the 13 capability folders its record's own description matches — create that
capability folder the first time something real needs it (lazily; see
`docs/proposals/schema-evolution-principles.md`), never all 13 up front. If nothing fits, raise
it with the user rather than inventing a new top-level name.

See `docs/proposals/schema-evolution-principles.md` for the four extension mechanisms once
inside a capability folder. See `architecture/decision/nested-claude-md-governance/` for why
this file exists.

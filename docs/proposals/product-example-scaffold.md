# First product scaffold: process-os reuse assessment + MCP/FastAPI/plugin shape

**Status:** reconnaissance only, gathered in an isolated worktree — no schemas, code, or plugins
implemented. This is grounding for a discuss-first design pass with the user, mirroring how
`docs/proposals/infrastructure-services-design.md` was built up this same session, not a plan
already executed.

## process-os upstream status

No commits in the local `process-os` checkout since `2026-09-23 20:46` — that predates the
`2026-09-24` process-kit port. The 7 ported packages (`types`, `schema`, `action`, `filesystem`,
`config`, `template`, `process`) are current with upstream `HEAD`; nothing to re-sync.

## `sdlc` / `sdlc.python`: the generic-dispatch-then-platform-specific shape

- `sdlc.build-application` (process): `resolve-platform` → `switch: platform.name` →
  `sdlc.python.build-application`. The generic layer knows nothing about Python; `node` is
  already recognized in `resolve-platform.sh`'s own language table but has no case yet
  (`stop: no build process for this platform yet`).
- `sdlc.resolve-platform` (action): validates language against platform (`python`→`uv`,
  `node`→`npm`), emits a `platform` record (name/language/framework/runtime/tool).
- `sdlc.build.yaml`'s own header comment is the real, working example of mechanism #4
  (`requires:` + sibling `extension/build.yaml`) that this repo's own
  `schema-evolution-principles.md` already cites — confirms that doc's claim against the actual
  source, not just its own restatement.
- `sdlc.python.build-application` (process): `create-project → install → test → package` — a
  real 4-step process. Its `create-project` action confirms `process-cli render ... --into
  <real-target-path>` writes to a genuine destination (e.g. `platform/python/apps/<name>`), not
  into `processos-workspace/output/` scratch — resolves an ambiguity in `process-cli render`'s
  own `--into` help text ("a folder of the runtime's output") by reading what the action actually
  passes it.

## Reuse assessment: cross-repo `uses:`/`libraries:` vs. adapt fresh

Two live options, per CLAUDE.md's own note that a `libraries:` entry in `processos.yaml`
(readonly, optionally version-pinned) is confirmed working:

- **Reuse `sdlc`/`sdlc.python` directly** — puts a readonly external scope in the dependency path
  of every product built here, and hard-codes assumptions `resolve-platform.sh` bakes in
  (python+typer only today).
- **Adapt the shape into `sbx-sdlc-kit`'s own namespace** — re-authors roughly 6 small
  definitions (a resolve-platform equivalent, `platform`/`build`/`build-artifact` schemas, a
  build-application dispatcher), but keeps this workspace self-contained and free to diverge for
  shapes `sdlc.python.build-application` never anticipated (a FastAPI+MCP service, not a typer
  CLI).

Not decided here — this is the first real question for the user once this lands.

## Plugin distribution: process-os's own proven mechanism

process-os already solves exactly what the user described this session — ship new reusable
capability as an installable plugin rather than embedding it in-repo:

- `process-os.create-plugin` (action) + `process-os.claude-plugin` (template) scaffold a
  marketplace-ready plugin skeleton (`.claude-plugin/plugin.json` + one skill stub + one command
  stub) into `products/<name>/`, register it in the repo-root `.claude-plugin/marketplace.json`
  (creating that file on the first plugin), and exclude the plugin folder from the uv workspace
  (it has no `pyproject.toml`).
- `process-claude-plugin`'s own real `.claude-plugin/plugin.json`:
  `mcpServers: {process-claude-plugin: {command: "process-cli", args: ["mcp"]}}` — a **local
  stdio MCP server**, spawned as a subprocess by Claude Code itself. Not HTTP, not FastAPI.

Same open question as above applies here too (reuse cross-repo vs. copy the shape) — not
re-litigated separately.

## Open gap: stdio vs. HTTP MCP

Every MCP exposure that exists in process-os today is local-stdio. Nothing here is a
FastAPI-fronted, network-reachable MCP server — that part is genuinely new ground, not reuse of
proven prior art. The user's own framing this session ("deploy to use with AI chats or any other
messengers via adapters") implies at least one deployment target needs network reach, not just
local stdio spawned per Claude Code session. Needs its own discussion: does the product need
*both* a stdio MCP (for Claude Code itself) and an HTTP one (for external chat/messenger
adapters), or just the HTTP one, with Claude Code also talking to it over HTTP?

## One concrete implementation fact, not a decision

`platform/python/pyproject.toml`'s uv workspace is `members = ["packages/*"]` only. A first app
under `platform/python/apps/` needs that glob extended (or a separate workspace file) before the
existing toolchain (ruff/mypy config, `dev` dependency group) covers it. Trivial, but real —
surfaces the moment a product scaffold is actually modeled, not before.

## Explicitly not decided or answered here

- What the example product's actual domain/subject is — the user's call, not pre-picked.
- `apps/` vs `applications/` under `platform/<lang>/` — already deferred workspace-wide
  (`architecture/decision/apps-vs-applications-naming`), not re-opened here.
- `platform-position-in-namespace` — deferred pending "real content to test namespace depth
  against"; that condition is now satisfied (process-kit is fully ported). Worth revisiting soon,
  but not answered in this doc.
- Whether FastAPI is the right choice at all vs. a stdio-only MCP server with no HTTP layer,
  given no real product requirement has been discussed yet.

# Architecture capability & namespace governance — proposal

Status: **proposal — nothing modeled yet**, same discipline as every other doc in `docs/proposals/`.
Nothing here gets `process-cli write`-ten without your explicit yes, item by item.

Last updated: 2026-09-24
tags: proposal, architecture, governance, capability, namespace

Resolves two decision points left open in `docs/proposals/filestructure-and-starter-set.html` (§2,
§3) — noted inline below; that file is left alone as the historical record of how we got here.

---

## 1. How to read this

Everything below came out of chat discussion this session. Sections 2–4 restate decisions you
already made in chat, formalized here so they're written down once instead of living only in
scrollback. Section 5 is new and is the actual thing that needs your approval: two schemas that
make "architecture" a real, checked entry point into this workspace instead of a folder of prose.

## 2. Filestructure — resolved

`products/` and `experiments/` nest under a single `projects/` parent, sibling to `platform/` —
matching `sbx-workspace`'s shape (`projects/{products,experiments,prototypes,tools}`), not the flatter
form the original proposal defaulted to. Root stays: `processos.yaml`, `processos-workspace/`,
`platform/`, `projects/`, `docs/`, `CLAUDE.md`.

**Still open, your call, not addressed this round:** `apps` vs `applications` as the folder name under
`platform/<lang>/`.

## 3. Namespace shape — resolved

Two layers, both settled:

- **Capability is the outermost namespace segment** (`sdlc.<capability>.…`), applied lazily — a path
  exists only when a real definition needs it, never pre-scaffolded. (Already resolved before this
  session; restated for completeness.)
- **Within a capability, namespace by framework, not platform** — `sdlc.<capability>.nextjs`, not
  `sdlc.<capability>.node` with framework as a field. Extend with a dotted segment only when framework
  alone doesn't disambiguate (e.g. `nextjs.javascript` vs `nextjs.typescript`, if that distinction is
  ever real). Fall back to a platform-level segment (`sdlc.<capability>.node`) only when there
  genuinely is no framework in play.

**Still open, deliberately deferred:** platform/framework position depth beyond this — left until the
`types` port gives real content to test it against, per `docs/proposals/porting-and-modeling-process.md`.

## 4. Governance principle: types and schemas never mix a name or a home

A schema (`type: mapping`) is one kind of type; a bare scalar/enum/sequence type is another. Both
already live in separate kind-folders per namespace (`type/` vs `schema/`) — the rule going forward is
to keep them that way on purpose: a primitive value-type and a domain mapping-concept never share a
name, and a primitive type that's genuinely needed gets its own clear, dedicated namespace rather than
being tucked wherever's convenient. (This is why `capability`, below, is a schema only — no
competing bare enum-type of the same name.)

This also means: the day we need a real cross-cutting primitive (a date pattern, a slug pattern), that
is also the day we resolve the still-open question of reusing process-os's own `std` scope via a
`libraries:` entry vs. growing our own `sbx-sdlc-kit.std` — not before, and not guessed at now.

---

## 5. New definitions proposed

Two schemas, both under a new `architecture` namespace — this is what actually makes "architecture"
the entry point you described: where capability names are governed, and where decisions get recorded
instead of living only in proposal-doc callouts or chat history.

### 5a. `sbx-sdlc-kit.architecture.capability`

Path: `processos-workspace/definitions/sbx-sdlc-kit/architecture/schema/capability.yaml`

```yaml
# One capability this workspace organizes namespaces by. Records under this schema ARE the closed
# list — before adding a new sdlc.<name>.* namespace, check what's recorded here; if it's not here,
# that's the signal to add a record (discussed first), not to freelance a namespace segment.
name: capability
type: mapping
fields:
  name:        {type: string, required: true}
  description: {type: string, required: true}
```

One record per capability (§6 drafts the content), created via `process-cli create` — duplicates are
structurally impossible, since `create` refuses to overwrite a file that already exists for a given
name.

### 5b. `sbx-sdlc-kit.architecture.decision`

Path: `processos-workspace/definitions/sbx-sdlc-kit/architecture/schema/decision.yaml`

```yaml
# A point-in-time call this workspace made — decided or deliberately deferred — recorded so it can
# be checked later instead of re-derived from chat history or proposal-doc callouts.
name: decision
type: mapping
fields:
  topic:      {type: string, required: true}   # stable slug for the underlying question — shared by
                                                # every decision ever made about it; this is what groups
                                                # related decisions instead of an unstructured flat list
  question:   {type: string, required: true}
  status:     {type: decision-status, required: true}
  resolution: {type: string, required: true}   # the decision itself, or what it's deferred until
  rationale:  {type: string, required: true}
  date:       {type: string, required: true}   # ISO date, e.g. 2026-09-24
  supersedes: {type: string, required: false}  # a prior decision record's path, if this replaces it
```

This needs one small companion primitive — `decision-status`, a genuine enum (`decided`, `deferred`,
`superseded`), which is exactly the case §4's principle is for: a real scalar type, its own file, never
sharing a name with a schema.

Path: `processos-workspace/definitions/sbx-sdlc-kit/architecture/type/decision-status.yaml`

```yaml
# Where a decision record currently stands.
name: decision-status
type: string
enum: [decided, deferred, superseded]
```

**Records nest one folder per topic**, not one flat folder of loosely-named slugs — this is the actual
fix for "how do you tell two decisions on the same question apart from two unrelated ones":

```
processos-workspace/records/sbx-sdlc-kit/architecture/decision/
  namespace-shape/
    2026-09-24.yaml       # status: decided
  filestructure/
    2026-09-24.yaml       # status: decided
```

If a topic is ever revisited, the new call is a new dated file in that same topic folder
(`--set supersedes=<old file's path>`), **and** the old file's own `status` field gets edited in place
to `superseded` — so `ls` on a topic folder always shows the full history, and "what's current" is
just "the one file in this folder whose status isn't `superseded`" (in practice also the most recent
date). Nothing is ever deleted; both files stay as the record of what changed and why.

Once this exists, this session's own decisions get backfilled as records: filestructure nesting,
framework-first namespace, capability-as-namespace, apps-vs-applications (deferred), platform position
in namespace (deferred), and the type/schema separation principle itself.

---

## 6. Draft capability descriptions (for your review — not yet written as records)

The 13 names are shredbx's own taxonomy (confirmed via research into shredbx's largest client
product, now kept in that client's own repo), adopted as a reference list, not inherited wholesale. The descriptions below are written fresh, in our own words, for our own context —
edit freely.

| Capability | Description |
|---|---|
| `modeling` | Defining what a thing *is* before it's built — the types, schemas, and records that describe an app, package, or process ahead of any code. |
| `architecture` | How the workspace itself is organized — namespace shape, scope structure, filestructure conventions, and the governance (this capability list, the decision log) that keeps those choices from being reinvented ad hoc. |
| `implementation` | Actually building the thing — actions, generated or hand-written code, the logic that makes a modeled entity real. |
| `infrastructure` | The underlying platforms and services something runs on — servers, hosting, databases, networking. Topology, not app code. |
| `configuration` | The settings that shape how a built thing behaves without changing its code — env vars, config files, template-rendered baseline files. |
| `testing` | Verifying a thing works — test strategy, fixtures, coverage, and the checks that catch drift from what was modeled. |
| `documentation` | Explaining a thing for humans — READMEs, proposals, research notes, usage guides. Meant to be read, not executed. |
| `security` | Protecting a thing from misuse — auth, access control, secret handling, hardening, vulnerability review. |
| `automation` | Repeatable execution of work that would otherwise be done by hand — process-os actions/processes, CI jobs, scripted pipelines. |
| `deployment` | Getting a built, tested thing into a running environment — releases, publishing, shipping. |
| `observability` | Seeing what a running thing is doing — logging, metrics, tracing, dashboards. |
| `operations` | Day-to-day running and maintaining a live thing once deployed — incident response, on-call, routine upkeep. |
| `continuity` | Keeping a thing recoverable and durable over time — backups, disaster recovery, versioning/rollback, long-term retention. |

## 7. Sequencing

1. `sbx-sdlc-kit.architecture.capability` schema + `decision-status` type + `sbx-sdlc-kit.architecture.decision`
   schema — the governance layer itself.
2. `process-cli check`.
3. 13 capability records (§6 content, as edited by you).
4. Backfill this session's own decisions as `decision` records.
5. Only then: the previously-agreed next real work (porting `types` into `platform/python/`, per
   `docs/proposals/porting-and-modeling-process.md`) — unaffected by any of this, still next in line
   after the governance layer lands.

## 8. Approval checklist

- [x] §2 filestructure nesting — formalizing what you already confirmed; recorded as decision
  `sbx-sdlc-kit/architecture/decision/filestructure-nesting/first.yaml`
- [x] §3 namespace shape — formalizing what you already confirmed; recorded as decisions
  `namespace-capability-outermost/first.yaml` and `namespace-dispatch-within-capability/first.yaml`
- [x] §4 type/schema separation principle — formalizing what you already confirmed; recorded as
  decision `type-schema-separation/first.yaml`
- [x] §5a `sbx-sdlc-kit.architecture.capability` schema — written, `process-cli check` clean
- [x] §5b `decision-status` type + `sbx-sdlc-kit.architecture.decision` schema (with the `topic` field
  added mid-milestone for grouping) — written, `process-cli check` clean
- [x] §6 capability descriptions — reviewed, approved as-is, all 13 written as records
- [ ] `apps` vs `applications` — still unresolved; now durably recorded as **deferred** in decision
  `apps-vs-applications-naming/first.yaml`, not just prose here
- [ ] platform position depth in namespace — still deliberately deferred until the `types` port; now
  durably recorded as **deferred** in decision `platform-position-in-namespace/first.yaml`

Milestone 1 (governance layer) is complete: `processos-workspace/definitions/sbx-sdlc-kit/architecture/`
holds 1 type + 2 schemas, `processos-workspace/records/sbx-sdlc-kit/architecture/` holds 13 capability
records + 6 decision records (one topic folder each), all validated by `process-cli check`. Full task
log: `docs/plans/2026-09-24-architecture-capability-governance-milestone-1.md`.

# Architecture capability governance — Milestone 1 implementation plan

> **For Claude:** this plan follows this repo's own `CLAUDE.md` approval discipline, **not** the
> generic executing-plans/subagent-driven flow. Every `type`/`schema` definition below (Tasks 1–3)
> needs the user's explicit yes on its exact content, in this same conversation, before
> `process-cli write` runs — never batched, never unattended. Records (Tasks 5–6) are created once
> their containing schema is approved and their content has been explicitly signed off (in
> `docs/proposals/architecture-capability-governance.md` or in chat).

**Goal:** stand up the `architecture` capability's governance layer in `sbx-sdlc-kit` — a schema
that lists and describes every capability namespace we're allowed to use, and a schema that records
point-in-time decisions — so a future session can check both instead of re-deriving them from chat
history.

**Architecture:** two new definitions under a new `sbx-sdlc-kit.architecture` namespace
(`processos-workspace/definitions/sbx-sdlc-kit/architecture/{type,schema}/`), populated by records
under `processos-workspace/records/sbx-sdlc-kit/architecture/{capability,decision}/`. No code, no
actions, no processes — pure declarative data, per `docs/proposals/architecture-capability-governance.md`.

**Tech stack:** process-os / `process-cli` (`write`, `check`, `create`, `list records`, `show`).

---

### Task 1: `decision-status` type

**Definition id:** `sbx-sdlc-kit.architecture.decision-status`
**Written to (by `process-cli write`, not by hand):** `processos-workspace/definitions/sbx-sdlc-kit/architecture/type/decision-status.yaml`

**Step 1 — content** (already agreed, proposal §5b):
```yaml
# Where a decision record currently stands.
name: decision-status
type: string
enum: [decided, deferred, superseded]
```

**Step 2 — explicit approval** on this exact content, before anything is written.

**Step 3 — write:**
```bash
process-cli write type sbx-sdlc-kit.architecture.decision-status <path-to-drafted-file>
```
Expected: `type sbx-sdlc-kit.architecture.decision-status: written`

**Step 4 — sanity check:**
```bash
process-cli show type sbx-sdlc-kit.architecture.decision-status
```
Expected: the file, verbatim.

---

### Task 2: `capability` schema

**Definition id:** `sbx-sdlc-kit.architecture.capability`
**Written to:** `processos-workspace/definitions/sbx-sdlc-kit/architecture/schema/capability.yaml`

**Step 1 — content** (already agreed, proposal §5a):
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

**Step 2 — explicit approval.**

**Step 3 — write:**
```bash
process-cli write schema sbx-sdlc-kit.architecture.capability <path-to-drafted-file>
```
Expected: `schema sbx-sdlc-kit.architecture.capability: written`

---

### Task 3: `decision` schema

Depends on Task 1 — its `status` field references `decision-status`, so Task 1 must be written
first.

**Definition id:** `sbx-sdlc-kit.architecture.decision`
**Written to:** `processos-workspace/definitions/sbx-sdlc-kit/architecture/schema/decision.yaml`

**Step 1 — content** (already agreed, proposal §5b):
```yaml
# A point-in-time call this workspace made — decided or deliberately deferred — recorded so it can
# be checked later instead of re-derived from chat history or proposal-doc callouts.
name: decision
type: mapping
fields:
  topic:      {type: string, required: true}
  question:   {type: string, required: true}
  status:     {type: decision-status, required: true}
  resolution: {type: string, required: true}
  rationale:  {type: string, required: true}
  date:       {type: string, required: true}
  supersedes: {type: string, required: false}
```

**Step 2 — explicit approval.**

**Step 3 — write:**
```bash
process-cli write schema sbx-sdlc-kit.architecture.decision <path-to-drafted-file>
```
Expected: `schema sbx-sdlc-kit.architecture.decision: written`

---

### Task 4: `process-cli check`

```bash
process-cli check
```
Expected: `<config>: ok`. If not: read which definition and why (per `using-process-os`), fix, and
re-run before Task 5 — do not create records against a schema that doesn't check out clean.

---

### Task 5: 13 capability records (one reviewed batch) — DONE

**Files:** `processos-workspace/records/sbx-sdlc-kit/architecture/capability/<name>.yaml`

Content = proposal §6, as edited by the user. For each of the 13 names (`modeling`, `architecture`,
`implementation`, `infrastructure`, `configuration`, `testing`, `documentation`, `security`,
`automation`, `deployment`, `observability`, `operations`, `continuity`):

```bash
process-cli create sbx-sdlc-kit.architecture.capability \
  --set name=<name> \
  --set description="<description>" \
  --name <name> \
  --into sbx-sdlc-kit/architecture/capability
```
**Correction (found while running this task):** `--into` is relative to `processos-workspace/records/`
already — passing the full `processos-workspace/records/...` path double-nests it. Use the bare
`sbx-sdlc-kit/architecture/capability` form above.

Expected each: `<file>: written`

**Spot check:**
```bash
process-cli list records sbx-sdlc-kit/architecture/capability
```
**Correction:** `list records` takes the prefix as a positional argument, not `--prefix`.

Expected: 13 paths. Confirmed 2026-09-24 — all 13 written, verified with `process-cli show record`,
`process-cli check` clean.

---

### Task 6: backfill this session's own decisions — DONE

**Files:** `processos-workspace/records/sbx-sdlc-kit/architecture/decision/<topic>/<name>.yaml` — one
subfolder per topic (per the topic-grouping decision made this session), one file per call made about
that topic. All six are first calls, so each is a lone file in a fresh topic folder, `supersedes` empty.

**Correction (found while running this task):** `--name` must be lower-case words joined by hyphens —
no digits, so a date like `2026-09-24` is rejected as a record name. The `date` field already inside
each record is the source of truth for timing; the filename just needs to be unique per topic folder.
Used `first` for all six (a future revisit of the same topic would be named `second`, etc.).

**1. `filestructure-nesting`**
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=filestructure-nesting \
  --set question="Where do product-facing folders (products/, experiments/) live in the repo root — flat at root, or nested under a parent folder?" \
  --set status=decided \
  --set resolution="products/ and experiments/ nest under a single projects/ parent, sibling to platform/ — matching sbx-workspace's shape (projects/{products,experiments,prototypes,tools})." \
  --set rationale="Keeps repo root clean as more folders (prototypes, tools) get added later; matches a shape already proven in sbx-workspace rather than inventing a new one." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/filestructure-nesting
```

**2. `namespace-capability-outermost`**
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=namespace-capability-outermost \
  --set question="Is 'capability' the outermost namespace segment for sdlc definitions, and are all capability paths pre-scaffolded up front or created lazily on real need?" \
  --set status=decided \
  --set resolution="Capability is the outermost namespace segment (sdlc.<capability>....), applied lazily — a path exists only when a real definition needs it, never pre-scaffolded." \
  --set rationale="Resolved before this session; restated here so it's checkable instead of only living in chat history. Pre-scaffolding all capability paths up front repeats the sbx.framework mistake of modeling structure ahead of real content (see CLAUDE.md's capability-scopes tension note)." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/namespace-capability-outermost
```

**3. `namespace-dispatch-within-capability`**
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=namespace-dispatch-within-capability \
  --set question="Within a capability namespace, do we dispatch by platform with framework as a field (sdlc.<capability>.node), or directly by framework (sdlc.<capability>.nextjs)?" \
  --set status=decided \
  --set resolution="Namespace by framework first (sdlc.<capability>.nextjs), not by platform. Extend with a dotted segment only when framework alone doesn't disambiguate (e.g. nextjs.javascript vs nextjs.typescript, if that distinction is ever real). Fall back to a platform-level segment (sdlc.<capability>.node) only when there's genuinely no framework in play." \
  --set rationale="Almost every action/template/package is framework-specific in practice; a platform-level field-based grouping would add an extra layer that rarely gets reused. Extending with a dotted segment only on real disambiguation need keeps names clean rather than pre-guessing splits that might never be needed." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/namespace-dispatch-within-capability
```

**4. `type-schema-separation`**
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=type-schema-separation \
  --set question="Can a mapping schema and a bare scalar/enum/sequence type ever share a name or a filesystem location?" \
  --set status=decided \
  --set resolution="No — a schema (type: mapping) and a primitive type are always kept in separate kind-folders (type/ vs schema/) per namespace, and never share a name. A primitive type that's genuinely needed gets its own clear, dedicated namespace rather than being tucked in wherever's convenient." \
  --set rationale="Keeps 'what defines the shape of X' (schema) and 'what constrains a single value' (type) unambiguous at a glance — this is why capability is a schema only, with no competing bare enum-type of the same name, and why decision-status is its own separate type file rather than inlined." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/type-schema-separation
```

**5. `apps-vs-applications-naming`** (deferred)
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=apps-vs-applications-naming \
  --set question="Under platform/<lang>/, is the folder for bootstrapped applications named apps/ or applications/?" \
  --set status=deferred \
  --set resolution="Not decided yet — deferred until we actually port or bootstrap the first application under platform/." \
  --set rationale="No real content exists yet to test either name against; picking now would be guessing ahead of need." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/apps-vs-applications-naming
```

**6. `platform-position-in-namespace`** (deferred)
```bash
process-cli create sbx-sdlc-kit.architecture.decision \
  --set topic=platform-position-in-namespace \
  --set question="Beyond capability-outermost and framework-not-platform dispatch, what's the full position/depth of platform vs framework segments in the namespace?" \
  --set status=deferred \
  --set resolution="Deferred until the types port (process-os's types package into platform/python/) gives real content to test namespace depth against." \
  --set rationale="Per docs/proposals/porting-and-modeling-process.md's build-to-prove-then-model rule — this shape shouldn't be guessed at from zero real ports." \
  --set date=2026-09-24 \
  --name first \
  --into sbx-sdlc-kit/architecture/decision/platform-position-in-namespace
```

**Spot check:**
```bash
process-cli list records sbx-sdlc-kit/architecture/decision
```
Expected: 6 paths, one per topic folder above. Confirmed 2026-09-24 — all 6 written, verified with
`process-cli show record`, `process-cli check` clean.

**If a topic is ever revisited later** (not part of this milestone): create a new file (`second`,
`third`, ...) in that same topic folder with `--set supersedes=<old file's path>`, and separately edit
the old file's `status` field to `superseded` — so the folder always shows full history with exactly
one current (non-superseded) entry.

**Final check:**
```bash
process-cli check
```
Ran clean: `processos.yaml: ok`.

---

### Task 7: wrap-up

- Mark `docs/proposals/architecture-capability-governance.md` §8's checklist fully checked.
- Ask the user whether to `git commit` this milestone (nothing is committed without being asked).
- Next milestone (separate, not part of this one): port process-os's `types` package into
  `platform/python/`, per `docs/proposals/porting-and-modeling-process.md` — unaffected by this
  work, still next in line after it lands.

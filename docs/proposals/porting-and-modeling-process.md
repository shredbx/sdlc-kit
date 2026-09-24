# Porting & modeling process — what comes first, code or schema?

Last updated: 2026-09-24
tags: process, methodology, proposal

Status: **proposal — needs your approval**, same as every other doc in `docs/proposals/`. The last
section is the part meant to actually go into `CLAUDE.md` if you agree with it; everything above it
is the reasoning.

## The question, restated

For any new piece of work — porting `types` this week, but really for everything after it — do we:

**(A)** build the real thing first (working code, run against the real environment/SDK/API), and
only afterward write the type/schema/action/process that describes what we actually built, or

**(B)** write the type/schema/action/process first, as a design, and then implement to match it?

You named the risk on each side precisely: (B) risks "freezing" a description of something that
doesn't survive contact with reality — an SDK quirk, a library that doesn't behave like its docs, a
constraint you only discover by trying — and then either the docs lie or you're re-modeling under
pressure. (A) risks never formalizing anything, or formalizing the *wrong* generalization too early
from a single example — modeling for modeling's sake, the thing your MAIN RULE already forbids.

## My answer: it depends on one question, not a universal rule

**Do we already know this shape works?**

- **If yes** (we're porting code that's already built, tested, and running — like `process-kit`'s
  `types` package right now) — there's no "will this even work" uncertainty left. Model and
  code can move together, because the schema is describing something already proven, not guessing
  at one.
- **If no** (a new SDK, a new integration, the first attempt at a pattern — the AI Assistant's
  provider integration is going to be exactly this) — **build it in `/experiments/` first, with no
  process-os ceremony, until it demonstrably works.** Only then extract the real, minimal contract
  (the inputs it actually needed, the outputs it actually produced, the failure modes it actually
  hit) into a type/schema/action/process. This is already how you've scoped the chat-UI spike, and
  I think it should be the standing rule for anything with real uncertainty in it, not a one-off
  exception.

This isn't a compromise between (A) and (B) — it's recognizing they're answers to two different
situations, and the mistake is applying either one universally.

## The "rule of three" — what actually earns a template or action

The other half of your question — when do we stop doing something manually and write the
process-os definition for it? — has a concrete, well-tested answer from software engineering in
general, and process-os's own history backs it specifically: **don't abstract from one example.
Do it by hand at least twice, then look at what was actually the same both times, and model that.**

This matches what the Python-quality research found in process-os itself: all seven `process-kit`
packages share an identical `pyproject.toml` shape, and only *then* does `process-os.create-package`
(the action that scaffolds a new one from a template) exist — the template describes a pattern that
was real across multiple packages already, not a guess at what a package "should" look like before
any existed. It's also the exact failure `sbx.framework` hit and documented: it modeled "capability"
as a namespace before there was enough real content to know if that shape composed — decision #0057
calls the result "1000 capability files… unnecessary brainstorming matrix." One spike doesn't earn a
template. A second real instance of the same shape does.

**Applied to `types` this week**: port the code, its tests, and the tooling gap-fixes
(ruff/mypy/CI) — manually, no process-os action wrapping it, because there's nothing repeated yet to
generalize from. The moment we port a *second* package, that's the natural point to look at what was
actually the same across both and write `sbx-sdlc-kit`'s own `create-package`/`port-package` action
and template — informed by two real data points instead of one imagined one.

## "Frozen" is the wrong mental model

A written definition isn't a permanent commitment — it's a checked, versioned file, same as code.
When the environment changes underneath it (an SDK breaks a process that used to work), the fixture
tests or the action's own checks fail, and that failure *is* the signal to revise the definition —
the same way `process-cli check` already catches a broken definition before anyone hits a confusing
runtime error. The Python-quality research flagged this same discipline in process-os's own
development notes: "once a test passes, break the code once on purpose to see it fail, then restore
it" — i.e., the check is only trustworthy if it's been proven to actually catch drift. Treat a
process-os definition the same way you'd treat any other code: revisable on evidence, not a
one-time-only spec that's embarrassing to admit was wrong.

## Governance rules surface the same way — noticed, drafted, graduated

The same discipline applies to the governance layer itself (`decision`, `guideline`,
`quality-attribute`), not just packages: don't stop mid-task to model a `guideline` the moment
something rule-like is noticed. Jot a one-line draft note in the current plan/milestone doc instead,
and only turn matured notes into real records at a milestone's wrap-up — or when a second real
instance of the same rule shows up. "We build what we use, and we instantly use what we build"
applies to governance exactly like it applies to `create-package`.

## The house rule (proposed addition to `CLAUDE.md`, pending your approval)

> **Build to prove, then model — never the reverse, and never from one example.** Where the shape is
> already proven (a port of working code, a library used exactly as documented), model and
> implement together. Where anything is genuinely uncertain (a new SDK, a new integration, a first
> attempt), build it working in `/experiments/` first, with no process-os ceremony, and only
> formalize the type/schema/action/process once it's demonstrably real — and only once the same
> shape has shown up at least twice, not from a single instance. A definition that stops matching
> reality is a normal, cheap-to-catch maintenance event (`process-cli check`, a failing fixture) —
> not proof the modeling was wrong forever. The same applies to governance rules noticed mid-task:
> note them where the work is happening, graduate the real ones into `decision`/`guideline` records
> at a milestone's wrap-up, not the instant they're spotted.

# Unit model, Scope U2b — Go, six dependency-free modules (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U2b, go, records, readme, plan

> Design: `docs/proposals/unit-model-design.md`. Follows `docs/plans/2026-09-26-unit-model-scope-U2a.md`. The six README texts were shown
> to the user and approved before anything was copied into `platform/`. Touches **no** Go code, `go.mod`, test or definition. Nothing is pushed.

## The slice

The six level-0 modules that have **no `require` at all**: `contacts/{personname,phonenumber,socialnetwork}`, `media/video`,
`notifications/notify`, `persistence/repository`. They are closed under `uses` and have no third-party requirements, so, as in U2a, no
definition changes. The other five level-0 modules (`identity/rbac`, `localization/language`, `news/rss`, `persistence/database`,
`media/image`) have third-party requirements and form U2c, where the `requirement` type is loosened once.

`persistence/repository` contains a **nested module** `postgres/` (its own `go.mod`; id `repository-postgres`; it requires `database`).
It is a later unit. The `repository` README says where the boundary is.

## Before → after (new and modified paths only)

```
BEFORE                                                      AFTER
records/.../modeling/package/ (11 units)                    {personname, phonenumber, socialnetwork, video, notify, repository}/
                                                              {package.yaml, readme.yaml}                NEW × 12
platform/go/packages/{contacts/{personname,phonenumber,socialnetwork}, media/video, notifications/notify, persistence/repository}
                                                            README.md  NEW × 6  (rendered from the records, copied in)
docs/plans/ (this file) NEW · unit-model-design.md MODIFIED (section 9) · bos design: one log line
definitions/ · platform/CLAUDE.md · Go code · go.mod · go.sum · tests · go.work   UNCHANGED (0 files)
```

## How it was made

- **Authoring: markdown, then convert** (the decision left open after U2a). Each README was written as plain markdown with two
  placeholders, `<<CODE>>` and `<<OUTPUT>>`. A scratch script filled them from a complete example program and its real run, converted the
  markdown to a `readme` record, validated both records, rendered the record back and required the render to equal the markdown byte for
  byte. All six passed on the first pass; a one-word edit to a record makes the check fail. The converter is scratch (about 60 lines); its
  repo form is decided in the process scope (U4).
- **Examples are wired the way the README says**: a scratch module with `require ... v0.0.0` and a `replace` to the unit's folder, no
  `go.work`. That tests the Install recipe as well as the code.
- **Every example makes no network call.** The `notify` example gives the Telegram adapter an `*http.Client` whose transport is a fake Bot
  API; the `video` example gives the registry a fake oEmbed client.

## What reading the code found (in the READMEs; no code was changed, per the verbatim rule)

- `notify`: the SMTP adapter writes the title, fields, `From` and `To` into the message without removing line breaks (proven on a scratch
  copy: a line break in a title adds a header); `SMTPConfig.TLS` only decides whether to authenticate, it does not turn encryption on;
  `Host` and `Port` are not defaulted; a nil Telegram client means `http.DefaultClient` (no timeout); the Telegram base URL is unexported;
  the SMTP adapter has no tests.
- `video`: the type comments cite `video.*` dictionary YAML that exists nowhere in the workspace (the value sets are the Go constants);
  TikTok `vm.` and `vt.` short links match but do not parse; Facebook is a valid `Platform` with no resolver.
- `phonenumber`, `socialnetwork`: the package comments describe a "3-column expansion" that no code in the workspace performs (the
  `repository/postgres` `Mapper` is only an interface).
- `personname` had no package comment at all.
- Fixes belong to each kit's own refinement slice, not to this scope.

## A correction to U2a

The U2a example gate ran the programs through a scratch `go.work` with a `require` and no `replace`. That layout makes the `go` tool itself
look up `github.com` (the user's `GOPRIVATE=github.com/shredbx/*` sends those paths to git, and `go list -m all` even ran `git ls-remote`),
so that gate was not network-silent. The READMEs' programs sent nothing; the lookup came from the tool. The three committed U2a examples
were re-run under the corrected harness (require + replace, resolver canary, network denied by `sandbox-exec`) and pass.

## Results

| Gate | Result |
|---|---|
| `process-cli check` | `processos.yaml: ok` |
| Records against their schemas | 17 of 17 `package`, 17 of 17 `readme` (8 Python + 9 Go) |
| `name` equals the last segment of `path`; `path` holds a README; `kit` matches its folder | 17 of 17; 17 of 17; 17 of 17 |
| Reference integrity: every `uses` id has a record | ok |
| Render vs the README in the record's own `path`, byte for byte | **17 of 17**; `conform` 17 of 17 |
| Markdown draft → record → render equals the approved draft, before anything was copied | 6 of 6 |
| README examples: `gofmt`-clean, run, 0 DNS lookups, identical with the network denied, output equals what the README prints | 9 of 9 |
| `go test` on the nine Go modules vs the baselines taken before (pass incl. subtests / top-level / non-pass) | equal: personname 21/5/0, phonenumber 19/4/0, socialnetwork 27/8/0, video 40/20/0, notify 4/4/0, repository 56/24/0, seo 2/2/0, money 120/34/0, geocoordinate 17/4/0 |
| Resolver canary (`GODEBUG=netdns=go+2`) during those runs | no name looked up |
| Negative controls, run on bad input | invalid `path` rejected; package without `stack` rejected; a one-word record edit makes the render differ; a README claiming the wrong output fails; a `uses` id with no record is flagged; `sandbox-exec` provably denies (a loopback listener works plain, fails sandboxed) |
| `platform/` | exactly 6 new `README.md` files; no modified file under `platform/go` |
| Kit-edge check | exit 0, unchanged (4 declared, 1 known violation) |
| Client-name scan of every changed file | no hits |

## Next

Scope U2c: `identity/rbac`, `localization/language`, `news/rss`, `persistence/database`, `media/image` (level 0, third-party requirements).
The `requirement` type is loosened once, at the moment the first record needs it.

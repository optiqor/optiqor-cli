# optiqor-cli — Claude Conventions

The open-source CLI: `@optiqor/cli`. Apache-2.0. Standalone repo so it stays independently auditable, which is the entire reason it doesn't live in the proprietary monorepo. Strategy reference: `docs/open_source_cli_playbook.md` in the Optiqor org docs (not public).

This file is the operating manual. Read it before writing or reviewing code.

---

## Stack

- Go 1.24, single module: `github.com/optiqor/optiqor-cli`
- Cobra for command parsing, charmbracelet/lipgloss for terminal rendering
- Pure stdlib for the rest. No HTTP framework, no logger library, no DI framework.
- npm wrapper (`@optiqor/cli`) downloads the platform-specific Go binary at install time
- GoReleaser for cross-platform builds (linux + darwin, amd64 + arm64)
- Cosign for release artifact signing
- SHA-256 verified on download (in progress, see open issues)

If you reach for a new dependency, justify it in the PR. Adding a transitive dep widens the audit surface of an Apache-2.0 binary that runs on customer laptops.

---

## Hard rules (non-negotiable)

These exist because the OSS funnel breaks if any of them slip. They are not preferences.

1. **No LLM calls.** The CLI is a deterministic rule engine. The Sonnet/Opus/Haiku-driven Apply Fix flow lives in the proprietary backend. If you find yourself wanting to call an LLM from here, the answer is "POST to the SaaS sandbox endpoint instead and let it return the LLM-enriched response."

2. **No telemetry by default.** Zero-config install must not phone home. The only opt-in network egress is `--share`, which POSTs a sanitised analysis to `optiqor.dev/r/<hash>`. Every new network call needs an explicit user opt-in and must consult `offlineMode()` in `cmd/optiqor/main.go`.

3. **Accuracy disclosure in every output.** Every renderer (text, JSON, HTML, roast) emits the mandatory disclosure: `Sandbox accuracy: ±40%. Install the Optiqor agent for exact numbers (optiqor.dev/get).` The string is canonical. The honesty is the whole pitch; don't water it down.

4. **No proprietary backend code may be imported.** `go.mod` must never reference `github.com/optiqor/optiqor`. The CLI is independently buildable, auditable, licensable. CI greps imports on every PR.

5. **`pkg/` is the stable public API.** External programs may import it, including the proprietary backend (which depends on `pkg/rules` and `pkg/parser`). Breaking changes go through semver and a deprecation notice. New detectors land in `pkg/rules` first; the backend follows automatically via `go get -u`.

6. **`internal/` is private.** Refactor freely. `internal/{analyze,render,share,config,roast}` is CLI-side composition that stays out of the public API.

---

## Distribution

- Primary: `npm install -g @optiqor/cli` or `npx @optiqor/cli analyze ./chart`. The npm `postinstall` script downloads the right binary from GitHub Releases.
- Secondary: `go install github.com/optiqor/optiqor-cli/cmd/optiqor@latest`.
- Releases are GoReleaser-built (`-trimpath`, version ldflag from `git describe`) and Cosign-signed.
- We do **not** publish to Homebrew or Cargo in Year 1. npm is where the platform engineers are.
- Windows is out of scope. `.goreleaser.yaml` and `package.json` both reflect this. Adding Windows handling needs the OSS-scope exclusion in the playbook removed first.

---

## Commands

12 commands phased over 12 months. Year 1 ships the deterministic core.

| Command | Status | Purpose |
| --- | --- | --- |
| `analyze <chart>` | shipped | Primary entry point. Reads a Helm chart dir or `values.yaml`, prints findings. |
| `demo` | shipped | Runs analysis on a bundled chart so `npx @optiqor/cli demo` shows output without input. |
| `diff <a> <b>` | shipped | Cost delta between two values files. |
| `score <chart>` | shipped | 0-100 efficiency score (qualitative band Year 1, numeric Year 2). |
| `audit <chart>` | shipped | Security-focused subset of `analyze`. Stricter `--fail-on` defaults. |
| `watch <chart>` | stub | Re-runs `analyze` on file change. Phase 2+. |
| `compare <a> <b>` | stub | Side-by-side renderer. Phase 2+. |
| `--roast` (flag) | shipped | Humorous tone on `analyze`. Findings stay accurate. |

Stubs return "not yet implemented" and exit 1. The surface is registered so the command name is locked in.

---

## Output

The CLI's output is the product. Treat it with the same care as code.

### Stream discipline

- `stdout` is data: the report, JSON, or HTML when piped.
- `stderr` is everything else: status, warnings, errors, share URLs, the accuracy disclosure when output is redirected.
- Mixing the two breaks pipelines. `optiqor analyze ./chart | jq` must always work without `2>/dev/null` tricks.

### Color

- Respect `NO_COLOR` (de-facto standard, https://no-color.org).
- Respect `--no-color`.
- Auto-detect via `isatty(stderr)`. Don't auto-color when piped.

### Exit codes

- `0` success, no findings at or above `--fail-on`
- `1` runtime error (parse failure, file not found, invalid flag)
- `2` findings at or above `--fail-on` severity

Reserved: `3-127` for future use. Don't repurpose Unix-standard codes (130 SIGINT, 143 SIGTERM).

### Error messages

Errors reach the user. Each one must include:

1. What happened, in user terms.
2. What to do next, specific.

**Bad:**

```
Error: invalid input
```

**Good:**

```
error: --severity must be one of low, med, high (got: "medium")
```

For multi-line context, use a newline and indent the actionable next step.

### Help text

Every command and flag has help text. It's product surface: what users read before they try anything.

- One-line summary on the command line, one paragraph in the long description.
- The long description names the input shape, the output shape, and at least one runnable example.
- Examples use `./my-chart`, not `path/to/chart`.

**Bad flag description:**

```
"output format"
```

**Good:**

```
"emit machine-readable JSON instead of the human table"
```

---

## Determinism

The CLI is deterministic. Same input, same output, byte-identical.

- No timestamps in output unless explicitly requested.
- No random map iteration. Sort before serialising.
- No reads of `$HOME` or `$USER` in the analyze path. Config file path is a separate concern.
- Time and randomness injected. Never `time.Now()` directly in any package with a state machine.
- Goroutine concurrency only for parallel rule execution, with results merged in deterministic order.

Golden tests in `testdata/golden/` assert byte-identical output across runs. A flake here is a determinism bug, not a test bug. Regenerate with `-update`; eyeball the diff before committing.

---

## Testing

- Unit tests beside the code (`foo.go` + `foo_test.go`).
- Golden tests in `testdata/golden/` for every renderer mode.
- One test command: `go test ./...`. No separate integration suite. The CLI is pure functions over Helm input.
- Race detector in CI.
- Every detector in `pkg/rules` needs at least one positive and one negative test.
- Every renderer mode has a golden fixture.

`pkg/` (the public surface) has stricter test discipline. Coverage target: 90%. External tests (`pkg/foo_test.go` in package `foo_test`) for any exported symbol with a non-trivial contract.

---

## Code style

### Naming

- Functions are verbs (`ParseValues`, `RenderReport`).
- Predicates: `isLeader`, `hasQuorum`, not `leaderFlag`.
- Acronyms keep case: `httpClient`, `parseURL`, `IDToken`. Not `HttpClient`.
- Cobra command vars: `analyzeCmd`. Constructors: `newAnalyzeCmd()`.

### Functions

- One thing per function. If the name has "and", split.
- Early return for guard clauses. No `else` after `return`.
- Receiver names: one or two letters, consistent across all methods of a type.

### Errors

- Wrap with `%w`, never `%s`, when the caller might `errors.Is` / `errors.As`.
- Sentinel errors exported only when callers branch on them.
- Never `panic` for control flow. Reserved for "this cannot happen".
- Distinguish bad input (user error, exit 1, friendly message) from bug (program error, exit 1, full context).

### Comments

Comments explain why, never what. Code says what; you say why it has to.

**Bad (states the obvious, AI-shaped):**

```go
// Iterates through detectors and runs each one.
for _, d := range detectors {
    d.Run(w)
}
```

**Good (explains the why):**

```go
// Sequential on purpose. Detectors share a finding map that's faster
// to merge than to lock per-write. ADR-0003.
for _, d := range detectors {
    d.Run(w)
}
```

**Bad (AI godoc):**

```go
// RunDetectors runs all detectors on the workloads.
// Returns a slice of findings.
func RunDetectors(ws []Workload, ds []Detector) []Finding
```

**Good (godoc that earns its keep):**

```go
// RunDetectors evaluates ds against every workload in ws and returns
// findings sorted by (severity desc, detector id asc) for stable
// golden tests. Detectors are evaluated in registration order; the
// public rules.All() returns them in canonical order for reproducible
// output.
func RunDetectors(ws []Workload, ds []Detector) []Finding
```

Reference real things: issue numbers, commit SHAs, ADR numbers, RFC sections, vendor docs.

```go
// Helm v3 sets release.Namespace at render time, not template time.
// We default to "default" so values.yaml that uses .Release.Namespace
// in a template doesn't crash. See helm/helm#7702.
```

**Banned in comments:**

- Markdown headers (`#`, `##`, `###`).
- `Note:`, `Important:`, `Caution:`, `Warning:` labels.
- Emojis. None, anywhere.
- Restating what the code says.
- "Helper function to...", "Utility for...". The name should describe it.
- Author tags. Git blame exists.
- TODOs without a name or issue link.

Godoc on every exported symbol. The first sentence starts with the symbol name and is one line. Examples for non-trivial APIs go in `example_test.go`.

### Files and packages

- Lowercase file names, no underscores except `_test.go`.
- `doc.go` per package, package comment lives there.
- Package names: short, single-word, no plurals, no `util`, no `helpers`.

---

## Brand and tone

The CLI ships in front of users. The text is the product.

- `brand/tokens.json` is the single source of truth for colors, font tokens, glyph references. Imported by `pkg/htmlrender` (Go) and by the backend's `web/src/lib/brand.ts` (TS). Drift here breaks visual parity.
- Tone is editorial × engineering. Direct, sharp, opinionated, never glib. No exclamation marks in default output. `--roast` exists for the snark.
- Numbers are accurate or absent. We never round up. `±40%` is canonical.
- No emojis in default output. `--roast` may use a small whitelist (in `internal/roast`).

---

## Security

CLI binaries run on customer laptops. The supply chain is a primary attack surface.

- Releases are GoReleaser-built with `-trimpath` and `-buildvcs=true`. Build determinism is asserted by the release job.
- Cosign signs the archive and `checksums.txt`. The npm postinstall verifies the SHA-256 of the downloaded archive against `checksums.txt` before extracting.
- The CLI never opens files outside the supplied chart path without an explicit flag. Path traversal in `--config`, `--html`, `--output` is a hard rejection, not a warning.
- YAML deserialization uses `gopkg.in/yaml.v3` with anchor depth bounded. Billion-laughs is rejected at parse time.
- npm `postinstall` writes only to the package's own `vendor/` directory. Any path that resolves above `node_modules/@optiqor/cli/` is a P0.
- No `exec.Command` in the binary. The CLI is a pure-Go process. No shelling out, no `kubectl` spawning, no helm-CLI dependency.

---

## Workflow

### Branches

- `<type>/<short-kebab-slug>`. Same prefixes as commits: `feat/`, `fix/`, `chore/`, `docs/`, `test/`, `refactor/`.
- Branch off `main`. Never branch off a feature branch.
- Delete merged branches.

### Commits

Conventional Commits. One concern per commit. DCO sign-off required (`-s` flag, enforced by CI).

See [.claude/skills/commit/SKILL.md](.claude/skills/commit/SKILL.md) for the rules and the local quality gate (gofmt + vet + build + test -race + lint).

### Pull requests

- Open against `main`. Branch protection ruleset is active. Admin bypass only for emergencies.
- Title is a Conventional Commits subject; it becomes the squash-merge subject.
- Body tells the reviewer the story: what changed, why, how to verify.
- Link the issue: `Closes #N`.
- DCO sign-off enforced by CI (`.github/workflows/commit-lint.yml`).
- PR size labels auto-apply (`size/XS` through `size/XL`). XL is a smell. Split.
- markdown-link-check runs in CI. Cross-repo relative links (`../`) won't resolve in standalone checkout; use absolute URLs or remove the link.

### Reviews

See [.claude/skills/pr-review/SKILL.md](.claude/skills/pr-review/SKILL.md) for voice, line-anchoring, and verdict rules.

### Issues

See [.claude/skills/open-issue/SKILL.md](.claude/skills/open-issue/SKILL.md) for the audit checklist, classification table, and issue-body shapes.

### Decisions

ADRs in `docs/adr/` for non-trivial architectural changes. Template at `docs/adr/0000-template.md`. Examples:

- A new exported package under `pkg/`
- A change to the deterministic-output contract
- A new dependency that adds network or filesystem reach
- A change to brand tokens that affects existing renderers
- A change to the public `rules.Finding` or `rules.Workload` shapes (downstream backend depends on these)

---

## Anti-patterns ("don't")

### Code

- Don't import any GitHub / GitLab API client. The CLI doesn't talk to source control.
- Don't add a daemon, watcher, or persistent process. CLI invocations are short-lived.
- Don't add config files unless absolutely necessary. Flags + env vars only. `OPTIQOR_*` env vars are documented in README.
- Don't add platform-specific code paths. The CLI runs identically on linux and macOS.
- Don't use `init()` except to register Cobra commands.
- Don't write a singleton. Pass dependencies through constructors.
- Don't depend on `init()` ordering.
- Don't return `interface{}` / `any` unless you've considered generics.
- Don't shell out (`exec.Command`). Pure-Go binary.
- Don't read `$HOME` or `$USER` in the analyze path. Config-file lookup is separate and documented.

### Output

- Don't print to stdout from anywhere except the renderer.
- Don't auto-color when piped.
- Don't restate the input in the output ("Analyzing ./chart...") unless `--verbose`.
- Don't print stack traces to users. They go to stderr only if `DEBUG=1`.

### Process

- Don't import any code from `github.com/optiqor/optiqor`. The CLI's audit story is "Apache-2.0 and you can read every line."
- Don't commit binary artifacts. Generated files go in `.gitignore`.
- Don't TODO without an issue. Open the issue, paste the link, then write the TODO.
- Don't merge to `main` without a PR.
- Don't bump dependencies without a one-line justification in the commit message.
- Don't break `pkg/` without a semver bump and a deprecation notice. The backend imports it directly via `go get -u`.

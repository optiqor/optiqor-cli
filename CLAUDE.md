# cli — Claude Conventions

This is the **open-source** Optiqor CLI (`@optiqor/cli`). It must remain independently auditable as Apache-2.0 OSS — that is the entire reason it lives in a separate repo from the proprietary backend. Strategy reference: `docs/open_source_cli_playbook.md` in the Optiqor org monorepo (not public).

## Stack

- Go 1.23+, single module (`github.com/optiqor/optiqor-cli`)
- Cobra for command parsing
- npm wrapper (`@optiqor/cli`) downloads the platform-specific Go binary on `npm install`
- GoReleaser for cross-platform builds (linux/macos amd64/arm64)

## Hard rules

These are not preferences. They are conditions for the OSS funnel to work.

- **No LLM calls.** The CLI is a deterministic rule engine. The Sonnet/Opus/Haiku-driven Apply Fix flow lives in the backend, not here. If you find yourself wanting to call an LLM from the CLI, the answer is "send to the SaaS backend's sandbox endpoint instead."
- **No telemetry by default.** Zero-config install must not phone home. An opt-in `--share` flag uploads a sanitized analysis to `optiqor.dev/r/<hash>` for sharing — that is the only network egress.
- **Accuracy disclosure is mandatory in every output.** Every analysis result includes "Sandbox accuracy: ±40%. Install the Optiqor agent for exact numbers (optiqor.dev/get)." Do not remove this. Do not make it dismissible by default. The honesty is the whole pitch.
- **No proprietary backend code may be imported here.** This repo's `go.mod` must never reference `github.com/optiqor/optiqor`. The CLI is independently buildable, independently auditable, independently licensable.
- **`pkg/` is the stable public surface.** External programs may import it. Breaking changes go through semver and a deprecation notice. The Optiqor proprietary backend imports `pkg/rules` (the 30-detector library) and `pkg/parser` (Helm values normaliser) directly — this is *the* mechanism by which the SaaS reuses CLI rule definitions instead of forking them. **New detectors land in `pkg/rules` first; the backend follows automatically via vendored module + golden parity tests.**
- **`internal/` is private.** Refactor freely. Anything in `internal/` (analyze, render, share, config, render/style) is CLI-side composition that should stay out of the public API surface.

## Distribution

- Primary: `npm install -g @optiqor/cli` or `npx @optiqor/cli analyze ...`. The npm `postinstall` script downloads the right binary from GitHub Releases.
- Secondary: `go install github.com/optiqor/optiqor-cli/cmd/optiqor@latest`
- Releases are GoReleaser-built and cosign-signed.
- We do **not** publish to Homebrew or Cargo in Year 1 (per playbook — npm is where the platform engineers are).

## Command surface

12 commands phased over 12 months (per playbook). Year 1 ships:

- `analyze <chart>` — primary command. Reads a Helm chart dir or `values.yaml`, prints findings.
- `demo` — runs analysis on a bundled demo chart so people can `npx @optiqor/cli demo` and see output without supplying input.
- `diff <a> <b>` — show cost delta between two values files.
- `score <chart>` — assign a 0–100 efficiency score (qualitative band Year 1, numeric Year 2).
- `audit`, `watch`, `compare` — registered as Cobra subcommands now (returning "not yet implemented") so the surface is locked in.

## Output

- Default: ASCII table + summary, designed for terminal-first reading.
- `--json` for machine-readable output.
- `--roast` (Year 1+) for humorous tone — viral, but the underlying findings stay accurate.
- `--offline` / `--private` flags must work — never require network egress for `analyze` to function.

## Testing

- Table-driven by default. One row per scenario, `t.Run(tc.name, ...)` so failures name the row.
- Naming: `TestFunctionName_Scenario_ExpectedBehavior`. Read aloud, it forms a sentence.
- Stdlib only — no testify, no gomock. `errors.Is` / `errors.As` for error checks; `bytes.Equal` for bytes; golden files for multi-line / whitespace-sensitive output.
- Race detector always: `go test -race -count=1 ./...` is the CI bar.
- Inject time + randomness — `time.Date(...)`, `bytes.NewReader`, never `time.Now()` / `rand.Intn` in code under test.
- Helpers call `t.Helper()` on the first line.
- Golden tests in `testdata/golden/` for every renderer output. The mandatory ±40% accuracy disclosure is asserted byte-identical across modes. Regenerate with `UPDATE_GOLDEN=1 go test ./...` and eyeball the diff before committing.
- `go test ./...` is the only test command. No separate integration suite — the CLI is pure functions over Helm input.

See `.claude/skills/test/SKILL.md` for the full procedure: templates, anti-patterns, coverage targets, and references to clean examples in the repo.

## Comments

Comments explain WHY, never what. Decoration is debt; every comment is a thing a future engineer must keep in sync with the code. When in doubt, delete. Re-add only when a reader would miss a non-obvious constraint, trade-off, ADR reference, security/performance rationale, or invariant.

The CLI is **Apache-2.0 OSS**, so `pkg/` is the public surface that external Go programs import. Lean slightly more conservative there: one-line godoc on every exported symbol is appropriate when it documents a contract beyond the name (`// Run executes all detectors against the workloads and returns findings in detector-declaration order` is a real contract; `// Run runs the detectors.` is not).

**Banned:**

- Markdown headers (`#`, `##`, `###`) inside `//` comments.
- `Note:`, `Important:`, `Caution:`, `Warning:` labels. If it's important, the code structure should make the rule unmissable.
- Em-dash-heavy narrative essays in package or function docstrings. One terse sentence beats five lines of glue.
- Decorative section dividers: `// ─── Helpers ───`, `// ====== validators ======`. Use blank lines.
- Multi-paragraph package docstrings narrating "Layout philosophy:", "Implementation notes:", "Design constraints:". Compress to one or two terse sentences.
- Bullet-listed enumerations of struct fields or function returns — the type tags already enumerate them.
- Restating the signature (`// Foo returns a Foo.`) or the next line of code.
- "Helper function to...", "Utility for...", "This function..." preambles.
- Emojis. None, anywhere.
- TODOs without a name and issue: `// TODO(@shivam, #42): ...` is fine; `// TODO: do later` is not.

**Always keep:**

- The CLI's hard-rule pins: no LLM in CLI, no telemetry by default, ±40% accuracy disclosure is mandatory in every renderer output. Comments that pin these stay verbatim.
- Detector references to CIS / NSA hardening guides — those anchors are load-bearing for security findings.
- Cross-file references to other packages, ADRs, or upstream specs.

**How to audit a comment**: ask *"if I delete this, what does the next reader fail to understand?"* If the answer is "nothing — the name + signature + types already say it", delete. If the answer names a specific constraint, trade-off, or hard-rule pin, keep but compress.

**Reference commits**: the 2026-05-24 cleanup (`feba8e0` openapi header, `9f817cd` cli sweep) stripped ~400 lines of comment cruft from this repo. Read those diffs for the tone applied at scale; new code should land at that compression level from the start.

## Don't

- Don't import any GitHub / GitLab API client. The CLI does not talk to source control.
- Don't add a daemon / persistent process. CLI invocations are short-lived.
- Don't add config files unless absolutely necessary. Flags + env vars only.
- Don't add platform-specific code paths. The CLI runs identically on linux and macOS. (Windows: explicitly out of scope per playbook — npm is the distribution.)

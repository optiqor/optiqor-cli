# optiqor-cli — repo-local todo

This file tracks CLI-only work. The org-level roadmap that wires both
repos and the strategy docs lives in the Optiqor org monorepo (not
public); items here are scoped to what lands inside this repo's
`cmd/`, `internal/`, or `pkg/`.

## Recently shipped

- [x] **2026-05-03 — Boxed cost-finding cards + signal bars** (playbook §4 Rule 1). `pkg/rules.Signal` carries quantitative evidence; renderer draws `█████░░░░` ratio bars inside per-finding cards. Graceful flat-layout fallback under 50 cols. Bug-fixed `visibleRuneCount` to iterate runes (was bytes) so multi-byte glyph alignment holds.
- [x] **2026-05-03 — `--roast` flag** (`internal/roast`). Static map of detector ID → snarky title for all 30 detectors. Hard rules preserved: no LLM, only `Title` mutated, accuracy disclosure exact.
- [x] **2026-05-03 — `score` letter grade + percentile** (`internal/analyze/grade.go`). Baked-in 100-sample calibration distribution; binary-search percentile lookup. `Grade` lands in JSON output too.
- [x] **2026-05-03 — `Category` first-class on Detector + Finding**. Drops the hardcoded `SecurityDetectorIDs` map; categorization is type-safe and audited in one place (`pkg/rules/categories.go`).
- [x] **2026-05-03 — Cost-first redesign of analyze output**. Branded header, executive summary, cost section sorted by USD savings descending, security section as a compact bonus block.
- [x] **2026-05-03 — Rebrand sevro → optiqor** across repo (151 files, including module path, package name, GitHub remote, tagline, README).

## Tier 1 — Launch anchors still open

- [ ] **Real `--share` upload endpoint** — CLI side already computes hash + posts to `sandbox.optiqor.dev/api/v1/share`; blocked on backend Phase 2 receiver. Don't over-build CLI side until the endpoint returns 2xx.
- [ ] **`compare` as side-by-side, not a `diff` alias** — playbook Feature 7 ("bitnami/postgresql vs cloudnative-pg") is press-bait. Needs a 2-column renderer + winner declaration.
- [ ] **Populate `Signal` on the remaining cost detectors** that have ratios but don't yet emit one: `oversized-cpu-limit`, `oversized-memory-limit`, `excessive-replica-count`, `tiny-cpu-request`, `tiny-memory-request`, `cpu-without-memory-request`, `memory-without-cpu-request`.

### Web-rendering surface (shared with backend share pages — `optiqor-cli/pkg/htmlrender`)

> The CLI is terminal-first, but two narrow web touch-points cross repos. Both ship as Apache-2.0 Go code from this repo so the backend can `import` them without contaminating the OSS audit story.

- [ ] **`pkg/htmlrender`** — Go `html/template` renderer that takes a `render.Report` and emits a single self-contained HTML document. Inline CSS, zero JS framework, zero external assets — openable from `file://`. Same package the backend uses to serve `optiqor.dev/r/<hash>` so the CLI's local HTML and the backend's share page are byte-equivalent.
- [ ] **`analyze --html <path>`** flag — wires `pkg/htmlrender` to a CLI flag. `optiqor analyze ./chart --html report.html` writes a shareable file the user can email or commit. Accuracy disclosure mandatory, same as text + JSON paths.
- [ ] **`brand/tokens.json`** — single source of truth for the brand palette (colors, font stack, logo file references). Apache-2.0. Consumed by `pkg/htmlrender` (Go) and by the backend's `web/lib/brand` (TypeScript) so terminal + HTML report + dashboard never drift on the hero color.
- [ ] **`docs/api/openapi.yaml`** — public API spec for `/v1/analyze`, `/r/{hash}`, `/v1/receipts/{id}`. Apache-2.0 so community tooling + the backend's TS client can be generated from one file. Drift between the spec and the backend handler is asserted by a CI check in the backend repo.

## Tier 2 — Distribution multipliers

- [ ] **`optiqor/actions` GitHub Action** wrapper (separate repo per playbook). Wraps `analyze --json`, posts a sticky PR comment.
- [ ] **Shell completions** — Cobra emits bash/zsh/fish for free; ship via goreleaser into the npm tarball.
- [ ] **Man page** — Cobra → `man optiqor` from the same registry.
- [ ] **`docs-site/`** — Astro + MDX static site for `docs.optiqor.dev`. Zero-JS by default (Lighthouse 100), React islands when interactivity is needed, output deploys to S3 + CloudFront. MDX source covers CLI commands, the `pkg/rules` detector catalogue, and the public OpenAPI reference. Stays Apache-2.0 so external contributors can edit docs without a CLA dance.

## Tier 3 — Trust / enterprise gates

- [ ] **SBOM in releases** — `.goreleaser.yaml` `sbom:` stanza. SOC 2 / vendor-review gating.
- [ ] **cosign keyless OIDC provenance** — config exists but isn't wired to GitHub OIDC.
- [ ] **`--version --verbose`** — include commit, build date, Go version, target. Trivial Cobra wiring; matters for bug reports.
- [ ] **Per-detector golden fixtures** — `testdata/golden/` covers commands, not detectors. With 30 detectors and `pkg/rules` being public API the backend imports, drift will be silent without per-detector goldens.

## Hard rules — never violate

These are conditions for the OSS funnel to work. See [CLAUDE.md](CLAUDE.md) for the canonical list.

- **No LLM calls.** The CLI is a deterministic rule engine.
- **No telemetry by default.** Only `--share` egresses (opt-in).
- **Accuracy disclosure mandatory in every output.** Verbatim string; renderers must include it.
- **No proprietary backend code imported.** `go.mod` must never reference `github.com/optiqor/optiqor`.
- **`pkg/` is the stable public API.** Breaking changes go through semver and a deprecation notice.

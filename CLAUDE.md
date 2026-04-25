# cli — Claude Conventions

This is the **open-source** Costify CLI (`@costify/cost`). It must remain independently auditable as Apache-2.0 OSS — that is the entire reason it lives in a separate repo from the proprietary backend. Strategy reference: [../docs/open_source_cli_playbook.md](../docs/open_source_cli_playbook.md).

## Stack

- Go 1.23+, single module (`github.com/lowplane/cli`)
- Cobra for command parsing
- npm wrapper (`@costify/cost`) downloads the platform-specific Go binary on `npm install`
- GoReleaser for cross-platform builds (linux/macos amd64/arm64)

## Hard rules

These are not preferences. They are conditions for the OSS funnel to work.

- **No LLM calls.** The CLI is a deterministic rule engine. The Sonnet/Opus/Haiku-driven Apply Fix flow lives in the backend, not here. If you find yourself wanting to call an LLM from the CLI, the answer is "send to the SaaS backend's sandbox endpoint instead."
- **No telemetry by default.** Zero-config install must not phone home. An opt-in `--share` flag uploads a sanitized analysis to `costify.dev/r/<hash>` for sharing — that is the only network egress.
- **Accuracy disclosure is mandatory in every output.** Every analysis result includes "Sandbox accuracy: ±40%. Install the Costify agent for exact numbers (costify.dev/get)." Do not remove this. Do not make it dismissible by default. The honesty is the whole pitch.
- **No proprietary backend code may be imported here.** This repo's `go.mod` must never reference `github.com/lowplane/backend`. The CLI is independently buildable, independently auditable, independently licensable.
- **`pkg/` is the stable public surface.** External programs may import it. Breaking changes go through semver and a deprecation notice.
- **`internal/` is private.** Refactor freely.

## Distribution

- Primary: `npm install -g @costify/cost` or `npx @costify/cost analyze ...`. The npm `postinstall` script downloads the right binary from GitHub Releases.
- Secondary: `go install github.com/lowplane/cli/cmd/costify@latest`
- Releases are GoReleaser-built and cosign-signed.
- We do **not** publish to Homebrew or Cargo in Year 1 (per playbook — npm is where the platform engineers are).

## Command surface

12 commands phased over 12 months (per playbook). Year 1 ships:

- `analyze <chart>` — primary command. Reads a Helm chart dir or `values.yaml`, prints findings.
- `demo` — runs analysis on a bundled demo chart so people can `npx @costify/cost demo` and see output without supplying input.
- `diff <a> <b>` — show cost delta between two values files.
- `score <chart>` — assign a 0–100 efficiency score (qualitative band Year 1, numeric Year 2).
- `audit`, `watch`, `compare` — registered as Cobra subcommands now (returning "not yet implemented") so the surface is locked in.

## Output

- Default: ASCII table + summary, designed for terminal-first reading.
- `--json` for machine-readable output.
- `--roast` (Year 1+) for humorous tone — viral, but the underlying findings stay accurate.
- `--offline` / `--private` flags must work — never require network egress for `analyze` to function.

## Testing

- Golden tests in `testdata/fixtures/` for each detector and each renderer.
- `go test ./...` is the only test command. No separate integration suite — the CLI is pure functions over Helm input.

## Don't

- Don't import any GitHub / GitLab API client. The CLI does not talk to source control.
- Don't add a daemon / persistent process. CLI invocations are short-lived.
- Don't add config files unless absolutely necessary. Flags + env vars only.
- Don't add platform-specific code paths. The CLI runs identically on linux and macOS. (Windows: explicitly out of scope per playbook — npm is the distribution.)

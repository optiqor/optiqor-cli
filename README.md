# Costify CLI

> Cost & security analysis for Kubernetes Helm charts, from your terminal. No login. No agent. No cluster connection required.

[![npm version](https://img.shields.io/npm/v/@costify/cost.svg)](https://www.npmjs.com/package/@costify/cost)
[![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

```sh
npx @costify/cost analyze ./my-helm-chart
```

## What it does

Reads a Helm chart (or just a `values.yaml`) and reports cost inefficiencies and security findings, with a shareable URL and a one-line install command for the full Costify agent if you want exact numbers.

```
$ npx @costify/cost demo

Costify Sandbox Analysis  ────────────────────  costify.dev
chart: bitnami/postgresql (demo)

Findings (3)
  [HIGH]  Overprovisioned CPU request    save ~$340/mo  confidence: medium
  [MED]   Missing memory limit            risk: OOM noisy-neighbor
  [LOW]   Image pinned to :latest         risk: unreproducible deploy

Estimated monthly cost     $1,140
Estimated monthly savings  $340 (30%)

Sandbox accuracy: ±40%. Install the Costify agent for exact numbers:
  costify.dev/get

Share this analysis: costify.dev/r/9f3a1c
```

## Install

### npx (recommended)

```sh
npx @costify/cost analyze ./chart
```

### Global

```sh
npm install -g @costify/cost
costify analyze ./chart
```

### From source

```sh
go install github.com/costify/cli/cmd/costify@latest
```

## Commands

| Command | What it does |
| --- | --- |
| `analyze <chart>` | Run cost + security analysis on a Helm chart or values file |
| `demo` | Run analysis on a bundled demo chart |
| `diff <a> <b>` | Show cost delta between two values files |
| `score <chart>` | Assign a 0–100 efficiency score |

More commands (`audit`, `watch`, `compare`) ship over the next 12 months.

## Honesty about accuracy

The CLI gives you **directional** signal (±40%) from Helm files alone. For exact numbers — backed by 30 days of real Prometheus data plus your AWS bill — install the Costify agent in your cluster. We deliberately keep the CLI honest about what it can and can't tell you from static files.

## Privacy

- **No telemetry.** The CLI does not phone home.
- **`--share` is opt-in.** Only when you explicitly pass `--share` do we upload a sanitized analysis to `costify.dev/r/<hash>` for sharing.
- **`--offline` works.** `analyze` runs entirely locally by default.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All contributions land under Apache-2.0; we use DCO sign-off.

## License

Apache 2.0. See [LICENSE](LICENSE).

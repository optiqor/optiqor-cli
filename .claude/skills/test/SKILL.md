---
name: test
description: Write rigorous unit + golden tests for the optiqor CLI. Table-driven by default; named TestFunctionName_Scenario_ExpectedBehavior; stdlib only (no testify); race detector always; time + randomness injected; t.Helper() in helpers; golden snapshots for every renderer / CLI output; YAML fixtures live in testdata/fixtures/. Use when the user says "test this", "add tests for X", "write a test", "cover this", "TDD this", or similar.
---

# test

Write tests a senior Go engineer would ship. Strict table-driven layout, stdlib comparators, race-clean, time-injected. The CLI's hard rule: every renderer output is pinned by a golden file.

## When to invoke

- "test this" / "add tests for X" / "cover X with tests"
- "write a test" / "TDD this"
- "the test for X is failing — fix it" (read, find root cause, write the minimal repro test alongside the fix)
- "update the golden snapshot" (only if a deliberate output change — see §Golden updates)

## Hard rules

### 1. Table-driven is the default

Any test with more than one case uses an inline table, one row per scenario, each row run inside `t.Run(tc.name, ...)`.

```go
for _, tc := range []struct {
    name string
    // …
}{
    {name: "happy path", /* … */},
    {name: "empty input", /* … */},
} {
    t.Run(tc.name, func(t *testing.T) { /* … */ })
}
```

No package-level fixtures. No shared mutable state across rows.

**No `_TableDriven` suffix in the test function name.** TDT is the default; the suffix is noise. `TestRender_Demo` consolidating multiple render scenarios into a table is named `TestRender_Demo`, not `TestRender_Demo_TableDriven`. The subtest names (`tc.name`) carry the scenario.

### 2. Naming

`TestFunctionName_Scenario_ExpectedBehavior`. Read aloud, it forms a sentence.

| Good | Bad |
|---|---|
| `TestParseValues_EmptyFile_ReturnsErrEmpty` | `TestParse` |
| `TestRender_DemoPlain_GoldenStable` | `TestRender` |
| `TestRoast_AllDetectors_HaveRewrite` | `TestRoastWorks` |
| `TestShare_PIIStrip_RemovesSourcePaths` | `TestShareUpload` |

### 3. Stdlib only — no testify, no gomock, no third-party assertion libs

CLAUDE.md hard rule: "Prefer stdlib." Comparators:

- `==` for primitives, strings, numbers.
- `reflect.DeepEqual` for structs/maps/slices.
- `errors.Is` / `errors.As` for error checks.
- `bytes.Equal` for byte slices.
- Golden files for anything multi-line or whitespace-sensitive.

### 4. Race detector always

`go test -race ./...` is the project's CI bar. Any test that wouldn't pass `-race` is broken; fix it before shipping.

### 5. Time + randomness injected

Never call `time.Now()`, `rand.Intn`, or `crypto/rand.Read` directly from code under test. Accept a `Clock`, `*rand.Rand`, or `io.Reader`. In tests, pin them:

```go
now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
clock := func() time.Time { return now }
```

The CLI's `share` package and any future scheduler use this pattern. A flaky clock = a flaky CI.

### 6. Helpers call `t.Helper()`

Any function taking `*testing.T` that isn't itself a `TestXxx` calls `t.Helper()` on its first line. Otherwise `t.Fatalf` reports the helper line, not the caller's.

### 7. Subtests are `t.Run`

Use `t.Run(tc.name, …)` so failures name the row (`TestParse/empty_file: …`), not the loop line.

### 8. Golden tests for every renderer output

The CLI's whole point is deterministic output that a customer can paste into a PR. Every renderer (text, JSON, HTML, roast, score) has a golden snapshot in `testdata/golden/`. The mandatory ±40% accuracy disclosure is asserted in every renderer test — that's the hard rule per CLAUDE.md, and the disclosure string must stay byte-identical across renderers.

Regenerate goldens with the project's update flag (see §Golden updates). Eyeball every diff.

### 9. Pure functions only (no LLM calls in tests)

CLAUDE.md hard rule: "No LLM calls." No telemetry, no HTTP egress (except the opt-in `--share` upload client, which has its own fake `Upload` interface). Tests must be hermetic — runnable on a plane, in an airgap, without internet.

### 10. One concern per test

A test asserts one thing. Three behaviours = three named subtests or three `TestXxx` functions. A failure must name the broken contract.

---

## Decision: unit vs golden vs CLI invocation

| Type | Where | Coverage | Example |
|---|---|---|---|
| **Unit** | `_test.go` next to the code, same package | Pure logic: parsers, rule engines, score math, share-URL hashing | `pkg/rules/rules_test.go` |
| **Golden** | `testdata/golden/*.txt` asserted via `_test.go` | Renderer output byte-stability | `internal/render/golden_test.go` |
| **CLI invocation** | `cmd/optiqor/golden_test.go` | Whole-binary invocation through Cobra; asserts stdout/exit-code against golden | `cmd/optiqor/golden_test.go` already exists with this pattern |

There is no integration-test boundary in the CLI — the CLI does not talk to a database. The closest thing is the `--share` upload, which is tested against an `httptest.Server` fake.

---

## Templates (copy-paste, edit)

### Template A — table-driven detector test

```go
func TestCPUOverprovisioned(t *testing.T) {
    for _, tc := range []struct {
        name string
        in   parser.Workload
        want []rules.Finding
    }{
        {
            name: "request half the limit triggers",
            in: parser.Workload{
                Name: "api",
                CPURequest: q("1"), CPULimit: q("4"),
            },
            want: []rules.Finding{{
                DetectorID: "cpu-overprovisioned",
                Workload:   "api",
                Severity:   rules.SeverityMed,
            }},
        },
        {
            name: "request equals limit does not trigger",
            in: parser.Workload{
                Name: "api",
                CPURequest: q("2"), CPULimit: q("2"),
            },
            want: nil,
        },
    } {
        t.Run(tc.name, func(t *testing.T) {
            got := CPUOverprovisioned().Run([]parser.Workload{tc.in})
            if !findingsMatch(got, tc.want) {
                t.Errorf("got %+v, want %+v", got, tc.want)
            }
        })
    }
}
```

Notes:
- Inline struct literal. `name` first. No shared `var cases = …`.
- Helper `q("2")` parses a Quantity — keep helpers small, `t.Helper()` inside.
- Comparator `findingsMatch` does field-equality on the ID + Severity + Workload — ignore cost cents and exact title text so renaming a title doesn't break the test (test the contract, not the prose).

### Template B — golden renderer test

```go
func TestRender_Demo_PlainGolden(t *testing.T) {
    report := loadFixtureReport(t, "demo_findings.json")
    var buf bytes.Buffer
    if err := Render(&buf, report, Options{Color: false, Width: 78}); err != nil {
        t.Fatal(err)
    }
    goldenAssert(t, "demo_plain", buf.Bytes())
}

func TestRender_Demo_JSONGolden(t *testing.T) {
    report := loadFixtureReport(t, "demo_findings.json")
    var buf bytes.Buffer
    if err := RenderJSON(&buf, report); err != nil {
        t.Fatal(err)
    }
    goldenAssert(t, "demo_json", buf.Bytes())
}
```

Both invoke the same fixture so a renderer-pair drift is caught immediately. Disclosure presence is asserted separately:

```go
func TestRender_DisclosureAlwaysPresent(t *testing.T) {
    for _, mode := range []string{"plain", "color", "json", "roast"} {
        t.Run(mode, func(t *testing.T) {
            out := renderForTest(t, mode)
            if !bytes.Contains(out, []byte(AccuracyDisclosure)) {
                t.Errorf("%s mode is missing the mandatory ±40%% disclosure", mode)
            }
        })
    }
}
```

### Template C — `httptest.Server` fake for `--share`

```go
func TestShare_UploadsSanitizedPayload(t *testing.T) {
    var got []byte
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        got, _ = io.ReadAll(r.Body)
        w.WriteHeader(http.StatusOK)
    }))
    t.Cleanup(srv.Close)

    t.Setenv("OPTIQOR_SHARE_URL", srv.URL+"/api/v1/share")

    if err := Upload(context.Background(), Report{ /* … */ }); err != nil {
        t.Fatal(err)
    }
    if bytes.Contains(got, []byte("/home/user/secrets")) {
        t.Error("source path leaked — sanitiser failed")
    }
}
```

Notes:
- `httptest.NewServer` is the only network-shaped test allowed.
- `t.Setenv` over `os.Setenv` so the env var auto-restores.
- `t.Cleanup` over `defer` so it survives subtest cancellation.

### Template D — CLI invocation through Cobra

The CLI repo already has a pattern at `cmd/optiqor/golden_test.go`. The shape:

```go
func TestCmd_Analyze_DemoChart_GoldenStable(t *testing.T) {
    var stdout, stderr bytes.Buffer
    rc := runCmd(t, []string{"analyze", "--no-color", "testdata/fixtures/basic-chart"},
        &stdout, &stderr)
    if rc != 0 {
        t.Fatalf("exit code: got %d, want 0; stderr=%s", rc, stderr.String())
    }
    goldenAssert(t, "analyze_basic_chart_plain", stdout.Bytes())
}
```

`runCmd` invokes the same `RootCmd()` the binary does, with redirected streams. Exit code is the Cobra `Execute()` return mapped to the project's exit-code matrix (0 clean / 1 findings / 2 invocation / 3 runtime).

---

## Process for a new test

```
1. Read the code under test
   - What's the contract? (godoc + signatures)
   - What's the input shape, what's the output shape?
   - For a detector: what counts as a positive, what counts as negative?
   - For a renderer: what's the byte-identity contract?

2. Enumerate scenarios
   - One row per behaviour the code must guarantee
   - Detector: positive case, negative case, boundary (exactly at threshold)
   - Renderer: every option combination that materially changes output

3. Pick the test type
   - Pure logic → unit (Template A)
   - Renderer / CLI output → golden (Template B or D)
   - Share upload → httptest.Server fake (Template C)

4. Write the table
   - Anonymous struct inline
   - name string field first

5. Wire t.Run subtests
   - Each row is a t.Run(tc.name, func(t *testing.T) { ... })
   - Use t.Helper() in any helpers

6. Inject time / randomness
   - Pin time.Date(...) for time-dependent assertions
   - Accept io.Reader / *rand.Rand instead of calling globals

7. Run the gate (see below)

8. Commit per the project's commit skill
   - `test(<pkg>): cover <thing>` is the conventional subject
```

## Quality gate (run before declaring done)

From the repo root:

```bash
gofmt -l .                          # must be empty
go vet ./...                        # must be clean
go test -race -count=1 ./...        # must pass
golangci-lint run --timeout=2m ./... # must report 0 issues
```

If golden snapshots diverge, the gate fails by design. Eyeball the diff:

- If the change is **intentional** (you changed the renderer / score formula / disclosure text), run `UPDATE_GOLDEN=1 go test ./...` to regenerate, eyeball the diff again, commit the regenerated files alongside the code change.
- If the change is **unintentional**, the test is doing its job — find what drifted and fix it.

## Coverage targets

- `pkg/rules/`, `pkg/parser/`, `pkg/htmlrender/`: **90%** (public surface; every detector has positive + negative cases minimum)
- `internal/analyze/`, `internal/render/`, `internal/share/`, `internal/config/`: **70%**
- `internal/roast/`, `internal/render/style/`: **70%** including golden coverage of every styled-mode permutation
- `cmd/optiqor/`: covered by the golden tests in `cmd/optiqor/golden_test.go` — direct unit tests on Cobra command construction are not worth writing

Check with:

```bash
go test -cover ./...
```

## Golden updates

If you intentionally changed a renderer or detector and a golden diff is the expected output:

```bash
UPDATE_GOLDEN=1 go test ./...
git diff testdata/golden/    # eyeball every line
git add testdata/golden/<file>.txt  # only the ones that actually changed
```

Never commit a golden update without reading the diff. A test that's failing only because the golden is stale is the same as no test — it tells you nothing.

The accuracy disclosure inside any golden file must remain byte-identical to `AccuracyDisclosure`. If the disclosure text moved across a column or the surrounding box characters changed, ensure the disclosure substring is intact.

## Anti-patterns

| ❌ Do not | ✅ Instead |
|---|---|
| Compare `err.Error() == "expected text"` | `errors.Is(err, ErrSentinel)` |
| Share a `var fixture = ...` across tests | Build fresh inside each test |
| `time.Sleep` to wait for async work | The CLI is synchronous — there is no async to wait for. If you're tempted, the design is wrong. |
| `os.Setenv` without `t.Setenv` | `t.Setenv` — auto-restores |
| Pull in `testify/assert` for a one-line test | `if got != want { t.Errorf(...) }` — stdlib hard rule |
| Hit a real network endpoint | `httptest.NewServer` — never the real `optiqor.dev/r/<hash>` |
| Run the LLM, even a "small" one | Per CLAUDE.md: no LLM in CLI. Tests are deterministic. |
| Commit golden updates without reading the diff | `git diff testdata/golden/` first, every time |
| Test colour-rendering by stripping ANSI codes in the assertion | Run the test twice: one golden for `Color: false`, one for `Color: true`. Different concerns, different snapshots. |
| `init()` in test files to register fixtures | Push into `setUpXxx(t)` helpers; no `init()` in tests |
| Asserting on stderr log output | Logs are for humans, not contracts. Surface contract via return value or exit code. |

## Examples in this repo (read these for tone)

- **Table-driven detector**: [`pkg/rules/rules_test.go`](../../../pkg/rules/rules_test.go) and friends — positive + negative + threshold-boundary in rows.
- **Renderer golden**: [`internal/render/text_test.go`](../../../internal/render/text_test.go) — golden assertions for plain + JSON; disclosure-presence test that loops over modes.
- **CLI golden**: [`cmd/optiqor/golden_test.go`](../../../cmd/optiqor/golden_test.go) — full Cobra invocation, exit-code + stdout assertion.
- **`httptest.Server` for share upload**: [`internal/share/share_test.go`](../../../internal/share/share_test.go) — fake server captures payload, asserts sanitiser stripped PII.
- **Helper with `t.Helper()`**: [`pkg/parser/helm_test.go`](../../../pkg/parser/helm_test.go) — `q("2Gi")` quantity helper, `loadFixture(t, "basic-chart")` chart helper.
- **Boundary assertion**: [`internal/analyze/grade_test.go`](../../../internal/analyze/grade_test.go) — explicit `// 100 - 20 = 80` math-restate comments are kept because the test rationale is the threshold itself.

When in doubt, read one of those and match its shape.

// Command sevro is the entrypoint for the open-source Sevro CLI.
//
// The CLI is a deterministic rule engine that analyzes Helm charts for cost
// inefficiencies and security findings. It does NOT call any LLM and does NOT
// phone home by default — see ../../CLAUDE.md for the hard rules.
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lowplane/sevro/internal/analyze"
	"github.com/lowplane/sevro/internal/config"
	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/internal/render/style"
	"github.com/lowplane/sevro/internal/rules"
	"github.com/lowplane/sevro/internal/share"
)

// Exit codes — stable contract for CI integration.
const (
	exitSuccess     = 0 // no findings ≥ threshold
	exitFindings    = 1 // findings ≥ threshold reported
	exitInvocation  = 2 // invocation / parse error
	exitInternal    = 3 // unexpected runtime error
)

// errFindings is a sentinel returned from RunE so main can map it to exitFindings.
var errFindings = errors.New("sevro: findings exceed threshold")

var version = "dev"

const accuracyDisclosure = "Sandbox accuracy: ±40%. Install the Sevro agent for exact numbers (sevro.dev/get)."

func main() {
	err := newRootCmd().Execute()
	switch {
	case err == nil:
		os.Exit(exitSuccess)
	case errors.Is(err, errFindings):
		// Already-rendered finding output; suppress an additional error line.
		os.Exit(exitFindings)
	default:
		printError(os.Stderr, err)
		os.Exit(exitInvocation)
	}
}

func newRootCmd() *cobra.Command {
	var (
		noColor    bool
		configPath string
	)

	root := &cobra.Command{
		Use:   "sevro",
		Short: "Cost & security analysis for Kubernetes Helm charts",
		Long: `sevro analyzes Helm charts (or values files) for cost inefficiencies and
security findings — entirely offline, no login, no agent.

` + accuracyDisclosure,
		Example: `  # Analyze a Helm chart directory
  sevro analyze ./my-chart

  # Analyze a single values file
  sevro analyze ./values.yaml

  # Demo with a bundled chart
  sevro demo

  # Pipe machine-readable JSON into jq
  sevro analyze ./chart --json | jq .findings`,
		Version:           version,
		SilenceUsage:      true,
		SilenceErrors:     true, // we render errors ourselves
		DisableAutoGenTag: true,
	}

	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output (also: NO_COLOR env)")
	root.PersistentFlags().StringVar(&configPath, "config", "", "path to .sevro.yaml (default: ./.sevro.yaml or $SEVRO_CONFIG)")

	// Stash the no-color decision and the loaded config in context so
	// subcommands can read both.
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}
		// Config-level no_color implies --no-color unless flag explicitly disabled.
		effectiveNoColor := noColor || cfg.NoColor
		ctx := cmd.Context()
		ctx = withColorPolicy(ctx, resolveColor(cmd, effectiveNoColor))
		ctx = withConfig(ctx, cfg)
		cmd.SetContext(ctx)
		return nil
	}

	root.SetVersionTemplate(versionTemplate())

	root.AddCommand(
		newAnalyzeCmd(),
		newDemoCmd(),
		newDiffCmd(),
		newScoreCmd(),
		newAuditCmd(),
		newWatchCmd(),
		newCompareCmd(),
	)

	return root
}

// versionTemplate prints a polished one-liner including the brand.
func versionTemplate() string {
	return fmt.Sprintf("sevro %s — %s\n", version, "Helm chart cost & security analysis")
}

func newAnalyzeCmd() *cobra.Command {
	var (
		jsonOut    bool
		offline    bool
		shareFlag  bool
		roast      bool
		minSev     string
		detectors  []string
		failOn     string // severity threshold that triggers exit code 1
		outputPath string
	)
	cmd := &cobra.Command{
		Use:   "analyze [chart]",
		Short: "Analyze a Helm chart or values file for cost & security findings",
		Long: `Reads a Helm chart directory or a single values file and reports cost
inefficiencies and security findings.

` + accuracyDisclosure,
		Example: `  sevro analyze ./my-chart
  sevro analyze ./values.yaml --json
  sevro analyze ./chart --severity=med --fail-on=high
  sevro analyze ./chart --detector cpu-overprovisioned --detector missing-memory-limit`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			rep, err := analyze.RunPath(abs)
			if err != nil {
				return err
			}

			// Merge config-file defaults with flags. Flags win when supplied.
			cfg := configFrom(cmd.Context())
			effSev := minSev
			if effSev == "" {
				effSev = cfg.MinSeverity
			}
			effDetectors := detectors
			if len(effDetectors) == 0 {
				effDetectors = cfg.Detectors
			}
			effFailOn := failOn
			if effFailOn == "" {
				effFailOn = cfg.FailOn
			}

			rep = analyze.Filter(rep, analyze.FilterOptions{
				MinSeverity: rules.Severity(toUpper(effSev)),
				DetectorIDs: effDetectors,
			})
			if err := emitReport(cmd, rep, jsonOut, outputPath); err != nil {
				return err
			}
			if shareFlag {
				emitShareURL(cmd, rep)
			}
			_ = offline
			_ = roast
			return checkFailOn(rep, effFailOn)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&offline, "offline", true, "do not perform any network calls (always true in Phase 1)")
	cmd.Flags().BoolVar(&shareFlag, "share", false, "print sevro.dev/r/<hash> for the sanitised analysis (no upload in Phase 1)")
	cmd.Flags().BoolVar(&roast, "roast", false, "humorous output (findings stay accurate)")
	cmd.Flags().StringVar(&minSev, "severity", "", "drop findings below this severity (low|med|high)")
	cmd.Flags().StringArrayVar(&detectors, "detector", nil, "only run findings from these detector IDs (repeatable)")
	cmd.Flags().StringVar(&failOn, "fail-on", "", "exit code 1 when any finding is at this severity or higher (low|med|high)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "write the rendered output to a file instead of stdout")
	return cmd
}

// emitReport renders the report in JSON or styled text. When
// outputPath is non-empty the rendered bytes go to that file instead
// of stdout (CI use case: `sevro analyze --json --output result.json`).
func emitReport(cmd *cobra.Command, rep render.Report, jsonOut bool, outputPath string) error {
	w, closeFn, err := openOutput(cmd, outputPath)
	if err != nil {
		return err
	}
	defer closeFn()
	if jsonOut {
		return render.JSON(w, rep)
	}
	return render.Text(w, rep, renderOpts(cmd))
}

// openOutput resolves the destination: stdout when path is empty;
// otherwise creates / truncates the file. The returned closer is a
// no-op for stdout so callers can defer it unconditionally.
func openOutput(cmd *cobra.Command, path string) (io.Writer, func(), error) {
	if path == "" {
		return cmd.OutOrStdout(), func() {}, nil
	}
	f, err := os.Create(path) //nolint:gosec // user-specified output path
	if err != nil {
		return nil, nil, fmt.Errorf("open --output: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

// emitShareURL prints `--share` provenance to stderr so it doesn't
// pollute --json or --output. Returns nil even on share errors;
// share is best-effort UX, not a blocking failure path.
func emitShareURL(cmd *cobra.Command, rep any) {
	url, err := share.URL(rep)
	if err != nil {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "share URL (Phase-1 stub, viewer ships with sandbox): %s\n", url)
}

// checkFailOn returns errFindings when any finding meets or exceeds
// the threshold severity. Empty threshold is a no-op.
func checkFailOn(rep render.Report, threshold string) error {
	if threshold == "" {
		return nil
	}
	min := rules.Severity(toUpper(threshold))
	if !validSeverity(min) {
		return fmt.Errorf("invalid --fail-on severity %q (want low|med|high)", threshold)
	}
	for _, f := range rep.Findings {
		if severityRank(f.Severity) >= severityRank(min) {
			return errFindings
		}
	}
	return nil
}

func severityRank(s rules.Severity) int {
	switch s {
	case rules.SeverityHigh:
		return 3
	case rules.SeverityMed:
		return 2
	case rules.SeverityLow:
		return 1
	}
	return 0
}

func validSeverity(s rules.Severity) bool {
	return s == rules.SeverityHigh || s == rules.SeverityMed || s == rules.SeverityLow
}

func toUpper(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// demoChart is the bundled demo values file. //go:embed lets us ship
// the fixture inside the binary so `npx @sevro/cli demo` works with
// no input.
//
//go:embed demo/values.yaml
var demoChart []byte

func newDemoCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "demo",
		Short:   "Run analysis on a bundled demo chart",
		Example: `  sevro demo`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rep, err := analyze.Run(bytesReader(demoChart), analyze.Options{Source: "demo"})
			if err != nil {
				return err
			}
			if jsonOut {
				return render.JSON(cmd.OutOrStdout(), rep)
			}
			return render.Text(cmd.OutOrStdout(), rep, renderOpts(cmd))
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}

// renderOpts builds a render.Options for the active command, picking up
// the colour-policy decision the persistent pre-run stashed in context.
func renderOpts(cmd *cobra.Command) render.Options {
	return render.Options{
		Color: colorPolicyFrom(cmd.Context()),
		Width: terminalWidth(),
	}
}

// resolveColor decides whether to emit ANSI for a given command.
// Order of precedence (highest to lowest):
//
//  1. --no-color flag
//  2. NO_COLOR env var (any non-empty value, per https://no-color.org)
//  3. CLICOLOR_FORCE=1 forces color even when not a TTY
//  4. stdout is a TTY → color on
//  5. otherwise → color off
func resolveColor(cmd *cobra.Command, noColor bool) bool {
	if noColor {
		return false
	}
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		return true
	}
	out, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return false
	}
	return style.IsTTY(out)
}

// terminalWidth returns the current terminal width (cols). Falls back
// to 80 when not a TTY or when reading $COLUMNS fails.
func terminalWidth() int {
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := atoi(v); err == nil && n > 20 {
			return n
		}
	}
	return 80
}

func atoi(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// printError renders an error in red on a TTY; plain on a pipe.
func printError(w io.Writer, err error) {
	if err == nil {
		return
	}
	useColor := false
	if f, ok := w.(*os.File); ok {
		useColor = style.IsTTY(f) && os.Getenv("NO_COLOR") == ""
	}
	t := style.NewTheme(useColor)
	fmt.Fprintln(w, t.SevHigh.Render(" ERROR ")+" "+err.Error())
}

// bytesReader is a tiny adapter so analyze.Run can read from a byte slice
// without pulling in bytes.NewReader at the import-graph root of main.
func bytesReader(b []byte) *bytesReaderImpl { return &bytesReaderImpl{b: b} }

type bytesReaderImpl struct {
	b []byte
	i int
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func newDiffCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "diff <a> <b>",
		Short:   "Show cost delta between two values files",
		Args:    cobra.ExactArgs(2),
		Example: `  sevro diff ./before/values.yaml ./after/values.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, err := analyze.DiffPaths(args[0], args[1])
			if err != nil {
				return err
			}
			if jsonOut {
				return rep.WriteJSON(cmd.OutOrStdout())
			}
			return rep.WriteText(cmd.OutOrStdout(), renderOpts(cmd))
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}

func newScoreCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "score [chart]",
		Short:   "Assign an efficiency score to a chart",
		Args:    cobra.MaximumNArgs(1),
		Example: "  sevro score ./my-chart\n  sevro score ./values.yaml --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			rep, err := analyze.RunPath(abs)
			if err != nil {
				return err
			}
			s := analyze.Compute(rep.Source, rep.Workloads, rep.Findings)
			if jsonOut {
				return s.WriteJSON(cmd.OutOrStdout())
			}
			return s.WriteText(cmd.OutOrStdout(), renderOpts(cmd))
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}

func newAuditCmd() *cobra.Command {
	var (
		jsonOut    bool
		failOn     string
		outputPath string
	)
	cmd := &cobra.Command{
		Use:     "audit [chart]",
		Short:   "Audit a chart for security findings only",
		Args:    cobra.MaximumNArgs(1),
		Example: "  sevro audit ./my-chart\n  sevro audit ./values.yaml --fail-on=high",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			rep, err := analyze.RunPath(abs)
			if err != nil {
				return err
			}
			rep = analyze.Filter(rep, analyze.FilterOptions{SecurityOnly: true})
			if err := emitReport(cmd, rep, jsonOut, outputPath); err != nil {
				return err
			}
			return checkFailOn(rep, failOn)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	cmd.Flags().StringVar(&failOn, "fail-on", "high", "exit code 1 when any finding is at this severity or higher (low|med|high)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "write the rendered output to a file instead of stdout")
	return cmd
}

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch [chart]",
		Short: "Watch a chart and re-analyze on change",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
}

func newCompareCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "compare <a> <b>",
		Short:   "Side-by-side comparison of two charts (currently a diff alias)",
		Args:    cobra.ExactArgs(2),
		Example: `  sevro compare ./before/values.yaml ./after/values.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, err := analyze.DiffPaths(args[0], args[1])
			if err != nil {
				return err
			}
			if jsonOut {
				return rep.WriteJSON(cmd.OutOrStdout())
			}
			return rep.WriteText(cmd.OutOrStdout(), renderOpts(cmd))
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}

func notYetImplemented(cmd *cobra.Command) error {
	return fmt.Errorf("`sevro %s` is not yet implemented (see https://sevro.dev/roadmap)", cmd.Name())
}

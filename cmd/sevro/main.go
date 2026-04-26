// Command sevro is the entrypoint for the open-source Sevro CLI.
//
// The CLI is a deterministic rule engine that analyzes Helm charts for cost
// inefficiencies and security findings. It does NOT call any LLM and does NOT
// phone home by default — see ../../CLAUDE.md for the hard rules.
package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lowplane/sevro/internal/analyze"
	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/internal/render/style"
)

var version = "dev"

const accuracyDisclosure = "Sandbox accuracy: ±40%. Install the Sevro agent for exact numbers (sevro.dev/get)."

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// Surface the underlying error in styled red on a TTY; otherwise
		// fall back to cobra's default plain stderr write.
		printError(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var noColor bool

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

	// Stash the no-color decision in a context so subcommands can read it.
	root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		cmd.SetContext(withColorPolicy(cmd.Context(), resolveColor(cmd, noColor)))
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
		jsonOut bool
		offline bool
		share   bool
		roast   bool
	)
	cmd := &cobra.Command{
		Use:   "analyze [chart]",
		Short: "Analyze a Helm chart or values file for cost & security findings",
		Long: `Reads a Helm chart directory or a single values file and reports cost
inefficiencies and security findings.

` + accuracyDisclosure,
		Example: `  sevro analyze ./my-chart
  sevro analyze ./values.yaml --json
  sevro analyze ./chart --no-color`,
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
			if jsonOut {
				return render.JSON(cmd.OutOrStdout(), rep)
			}
			// `--share` and `--roast` are accepted now so flags land in
			// muscle memory; their behavior arrives in later phases.
			_ = offline
			_ = share
			_ = roast
			return render.Text(cmd.OutOrStdout(), rep, renderOpts(cmd))
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&offline, "offline", true, "do not perform any network calls (always true in Phase 1)")
	cmd.Flags().BoolVar(&share, "share", false, "upload sanitized analysis to sevro.dev/r/<hash> (opt-in)")
	cmd.Flags().BoolVar(&roast, "roast", false, "humorous output (findings stay accurate)")
	return cmd
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
	return &cobra.Command{
		Use:   "diff <a> <b>",
		Short: "Show cost delta between two values files",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
}

func newScoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "score [chart]",
		Short: "Assign an efficiency score to a chart",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
}

func newAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "audit",
		Short:  "Audit a chart for security findings only",
		Hidden: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
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
	return &cobra.Command{
		Use:   "compare <a> <b>",
		Short: "Side-by-side comparison of two charts",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
}

func notYetImplemented(cmd *cobra.Command) error {
	return fmt.Errorf("`sevro %s` is not yet implemented (see https://sevro.dev/roadmap)", cmd.Name())
}

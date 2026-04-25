// Command costify is the entrypoint for the open-source Costify CLI.
//
// The CLI is a deterministic rule engine that analyzes Helm charts for cost
// inefficiencies and security findings. It does NOT call any LLM and does NOT
// phone home by default — see ../../CLAUDE.md for the hard rules.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

const accuracyDisclosure = "Sandbox accuracy: ±40%. Install the Costify agent for exact numbers (costify.dev/get)."

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "costify",
		Short:         "Cost & security analysis for Kubernetes Helm charts",
		Long:          "costify analyzes Helm charts (or values files) for cost inefficiencies and security findings.\n\n" + accuracyDisclosure,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(phase-3): wire internal/analyze, internal/parser, internal/rules, internal/render.
			_ = jsonOut
			_ = offline
			_ = share
			_ = roast
			return notYetImplemented(cmd)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&offline, "offline", false, "do not perform any network calls (default true for analyze)")
	cmd.Flags().BoolVar(&share, "share", false, "upload sanitized analysis to costify.dev/r/<hash> (opt-in)")
	cmd.Flags().BoolVar(&roast, "roast", false, "humorous output (findings stay accurate)")
	return cmd
}

func newDemoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "Run analysis on a bundled demo chart",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notYetImplemented(cmd)
		},
	}
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
	return fmt.Errorf("`costify %s` is not yet implemented (see https://costify.dev/roadmap)", cmd.Name())
}

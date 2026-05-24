// Package analyze orchestrates one CLI run: parser → rules → render.
package analyze

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/optiqor/optiqor-cli/internal/render"
	"github.com/optiqor/optiqor-cli/pkg/parser"
	"github.com/optiqor/optiqor-cli/pkg/rules"
)

// Options controls a single analysis run.
type Options struct {
	Source    string           // file path or "stdin", surfaced in the report header
	Detectors []rules.Detector // nil → rules.All()
}

// Run reads a Helm values document from r and returns the report.
func Run(r io.Reader, opts Options) (render.Report, error) {
	wls, err := parser.ParseValues(r)
	if err != nil {
		return render.Report{}, err
	}
	dets := opts.Detectors
	if dets == nil {
		dets = rules.All()
	}
	return render.Report{
		Source:    opts.Source,
		Workloads: len(wls),
		Findings:  rules.Run(wls, dets),
	}, nil
}

// RunPath accepts a values file or a directory containing
// values.yaml at the root.
func RunPath(path string) (render.Report, error) {
	info, err := os.Stat(path)
	if err != nil {
		return render.Report{}, fmt.Errorf("analyze: stat %s: %w", path, err)
	}
	target := path
	if info.IsDir() {
		target = filepath.Join(path, "values.yaml")
	}
	f, err := os.Open(target) //nolint:gosec // user-specified analysis input
	if err != nil {
		return render.Report{}, fmt.Errorf("analyze: open %s: %w", target, err)
	}
	defer func() { _ = f.Close() }()
	return Run(f, Options{Source: target})
}

package rules

import "github.com/optiqor/optiqor-cli/pkg/parser"

// runAsRoot fires when securityContext.runAsNonRoot is explicitly
// false, OR when neither pod nor container security context declares
// runAsNonRoot. Modern Kubernetes guidance (CIS 5.2.5) is to run all
// workloads as a non-root UID; charts that omit the field fall back
// to whatever the image's USER directive says, which is almost always
// 0 for public images.
type runAsRoot struct{}

func newRunAsRoot() Detector { return runAsRoot{} }

func (runAsRoot) ID() string   { return "run-as-root" }
func (runAsRoot) Name() string { return "Container runs as root" }

func (runAsRoot) Run(w parser.Workload) []Finding {
	// Explicit false → HIGH; unset → MED (image USER could be non-zero,
	// can't tell from values.yaml alone).
	if w.Security.RunAsNonRoot != nil && !*w.Security.RunAsNonRoot {
		return []Finding{{
			DetectorID: "run-as-root",
			Workload:   w.Name,
			Title:      "Container runs as root",
			Detail:     "securityContext.runAsNonRoot is set to false. Running as UID 0 expands the blast radius of any container escape. Set runAsNonRoot=true and runAsUser to a non-zero UID.",
			Severity:   SeverityHigh,
			Confidence: ConfidenceHigh,
		}}
	}
	if w.Security.RunAsNonRoot == nil {
		return []Finding{{
			DetectorID: "run-as-root",
			Workload:   w.Name,
			Title:      "runAsNonRoot not declared",
			Detail:     "securityContext.runAsNonRoot is not declared. Without it, Kubernetes defers to the image's USER directive — which is UID 0 for most public images. Set runAsNonRoot=true to make the intent explicit.",
			Severity:   SeverityMed,
			Confidence: ConfidenceMed,
		}}
	}
	return nil
}

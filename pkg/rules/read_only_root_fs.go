package rules

import "github.com/optiqor/optiqor-cli/pkg/parser"

// readOnlyRootFSMissing fires when the workload does not declare a
// read-only root filesystem. Mounting `/` as read-only is a cheap
// hardening step that defeats most container-escape persistence
// techniques (the attacker can't write a backdoor to disk).
type readOnlyRootFSMissing struct{}

func newReadOnlyRootFSMissing() Detector { return readOnlyRootFSMissing{} }

func (readOnlyRootFSMissing) ID() string   { return "read-only-root-fs-missing" }
func (readOnlyRootFSMissing) Name() string { return "Root filesystem not read-only" }

func (readOnlyRootFSMissing) Run(w parser.Workload) []Finding {
	if w.Security.ReadOnlyRootFilesystem != nil && !*w.Security.ReadOnlyRootFilesystem {
		return []Finding{{
			DetectorID: "read-only-root-fs-missing",
			Workload:   w.Name,
			Title:      "Container root filesystem is writable",
			Detail:     "securityContext.readOnlyRootFilesystem is set to false. A writable rootfs lets an attacker persist files after a container escape. Set readOnlyRootFilesystem=true and mount tmpfs volumes for any directories the app legitimately writes to.",
			Severity:   SeverityMed,
			Confidence: ConfidenceHigh,
		}}
	}
	if w.Security.ReadOnlyRootFilesystem == nil {
		return []Finding{{
			DetectorID: "read-only-root-fs-missing",
			Workload:   w.Name,
			Title:      "readOnlyRootFilesystem not declared",
			Detail:     "securityContext.readOnlyRootFilesystem is not declared. Declare readOnlyRootFilesystem=true to make hardening explicit; mount tmpfs volumes for legitimate write paths.",
			Severity:   SeverityLow,
			Confidence: ConfidenceMed,
		}}
	}
	return nil
}

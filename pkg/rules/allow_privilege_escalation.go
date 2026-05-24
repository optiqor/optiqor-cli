package rules

import "github.com/optiqor/optiqor-cli/pkg/parser"

// allowPrivilegeEscalation fires when allowPrivilegeEscalation is true
// (or unset, which defaults to true). The flag is the kernel's
// "no_new_privs" bit; setting it false closes a class of setuid-based
// privilege-escalation bugs at almost zero compatibility cost.
type allowPrivilegeEscalation struct{}

func newAllowPrivilegeEscalation() Detector { return allowPrivilegeEscalation{} }

func (allowPrivilegeEscalation) ID() string   { return "allow-privilege-escalation" }
func (allowPrivilegeEscalation) Name() string { return "Privilege escalation allowed" }

func (allowPrivilegeEscalation) Run(w parser.Workload) []Finding {
	// Explicit true → HIGH; unset (defaults to true at runtime) → MED.
	if w.Security.AllowPrivilegeEscalation != nil && *w.Security.AllowPrivilegeEscalation {
		return []Finding{{
			DetectorID: "allow-privilege-escalation",
			Workload:   w.Name,
			Title:      "allowPrivilegeEscalation explicitly enabled",
			Detail:     "securityContext.allowPrivilegeEscalation=true lets a process gain capabilities its parent didn't have (setuid, file capabilities). Set allowPrivilegeEscalation=false unless you're running an init system or known-good legacy binary that requires it.",
			Severity:   SeverityHigh,
			Confidence: ConfidenceHigh,
		}}
	}
	if w.Security.AllowPrivilegeEscalation == nil {
		return []Finding{{
			DetectorID: "allow-privilege-escalation",
			Workload:   w.Name,
			Title:      "allowPrivilegeEscalation not declared",
			Detail:     "securityContext.allowPrivilegeEscalation is undeclared; Kubernetes defaults it to true. Set allowPrivilegeEscalation=false to make the no_new_privs bit explicit.",
			Severity:   SeverityMed,
			Confidence: ConfidenceMed,
		}}
	}
	return nil
}

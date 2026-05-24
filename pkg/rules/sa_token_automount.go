package rules

import "github.com/optiqor/optiqor-cli/pkg/parser"

// serviceAccountTokenAutomount fires when the workload has not
// explicitly disabled service-account-token auto-mount. By default
// Kubernetes mounts a token at `/var/run/secrets/kubernetes.io/...`
// for every pod; for a workload that never talks to the K8s API,
// that token is a free credential for any attacker who lands on
// the pod.
//
// CIS Kubernetes Benchmark 5.1.5 / 5.1.6.
type serviceAccountTokenAutomount struct{}

func newServiceAccountTokenAutomount() Detector { return serviceAccountTokenAutomount{} }

func (serviceAccountTokenAutomount) ID() string {
	return "service-account-token-automount"
}
func (serviceAccountTokenAutomount) Name() string {
	return "ServiceAccount token auto-mount not disabled"
}

func (serviceAccountTokenAutomount) Run(w parser.Workload) []Finding {
	if w.Security.AutomountServiceAccountToken != nil && !*w.Security.AutomountServiceAccountToken {
		return nil
	}
	severity := SeverityMed
	confidence := ConfidenceMed
	detail := "automountServiceAccountToken is undeclared, so Kubernetes auto-mounts a service-account credential at /var/run/secrets/kubernetes.io/serviceaccount/. Most application workloads never read it; an attacker on the pod gets a free pivot to the K8s API. Set automountServiceAccountToken=false unless the workload genuinely calls the API."
	if w.Security.AutomountServiceAccountToken != nil && *w.Security.AutomountServiceAccountToken {
		severity = SeverityHigh
		confidence = ConfidenceHigh
		detail = "automountServiceAccountToken is explicitly true. The pod gets a service-account credential at /var/run/secrets/kubernetes.io/serviceaccount/; an attacker who lands on the pod gets a free pivot to the K8s API. Confirm the workload genuinely needs the API; otherwise set automountServiceAccountToken=false."
	}
	return []Finding{{
		DetectorID: "service-account-token-automount",
		Workload:   w.Name,
		Title:      "ServiceAccount token automatically mounted",
		Detail:     detail,
		Severity:   severity,
		Confidence: confidence,
	}}
}

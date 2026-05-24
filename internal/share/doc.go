// Package share uploads sanitised analyses to optiqor.dev/r/<hash>
// when the user opts in via --share. Only outbound network call the
// CLI makes (CLAUDE.md hard rule: no telemetry by default).
package share

// Package parser reads Helm values + templates and Kustomize overlays
// into a normalised in-memory representation that the rule engine
// consumes.
//
// Phase 1 supports flat values.yaml structures (`<workload>.resources.{requests,limits}.{cpu,memory}`).
// Sub-chart and template parsing land in Phase 2.
package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CPU is millicores. 1 CPU == 1000 milli-CPU.
//
// Memory is bytes.
type Quantity struct {
	Value    int64  // canonical units (millicores for CPU, bytes for memory)
	Set      bool   // whether the source actually had a value
	Original string // original source string, preserved for display
}

// String renders the original input when present; otherwise a sentinel.
func (q Quantity) String() string {
	if !q.Set {
		return "(unset)"
	}
	return q.Original
}

// ParseCPU parses Kubernetes CPU quantities: "500m" -> 500, "1" -> 1000,
// "2.5" -> 2500. An empty string yields a zero, unset Quantity.
func ParseCPU(s string) (Quantity, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Quantity{}, nil
	}
	if strings.HasSuffix(s, "m") {
		raw := strings.TrimSuffix(s, "m")
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Quantity{}, fmt.Errorf("parser: cpu %q: %w", s, err)
		}
		if n < 0 {
			return Quantity{}, fmt.Errorf("parser: cpu %q: negative", s)
		}
		return Quantity{Value: n, Set: true, Original: s}, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Quantity{}, fmt.Errorf("parser: cpu %q: %w", s, err)
	}
	if f < 0 {
		return Quantity{}, fmt.Errorf("parser: cpu %q: negative", s)
	}
	return Quantity{Value: int64(f * 1000), Set: true, Original: s}, nil
}

// ParseMemory parses Kubernetes memory quantities: "512Mi" -> 512*1024^2,
// "2Gi" -> 2*1024^3, "1G" -> 10^9, "1500" -> 1500. Empty -> unset.
func ParseMemory(s string) (Quantity, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Quantity{}, nil
	}
	mult, raw := splitMemorySuffix(s)
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return Quantity{}, fmt.Errorf("parser: memory %q: %w", s, err)
	}
	if f < 0 {
		return Quantity{}, fmt.Errorf("parser: memory %q: negative", s)
	}
	v := int64(f * float64(mult))
	if v < 0 { // overflow
		return Quantity{}, errors.New("parser: memory overflow")
	}
	return Quantity{Value: v, Set: true, Original: s}, nil
}

// splitMemorySuffix returns the (multiplier, numeric-part) pair for a
// Kubernetes memory string. Recognises Ki/Mi/Gi/Ti/Pi (binary) and
// k/M/G/T/P (decimal).
func splitMemorySuffix(s string) (mult int64, numeric string) {
	suffixes := []struct {
		s    string
		mult int64
	}{
		{"Ki", 1024},
		{"Mi", 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Pi", 1024 * 1024 * 1024 * 1024 * 1024},
		{"k", 1_000},
		{"M", 1_000_000},
		{"G", 1_000_000_000},
		{"T", 1_000_000_000_000},
		{"P", 1_000_000_000_000_000},
	}
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf.s) {
			return suf.mult, strings.TrimSuffix(s, suf.s)
		}
	}
	return 1, s
}

// FormatCPU prints a Quantity that represents CPU millicores.
func FormatCPU(q Quantity) string {
	if !q.Set {
		return "(unset)"
	}
	if q.Value%1000 == 0 {
		return fmt.Sprintf("%d", q.Value/1000)
	}
	return fmt.Sprintf("%dm", q.Value)
}

// FormatMemory prints a Quantity in a human-friendly base-2 unit.
func FormatMemory(q Quantity) string {
	if !q.Set {
		return "(unset)"
	}
	v := q.Value
	switch {
	case v >= 1024*1024*1024*1024:
		return fmt.Sprintf("%.1fTi", float64(v)/float64(1024*1024*1024*1024))
	case v >= 1024*1024*1024:
		return fmt.Sprintf("%.1fGi", float64(v)/float64(1024*1024*1024))
	case v >= 1024*1024:
		return fmt.Sprintf("%.1fMi", float64(v)/float64(1024*1024))
	case v >= 1024:
		return fmt.Sprintf("%.1fKi", float64(v)/float64(1024))
	default:
		return fmt.Sprintf("%dB", v)
	}
}

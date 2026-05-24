package analyze

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// DiffEntry is the per-workload change between two values files.
// Workloads are matched by name: B-only → NewWorkload,
// A-only → RemovedWorkload.
type DiffEntry struct {
	Name                 string
	NewWorkload          bool
	RemovedWorkload      bool
	CPURequestDelta      int64 // millicores; B - A
	CPULimitDelta        int64
	MemoryRequestDelta   int64 // bytes; B - A
	MemoryLimitDelta     int64
	MonthlyUSDCentsDelta int64 // sandbox estimate; ±40% disclosure applies
}

// DiffReport is the wire shape Diff returns.
type DiffReport struct {
	A       string      `json:"a"`
	B       string      `json:"b"`
	Entries []DiffEntry `json:"entries"`
}

// Pricing constants must stay in lockstep with internal/rules so the
// diff's monthly delta matches the analyze report's savings.
const (
	cpuPriceCentsPerCoreHour = 4   // $/vCPU-hour, AWS m5 baseline
	cpuMonthlyHours          = 730 // hours/month, AWS billing convention
	memPriceCentsPerGiBMonth = 350 // $3.50 per GiB-month
)

// Diff returns the DiffReport between two streams of Helm values.
func Diff(a, b io.Reader, aLabel, bLabel string) (DiffReport, error) {
	wlA, err := parser.ParseValues(a)
	if err != nil {
		return DiffReport{}, fmt.Errorf("diff: parse %q: %w", aLabel, err)
	}
	wlB, err := parser.ParseValues(b)
	if err != nil {
		return DiffReport{}, fmt.Errorf("diff: parse %q: %w", bLabel, err)
	}

	byName := map[string]*pair{}
	for i := range wlA {
		byName[wlA[i].Name] = &pair{a: &wlA[i]}
	}
	for i := range wlB {
		if p, ok := byName[wlB[i].Name]; ok {
			p.b = &wlB[i]
		} else {
			byName[wlB[i].Name] = &pair{b: &wlB[i]}
		}
	}

	entries := make([]DiffEntry, 0, len(byName))
	for name, p := range byName {
		entry := DiffEntry{Name: name}
		switch {
		case p.a == nil:
			entry.NewWorkload = true
		case p.b == nil:
			entry.RemovedWorkload = true
		}
		entry.CPURequestDelta = qDelta(get(p.a, reqCPU), get(p.b, reqCPU))
		entry.CPULimitDelta = qDelta(get(p.a, limCPU), get(p.b, limCPU))
		entry.MemoryRequestDelta = qDelta(get(p.a, reqMem), get(p.b, reqMem))
		entry.MemoryLimitDelta = qDelta(get(p.a, limMem), get(p.b, limMem))
		entry.MonthlyUSDCentsDelta = monthlyDelta(entry)
		// Suppress no-op entries (same on both sides, all zero deltas).
		// Keep New/Removed even when math nets to zero — an added
		// workload with no resources is still useful to surface.
		if !entry.NewWorkload && !entry.RemovedWorkload &&
			entry.CPURequestDelta == 0 && entry.CPULimitDelta == 0 &&
			entry.MemoryRequestDelta == 0 && entry.MemoryLimitDelta == 0 {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	return DiffReport{A: aLabel, B: bLabel, Entries: entries}, nil
}

// DiffPaths opens both files and runs Diff.
func DiffPaths(a, b string) (DiffReport, error) {
	fa, err := openValues(a)
	if err != nil {
		return DiffReport{}, err
	}
	defer func() { _ = fa.Close() }()
	fb, err := openValues(b)
	if err != nil {
		return DiffReport{}, err
	}
	defer func() { _ = fb.Close() }()
	return Diff(fa, fb, a, b)
}

// MonthlyUSDCentsDelta totals the per-entry deltas. Negative is a
// saving (B < A); positive is a regression.
func (r DiffReport) MonthlyUSDCentsDelta() int64 {
	var sum int64
	for _, e := range r.Entries {
		sum += e.MonthlyUSDCentsDelta
	}
	return sum
}

type pair struct {
	a, b *parser.Workload
}

type which int

const (
	reqCPU which = iota
	limCPU
	reqMem
	limMem
)

func get(w *parser.Workload, k which) parser.Quantity {
	if w == nil {
		return parser.Quantity{}
	}
	switch k {
	case reqCPU:
		return w.Requests.CPU
	case limCPU:
		return w.Limits.CPU
	case reqMem:
		return w.Requests.Memory
	case limMem:
		return w.Limits.Memory
	}
	return parser.Quantity{}
}

func qDelta(a, b parser.Quantity) int64 {
	var av, bv int64
	if a.Set {
		av = a.Value
	}
	if b.Set {
		bv = b.Value
	}
	return bv - av
}

// monthlyDelta is sandbox-grade per the ±40% disclosure; the agent
// replaces it with measured numbers.
func monthlyDelta(e DiffEntry) int64 {
	cpuMillicoreDelta := e.CPURequestDelta // request drives reserved capacity
	memBytesDelta := e.MemoryRequestDelta

	cpuCents := cpuMillicoreDelta * cpuMonthlyHours * cpuPriceCentsPerCoreHour / 1000
	memCents := int64(float64(memBytesDelta) / float64(1024*1024*1024) * float64(memPriceCentsPerGiBMonth))
	return cpuCents + memCents
}

func openValues(path string) (*os.File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("diff: abs %s: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("diff: stat %s: %w", abs, err)
	}
	target := abs
	if info.IsDir() {
		target = filepath.Join(abs, "values.yaml")
	}
	return os.Open(target) //nolint:gosec // user-specified analysis input
}

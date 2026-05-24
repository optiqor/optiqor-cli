package parser

import "testing"

func TestParseCPU(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		want    int64
		set     bool
		wantErr bool
	}{
		{name: "empty-yields-unset", in: "", want: 0, set: false},
		{name: "millicore-suffix", in: "500m", want: 500, set: true},
		{name: "whole-core", in: "1", want: 1000, set: true},
		{name: "fractional-core", in: "2.5", want: 2500, set: true},
		{name: "zero-is-set", in: "0", want: 0, set: true},
		{name: "negative-errors", in: "-1", wantErr: true},
		{name: "trailing-junk-errors", in: "100x", wantErr: true},
		{name: "non-numeric-errors", in: "abc", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseCPU(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseCPU(%q): expected error", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCPU(%q): unexpected error: %v", tc.in, err)
			}
			if q.Value != tc.want || q.Set != tc.set {
				t.Errorf("ParseCPU(%q) = {%d %v}, want {%d %v}", tc.in, q.Value, q.Set, tc.want, tc.set)
			}
		})
	}
}

func TestParseMemory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		want    int64
		set     bool
		wantErr bool
	}{
		{name: "empty-yields-unset", in: "", want: 0, set: false},
		{name: "plain-bytes", in: "1024", want: 1024, set: true},
		{name: "Ki-binary", in: "1Ki", want: 1024, set: true},
		{name: "Mi-binary", in: "1Mi", want: 1024 * 1024, set: true},
		{name: "512Mi", in: "512Mi", want: 512 * 1024 * 1024, set: true},
		{name: "Gi-binary", in: "2Gi", want: 2 * 1024 * 1024 * 1024, set: true},
		{name: "G-decimal", in: "1G", want: 1_000_000_000, set: true},
		{name: "M-decimal", in: "100M", want: 100_000_000, set: true},
		{name: "negative-errors", in: "-1Mi", wantErr: true},
		{name: "non-numeric-errors", in: "abcMi", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseMemory(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMemory(%q): expected error", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMemory(%q): unexpected error: %v", tc.in, err)
			}
			if q.Value != tc.want || q.Set != tc.set {
				t.Errorf("ParseMemory(%q) = {%d %v}, want {%d %v}", tc.in, q.Value, q.Set, tc.want, tc.set)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Quantity
		want string
	}{
		{name: "zero", in: Quantity{Value: 0, Set: true}, want: "0"},
		{name: "sub-core-millis", in: Quantity{Value: 500, Set: true}, want: "500m"},
		{name: "whole-core", in: Quantity{Value: 1000, Set: true}, want: "1"},
		{name: "fractional-stays-millis", in: Quantity{Value: 1500, Set: true}, want: "1500m"},
		{name: "two-cores", in: Quantity{Value: 2000, Set: true}, want: "2"},
		{name: "unset-sentinel", in: Quantity{}, want: "(unset)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatCPU(tc.in); got != tc.want {
				t.Errorf("FormatCPU(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    int64
		want string
	}{
		{name: "zero-bytes", v: 0, want: "0B"},
		{name: "one-Ki", v: 1024, want: "1.0Ki"},
		{name: "one-Mi", v: 1024 * 1024, want: "1.0Mi"},
		{name: "two-Gi", v: 2 * 1024 * 1024 * 1024, want: "2.0Gi"},
		{name: "one-Ti", v: 1024 * 1024 * 1024 * 1024, want: "1.0Ti"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatMemory(Quantity{Value: tc.v, Set: true})
			if got != tc.want {
				t.Errorf("FormatMemory(%d) = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

func TestQuantityString(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Quantity
		want string
	}{
		{name: "set-returns-original", in: Quantity{Original: "500m", Set: true}, want: "500m"},
		{name: "unset-returns-sentinel", in: Quantity{}, want: "(unset)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

package parser

import "testing"

func TestParseCPU(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		set     bool
		wantErr bool
	}{
		{"", 0, false, false},
		{"500m", 500, true, false},
		{"1", 1000, true, false},
		{"2.5", 2500, true, false},
		{"0", 0, true, false},
		{"-1", 0, false, true},
		{"100x", 0, false, true},
		{"abc", 0, false, true},
	}
	for _, tc := range cases {
		q, err := ParseCPU(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseCPU(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCPU(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if q.Value != tc.want || q.Set != tc.set {
			t.Errorf("ParseCPU(%q) = {%d %v}, want {%d %v}", tc.in, q.Value, q.Set, tc.want, tc.set)
		}
	}
}

func TestParseMemory(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		set     bool
		wantErr bool
	}{
		{"", 0, false, false},
		{"1024", 1024, true, false},
		{"1Ki", 1024, true, false},
		{"1Mi", 1024 * 1024, true, false},
		{"512Mi", 512 * 1024 * 1024, true, false},
		{"2Gi", 2 * 1024 * 1024 * 1024, true, false},
		{"1G", 1_000_000_000, true, false},
		{"100M", 100_000_000, true, false},
		{"-1Mi", 0, false, true},
		{"abcMi", 0, false, true},
	}
	for _, tc := range cases {
		q, err := ParseMemory(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseMemory(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMemory(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if q.Value != tc.want || q.Set != tc.set {
			t.Errorf("ParseMemory(%q) = {%d %v}, want {%d %v}", tc.in, q.Value, q.Set, tc.want, tc.set)
		}
	}
}

func TestFormatCPU(t *testing.T) {
	cases := map[int64]string{
		0:    "0",
		500:  "500m",
		1000: "1",
		1500: "1500m",
		2000: "2",
	}
	for in, want := range cases {
		got := FormatCPU(Quantity{Value: in, Set: true})
		if got != want {
			t.Errorf("FormatCPU(%d) = %q, want %q", in, got, want)
		}
	}
	if got := FormatCPU(Quantity{}); got != "(unset)" {
		t.Errorf("FormatCPU(unset) = %q", got)
	}
}

func TestFormatMemory(t *testing.T) {
	cases := []struct {
		v    int64
		want string
	}{
		{0, "0B"},
		{1024, "1.0Ki"},
		{1024 * 1024, "1.0Mi"},
		{2 * 1024 * 1024 * 1024, "2.0Gi"},
		{1024 * 1024 * 1024 * 1024, "1.0Ti"},
	}
	for _, tc := range cases {
		got := FormatMemory(Quantity{Value: tc.v, Set: true})
		if got != tc.want {
			t.Errorf("FormatMemory(%d) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestQuantityString(t *testing.T) {
	q := Quantity{Original: "500m", Set: true}
	if q.String() != "500m" {
		t.Errorf("String() = %q", q.String())
	}
	if (Quantity{}).String() != "(unset)" {
		t.Error("unset String() should be (unset)")
	}
}

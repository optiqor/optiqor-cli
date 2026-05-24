package style

import (
	"strings"
	"testing"
)

func TestNewTheme_PlainOmitsAnsi(t *testing.T) {
	th := NewTheme(false)
	for _, s := range []string{
		th.SeverityBadge("HIGH"),
		th.ConfidenceDots("high"),
		th.Brand.Render("Optiqor"),
		th.DividerLine(40),
	} {
		if strings.Contains(s, "\x1b") {
			t.Errorf("plain theme leaked ANSI: %q", s)
		}
	}
}

func TestNewTheme_ColoredEmitsAnsi(t *testing.T) {
	th := NewTheme(true)
	got := th.SeverityBadge("HIGH")
	if !strings.Contains(got, "\x1b") {
		t.Errorf("colored theme should emit ANSI; got %q", got)
	}
}

func TestSeverityBadge_LabelsPresent(t *testing.T) {
	for _, s := range []string{"HIGH", "MED", "LOW", "INFO"} {
		got := NewTheme(false).SeverityBadge(s)
		if !strings.Contains(got, s) {
			t.Errorf("SeverityBadge(%q) = %q; should contain label", s, got)
		}
	}
}

func TestConfidenceDots_LabelsPresent(t *testing.T) {
	cases := []string{"high", "medium", "low", "unknown"}
	for _, c := range cases {
		got := NewTheme(false).ConfidenceDots(c)
		if !strings.Contains(got, c) {
			t.Errorf("ConfidenceDots(%q) = %q; should contain word", c, got)
		}
	}
}

func TestHyperlink_PlainFallback(t *testing.T) {
	got := NewTheme(false).Hyperlink("link", "https://optiqor.dev")
	if !strings.Contains(got, "https://optiqor.dev") {
		t.Errorf("plain hyperlink should expose URL: %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("plain hyperlink leaked ANSI: %q", got)
	}
}

func TestHyperlink_ColorEmitsOSC8(t *testing.T) {
	got := NewTheme(true).Hyperlink("link", "https://optiqor.dev")
	if !strings.Contains(got, "\x1b]8;") {
		t.Errorf("OSC 8 missing: %q", got)
	}
}

func TestDividerLine_Width(t *testing.T) {
	th := NewTheme(false)
	got := th.DividerLine(10)
	// Box-drawing chars are multi-byte; count runes not bytes.
	runes := []rune(got)
	if len(runes) != 10 {
		t.Errorf("DividerLine(10) rune count = %d, want 10", len(runes))
	}
}

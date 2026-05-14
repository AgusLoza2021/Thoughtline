package dashboard

import "testing"

// TestRoadmap_HasAllExpectedMilestones verifies that the roadmap YAML is
// present and correctly loaded. Milestones M0–M6 must all exist and be
// in the expected status states.
func TestRoadmap_HasAllExpectedMilestones(t *testing.T) {
	got := Roadmap()
	want := []string{"M0", "M1", "M2", "M3", "M4", "M5", "M6"}
	if len(got) != len(want) {
		t.Fatalf("expected %d milestones, got %d", len(want), len(got))
	}
	for i, m := range got {
		if m.ID != want[i] {
			t.Errorf("milestone %d: got %s want %s", i, m.ID, want[i])
		}
	}
	// M0–M5 are done at v0.0.1.
	for _, m := range got[:6] {
		if m.Status != "done" {
			t.Errorf("milestone %s should be done, got %s", m.ID, m.Status)
		}
	}
	// M6 is deferred.
	if got[6].Status != "deferred" {
		t.Errorf("M6 should be deferred, got %s", got[6].Status)
	}
}

// TestStatusGlyph_KnownStatuses verifies the StatusGlyph mapping for all
// known status values.
func TestStatusGlyph_KnownStatuses(t *testing.T) {
	tests := []struct {
		status string
		glyph  string
	}{
		{"done", "✓"},
		{"next", "→"},
		{"deferred", "·"},
		{"in-progress", "*"},
		{"unknown", "?"},
	}
	for _, tt := range tests {
		if got := StatusGlyph(tt.status); got != tt.glyph {
			t.Errorf("StatusGlyph(%q) = %q, want %q", tt.status, got, tt.glyph)
		}
	}
}

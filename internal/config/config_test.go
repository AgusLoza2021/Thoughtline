package config_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/config"
)

// silence slog warnings in tests that trigger the unknown-field path
func silenceSlog() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// Task 2.1 — TestConfig_Defaults
func TestConfig_Defaults(t *testing.T) {
	t.Helper()
	cfg := config.DefaultGlobal()

	if cfg.Ranking.BM25 != 0.40 {
		t.Errorf("Ranking.BM25: want 0.40, got %v", cfg.Ranking.BM25)
	}
	if cfg.Ranking.Semantic != 0.25 {
		t.Errorf("Ranking.Semantic: want 0.25, got %v", cfg.Ranking.Semantic)
	}
	if cfg.Ranking.Recency != 0.15 {
		t.Errorf("Ranking.Recency: want 0.15, got %v", cfg.Ranking.Recency)
	}
	if cfg.Ranking.Importance != 0.15 {
		t.Errorf("Ranking.Importance: want 0.15, got %v", cfg.Ranking.Importance)
	}
	if cfg.Ranking.Access != 0.05 {
		t.Errorf("Ranking.Access: want 0.05, got %v", cfg.Ranking.Access)
	}
	if cfg.Decay.HalfLifeDays != 30 {
		t.Errorf("Decay.HalfLifeDays: want 30, got %v", cfg.Decay.HalfLifeDays)
	}
	if cfg.Decay.Floor != 0.1 {
		t.Errorf("Decay.Floor: want 0.1, got %v", cfg.Decay.Floor)
	}
	if cfg.Graph.AutoLinkOnContradiction != false {
		t.Errorf("Graph.AutoLinkOnContradiction: want false, got %v", cfg.Graph.AutoLinkOnContradiction)
	}
	if cfg.Graph.AutoLinkThreshold != 0.85 {
		t.Errorf("Graph.AutoLinkThreshold: want 0.85, got %v", cfg.Graph.AutoLinkThreshold)
	}
	if cfg.Graph.MaxNeighborsPerNode != 50 {
		t.Errorf("Graph.MaxNeighborsPerNode: want 50, got %v", cfg.Graph.MaxNeighborsPerNode)
	}
	if cfg.Simulation.Enabled != false {
		t.Errorf("Simulation.Enabled: want false, got %v", cfg.Simulation.Enabled)
	}
	if cfg.Simulation.SyntheticDecay != 1.0 {
		t.Errorf("Simulation.SyntheticDecay: want 1.0, got %v", cfg.Simulation.SyntheticDecay)
	}
	if cfg.Simulation.TimeMultiplier != 1.0 {
		t.Errorf("Simulation.TimeMultiplier: want 1.0, got %v", cfg.Simulation.TimeMultiplier)
	}
	if cfg.Simulation.AllowDestructive != false {
		t.Errorf("Simulation.AllowDestructive: want false, got %v", cfg.Simulation.AllowDestructive)
	}
}

// Task 2.3 — TestEffective_EmptyOverride
func TestEffective_EmptyOverride(t *testing.T) {
	global := config.DefaultGlobal()

	cases := []struct {
		name     string
		override json.RawMessage
	}{
		{"nil override", nil},
		{"empty bytes", json.RawMessage{}},
		{"empty object", json.RawMessage(`{}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eff, err := config.Effective(global, tc.override)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if eff.Ranking.BM25 != global.Ranking.BM25 {
				t.Errorf("Ranking.BM25: want %v, got %v", global.Ranking.BM25, eff.Ranking.BM25)
			}
			if eff.Decay.HalfLifeDays != global.Decay.HalfLifeDays {
				t.Errorf("Decay.HalfLifeDays: want %v, got %v", global.Decay.HalfLifeDays, eff.Decay.HalfLifeDays)
			}
			if eff.Simulation.Enabled != global.Simulation.Enabled {
				t.Errorf("Simulation.Enabled: want %v, got %v", global.Simulation.Enabled, eff.Simulation.Enabled)
			}
		})
	}
}

// Task 2.4 — TestEffective_PartialOverride
func TestEffective_PartialOverride(t *testing.T) {
	global := config.DefaultGlobal()
	override := json.RawMessage(`{"decay":{"half_life_days":1}}`)

	eff, err := config.Effective(global, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Decay.HalfLifeDays != 1 {
		t.Errorf("Decay.HalfLifeDays: want 1, got %v", eff.Decay.HalfLifeDays)
	}
	// Ranking.BM25 must be inherited from global
	if eff.Ranking.BM25 != 0.40 {
		t.Errorf("Ranking.BM25: want 0.40 (inherited), got %v", eff.Ranking.BM25)
	}
	// Decay.Floor must be inherited from global
	if eff.Decay.Floor != 0.1 {
		t.Errorf("Decay.Floor: want 0.1 (inherited), got %v", eff.Decay.Floor)
	}
}

// Task 2.4 — covers "global change propagates to non-overriding brain" scenario
func TestEffective_GlobalChange_Propagates(t *testing.T) {
	global := config.DefaultGlobal()
	global.Ranking.BM25 = 0.50 // global updated

	override := json.RawMessage(`{}`) // brain has no override

	eff, err := config.Effective(global, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Ranking.BM25 != 0.50 {
		t.Errorf("Ranking.BM25: want 0.50 (global change), got %v", eff.Ranking.BM25)
	}
}

// Task 2.4 — covers "brain override survives global change" scenario
func TestEffective_Override_SurvivesGlobalChange(t *testing.T) {
	global := config.DefaultGlobal()
	global.Ranking.BM25 = 0.50 // global updated

	override := json.RawMessage(`{"ranking":{"bm25":0.60}}`) // brain explicitly overrides

	eff, err := config.Effective(global, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Ranking.BM25 != 0.60 {
		t.Errorf("Ranking.BM25: want 0.60 (brain override), got %v", eff.Ranking.BM25)
	}
}

// Task 2.6 — TestEffective_UnknownField
func TestEffective_UnknownField(t *testing.T) {
	// Redirect slog to discard so the warning doesn't clutter test output,
	// but we still exercise the path.
	silenceSlog()

	global := config.DefaultGlobal()
	override := json.RawMessage(`{"experimental":{"foo":true},"decay":{"floor":0.2}}`)

	eff, err := config.Effective(global, override)
	if err != nil {
		t.Fatalf("expected no error for unknown field, got: %v", err)
	}
	// Known field from override must be applied
	if eff.Decay.Floor != 0.2 {
		t.Errorf("Decay.Floor: want 0.2, got %v", eff.Decay.Floor)
	}
	// Other fields from global
	if eff.Ranking.BM25 != 0.40 {
		t.Errorf("Ranking.BM25: want 0.40 (global), got %v", eff.Ranking.BM25)
	}
}

// Task 2.8 — TestEffective_MalformedJSON
func TestEffective_MalformedJSON(t *testing.T) {
	global := config.DefaultGlobal()
	override := json.RawMessage(`not-json`)

	_, err := config.Effective(global, override)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// Verify DefaultGlobalJSON returns valid JSON that unmarshals back to DefaultConfig
func TestDefaultGlobalJSON_RoundTrip(t *testing.T) {
	jsonStr := config.DefaultGlobalJSON()
	if jsonStr == "" {
		t.Fatal("DefaultGlobalJSON returned empty string")
	}

	var cfg config.Config
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("DefaultGlobalJSON is not valid JSON: %v", err)
	}

	defaults := config.DefaultGlobal()
	if cfg.Ranking.BM25 != defaults.Ranking.BM25 {
		t.Errorf("round-trip Ranking.BM25: want %v, got %v", defaults.Ranking.BM25, cfg.Ranking.BM25)
	}
	if cfg.Decay.HalfLifeDays != defaults.Decay.HalfLifeDays {
		t.Errorf("round-trip Decay.HalfLifeDays: want %v, got %v", defaults.Decay.HalfLifeDays, cfg.Decay.HalfLifeDays)
	}
}

// DeepMerge with nested patch — only changes targeted field
func TestEffective_DeepMergePatch(t *testing.T) {
	global := config.DefaultGlobal()
	override := json.RawMessage(`{"ranking":{"bm25":0.6}}`)

	eff, err := config.Effective(global, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Ranking.BM25 != 0.6 {
		t.Errorf("Ranking.BM25: want 0.6, got %v", eff.Ranking.BM25)
	}
	// Other ranking fields must stay at global defaults
	if eff.Ranking.Semantic != 0.25 {
		t.Errorf("Ranking.Semantic: want 0.25 (untouched), got %v", eff.Ranking.Semantic)
	}
	if eff.Ranking.Recency != 0.15 {
		t.Errorf("Ranking.Recency: want 0.15 (untouched), got %v", eff.Ranking.Recency)
	}
	if eff.Ranking.Importance != 0.15 {
		t.Errorf("Ranking.Importance: want 0.15 (untouched), got %v", eff.Ranking.Importance)
	}
	if eff.Ranking.Access != 0.05 {
		t.Errorf("Ranking.Access: want 0.05 (untouched), got %v", eff.Ranking.Access)
	}
}

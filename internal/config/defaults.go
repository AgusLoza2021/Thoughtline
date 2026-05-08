package config

import "encoding/json"

// DefaultGlobal returns the compiled-in default Config values.
// These are the authoritative defaults for every brain unless explicitly
// overridden in brain.config_json.
func DefaultGlobal() Config {
	return Config{
		Ranking: Ranking{
			BM25:       0.40,
			Semantic:   0.25,
			Recency:    0.15,
			Importance: 0.15,
			Access:     0.05,
		},
		Decay: Decay{
			HalfLifeDays: 30,
			Floor:        0.1,
		},
		Graph: Graph{
			AutoLinkOnContradiction: false,
			AutoLinkThreshold:       0.85,
			MaxNeighborsPerNode:     50,
		},
		Simulation: Simulation{
			Enabled:          false,
			SyntheticDecay:   1.0,
			TimeMultiplier:   1.0,
			AllowDestructive: false,
		},
	}
}

// DefaultGlobalJSON returns the JSON representation of DefaultGlobal().
// Used by migrateV4 to seed the global_config singleton row.
// Panics if marshalling fails — this is a compile-time constant, not user input.
func DefaultGlobalJSON() string {
	b, err := json.Marshal(DefaultGlobal())
	if err != nil {
		panic("config: failed to marshal default config: " + err.Error())
	}
	return string(b)
}

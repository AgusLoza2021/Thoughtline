// Package config defines the two-level configuration model for Thoughtline.
// A singleton global_config row holds system-wide defaults; each brain's
// config_json holds only field overrides. Effective config is computed as
// deep-merge(global, brain_overrides) at read time (lazy, no cache in Phase 0).
package config

// Ranking holds the weights used when scoring memories in search results.
// The weights should sum to 1.0; callers are responsible for enforcing this
// constraint — the struct itself does not validate.
type Ranking struct {
	BM25       float64 `json:"bm25"`
	Semantic   float64 `json:"semantic"`
	Recency    float64 `json:"recency"`
	Importance float64 `json:"importance"`
	Access     float64 `json:"access"`
}

// Decay defines how quickly a memory's relevance decays over time.
type Decay struct {
	HalfLifeDays int     `json:"half_life_days"`
	Floor        float64 `json:"floor"`
}

// Graph controls automatic link creation and graph traversal limits.
type Graph struct {
	AutoLinkOnContradiction bool    `json:"auto_link_on_contradiction"`
	AutoLinkThreshold       float64 `json:"auto_link_threshold"`
	MaxNeighborsPerNode     int     `json:"max_neighbors_per_node"`
}

// Simulation configures synthetic time-acceleration features.
// Only relevant when Enabled = true; production brains leave this at defaults.
type Simulation struct {
	Enabled          bool    `json:"enabled"`
	SyntheticDecay   float64 `json:"synthetic_decay"`
	TimeMultiplier   float64 `json:"time_multiplier"`
	AllowDestructive bool    `json:"allow_destructive"`
}

// Config is the top-level configuration structure for a brain.
// Both the global singleton and per-brain overrides use this type.
// JSON field names are snake_case to match the database column and wire format.
type Config struct {
	Ranking    Ranking    `json:"ranking"`
	Decay      Decay      `json:"decay"`
	Graph      Graph      `json:"graph"`
	Simulation Simulation `json:"simulation"`
}

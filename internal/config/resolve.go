package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// knownTopLevelKeys is the set of top-level JSON keys that Config recognises.
// Any key not in this set is logged as a warning (forward-compatibility path).
var knownTopLevelKeys = map[string]struct{}{
	"ranking":    {},
	"decay":      {},
	"graph":      {},
	"simulation": {},
}

// Effective computes the effective Config for a brain by deep-merging the
// per-brain override JSON on top of the global config.
//
// Rules:
//   - nil / empty / "{}" override → return global unchanged.
//   - Unknown top-level keys in the override → log slog.Warn, continue (do not error).
//   - Nested objects merge field-by-field (patch wins on scalars; sibling fields
//     in the base are preserved). This is a shallow two-level merge matching the
//     Config shape (Config → Section → scalar), which is sufficient for our schema.
//   - Malformed JSON → return an error with "invalid JSON" in the message.
func Effective(global Config, override json.RawMessage) (Config, error) {
	// Fast path: empty or trivially empty override.
	if len(override) == 0 || string(override) == "{}" {
		return global, nil
	}

	// Unmarshal override into a raw map so we can inspect unknown keys.
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(override, &patch); err != nil {
		return Config{}, fmt.Errorf("config.Effective: invalid JSON in override: %w", err)
	}

	if len(patch) == 0 {
		return global, nil
	}

	// Warn about unknown top-level keys (forward-compatibility).
	for k := range patch {
		if _, known := knownTopLevelKeys[k]; !known {
			slog.Warn("config: unknown field in brain override — ignored", "key", k)
		}
	}

	// Marshal global to a map[string]any so we can merge patch on top.
	globalBytes, err := json.Marshal(global)
	if err != nil {
		return Config{}, fmt.Errorf("config.Effective: marshal global: %w", err)
	}
	var base map[string]any
	if err := json.Unmarshal(globalBytes, &base); err != nil {
		return Config{}, fmt.Errorf("config.Effective: unmarshal global to map: %w", err)
	}

	// Deep-merge: for each patch key, merge into base.
	for k, rawVal := range patch {
		if _, known := knownTopLevelKeys[k]; !known {
			// Unknown key — already warned; skip so it doesn't end up in Config.
			continue
		}

		// Both base[k] and rawVal should be objects (our Config is two-level deep).
		// Unmarshal the patch section.
		var patchSection map[string]any
		if err := json.Unmarshal(rawVal, &patchSection); err != nil {
			// If it's not an object (shouldn't happen for valid config), replace wholesale.
			var scalar any
			if err2 := json.Unmarshal(rawVal, &scalar); err2 == nil {
				base[k] = scalar
			}
			continue
		}

		// Merge patchSection into the existing base section.
		if baseSection, ok := base[k].(map[string]any); ok {
			for field, val := range patchSection {
				baseSection[field] = val
			}
			base[k] = baseSection
		} else {
			// Base section didn't exist or wasn't a map — just use patch.
			base[k] = patchSection
		}
	}

	// Marshal merged map back to JSON, then unmarshal into Config.
	mergedBytes, err := json.Marshal(base)
	if err != nil {
		return Config{}, fmt.Errorf("config.Effective: marshal merged: %w", err)
	}
	var result Config
	if err := json.Unmarshal(mergedBytes, &result); err != nil {
		return Config{}, fmt.Errorf("config.Effective: unmarshal result: %w", err)
	}
	return result, nil
}

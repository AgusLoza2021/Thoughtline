package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// keysFileName is the optional sidecar that curates, per entry, the words a
// developer actually types when doing that work.
//
// WHY a sidecar instead of editing the entries: the recipe files are
// hand-written and stay untouched. Measured on real commits, the lexical gate
// fired on 5 of 20 commits and the top hit was right in 3, because the gate
// guessed an entry's subject from its title and tags. A curated key list is a
// claim by the author ("this entry is what you want when you type
// camera-shake"), so a key match is intent, not an accident of shared words.
const keysFileName = "keys.yaml"

// Tag prefixes Sync adds for curated keys. They are tags rather than new
// columns because check reads rows through Search, which already returns tags,
// and because a re-sync is what carries a keys.yaml edit into the database.
//
// A tag cannot start with "!", so a negative key ("!spine-anatomy") travels
// under its own prefix instead.
const (
	KeyTagPrefix    = "key:"
	NegKeyTagPrefix = "notkey:"
)

// entryKeys is one line of keys.yaml: the keys that select an entry and the
// negative keys that must veto it.
type entryKeys struct {
	Keys    []string
	NegKeys []string
}

// parseKeysFile reads the "slug: [a, b, !c]" shape of keys.yaml.
//
// It is deliberately not YAML: the file holds one flat line per entry, so a
// hundred-line parser would add a dependency for nothing. Unknown lines are
// skipped, because a half-written keys.yaml must not fail a sync.
func parseKeysFile(raw []byte) map[string]entryKeys {
	out := make(map[string]entryKeys)
	for _, line := range strings.Split(normalizeText(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		slug, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			continue
		}
		out[slug] = parseKeyList(value)
	}
	return out
}

// parseKeyList turns "[camera-shake, screen-shake, !spine-anatomy]" into the
// two lists. A leading "!" marks a negative key.
func parseKeyList(value string) entryKeys {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")

	var out entryKeys
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		negative := strings.HasPrefix(part, "!")
		key := normalizeKey(strings.TrimPrefix(part, "!"))
		if key == "" {
			continue
		}
		if negative {
			out.NegKeys = append(out.NegKeys, key)
			continue
		}
		out.Keys = append(out.Keys, key)
	}
	return out
}

// normalizeKey folds a key into the shape the matcher and the tag regex both
// accept: lowercase, words joined by hyphens, nothing else.
func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastHyphen := true // a leading hyphen would make the tag invalid
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '\t' || r == '_' || r == ':' || r == '-':
			if !lastHyphen {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// applyKeys attaches the curated keys to the scanned entries. A corpus without
// keys.yaml keeps working exactly as before: its entries carry no keys and the
// check falls back to the coverage gate.
//
// A curated entry's revision hash absorbs its keys, because that hash is what
// Sync compares to decide a row is unchanged (via the footer). Without it,
// editing keys.yaml would never reach an already-synced database: the recipe
// file did not change, so every row would report "unchanged" and keep its old
// tags.
func applyKeys(dir string, entries []Entry) {
	raw, err := os.ReadFile(filepath.Join(dir, keysFileName))
	if err != nil {
		return
	}
	keys := parseKeysFile(raw)
	for i := range entries {
		k, ok := keys[entries[i].Slug]
		if !ok {
			continue
		}
		entries[i].Keys = k.Keys
		entries[i].NegKeys = k.NegKeys
		entries[i].Hash = withKeys(entries[i].Hash, k)
	}
}

// withKeys folds the curated keys into an entry's revision hash.
func withKeys(hash string, k entryKeys) string {
	sum := sha256.New()
	sum.Write([]byte(hash))
	sum.Write([]byte("\nkeys:"))
	sum.Write([]byte(strings.Join(k.Keys, ",")))
	sum.Write([]byte("\nnotkeys:"))
	sum.Write([]byte(strings.Join(k.NegKeys, ",")))
	return hex.EncodeToString(sum.Sum(nil))
}

// keyGate decides a hit by its curated keys.
//
// keyed reports that the row carries keys at all (so the caller must not fall
// back to the coverage gate), matched lists the positive keys that fired, and
// blocked says a negative key fired: a visible distractor ("spine-anatomy" on a
// skeleton-recolour entry) vetoes the entry even when a positive one matched.
func keyGate(tags, tokens []string) (matched []string, blocked, keyed bool) {
	words := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		words[tok] = true
	}
	for _, tag := range tags {
		if key, ok := strings.CutPrefix(tag, KeyTagPrefix); ok {
			keyed = true
			if keyMatches(key, words) {
				matched = append(matched, key)
			}
			continue
		}
		if key, ok := strings.CutPrefix(tag, NegKeyTagPrefix); ok {
			keyed = true
			if keyMatches(key, words) {
				blocked = true
			}
		}
	}
	return matched, blocked, keyed
}

// keyMatches reports whether every hyphen part of a key appears as a whole word
// in the query. Whole words means the same tokenizer the query went through, so
// "CameraShake" in a path satisfies camera-shake while "hits" never satisfies
// hit.
//
// A trailing "s" is tolerated in both directions: a developer writes "fill
// bars" and "sprites", the key says fill-bar and sprite. Nothing else is
// stemmed, so keys stay predictable.
func keyMatches(key string, words map[string]bool) bool {
	parts := strings.Split(key, "-")
	found := false
	for _, p := range parts {
		if p == "" {
			continue
		}
		if !partPresent(p, words) {
			return false
		}
		found = true
	}
	return found
}

func partPresent(part string, words map[string]bool) bool {
	if words[part] {
		return true
	}
	if words[part+"s"] {
		return true
	}
	if trimmed := strings.TrimSuffix(part, "s"); trimmed != part && len(trimmed) > 1 && words[trimmed] {
		return true
	}
	return false
}

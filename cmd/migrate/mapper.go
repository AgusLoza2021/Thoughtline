package migrate

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// topicKeyRe mirrors the pattern in internal/memory/validate.go so the mapper
// can clear invalid topic_keys without importing the private regex.
var topicKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]{1,128}$`)

// MapType converts an Engram type string to a Thoughtline memory.Type plus any
// provenance tags to add to the migrated row.
//
// Mapping table (spec engram-migration Requirement 4):
//
//	bugfix       → bugfix          (no tag)
//	preference   → preference      (no tag)
//	decision     → decision        (no tag)
//	architecture → architecture    (no tag)
//	pattern      → convention      + origin-type:pattern
//	config       → convention      + origin-type:config
//	discovery    → convention      + origin-type:discovery
//	manual       → convention      + origin-type:manual
//	<anything>   → convention      + origin-type:<anything>
func MapType(engramType string) (memory.Type, []string, error) {
	switch engramType {
	case "bugfix":
		return memory.TypeBugfix, nil, nil
	case "preference":
		return memory.TypePreference, nil, nil
	case "decision":
		return memory.TypeDecision, nil, nil
	case "architecture":
		return memory.TypeArchitecture, nil, nil
	default:
		// Unknown or coercible types fall through to convention + provenance tag.
		tag := "origin-type:" + engramType
		return memory.TypeConvention, []string{tag}, nil
	}
}

// MapTimestamp parses an ISO-8601 string and returns the Unix epoch in
// milliseconds. Always interprets the input in UTC to avoid timezone
// corruption.
//
// Accepted formats:
//   - ISO 8601 with T separator: "2024-03-15T10:30:00Z", "2024-03-15T10:30:00+00:00"
//   - SQLite default datetime: "2024-03-15 10:30:00" (space separator, no timezone; assumed UTC)
func MapTimestamp(iso string) (int64, error) {
	if iso == "" {
		return 0, fmt.Errorf("migrate: empty timestamp string")
	}
	// Try all formats Engram may write. The SQLite default format uses a space
	// separator with no timezone marker; we treat it as UTC since Engram stores
	// all timestamps in UTC internally.
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05", // SQLite default datetime format (UTC assumed)
	}
	var (
		t   time.Time
		err error
	)
	for _, f := range formats {
		t, err = time.Parse(f, iso)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, fmt.Errorf("migrate: parse timestamp %q: %w", iso, err)
	}
	return t.UTC().UnixMilli(), nil
}

// MapRow applies all column and type mappings, returning a memory.Memory ready
// for storage.Save(). It enforces the following transformations:
//
//   - project is lowercased (Engram stores mixed case in some old rows)
//   - preference scope is forced to personal regardless of source
//   - session_id is always cleared (Engram session IDs are not UUIDv7)
//   - NormalizedHash is left zero (storage.Save() recomputes it)
//   - title > 200 runes is truncated at rune boundary; logger receives one WARN
//   - content > MaxContentBytes causes MapRow to return an error
//   - invalid topic_key (fails Thoughtline's regex) is silently cleared
//   - origin-type:<x> tag is merged into Tags for coerced types
//   - bogus scope value returns error
func MapRow(o EngramObservation, logger Logger) (memory.Memory, error) {
	// --- Type + provenance tags ---
	tlType, originTags, err := MapType(o.Type)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("migrate: map type for %s: %w", o.SyncID, err)
	}

	// --- Scope ---
	var scope memory.Scope
	switch o.Scope {
	case "project":
		scope = memory.ScopeProject
	case "personal":
		scope = memory.ScopePersonal
	default:
		return memory.Memory{}, fmt.Errorf("migrate: unknown scope %q for sync_id %s", o.Scope, o.SyncID)
	}
	// preference must always be personal (Engram may have stored it wrong)
	if tlType == memory.TypePreference {
		scope = memory.ScopePersonal
	}

	// --- Project ---
	project := strings.ToLower(strings.TrimSpace(o.Project))

	// --- Title (truncate at rune boundary if > 200 runes) ---
	title := o.Title
	if utf8.RuneCountInString(title) > memory.MaxTitleChars {
		runes := []rune(title)
		title = string(runes[:memory.MaxTitleChars])
		logger.Log("WARN", "title truncated", map[string]string{
			"sync_id":  o.SyncID,
			"original": fmt.Sprintf("%d runes", len(runes)),
		})
	}

	// --- Content (reject if oversized; never silently truncate) ---
	if len(o.Content) > memory.MaxContentBytes {
		return memory.Memory{}, fmt.Errorf("migrate: content too large (%d bytes > %d) for sync_id %s",
			len(o.Content), memory.MaxContentBytes, o.SyncID)
	}

	// --- TopicKey (clear if it fails Thoughtline's regex) ---
	topicKey := o.TopicKey
	if topicKey != "" && !topicKeyRe.MatchString(topicKey) {
		topicKey = ""
	}

	// --- Tags (merge origin-type tags into any existing tags) ---
	tags := append([]string(nil), o.Tags...) // copy to avoid aliasing
	tags = append(tags, originTags...)

	// --- Timestamps ---
	createdAt, err := MapTimestamp(o.CreatedAt)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("migrate: parse created_at for %s: %w", o.SyncID, err)
	}
	updatedAt, err := MapTimestamp(o.UpdatedAt)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("migrate: parse updated_at for %s: %w", o.SyncID, err)
	}

	return memory.Memory{
		// SyncID is intentionally NOT set here — writer.go handles the
		// post-UPDATE that stamps the Engram sync_id after storage.Save().
		Project:       project,
		Scope:         scope,
		Type:          tlType,
		TopicKey:      topicKey,
		Title:         title,
		Content:       o.Content,
		Tags:          tags,
		RevisionCount: o.RevisionCount,
		CreatedAt:     time.UnixMilli(createdAt),
		UpdatedAt:     time.UnixMilli(updatedAt),
		// SessionID left empty — Engram IDs are not UUIDv7; would fail FK.
		// NormalizedHash left zero — storage.Save() recomputes via sha256.
	}, nil
}

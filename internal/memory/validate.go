package memory

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// topicKeyRe enforces the topic-key shape this server accepts: lowercase letters,
// digits, slash, underscore, hyphen; must start with [a-z0-9]; total length
// 2..129 chars (1 lead + 1..128 trail). docs/design/memory-domain.md also permits
// a dot, for version numbers, which this regex rejects.
var topicKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]{1,128}$`)

// tagRe enforces the tag shape: lowercase, digit, colon, underscore, hyphen;
// must start with [a-z0-9]; total length 1..41 chars.
var tagRe = regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]{0,40}$`)

// Validate enforces the domain rules the server applies to every save. It is
// narrower than the catalogue in docs/design/memory-domain.md: it admits the
// eleven types this package closes and the two scopes below.
// It mutates nothing — callers can apply trimming themselves before saving.
//
// Errors returned are sentinel values from this package (e.g. ErrEmptyTitle).
// Use errors.Is to match.
func Validate(m Memory) error {
	if !m.Type.Valid() {
		return ErrInvalidType
	}
	if !m.Scope.Valid() {
		return ErrInvalidScope
	}

	// Type/scope coupling. The catalogue permits a `global` scope; this does
	// not, so every non-preference save is project-scoped.
	if m.Type == TypePreference && m.Scope != ScopePersonal {
		return ErrPreferenceMustBePersonal
	}
	if m.Type != TypePreference && m.Scope != ScopeProject {
		return ErrNonPreferenceMustBeProject
	}

	if strings.TrimSpace(m.Project) == "" {
		return ErrEmptyProject
	}

	title := strings.TrimSpace(m.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if utf8.RuneCountInString(title) > MaxTitleChars {
		return ErrTitleTooLong
	}

	if strings.TrimSpace(m.Content) == "" {
		return ErrEmptyContent
	}
	if len(m.Content) > MaxContentBytes {
		return ErrContentTooLong
	}

	if m.TopicKey != "" && !topicKeyRe.MatchString(m.TopicKey) {
		return ErrInvalidTopicKey
	}

	for _, tag := range m.Tags {
		if !tagRe.MatchString(tag) {
			return ErrInvalidTag
		}
	}

	// Optional session linkage. Empty = unattached. When set, must be a
	// valid UUIDv7 — the same shape Session.ID takes.
	if m.SessionID != "" {
		parsed, err := uuid.Parse(m.SessionID)
		if err != nil || parsed.Version() != 7 {
			return ErrInvalidSessionID
		}
	}

	return nil
}

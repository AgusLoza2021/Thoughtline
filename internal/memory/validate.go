package memory

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// topicKeyRe enforces the topic-key shape documented in
// docs/design/memory-domain.md: lowercase letters, digits, slash, underscore,
// hyphen; must start with [a-z0-9]; total length 2..129 chars (1 lead + 1..128
// trail).
var topicKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]{1,128}$`)

// tagRe enforces the tag shape: lowercase, digit, colon, underscore, hyphen;
// must start with [a-z0-9]; total length 1..41 chars.
var tagRe = regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]{0,40}$`)

// Validate enforces every domain rule documented in docs/design/memory-domain.md.
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

	// Type/scope coupling — see memory-domain.md "Validation rules" #2.
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

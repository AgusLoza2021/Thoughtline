// Package brain provides the first-class Brain entity for Thoughtline.
// A Brain is a scoped, namespaced container for memories. Every memory
// belongs to exactly one brain; no cross-brain reads are permitted by default.
package brain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Kind is the trust classification of a brain.
type Kind string

const (
	// KindReal represents the user's actual cognition / production data.
	KindReal Kind = "real"
	// KindSynthetic represents experiment or generated data.
	KindSynthetic Kind = "synthetic"
	// KindSandbox represents a playground brain, safe to wipe.
	KindSandbox Kind = "sandbox"
)

// Valid reports whether k is one of the three accepted kind values.
func (k Kind) Valid() bool {
	switch k {
	case KindReal, KindSynthetic, KindSandbox:
		return true
	}
	return false
}

// Brain is a first-class entity that owns a collection of memories.
type Brain struct {
	ID          int64
	Slug        string
	DisplayName string
	Kind        Kind
	Description string
	ConfigJSON  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

// slugRe enforces the slug rules:
//   - only lowercase a-z, digits 0-9, and hyphens
//   - length 1–64
//   - must not start or end with a hyphen
//
// The two alternatives handle single-char slugs (no hyphen possible) and
// multi-char slugs (first and last char must be alphanum).
var slugRe = regexp.MustCompile(`^(?:[a-z0-9]|[a-z0-9][a-z0-9-]{0,62}[a-z0-9])$`)

// ValidateSlug returns ErrInvalidSlug wrapping the slug value if the slug
// does not satisfy the brain slug rules. Returns nil on success.
func ValidateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("%w: %q", ErrInvalidSlug, slug)
	}
	return nil
}

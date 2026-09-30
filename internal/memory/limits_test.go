package memory

import (
	"strings"
	"testing"
)

// TestLimitsMatchPatterns pins the one claim that used to be two.
//
// MaxTopicKeyLen and MaxTagLen are exported so callers can render a message, and
// the patterns in validate.go are what actually reject. They disagreed once — the
// constants said 128 and 40 while the patterns allowed 129 and 41 — so a caller
// reading the constant could be one character over the limit it named. This walks
// the boundary in both directions: the longest value the constant promises is
// accepted, and one character more is rejected.
func TestLimitsMatchPatterns(t *testing.T) {
	longest := func(n int) string { return "a" + strings.Repeat("b", n-1) }

	t.Run("topic_key", func(t *testing.T) {
		m := validMemory()
		m.TopicKey = longest(MaxTopicKeyLen)
		if err := Validate(m); err != nil {
			t.Errorf("topic_key of exactly MaxTopicKeyLen (%d) should be accepted, got %v", MaxTopicKeyLen, err)
		}

		m.TopicKey = longest(MaxTopicKeyLen + 1)
		if err := Validate(m); err == nil {
			t.Errorf("topic_key of MaxTopicKeyLen+1 (%d) should be rejected", MaxTopicKeyLen+1)
		}
	})

	t.Run("tag", func(t *testing.T) {
		m := validMemory()
		m.Tags = []string{longest(MaxTagLen)}
		if err := Validate(m); err != nil {
			t.Errorf("tag of exactly MaxTagLen (%d) should be accepted, got %v", MaxTagLen, err)
		}

		m.Tags = []string{longest(MaxTagLen + 1)}
		if err := Validate(m); err == nil {
			t.Errorf("tag of MaxTagLen+1 (%d) should be rejected", MaxTagLen+1)
		}
	})
}

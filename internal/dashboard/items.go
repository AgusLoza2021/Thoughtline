package dashboard

import (
	"fmt"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// memoryItem adapts a storage.SearchResult for the bubbles/list default
// delegate. Title/Description satisfy list.DefaultItem; FilterValue lets
// the list's built-in filter input grep across the noisy fields.
type memoryItem struct {
	r storage.SearchResult
}

func (m memoryItem) FilterValue() string {
	return m.r.Title + " " + m.r.TopicKey + " " + string(m.r.Type) + " " + m.r.Project
}

func (m memoryItem) Title() string { return truncate(m.r.Title, 80) }

func (m memoryItem) Description() string {
	parts := fmt.Sprintf("[%s] %s", m.r.Type, m.r.Project)
	if m.r.TopicKey != "" {
		parts += " · " + m.r.TopicKey
	}
	parts += " · " + relativeTime(m.r.UpdatedAt)
	return parts
}

type sessionItem struct {
	s memory.Session
}

func (si sessionItem) FilterValue() string {
	return si.s.ID + " " + si.s.Project + " " + si.s.AgentLabel
}

func (si sessionItem) Title() string {
	state := "open"
	if si.s.EndedAt != nil {
		state = "closed"
	}
	label := si.s.AgentLabel
	if label == "" {
		label = "—"
	}
	return fmt.Sprintf("%s · %s", state, label)
}

func (si sessionItem) Description() string {
	return fmt.Sprintf("%s · started %s · %s",
		si.s.ID, relativeTime(si.s.StartedAt), si.s.Project)
}

type tagItem struct {
	tc storage.TagCount
}

func (ti tagItem) FilterValue() string { return ti.tc.Tag }
func (ti tagItem) Title() string       { return ti.tc.Tag }
func (ti tagItem) Description() string {
	return fmt.Sprintf("%d use%s", ti.tc.Count, plural(ti.tc.Count))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// relativeTime renders t as a short human-readable duration relative to now.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

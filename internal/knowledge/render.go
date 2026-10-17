package knowledge

import (
	"fmt"
	"strings"
)

// CompactFooter closes a compact answer. The point of the two-step output is
// that a check is cheap to read and the entry itself is one call away, so the
// last line names the tool that turns an id into the full recipe.
const CompactFooter = "Call tl_get_observation(id) for the full recipe."

// FormatHitLine is the compact, one-line form of a hit:
//
//	[knowledge] Camera Shake as an Additive Local Position Offset — keys: camera-shake — id 42 (knowledge/camera-shake)
//
// It carries exactly what a second call needs (the id and the topic key) plus
// the curated keys that made the entry match, so an agent can judge the hit
// without spending prompt budget on sections it may not need.
func FormatHitLine(h CheckHit) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", h.Source, h.Title)
	if len(h.MatchedKeys) > 0 {
		fmt.Fprintf(&b, " — keys: %s", strings.Join(h.MatchedKeys, ","))
	}
	fmt.Fprintf(&b, " — id %d (%s)", h.ID, h.TopicKey)
	return b.String()
}

// FormatHitDetail is the full block a caller gets with detail=true: the gate,
// the validation step and the limits, which is what tuning and hook mode read.
func FormatHitDetail(h CheckHit) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### [%s] %s (%s, score %.2f) — coverage %d",
		h.Source, h.Title, h.TopicKey, h.Score, h.Coverage)
	if len(h.Matched) > 0 {
		fmt.Fprintf(&b, ": %s", strings.Join(h.Matched, ", "))
	}
	fmt.Fprintf(&b, " (weight %.1f)\n", h.Weight)
	if len(h.MatchedKeys) > 0 {
		fmt.Fprintf(&b, "**Keys** %s\n", strings.Join(h.MatchedKeys, ", "))
	}
	writeSection(&b, "Applies when", h.AppliesWhen)
	writeSection(&b, "Validate", h.Validate)
	writeSection(&b, "Limits", h.Limits)
	if len(h.Stacks) > 0 {
		fmt.Fprintf(&b, "**Stacks**: %s\n", strings.Join(h.Stacks, ", "))
	}
	return b.String()
}

// writeSection omits an empty section instead of printing a heading with
// nothing under it: an empty "Limits" block reads as "there are no limits",
// which is a lie.
func writeSection(b *strings.Builder, label, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	fmt.Fprintf(b, "**%s**\n%s\n", label, body)
}

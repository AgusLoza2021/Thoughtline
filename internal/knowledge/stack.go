package knowledge

import (
	"regexp"
	"sort"
)

// stackMarker is one foreign-stack signal. These corpora were distilled from a
// different project, so entries referencing SVN, FiberCtrl, Tactile, belt
// paths, Spine or NGUI are usually inapplicable and must be filterable.
type stackMarker struct {
	ID string
	Re *regexp.Regexp
	// CoOccurs gates ID on a second pattern. "spine" alone is ordinary English
	// prose ("the spine of the book"), so it only counts as the Spine runtime
	// when skeleton terminology is nearby.
	CoOccurs *regexp.Regexp
}

var stackMarkers = []stackMarker{
	{ID: "svn", Re: regexp.MustCompile(`(?i)\bsvn\b`)},
	{ID: "fiber", Re: regexp.MustCompile(`(?i)FiberCtrl|\bfibers?\b`)},
	{ID: "tactile", Re: regexp.MustCompile(`(?i)\btactile\b`)},
	{ID: "belt", Re: regexp.MustCompile(`(?i)\bbelts?\b`)},
	{
		ID:       "spine",
		Re:       regexp.MustCompile(`(?i)\bspine\b`),
		CoOccurs: regexp.MustCompile(`(?i)skeleton|setskin|setslot`),
	},
	{ID: "level-editor", Re: regexp.MustCompile(`(?i)level[\s-]+editor`)},
	{ID: "ngui", Re: regexp.MustCompile(`(?i)\bngui\b`)},
}

// DetectStacks returns the sorted, deduped ids of every foreign stack the text
// mentions. Ids are stable so they can be persisted in config and compared
// across runs.
func DetectStacks(text string) []string {
	var ids []string
	for _, marker := range stackMarkers {
		if !marker.Re.MatchString(text) {
			continue
		}
		if marker.CoOccurs != nil && !marker.CoOccurs.MatchString(text) {
			continue
		}
		ids = append(ids, marker.ID)
	}
	sort.Strings(ids)
	return ids
}

// KnownStacks lists every stack id DetectStacks can emit, for config
// validation and CLI help.
func KnownStacks() []string {
	ids := make([]string, 0, len(stackMarkers))
	for _, marker := range stackMarkers {
		ids = append(ids, marker.ID)
	}
	sort.Strings(ids)
	return ids
}

package knowledge

import (
	"reflect"
	"testing"
)

func TestDetectStacks(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"svn word", "Commit through SVN externals first.", []string{"svn"}},
		{"svn substring is not a match", "The svnserve config lives elsewhere.", nil},
		{"fiber class", "Wrap the routine in FiberCtrl.", []string{"fiber"}},
		{"fiber plural", "Two fibers run in parallel.", []string{"fiber"}},
		{"tactile", "Tactile create-art workflow.", []string{"tactile"}},
		{"belt", "Move the piece along the belt path.", []string{"belt"}},
		{"level editor", "Open the Level Editor tool window.", []string{"level-editor"}},
		{"level-editor hyphen", "Authored in the level-editor.", []string{"level-editor"}},
		{"ngui", "Legacy NGUI atlas.", []string{"ngui"}},
		{"spine with skeleton", "Spine skeleton data drives the character.", []string{"spine"}},
		{"spine with setskin", "Call setSkin on the Spine component.", []string{"spine"}},
		{"spine prose only", "Chapter one: the spine of the book.", nil},
		{"no markers", "A plain Unity camera recipe.", nil},
		{"multiple markers sorted", "SVN outside, FiberCtrl inside, NGUI on top.", []string{"fiber", "ngui", "svn"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectStacks(tc.text)
			if len(got) != len(tc.want) {
				t.Fatalf("DetectStacks(%q) = %v, want %v", tc.text, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("DetectStacks(%q) = %v, want %v", tc.text, got, tc.want)
				}
			}
		})
	}
}

func TestKnownStacksCoversEverythingDetectable(t *testing.T) {
	known := KnownStacks()
	for _, id := range known {
		if id == "" {
			t.Fatal("empty stack id")
		}
	}
	// Every marker must be discoverable through the exported list, otherwise
	// config validation would reject a stack the parser can report.
	var detected []string
	for _, marker := range stackMarkers {
		if _, ok := indexOf(known, marker.ID); !ok {
			t.Fatalf("%q missing from KnownStacks() = %v", marker.ID, known)
		}
		detected = append(detected, marker.ID)
	}
	if len(detected) != len(known) {
		t.Fatalf("KnownStacks() has ids with no marker: %v", known)
	}
	if !reflect.DeepEqual(known, []string{"belt", "fiber", "level-editor", "ngui", "spine", "svn", "tactile"}) {
		t.Fatalf("KnownStacks() = %v", known)
	}
}

func indexOf(haystack []string, needle string) (int, bool) {
	for i, s := range haystack {
		if s == needle {
			return i, true
		}
	}
	return -1, false
}

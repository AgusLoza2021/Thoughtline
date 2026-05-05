package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestTopTags_AggregatesAcrossMemories(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tags.db")
	st, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	seed := []memory.Memory{
		{Project: "p1", Scope: memory.ScopeProject, Type: memory.TypeDecision, Title: "a", Content: "x", Tags: []string{"go", "sqlite"}},
		{Project: "p1", Scope: memory.ScopeProject, Type: memory.TypeDecision, Title: "b", Content: "y", Tags: []string{"go", "tui"}},
		{Project: "p2", Scope: memory.ScopeProject, Type: memory.TypeDecision, Title: "c", Content: "z", Tags: []string{"go"}},
	}
	for _, m := range seed {
		if _, _, err := st.Save(context.Background(), m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	all, err := st.TopTags(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("TopTags all: %v", err)
	}
	if len(all) == 0 || all[0].Tag != "go" || all[0].Count != 3 {
		t.Errorf("expected 'go' top with count 3, got %+v", all)
	}

	scoped, err := st.TopTags(context.Background(), "p1", 0)
	if err != nil {
		t.Fatalf("TopTags p1: %v", err)
	}
	var goCount int
	for _, tc := range scoped {
		if tc.Tag == "go" {
			goCount = tc.Count
		}
	}
	if goCount != 2 {
		t.Errorf("p1 'go' count = %d, want 2", goCount)
	}
}

func TestTopTags_Empty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tags.db")
	st, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	out, err := st.TopTags(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("TopTags: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty, got %+v", out)
	}
}

package storage

import (
	"context"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// saveTopic saves a minimal valid memory under an explicit topic_key and
// returns it. Keeps the prefix tests focused on matching, not on fixtures.
func saveTopic(t *testing.T, st *Storage, brainID int64, topicKey string) memory.Memory {
	t.Helper()
	m := sampleMemory()
	m.TopicKey = topicKey
	saved, _, err := st.Save(context.Background(), brainID, m)
	if err != nil {
		t.Fatalf("save %q: %v", topicKey, err)
	}
	return saved
}

func TestListByTopicPrefix_MatchesPrefixOrderedByTopicKey(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	saveTopic(t, st, brainID, "knowledge/zeta")
	saveTopic(t, st, brainID, "knowledge/alpha")
	saveTopic(t, st, brainID, "knowledge/deep/nested")
	saveTopic(t, st, brainID, "scene/inn-cellar")

	got, err := st.ListByTopicPrefix(ctx, brainID, "knowledge/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []string{"knowledge/alpha", "knowledge/deep/nested", "knowledge/zeta"}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, key := range want {
		if got[i].TopicKey != key {
			t.Errorf("row %d topic_key = %q, want %q", i, got[i].TopicKey, key)
		}
		if got[i].Content != "world/static for chairs and walls; world/interactive for the cellar door." {
			t.Errorf("row %d content not loaded", i)
		}
	}
}

// A raw LIKE would treat "_" as "any single character" and match knowledge/axb,
// which would make a prune of one namespace delete rows of another.
func TestListByTopicPrefix_EscapesLikeWildcards(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	saveTopic(t, st, brainID, "knowledge/a_b")
	saveTopic(t, st, brainID, "knowledge/axb")
	saveTopic(t, st, brainID, "knowledge/a%b")
	saveTopic(t, st, brainID, "knowledge/ayb")

	got, err := st.ListByTopicPrefix(ctx, brainID, "knowledge/a_b")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].TopicKey != "knowledge/a_b" {
		t.Fatalf("got %+v, want only knowledge/a_b", got)
	}

	got, err = st.ListByTopicPrefix(ctx, brainID, "knowledge/a%b")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].TopicKey != "knowledge/a%b" {
		t.Fatalf("got %+v, want only knowledge/a%%b", got)
	}
}

func TestListByTopicPrefix_ExcludesSoftDeletedAndOtherBrains(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)
	otherID, err := st.ResolveOrCreateBrainID(ctx, "other-project")
	if err != nil {
		t.Fatalf("resolve other brain: %v", err)
	}

	keep := saveTopic(t, st, brainID, "knowledge/keep")
	drop := saveTopic(t, st, brainID, "knowledge/drop")
	if err := st.SoftDelete(ctx, brainID, drop.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	foreign := sampleMemory()
	foreign.Project = "other-project"
	foreign.TopicKey = "knowledge/foreign"
	if _, _, err := st.Save(ctx, otherID, foreign); err != nil {
		t.Fatalf("save foreign: %v", err)
	}

	got, err := st.ListByTopicPrefix(ctx, brainID, "knowledge/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != keep.ID {
		t.Fatalf("got %+v, want only the active row of this brain", got)
	}
}

func TestListByTopicPrefix_RequiresBrain(t *testing.T) {
	st := newTestStorage(t)
	if _, err := st.ListByTopicPrefix(context.Background(), 0, "knowledge/"); err == nil {
		t.Fatal("expected an error for brainID 0")
	}
}

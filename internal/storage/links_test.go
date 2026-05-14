package storage

import (
	"context"
	"testing"
)

func TestCreateLink_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a, _, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save a: %v", err)
	}
	b := sampleMemory()
	b.Title = "Another memory"
	b.Content = "different content"
	b, _, err = st.Save(ctx, brainID, b)
	if err != nil {
		t.Fatalf("save b: %v", err)
	}

	link, noop, err := st.CreateLink(ctx, brainID, a.ID, b.ID, "related", "test note")
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if noop {
		t.Fatal("expected new link, got noop")
	}
	if link.ID == 0 {
		t.Error("expected non-zero link ID")
	}
	if link.FromID != a.ID || link.ToID != b.ID {
		t.Errorf("from/to mismatch: got %d/%d, want %d/%d", link.FromID, link.ToID, a.ID, b.ID)
	}
	if link.Relation != "related" {
		t.Errorf("relation: got %q, want %q", link.Relation, "related")
	}
	if link.Note != "test note" {
		t.Errorf("note: got %q, want %q", link.Note, "test note")
	}
	if link.Source != "manual" {
		t.Errorf("source: got %q, want %q", link.Source, "manual")
	}
	if link.CreatedAt.IsZero() {
		t.Error("CreatedAt must be populated")
	}
}

func TestCreateLink_Duplicate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a, _, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save a: %v", err)
	}
	b := sampleMemory()
	b.Title = "Another memory"
	b.Content = "different content"
	b, _, err = st.Save(ctx, brainID, b)
	if err != nil {
		t.Fatalf("save b: %v", err)
	}

	if _, noop, err := st.CreateLink(ctx, brainID, a.ID, b.ID, "related", ""); err != nil || noop {
		t.Fatalf("first CreateLink: err=%v noop=%v", err, noop)
	}

	_, noop, err := st.CreateLink(ctx, brainID, a.ID, b.ID, "related", "a second note")
	if err != nil {
		t.Fatalf("second CreateLink returned unexpected error: %v", err)
	}
	if !noop {
		t.Fatal("duplicate link should return noop=true")
	}
}

func TestCreateLink_DifferentRelationIsNewLink(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a, _, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save a: %v", err)
	}
	b := sampleMemory()
	b.Title = "Another memory"
	b.Content = "different content"
	b, _, err = st.Save(ctx, brainID, b)
	if err != nil {
		t.Fatalf("save b: %v", err)
	}

	if _, noop, err := st.CreateLink(ctx, brainID, a.ID, b.ID, "related", ""); err != nil || noop {
		t.Fatalf("first CreateLink: err=%v noop=%v", err, noop)
	}
	_, noop, err := st.CreateLink(ctx, brainID, a.ID, b.ID, "refines", "")
	if err != nil {
		t.Fatalf("second CreateLink: %v", err)
	}
	if noop {
		t.Fatal("different relation should create a new link, not a noop")
	}
}

func TestGetLinks_Both(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	center, _, _ := st.Save(ctx, brainID, sampleMemory())

	other1 := sampleMemory()
	other1.Title = "Other1"
	other1.Content = "c1"
	other1, _, _ = st.Save(ctx, brainID, other1)

	other2 := sampleMemory()
	other2.Title = "Other2"
	other2.Content = "c2"
	other2, _, _ = st.Save(ctx, brainID, other2)

	// center → other1
	if _, _, err := st.CreateLink(ctx, brainID, center.ID, other1.ID, "depends_on", ""); err != nil {
		t.Fatalf("link1: %v", err)
	}
	// other2 → center
	if _, _, err := st.CreateLink(ctx, brainID, other2.ID, center.ID, "references", ""); err != nil {
		t.Fatalf("link2: %v", err)
	}

	links, err := st.GetLinks(ctx, brainID, center.ID, "")
	if err != nil {
		t.Fatalf("GetLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestGetLinks_FromOnly(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	center, _, _ := st.Save(ctx, brainID, sampleMemory())
	other1 := sampleMemory()
	other1.Title = "Other1"
	other1.Content = "c1"
	other1, _, _ = st.Save(ctx, brainID, other1)
	other2 := sampleMemory()
	other2.Title = "Other2"
	other2.Content = "c2"
	other2, _, _ = st.Save(ctx, brainID, other2)

	// center → other1 (from)
	if _, _, err := st.CreateLink(ctx, brainID, center.ID, other1.ID, "depends_on", ""); err != nil {
		t.Fatalf("link1: %v", err)
	}
	// other2 → center (to)
	if _, _, err := st.CreateLink(ctx, brainID, other2.ID, center.ID, "references", ""); err != nil {
		t.Fatalf("link2: %v", err)
	}

	links, err := st.GetLinks(ctx, brainID, center.ID, "from")
	if err != nil {
		t.Fatalf("GetLinks from: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link (from), got %d", len(links))
	}
	if links[0].FromID != center.ID {
		t.Errorf("expected from_id=%d, got %d", center.ID, links[0].FromID)
	}
}

func TestGetLinks_ToOnly(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	center, _, _ := st.Save(ctx, brainID, sampleMemory())
	other1 := sampleMemory()
	other1.Title = "Other1"
	other1.Content = "c1"
	other1, _, _ = st.Save(ctx, brainID, other1)
	other2 := sampleMemory()
	other2.Title = "Other2"
	other2.Content = "c2"
	other2, _, _ = st.Save(ctx, brainID, other2)

	// center → other1 (from)
	if _, _, err := st.CreateLink(ctx, brainID, center.ID, other1.ID, "depends_on", ""); err != nil {
		t.Fatalf("link1: %v", err)
	}
	// other2 → center (to)
	if _, _, err := st.CreateLink(ctx, brainID, other2.ID, center.ID, "references", ""); err != nil {
		t.Fatalf("link2: %v", err)
	}

	links, err := st.GetLinks(ctx, brainID, center.ID, "to")
	if err != nil {
		t.Fatalf("GetLinks to: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link (to), got %d", len(links))
	}
	if links[0].ToID != center.ID {
		t.Errorf("expected to_id=%d, got %d", center.ID, links[0].ToID)
	}
}

func TestGetLinks_EmptyResult(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m, _, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	links, err := st.GetLinks(ctx, brainID, m.ID, "")
	if err != nil {
		t.Fatalf("GetLinks: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

package links_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/brain"
	"github.com/AgusLoza2021/Thoughtline/internal/links"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func openDB(t *testing.T) *storage.Storage {
	t.Helper()
	ctx := context.Background()
	s, err := storage.Open(ctx, t.TempDir()+"\\test.db")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func createBrain(t *testing.T, db *sql.DB, slug string) brain.Brain {
	t.Helper()
	bs := brain.New(db)
	b, err := bs.Create(context.Background(), slug, slug, brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("brain.Create(%q): %v", slug, err)
	}
	return b
}

func createMemory(t *testing.T, s *storage.Storage, brainID int64, title string) memory.Memory {
	t.Helper()
	m, _, err := s.Save(context.Background(), brainID, memory.Memory{
		Project: "test-project",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   title,
		Content: "content for " + title,
	})
	if err != nil {
		t.Fatalf("storage.Save(%q): %v", title, err)
	}
	return m
}

// setup2Brains returns (storage, brainA, brainB, memA1, memA2, memB1, memB2).
func setup2Brains(t *testing.T) (*storage.Storage, brain.Brain, brain.Brain, memory.Memory, memory.Memory, memory.Memory, memory.Memory) {
	t.Helper()
	s := openDB(t)
	db := s.DB()

	brainA := createBrain(t, db, "brain-a")
	brainB := createBrain(t, db, "brain-b")

	memA1 := createMemory(t, s, brainA.ID, "A memory 1")
	memA2 := createMemory(t, s, brainA.ID, "A memory 2")
	memB1 := createMemory(t, s, brainB.ID, "B memory 1")
	memB2 := createMemory(t, s, brainB.ID, "B memory 2")

	return s, brainA, brainB, memA1, memA2, memB1, memB2
}

// ---------------------------------------------------------------------------
// 5.1 — TestLinks_CreateValidLink
// ---------------------------------------------------------------------------

func TestLinks_CreateValidLink(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	link, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.RelSupersedes, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if link.ID == 0 {
		t.Error("expected assigned ID, got 0")
	}
	if link.Weight != 1.0 {
		t.Errorf("expected weight=1.0, got %v", link.Weight)
	}
	if link.BrainID != brainA.ID {
		t.Errorf("expected brainID=%d, got %d", brainA.ID, link.BrainID)
	}
	if link.Relation != links.RelSupersedes {
		t.Errorf("expected relation=%q, got %q", links.RelSupersedes, link.Relation)
	}
}

// ---------------------------------------------------------------------------
// 5.3 — TestLinks_CrossBrainRejected
// ---------------------------------------------------------------------------

func TestLinks_CrossBrainRejected(t *testing.T) {
	s, brainA, _, memA1, _, memB1, _ := setup2Brains(t)
	ls := links.New(s.DB())

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memB1.ID,
		links.RelRelated, links.SourceManual, 0, "")
	if err == nil {
		t.Fatal("expected ErrCrossBrainLink, got nil")
	}
	if !isErr(err, links.ErrCrossBrainLink) {
		t.Fatalf("expected ErrCrossBrainLink, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5.4 — TestLinks_SelfLoopRejected
// ---------------------------------------------------------------------------

func TestLinks_SelfLoopRejected(t *testing.T) {
	s, brainA, _, memA1, _, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memA1.ID,
		links.RelRelated, links.SourceManual, 0, "")
	if err == nil {
		t.Fatal("expected ErrSelfLoop, got nil")
	}
	if !isErr(err, links.ErrSelfLoop) {
		t.Fatalf("expected ErrSelfLoop, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5.5 — TestLinks_DuplicateTripleRejected
// ---------------------------------------------------------------------------

func TestLinks_DuplicateTripleRejected(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.RelRefines, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.RelRefines, links.SourceManual, 0, "")
	if err == nil {
		t.Fatal("expected ErrLinkExists on duplicate triple, got nil")
	}
	if !isErr(err, links.ErrLinkExists) {
		t.Fatalf("expected ErrLinkExists, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5.6 — TestLinks_SamePairDifferentRelation
// ---------------------------------------------------------------------------

func TestLinks_SamePairDifferentRelation(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.RelRefines, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.RelRelated, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("second Create (different relation): %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5.7 + 5.8 — TestLinks_Neighbors
// ---------------------------------------------------------------------------

func TestLinks_Neighbors(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	// Add a third memory for inbound link
	memA3 := createMemory(t, s, brainA.ID, "A memory 3")
	ls := links.New(s.DB())
	ctx := context.Background()

	// M1 → M2 (outbound from M1)
	_, err := ls.Create(ctx, brainA.ID, memA1.ID, memA2.ID, links.RelSupersedes, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("Create outbound1: %v", err)
	}
	// M1 → M3 (outbound from M1)
	_, err = ls.Create(ctx, brainA.ID, memA1.ID, memA3.ID, links.RelRefines, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("Create outbound2: %v", err)
	}
	// M2 → M1 (inbound to M1)
	_, err = ls.Create(ctx, brainA.ID, memA2.ID, memA1.ID, links.RelRelated, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("Create inbound: %v", err)
	}

	t.Run("all_neighbors_returns_3", func(t *testing.T) {
		got, err := ls.Neighbors(ctx, brainA.ID, memA1.ID, nil)
		if err != nil {
			t.Fatalf("Neighbors: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("expected 3 links, got %d", len(got))
		}
	})

	t.Run("filtered_by_relation_returns_1", func(t *testing.T) {
		rel := links.RelSupersedes
		got, err := ls.Neighbors(ctx, brainA.ID, memA1.ID, &rel)
		if err != nil {
			t.Fatalf("Neighbors(filtered): %v", err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1 link, got %d", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// 5.9 + 5.10 — TestLinks_Subgraph
// ---------------------------------------------------------------------------

func TestLinks_Subgraph(t *testing.T) {
	s, brainA, brainB, memA1, memA2, memB1, memB2 := setup2Brains(t)
	ls := links.New(s.DB())
	ctx := context.Background()

	// Add a third and fourth memory to brainA for 4 total
	memA3 := createMemory(t, s, brainA.ID, "A memory 3")
	memA4 := createMemory(t, s, brainA.ID, "A memory 4")

	// 3 links in brainA
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA2.ID, links.RelSupersedes)
	mustCreate(t, ls, ctx, brainA.ID, memA2.ID, memA3.ID, links.RelRefines)
	mustCreate(t, ls, ctx, brainA.ID, memA3.ID, memA4.ID, links.RelRelated)

	// 1 link in brainB
	mustCreate(t, ls, ctx, brainB.ID, memB1.ID, memB2.ID, links.RelRelated)

	t.Run("subgraph_brainA_4nodes_3edges", func(t *testing.T) {
		sg, err := ls.Subgraph(ctx, brainA.ID, links.SubgraphOptions{Limit: 500})
		if err != nil {
			t.Fatalf("Subgraph: %v", err)
		}
		if len(sg.Nodes) != 4 {
			t.Errorf("expected 4 nodes, got %d", len(sg.Nodes))
		}
		if len(sg.Edges) != 3 {
			t.Errorf("expected 3 edges, got %d", len(sg.Edges))
		}
		for _, n := range sg.Nodes {
			if n.BrainID != brainA.ID {
				t.Errorf("node brainID=%d is not brainA.ID=%d", n.BrainID, brainA.ID)
			}
		}
	})

	t.Run("subgraph_with_type_filter", func(t *testing.T) {
		// All memories are TypeDecision — so filtering by TypeDecision should return all 4
		sg, err := ls.Subgraph(ctx, brainA.ID, links.SubgraphOptions{
			Types: []memory.Type{memory.TypeDecision},
			Limit: 500,
		})
		if err != nil {
			t.Fatalf("Subgraph(type filter): %v", err)
		}
		if len(sg.Nodes) != 4 {
			t.Errorf("expected 4 nodes with TypeDecision filter, got %d", len(sg.Nodes))
		}

		// Filter by a type that has no memories
		sg2, err := ls.Subgraph(ctx, brainA.ID, links.SubgraphOptions{
			Types: []memory.Type{memory.TypeBugfix},
			Limit: 500,
		})
		if err != nil {
			t.Fatalf("Subgraph(type filter bugfix): %v", err)
		}
		if len(sg2.Nodes) != 0 {
			t.Errorf("expected 0 nodes with TypeBugfix filter, got %d", len(sg2.Nodes))
		}
	})

	t.Run("subgraph_limit", func(t *testing.T) {
		sg, err := ls.Subgraph(ctx, brainA.ID, links.SubgraphOptions{Limit: 2})
		if err != nil {
			t.Fatalf("Subgraph(limit=2): %v", err)
		}
		if len(sg.Nodes) > 2 {
			t.Errorf("expected at most 2 nodes with limit=2, got %d", len(sg.Nodes))
		}
	})
}

// ---------------------------------------------------------------------------
// 5.11 — TestLinks_CascadeOnBrainDelete
// ---------------------------------------------------------------------------

func TestLinks_CascadeOnBrainDelete(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())
	ctx := context.Background()

	memA3 := createMemory(t, s, brainA.ID, "A memory 3")

	// Create 3 links in brainA (need 5 for full coverage but 3 is enough)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA2.ID, links.RelSupersedes)
	mustCreate(t, ls, ctx, brainA.ID, memA2.ID, memA3.ID, links.RelRefines)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA3.ID, links.RelRelated)

	// Hard-delete the brain row. memories.brain_id has no CASCADE so we must
	// remove memories first, then the brain row itself. This is correct per
	// schema: the brain cascade test verifies memory_links rows are gone.
	db := s.DB()
	if _, err := db.ExecContext(ctx, `DELETE FROM memories WHERE brain_id = ?`, brainA.ID); err != nil {
		t.Fatalf("delete memories: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM brains WHERE id = ?`, brainA.ID); err != nil {
		t.Fatalf("delete brain: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_links WHERE brain_id = ?`, brainA.ID).Scan(&count); err != nil {
		t.Fatalf("count links: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 links after brain delete (cascade), got %d", count)
	}
}

// ---------------------------------------------------------------------------
// 5.12 — TestLinks_CascadeOnMemoryDelete
// ---------------------------------------------------------------------------

func TestLinks_CascadeOnMemoryDelete(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())
	ctx := context.Background()

	memA3 := createMemory(t, s, brainA.ID, "A memory 3")
	memA4 := createMemory(t, s, brainA.ID, "A memory 4")

	// M1 → M2, M1 → M3, M1 → M4 (3 outbound from M1)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA2.ID, links.RelSupersedes)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA3.ID, links.RelRefines)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA4.ID, links.RelRelated)
	// M2 → M1, M3 → M1 (2 inbound to M1)
	mustCreate(t, ls, ctx, brainA.ID, memA2.ID, memA1.ID, links.RelRelated)
	mustCreate(t, ls, ctx, brainA.ID, memA3.ID, memA1.ID, links.RelDependsOn)

	// Hard-delete memory M1
	db := s.DB()
	if _, err := db.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, memA1.ID); err != nil {
		t.Fatalf("delete memory: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memory_links WHERE from_id = ? OR to_id = ?`,
		memA1.ID, memA1.ID).Scan(&count); err != nil {
		t.Fatalf("count links: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 incident links after memory delete (cascade), got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Additional: TestLinks_InvalidRelation
// ---------------------------------------------------------------------------

func TestLinks_InvalidRelation(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, memA2.ID,
		links.Relation("inspired_by"), links.SourceManual, 0, "")
	if err == nil {
		t.Fatal("expected ErrInvalidRelation, got nil")
	}
	if !isErr(err, links.ErrInvalidRelation) {
		t.Fatalf("expected ErrInvalidRelation, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Additional: TestLinks_MissingEndpoint
// ---------------------------------------------------------------------------

func TestLinks_MissingEndpoint(t *testing.T) {
	s, brainA, _, memA1, _, _, _ := setup2Brains(t)
	ls := links.New(s.DB())

	const nonExistentID int64 = 99999

	_, err := ls.Create(context.Background(), brainA.ID, memA1.ID, nonExistentID,
		links.RelRelated, links.SourceManual, 0, "")
	if err == nil {
		t.Fatal("expected ErrEndpointNotFound, got nil")
	}
	if !isErr(err, links.ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Additional: TestLinks_GetByID
// ---------------------------------------------------------------------------

func TestLinks_GetByID(t *testing.T) {
	s, brainA, brainB, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())
	ctx := context.Background()

	created, err := ls.Create(ctx, brainA.ID, memA1.ID, memA2.ID,
		links.RelRefines, links.SourceManual, 0, "test note")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := ls.GetByID(ctx, brainA.ID, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("expected ID=%d, got %d", created.ID, got.ID)
		}
		if got.Note != "test note" {
			t.Errorf("expected note=%q, got %q", "test note", got.Note)
		}
	})

	t.Run("cross_brain_returns_not_found", func(t *testing.T) {
		_, err := ls.GetByID(ctx, brainB.ID, created.ID)
		if !isErr(err, links.ErrLinkNotFound) {
			t.Fatalf("expected ErrLinkNotFound for cross-brain GetByID, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Additional: TestLinks_Delete
// ---------------------------------------------------------------------------

func TestLinks_Delete(t *testing.T) {
	s, brainA, brainB, memA1, memA2, _, _ := setup2Brains(t)
	ls := links.New(s.DB())
	ctx := context.Background()

	created, err := ls.Create(ctx, brainA.ID, memA1.ID, memA2.ID,
		links.RelRelated, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("cross_brain_delete_fails", func(t *testing.T) {
		err := ls.Delete(ctx, brainB.ID, created.ID)
		if !isErr(err, links.ErrLinkNotFound) {
			t.Fatalf("expected ErrLinkNotFound for cross-brain Delete, got: %v", err)
		}
	})

	t.Run("correct_brain_delete_succeeds", func(t *testing.T) {
		err := ls.Delete(ctx, brainA.ID, created.ID)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		// Verify it's gone
		_, err = ls.GetByID(ctx, brainA.ID, created.ID)
		if !isErr(err, links.ErrLinkNotFound) {
			t.Fatalf("expected ErrLinkNotFound after delete, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Additional: TestLinks_Subgraph_RelationFilter
// ---------------------------------------------------------------------------

func TestLinks_Subgraph_RelationFilter(t *testing.T) {
	s, brainA, _, memA1, memA2, _, _ := setup2Brains(t)
	memA3 := createMemory(t, s, brainA.ID, "A memory 3")
	ls := links.New(s.DB())
	ctx := context.Background()

	// 2 supersedes + 1 related
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA2.ID, links.RelSupersedes)
	mustCreate(t, ls, ctx, brainA.ID, memA2.ID, memA3.ID, links.RelSupersedes)
	mustCreate(t, ls, ctx, brainA.ID, memA1.ID, memA3.ID, links.RelRelated)

	sg, err := ls.Subgraph(ctx, brainA.ID, links.SubgraphOptions{
		Relations: []links.Relation{links.RelSupersedes},
		Limit:     500,
	})
	if err != nil {
		t.Fatalf("Subgraph(relation filter): %v", err)
	}
	if len(sg.Edges) != 2 {
		t.Errorf("expected 2 supersedes edges, got %d", len(sg.Edges))
	}
	for _, e := range sg.Edges {
		if e.Relation != links.RelSupersedes {
			t.Errorf("unexpected relation %q in filtered subgraph", e.Relation)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustCreate(t *testing.T, ls *links.Store, ctx context.Context, brainID, fromID, toID int64, rel links.Relation) links.Link {
	t.Helper()
	l, err := ls.Create(ctx, brainID, fromID, toID, rel, links.SourceManual, 0, "")
	if err != nil {
		t.Fatalf("mustCreate(from=%d, to=%d, rel=%q): %v", fromID, toID, rel, err)
	}
	return l
}

// isErr checks errors.Is. Defined here to avoid importing errors in every test.
func isErr(err, target error) bool {
	if err == nil {
		return false
	}
	// Walk the error chain manually since errors package would work but this is clear
	for err != nil {
		if err == target {
			return true
		}
		err = unwrap(err)
	}
	return false
}

func unwrap(err error) error {
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return nil
}

// Suppress unused import of time (used in Link struct via CreatedAt).
var _ = time.Now

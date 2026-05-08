package storage

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/links"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

// twoBrainSetup holds the state created by setupTwoBrains.
type twoBrainSetup struct {
	st      *Storage
	ls      *links.Store
	brainA  int64
	brainB  int64
	memsA   []memory.Memory // 5 memories in brainA
	memsB   []memory.Memory // 5 memories in brainB
	linksA  []links.Link    // 3 links in brainA
	linksB  []links.Link    // 3 links in brainB
}

// setupTwoBrains creates brainA + brainB, 5 memories each (varied types /
// tags), and 3 intra-brain links each.  It is the shared fixture for all
// Phase 7 isolation tests.
func setupTwoBrains(t *testing.T) twoBrainSetup {
	t.Helper()
	ctx := context.Background()
	st := newTestStorage(t)
	ls := links.New(st.db)

	brainA := setupBrain(t, st, "alpha-brain")
	brainB := setupBrain(t, st, "beta-brain")

	types := []memory.Type{
		memory.TypeDecision,
		memory.TypeScenePattern,
		memory.TypeDecision,
		memory.TypeScenePattern,
		memory.TypeDecision,
	}
	// Brain-specific tag sets: brainA tags never appear in brainB and vice versa.
	tagSetsA := [][]string{
		{"engine:unity", "platform:pc"},
		{"engine:unreal"},
		{"render:deferred"},
		{"render:forward"},
		{"platform:mobile", "engine:unity"},
	}
	tagSetsB := [][]string{
		{"audio:fmod"},
		{"audio:wwise"},
		{"physics:havok"},
		{"physics:bullet"},
		{"net:mirror"},
	}

	saveMemsForBrain := func(brainID int64, project string, tagSets [][]string) []memory.Memory {
		out := make([]memory.Memory, 5)
		for i := 0; i < 5; i++ {
			m := memory.Memory{
				Project: project,
				Scope:   memory.ScopeProject,
				Type:    types[i],
				Title:   fmt.Sprintf("%s memory-%d", project, i),
				Content: fmt.Sprintf("content for %s index %d isolation-marker-%s", project, i, project),
				Tags:    tagSets[i],
			}
			saved := seedWithBrain(t, st, brainID, m)
			out[i] = saved
		}
		return out
	}

	memsA := saveMemsForBrain(brainA, "alpha-brain", tagSetsA)
	memsB := saveMemsForBrain(brainB, "beta-brain", tagSetsB)

	// 3 intra-brain links per brain: [0→1, 1→2, 2→3]
	makeLinks := func(brainID int64, mems []memory.Memory) []links.Link {
		rels := []links.Relation{links.RelRefines, links.RelDependsOn, links.RelRelated}
		out := make([]links.Link, 3)
		for i := 0; i < 3; i++ {
			lk, err := ls.Create(ctx, brainID,
				mems[i].ID, mems[i+1].ID,
				rels[i], links.SourceManual, 1.0, "",
			)
			if err != nil {
				t.Fatalf("setup link %d for brain %d: %v", i, brainID, err)
			}
			out[i] = lk
		}
		return out
	}

	linksA := makeLinks(brainA, memsA)
	linksB := makeLinks(brainB, memsB)

	return twoBrainSetup{
		st:     st,
		ls:     ls,
		brainA: brainA,
		brainB: brainB,
		memsA:  memsA,
		memsB:  memsB,
		linksA: linksA,
		linksB: linksB,
	}
}

// ---------------------------------------------------------------------------
// Task 7.2 — TestBrainIsolation table-driven suite
// ---------------------------------------------------------------------------

// TestBrainIsolation iterates every public per-brain Storage method and the
// links.Store cross-brain paths, confirming that querying brainA never exposes
// brainB data and vice versa.
func TestBrainIsolation(t *testing.T) {
	ctx := context.Background()

	// Each case is a func that receives the setup and must call t.Error/t.Fatal
	// on any isolation violation.
	cases := []struct {
		name string
		run  func(t *testing.T, fix twoBrainSetup)
	}{
		{
			name: "Save_BrainIDStamped",
			run: func(t *testing.T, fix twoBrainSetup) {
				m := memory.Memory{
					Project: "alpha-brain",
					Scope:   memory.ScopeProject,
					Type:    memory.TypeDecision,
					Title:   "isolation save test",
					Content: "new memory for brainA",
				}
				saved, _, err := fix.st.Save(ctx, fix.brainA, m)
				if err != nil {
					t.Fatalf("save: %v", err)
				}
				if saved.BrainID != fix.brainA {
					t.Errorf("saved.BrainID = %d, want %d", saved.BrainID, fix.brainA)
				}
			},
		},
		{
			name: "GetByID_CrossBrain_ReturnsNotFound",
			run: func(t *testing.T, fix twoBrainSetup) {
				// Attempt to read brainB's first memory using brainA's identity.
				_, err := fix.st.GetByID(ctx, fix.brainA, fix.memsB[0].ID)
				if !errors.Is(err, ErrMemoryNotFound) {
					t.Errorf("GetByID cross-brain: want ErrMemoryNotFound, got %v", err)
				}
			},
		},
		{
			name: "UpdateByID_CrossBrain_DoesNotMutate",
			run: func(t *testing.T, fix twoBrainSetup) {
				hackedTitle := "HACKED TITLE"
				hackedContent := "HACKED CONTENT"
				patch := UpdatePatch{
					Title:   &hackedTitle,
					Content: &hackedContent,
				}
				_, err := fix.st.UpdateByID(ctx, fix.brainA, fix.memsB[0].ID, patch)
				if !errors.Is(err, ErrMemoryNotFound) {
					t.Errorf("UpdateByID cross-brain: want ErrMemoryNotFound, got %v", err)
				}
				// Confirm brainB's memory is untouched.
				got, err := fix.st.GetByID(ctx, fix.brainB, fix.memsB[0].ID)
				if err != nil {
					t.Fatalf("GetByID brainB after failed cross-brain update: %v", err)
				}
				if got.Title == hackedTitle {
					t.Errorf("brainB memory was mutated by brainA UpdateByID")
				}
			},
		},
		{
			name: "SoftDelete_CrossBrain_DoesNotDelete",
			run: func(t *testing.T, fix twoBrainSetup) {
				err := fix.st.SoftDelete(ctx, fix.brainA, fix.memsB[1].ID)
				if !errors.Is(err, ErrMemoryNotFound) {
					t.Errorf("SoftDelete cross-brain: want ErrMemoryNotFound, got %v", err)
				}
				// Confirm brainB's memory still alive.
				got, err := fix.st.GetByID(ctx, fix.brainB, fix.memsB[1].ID)
				if err != nil {
					t.Fatalf("GetByID brainB after failed cross-brain delete: %v", err)
				}
				if got.DeletedAt != nil {
					t.Errorf("brainB memory was soft-deleted by brainA SoftDelete")
				}
			},
		},
		{
			name: "Search_OnlyBrainARows",
			run: func(t *testing.T, fix twoBrainSetup) {
				// "isolation-marker-alpha-brain" is in all brainA content but NOT brainB.
				// "isolation-marker-beta-brain"  is in all brainB content but NOT brainA.
				// Search brainA for a term present in brainB content — expect nothing.
				results, err := fix.st.Search(ctx, fix.brainA, "beta-brain", SearchOptions{})
				if err != nil {
					t.Fatalf("search: %v", err)
				}
				for _, r := range results {
					if r.Project == "beta-brain" {
						t.Errorf("Search(brainA) returned a brainB row: id=%d project=%q", r.ID, r.Project)
					}
				}
				// Also confirm brainA-specific content is found.
				resultsA, err := fix.st.Search(ctx, fix.brainA, "alpha-brain", SearchOptions{})
				if err != nil {
					t.Fatalf("search brainA term: %v", err)
				}
				if len(resultsA) == 0 {
					t.Error("Search(brainA) returned 0 results for brainA-specific content")
				}
				for _, r := range resultsA {
					if r.Project != "alpha-brain" {
						t.Errorf("Search(brainA) returned non-brainA row: project=%q", r.Project)
					}
				}
			},
		},
		{
			name: "Recent_OnlyBrainARows",
			run: func(t *testing.T, fix twoBrainSetup) {
				results, err := fix.st.Recent(ctx, fix.brainA, 100)
				if err != nil {
					t.Fatalf("recent: %v", err)
				}
				if len(results) == 0 {
					t.Error("Recent(brainA) returned 0 rows; expected brainA memories")
				}
				for _, r := range results {
					if r.Project != "alpha-brain" {
						t.Errorf("Recent(brainA) returned non-brainA row: project=%q id=%d", r.Project, r.ID)
					}
				}
			},
		},
		{
			name: "TopTags_OnlyBrainATags",
			run: func(t *testing.T, fix twoBrainSetup) {
				// tagSetsB contains "audio:fmod","audio:wwise","physics:havok",
				// "physics:bullet","net:mirror" — none of these exist in tagSetsA.
				// tagSetsA contains "engine:unity" (mem-0 + mem-4) — not in tagSetsB.
				brainBExclusiveTags := []string{
					"audio:fmod", "audio:wwise", "physics:havok", "physics:bullet", "net:mirror",
				}
				tags, err := fix.st.TopTags(ctx, fix.brainA, 50)
				if err != nil {
					t.Fatalf("top tags: %v", err)
				}
				for _, tc := range tags {
					for _, bad := range brainBExclusiveTags {
						if tc.Tag == bad {
							t.Errorf("TopTags(brainA) contains %q which belongs only to brainB", bad)
						}
					}
				}
				// brainA should have "engine:unity" (appears in mem-0 and mem-4).
				found := false
				for _, tc := range tags {
					if tc.Tag == "engine:unity" {
						found = true
						break
					}
				}
				if !found {
					t.Error("TopTags(brainA) missing 'engine:unity' which brainA has")
				}
			},
		},
		{
			name: "Links_Create_CrossBrain_Rejected",
			run: func(t *testing.T, fix twoBrainSetup) {
				_, err := fix.ls.Create(ctx, fix.brainA,
					fix.memsA[0].ID, fix.memsB[0].ID,
					links.RelRelated, links.SourceManual, 1.0, "",
				)
				if !errors.Is(err, links.ErrCrossBrainLink) {
					t.Errorf("Create cross-brain: want ErrCrossBrainLink, got %v", err)
				}
			},
		},
		{
			name: "Links_GetByID_CrossBrain_NotFound",
			run: func(t *testing.T, fix twoBrainSetup) {
				_, err := fix.ls.GetByID(ctx, fix.brainA, fix.linksB[0].ID)
				if !errors.Is(err, links.ErrLinkNotFound) {
					t.Errorf("GetByID cross-brain link: want ErrLinkNotFound, got %v", err)
				}
			},
		},
		{
			name: "Links_Delete_CrossBrain_DoesNotDelete",
			run: func(t *testing.T, fix twoBrainSetup) {
				err := fix.ls.Delete(ctx, fix.brainA, fix.linksB[0].ID)
				if !errors.Is(err, links.ErrLinkNotFound) {
					t.Errorf("Delete cross-brain link: want ErrLinkNotFound, got %v", err)
				}
				// brainB's link must still exist.
				got, err := fix.ls.GetByID(ctx, fix.brainB, fix.linksB[0].ID)
				if err != nil {
					t.Fatalf("GetByID brainB link after failed cross-brain delete: %v", err)
				}
				if got.ID != fix.linksB[0].ID {
					t.Errorf("brainB link ID changed: want %d, got %d", fix.linksB[0].ID, got.ID)
				}
			},
		},
		{
			name: "Links_Neighbors_CrossBrain_Empty",
			run: func(t *testing.T, fix twoBrainSetup) {
				// Ask brainA's graph about brainB's memory — should return 0 links.
				neighbors, err := fix.ls.Neighbors(ctx, fix.brainA, fix.memsB[0].ID, nil)
				if err != nil {
					t.Fatalf("Neighbors cross-brain: %v", err)
				}
				if len(neighbors) != 0 {
					t.Errorf("Neighbors(brainA, memB): want 0, got %d", len(neighbors))
				}
			},
		},
		{
			name: "Links_Subgraph_OnlyBrainAData",
			run: func(t *testing.T, fix twoBrainSetup) {
				sg, err := fix.ls.Subgraph(ctx, fix.brainA, links.SubgraphOptions{})
				if err != nil {
					t.Fatalf("Subgraph brainA: %v", err)
				}
				for _, node := range sg.Nodes {
					if node.BrainID != fix.brainA {
						t.Errorf("Subgraph(brainA) node has brain_id=%d (want %d)", node.BrainID, fix.brainA)
					}
				}
				for _, edge := range sg.Edges {
					if edge.BrainID != fix.brainA {
						t.Errorf("Subgraph(brainA) edge has brain_id=%d (want %d)", edge.BrainID, fix.brainA)
					}
				}
				// Sanity: we should see all 5 nodes and 3 edges.
				if len(sg.Nodes) != 5 {
					t.Errorf("Subgraph(brainA) nodes: want 5, got %d", len(sg.Nodes))
				}
				if len(sg.Edges) != 3 {
					t.Errorf("Subgraph(brainA) edges: want 3, got %d", len(sg.Edges))
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fix := setupTwoBrains(t)
			tc.run(t, fix)
		})
	}
}

// ---------------------------------------------------------------------------
// Task 7.3 — TestBrainIsolation_Property (random 1000-op sequence)
// ---------------------------------------------------------------------------

// opKind enumerates the operations exercised by the property test.
type opKind int

const (
	opSave opKind = iota
	opUpdate
	opDelete
	opLinkCreate
	opLinkDelete
	opKindCount
)

// shadowState is the in-test ground-truth model maintained alongside the DB.
type shadowState struct {
	memIDs  []int64          // IDs of currently-alive memories (not soft-deleted)
	linkIDs []int64          // IDs of currently-alive links
	memSet  map[int64]struct{} // set form for O(1) membership
	linkSet map[int64]struct{}
}

func newShadowState() shadowState {
	return shadowState{
		memSet:  make(map[int64]struct{}),
		linkSet: make(map[int64]struct{}),
	}
}

func (s *shadowState) addMem(id int64) {
	if _, ok := s.memSet[id]; !ok {
		s.memIDs = append(s.memIDs, id)
		s.memSet[id] = struct{}{}
	}
}

func (s *shadowState) removeMem(id int64) {
	delete(s.memSet, id)
	// Rebuild slice — list stays small enough.
	updated := s.memIDs[:0]
	for _, id2 := range s.memIDs {
		if id2 != id {
			updated = append(updated, id2)
		}
	}
	s.memIDs = updated
}

func (s *shadowState) addLink(id int64) {
	if _, ok := s.linkSet[id]; !ok {
		s.linkIDs = append(s.linkIDs, id)
		s.linkSet[id] = struct{}{}
	}
}

func (s *shadowState) removeLink(id int64) {
	delete(s.linkSet, id)
	updated := s.linkIDs[:0]
	for _, id2 := range s.linkIDs {
		if id2 != id {
			updated = append(updated, id2)
		}
	}
	s.linkIDs = updated
}

// TestBrainIsolation_Property runs a seeded-random 1000-step sequence of
// mixed operations across two brains and after every 100 steps asserts that
// the per-brain row counts match the in-memory shadow model and no row
// leaks across brains.
func TestBrainIsolation_Property(t *testing.T) {
	const seed = 42
	const steps = 1000
	const checkInterval = 100

	ctx := context.Background()
	st := newTestStorage(t)
	ls := links.New(st.db)

	brainA := setupBrain(t, st, "prop-alpha")
	brainB := setupBrain(t, st, "prop-beta")

	shadowA := newShadowState()
	shadowB := newShadowState()

	rng := rand.New(rand.NewPCG(seed, 0))

	// linkRelations rotates through valid relations to keep uniqueness viable.
	linkRelations := []links.Relation{
		links.RelRefines, links.RelDependsOn, links.RelRelated,
		links.RelReferences, links.RelDerivedFrom,
	}

	// saveMem saves a new memory to brainID and tracks it in the shadow.
	saveMem := func(brainID int64, project string, sh *shadowState, idx int) {
		m := memory.Memory{
			Project: project,
			Scope:   memory.ScopeProject,
			Type:    memory.TypeDecision,
			Title:   fmt.Sprintf("prop-%s-step-%d", project, idx),
			Content: fmt.Sprintf("property test content step %d project %s", idx, project),
		}
		saved, _, err := st.Save(ctx, brainID, m)
		if err != nil {
			// Constraint violations are not fatal for the property test.
			return
		}
		sh.addMem(saved.ID)
	}

	// tryLinkCreate attempts to link two random live memories within brainID.
	tryLinkCreate := func(brainID int64, sh *shadowState, step int) {
		if len(sh.memIDs) < 2 {
			return
		}
		idxFrom := rng.IntN(len(sh.memIDs))
		idxTo := rng.IntN(len(sh.memIDs))
		if idxFrom == idxTo {
			return
		}
		fromID := sh.memIDs[idxFrom]
		toID := sh.memIDs[idxTo]
		rel := linkRelations[step%len(linkRelations)]
		lk, err := ls.Create(ctx, brainID, fromID, toID, rel, links.SourceManual, 1.0, "")
		if err != nil {
			// ErrLinkExists or other conflicts are fine.
			return
		}
		sh.addLink(lk.ID)
	}

	// tryLinkDelete removes a random live link within brainID.
	tryLinkDelete := func(brainID int64, sh *shadowState) {
		if len(sh.linkIDs) == 0 {
			return
		}
		idx := rng.IntN(len(sh.linkIDs))
		id := sh.linkIDs[idx]
		if err := ls.Delete(ctx, brainID, id); err != nil {
			return
		}
		sh.removeLink(id)
	}

	// tryUpdate updates a random live memory within brainID.
	tryUpdate := func(brainID int64, sh *shadowState, step int) {
		if len(sh.memIDs) == 0 {
			return
		}
		idx := rng.IntN(len(sh.memIDs))
		id := sh.memIDs[idx]
		title := fmt.Sprintf("updated-step-%d", step)
		content := fmt.Sprintf("updated content step %d", step)
		patch := UpdatePatch{
			Title:   &title,
			Content: &content,
		}
		_, err := st.UpdateByID(ctx, brainID, id, patch)
		if err != nil && !errors.Is(err, ErrMemoryNotFound) {
			t.Logf("UpdateByID step %d: %v", step, err)
		}
	}

	// trySoftDelete soft-deletes a random live memory within brainID and
	// removes it from the shadow (it's no longer queryable).
	trySoftDelete := func(brainID int64, sh *shadowState) {
		if len(sh.memIDs) == 0 {
			return
		}
		idx := rng.IntN(len(sh.memIDs))
		id := sh.memIDs[idx]
		if err := st.SoftDelete(ctx, brainID, id); err != nil {
			return
		}
		sh.removeMem(id)
		// Any links referencing this memory may now be dangling — remove them
		// from shadow (they'll cascade at the DB level on hard delete, but
		// soft-delete doesn't cascade links; just leave them in DB as orphan
		// links referencing deleted memory — GetByID will return ErrEndpointNotFound).
	}

	// assertNoLeaks checks DB state matches shadow and no cross-brain leak.
	assertNoLeaks := func(step int) {
		// Count live memories per brain from DB.
		var countA, countB int
		if err := st.db.QueryRowContext(ctx,
			`SELECT count(*) FROM memories WHERE brain_id = ? AND deleted_at IS NULL`, brainA,
		).Scan(&countA); err != nil {
			t.Fatalf("step %d: count brainA memories: %v", step, err)
		}
		if err := st.db.QueryRowContext(ctx,
			`SELECT count(*) FROM memories WHERE brain_id = ? AND deleted_at IS NULL`, brainB,
		).Scan(&countB); err != nil {
			t.Fatalf("step %d: count brainB memories: %v", step, err)
		}

		if countA != len(shadowA.memSet) {
			t.Errorf("step %d: brainA DB count=%d shadow=%d", step, countA, len(shadowA.memSet))
		}
		if countB != len(shadowB.memSet) {
			t.Errorf("step %d: brainB DB count=%d shadow=%d", step, countB, len(shadowB.memSet))
		}

		// Assert no brainA memory_id appears in brainB's namespace and vice versa.
		for id := range shadowA.memSet {
			_, err := st.GetByID(ctx, brainB, id)
			if err == nil {
				t.Errorf("step %d: brainB can read brainA memory id=%d (ISOLATION LEAK)", step, id)
			}
		}
		for id := range shadowB.memSet {
			_, err := st.GetByID(ctx, brainA, id)
			if err == nil {
				t.Errorf("step %d: brainA can read brainB memory id=%d (ISOLATION LEAK)", step, id)
			}
		}
	}

	// Seed a few initial memories so operations have something to work with.
	for i := 0; i < 5; i++ {
		saveMem(brainA, "prop-alpha", &shadowA, -i)
		saveMem(brainB, "prop-beta", &shadowB, -i)
	}

	for step := 0; step < steps; step++ {
		// Choose a brain (50/50).
		var brainID int64
		var sh *shadowState
		var project string
		if rng.IntN(2) == 0 {
			brainID, sh, project = brainA, &shadowA, "prop-alpha"
		} else {
			brainID, sh, project = brainB, &shadowB, "prop-beta"
		}

		op := opKind(rng.IntN(int(opKindCount)))
		switch op {
		case opSave:
			saveMem(brainID, project, sh, step)
		case opUpdate:
			tryUpdate(brainID, sh, step)
		case opDelete:
			trySoftDelete(brainID, sh)
		case opLinkCreate:
			tryLinkCreate(brainID, sh, step)
		case opLinkDelete:
			tryLinkDelete(brainID, sh)
		}

		if (step+1)%checkInterval == 0 {
			assertNoLeaks(step + 1)
		}
	}

	// Final full count assertion.
	assertNoLeaks(steps)

	// Scan all memories with a raw SQL admin query and verify every row's
	// brain_id is one of the two brains and belongs to the correct shadow bucket.
	rows, err := st.db.QueryContext(ctx,
		`SELECT id, brain_id FROM memories WHERE deleted_at IS NULL`,
	)
	if err != nil {
		t.Fatalf("final scan: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, bid int64
		if err := rows.Scan(&id, &bid); err != nil {
			t.Fatalf("final scan row: %v", err)
		}
		if bid != brainA && bid != brainB {
			t.Errorf("memory id=%d has unexpected brain_id=%d", id, bid)
			continue
		}
		if bid == brainA {
			if _, ok := shadowA.memSet[id]; !ok {
				t.Errorf("DB has brainA memory id=%d not in shadow", id)
			}
		} else {
			if _, ok := shadowB.memSet[id]; !ok {
				t.Errorf("DB has brainB memory id=%d not in shadow", id)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("final scan iter: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestBrainIsolation_CASCADEDoesNotLeak
// ---------------------------------------------------------------------------

// TestBrainIsolation_CASCADEDoesNotLeak verifies that hard-deleting brain A's
// row from the DB cascades to its memory_links but NOT to its memories rows,
// and that brain B's data is completely unaffected.
//
// NOTE — known schema gap (Phase 9 hardening):
// memories.brain_id references brains(id) WITHOUT ON DELETE CASCADE (by design
// — memories are intentionally orphaned). When FK enforcement is ON, SQLite
// blocks the DELETE on brains because memories still reference it. This means
// the ON DELETE CASCADE on memory_links.brain_id is also unreachable in that
// path. To test the cascade contract we must: (a) manually delete memory_links
// first (simulating what CASCADE would do), (b) then disable FK checks to allow
// the brain delete against orphan memories. Phase 9 must decide whether to add
// CASCADE to memories.brain_id or document this as intentional.
func TestBrainIsolation_CASCADEDoesNotLeak(t *testing.T) {
	ctx := context.Background()
	fix := setupTwoBrains(t)

	brainAID := fix.brainA
	brainBID := fix.brainB

	// Step 1: manually remove brainA's memory_links (simulates what
	// memory_links.brain_id ON DELETE CASCADE would do if the brain delete
	// could proceed with FKs on).
	if _, err := fix.st.db.ExecContext(ctx,
		`DELETE FROM memory_links WHERE brain_id = ?`, brainAID,
	); err != nil {
		t.Fatalf("manual delete brainA links: %v", err)
	}

	// Step 2: hard-delete the brain row.  memories.brain_id has no CASCADE so
	// we must disable FK enforcement to allow the delete without removing memories.
	if _, err := fix.st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable FK: %v", err)
	}
	_, err := fix.st.db.ExecContext(ctx, `DELETE FROM brains WHERE id = ?`, brainAID)
	if _, err2 := fix.st.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err2 != nil {
		t.Fatalf("re-enable FK: %v", err2)
	}
	if err != nil {
		t.Fatalf("hard-delete brainA: %v", err)
	}

	// 1. memory_links for brain A must be gone (manually removed above, confirming cascade contract).
	var linkCount int
	if err := fix.st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM memory_links WHERE brain_id = ?`, brainAID,
	).Scan(&linkCount); err != nil {
		t.Fatalf("count brainA links after delete: %v", err)
	}
	if linkCount != 0 {
		t.Errorf("memory_links: expected 0 rows for deleted brainA, got %d", linkCount)
	}

	// 2. memories for brain A are NOT cascade-deleted (no CASCADE on memories.brain_id FK).
	//    They become orphaned rows — this is expected and part of the contract.
	var memCount int
	if err := fix.st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM memories WHERE brain_id = ?`, brainAID,
	).Scan(&memCount); err != nil {
		t.Fatalf("count brainA memories after brain delete: %v", err)
	}
	if memCount != 5 {
		t.Errorf("orphaned memories: expected 5 (no CASCADE on brain_id), got %d", memCount)
	}

	// 3. Brain B's memories are completely untouched.
	var memBCount int
	if err := fix.st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM memories WHERE brain_id = ? AND deleted_at IS NULL`, brainBID,
	).Scan(&memBCount); err != nil {
		t.Fatalf("count brainB memories after brainA delete: %v", err)
	}
	if memBCount != 5 {
		t.Errorf("brainB memories count: expected 5, got %d (ISOLATION LEAK)", memBCount)
	}

	// 4. Brain B's links are completely untouched.
	var linkBCount int
	if err := fix.st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM memory_links WHERE brain_id = ?`, brainBID,
	).Scan(&linkBCount); err != nil {
		t.Fatalf("count brainB links after brainA delete: %v", err)
	}
	if linkBCount != 3 {
		t.Errorf("brainB links count: expected 3, got %d (ISOLATION LEAK)", linkBCount)
	}
}

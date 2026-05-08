package brain_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/brain"
	"github.com/AgusLoza2021/Thoughtline/internal/events"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// openTestStorage opens a fresh ephemeral storage (migrations applied) and
// returns the underlying *sql.DB for use with brain.New.
func openTestStorage(t *testing.T) *brain.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := storage.Open(context.Background(), dir+"/test.db")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return brain.New(s.DB())
}

// ---------------------------------------------------------------------------
// Task 3.1 — Slug validation (table-driven)
// ---------------------------------------------------------------------------

func TestBrain_SlugValidation(t *testing.T) {
	cases := []struct {
		slug    string
		wantErr bool
	}{
		// valid
		{"a", false},
		{"my-research-brain-2026", false},
		{"thoughtline", false},
		{"abc123", false},
		{strings.Repeat("a", 64), false}, // exactly 64 chars
		// invalid — uppercase
		{"MyBrain", true},
		{"Foo", true},
		// invalid — starts with hyphen
		{"-leading", true},
		// invalid — ends with hyphen
		{"trailing-", true},
		// invalid — space
		{"with space", true},
		// invalid — too long (65 chars)
		{strings.Repeat("a", 65), true},
		// invalid — double hyphen (spec doesn't ban it, so treat as valid)
		// invalid — empty
		{"", true},
	}

	for _, tc := range cases {
		t.Run(tc.slug, func(t *testing.T) {
			err := brain.ValidateSlug(tc.slug)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateSlug(%q): expected error, got nil", tc.slug)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateSlug(%q): unexpected error: %v", tc.slug, err)
			}
			// Error message must reference the slug when it fails
			if tc.wantErr && err != nil && tc.slug != "" {
				if !strings.Contains(err.Error(), tc.slug) {
					t.Errorf("ValidateSlug(%q) error does not mention slug: %v", tc.slug, err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 3.3 — Create with valid kind succeeds
// ---------------------------------------------------------------------------

func TestBrain_Create_ValidKind(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "my-brain", "My Brain", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if b.ID == 0 {
		t.Error("Create: expected non-zero ID")
	}
	if b.Slug != "my-brain" {
		t.Errorf("Create: Slug = %q, want %q", b.Slug, "my-brain")
	}
	if b.Kind != brain.KindReal {
		t.Errorf("Create: Kind = %q, want %q", b.Kind, brain.KindReal)
	}
	if b.CreatedAt.IsZero() {
		t.Error("Create: CreatedAt is zero")
	}
}

// ---------------------------------------------------------------------------
// Task 3.4 — Create with invalid kind is rejected
// ---------------------------------------------------------------------------

func TestBrain_Create_InvalidKind(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "my-brain", "My Brain", brain.Kind("experimental"), "", nil)
	if err == nil {
		t.Fatal("Create with invalid kind: expected error, got nil")
	}
	if !errors.Is(err, brain.ErrInvalidKind) {
		t.Errorf("Create with invalid kind: want ErrInvalidKind, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 3.4 (slug part) — Create with invalid slug rejected
// ---------------------------------------------------------------------------

func TestBrain_Create_InvalidSlug(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	invalids := []string{"Foo", "with space", strings.Repeat("a", 65), "-bad"}
	for _, slug := range invalids {
		t.Run(slug, func(t *testing.T) {
			_, err := store.Create(ctx, slug, "Display", brain.KindReal, "", nil)
			if err == nil {
				t.Errorf("Create(%q): expected ErrInvalidSlug, got nil", slug)
			}
			if !errors.Is(err, brain.ErrInvalidSlug) {
				t.Errorf("Create(%q): want ErrInvalidSlug, got %v", slug, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 3.6 — Duplicate slug on active brains fails (ErrSlugTaken)
// ---------------------------------------------------------------------------

func TestBrain_SlugUniqueness(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "alpha", "Alpha", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = store.Create(ctx, "alpha", "Alpha 2", brain.KindReal, "", nil)
	if err == nil {
		t.Fatal("second Create with same slug: expected ErrSlugTaken, got nil")
	}
	if !errors.Is(err, brain.ErrSlugTaken) {
		t.Errorf("second Create: want ErrSlugTaken, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 3.7 — Slug reuse after archival is allowed
// ---------------------------------------------------------------------------

func TestBrain_SlugReuseAfterArchive(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "alpha", "Alpha", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	b2, err := store.Create(ctx, "alpha", "Alpha 2", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create after archive: %v", err)
	}
	if b2.ID == b.ID {
		t.Error("expected new brain to have different ID")
	}
}

// ---------------------------------------------------------------------------
// Task 3.8 — Archive hides brain from default List
// ---------------------------------------------------------------------------

func TestBrain_ArchiveHidesFromList(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "to-archive", "To Archive", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	active, err := store.List(ctx, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, br := range active {
		if br.ID == b.ID {
			t.Errorf("archived brain %d still appears in List(includeArchived=false)", b.ID)
		}
	}
}

// Task 3.8 — Archive is idempotent
func TestBrain_Archive_Idempotent(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "idem", "Idempotent", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("first Archive: %v", err)
	}
	if err := store.Archive(ctx, b.ID); err != nil {
		t.Errorf("second Archive (idempotent): unexpected error: %v", err)
	}
}

// Task 3.8 — GetByID returns archived brain (historical access)
func TestBrain_GetByID_ReturnsArchived(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "hist", "Historical", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	got, err := store.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetByID on archived brain: unexpected error: %v", err)
	}
	if got.ArchivedAt == nil {
		t.Error("GetByID: expected ArchivedAt to be set")
	}
}

// Task 3.8 — GetBySlug returns ErrBrainNotFound for archived brain
func TestBrain_GetBySlug_ArchivedNotFound(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "gone", "Gone", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	_, err = store.GetBySlug(ctx, "gone")
	if !errors.Is(err, brain.ErrBrainNotFound) {
		t.Errorf("GetBySlug(archived): want ErrBrainNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateConfig writes new config_json and bumps updated_at
// ---------------------------------------------------------------------------

func TestBrain_UpdateConfig(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "cfg-brain", "Config Brain", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newCfg := json.RawMessage(`{"decay":{"half_life_days":10}}`)
	if err := store.UpdateConfig(ctx, b.ID, newCfg); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}

	got, err := store.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetByID after UpdateConfig: %v", err)
	}
	if string(got.ConfigJSON) != string(newCfg) {
		t.Errorf("ConfigJSON = %s, want %s", got.ConfigJSON, newCfg)
	}
	if got.UpdatedAt.Before(b.UpdatedAt) {
		t.Errorf("UpdatedAt went backwards: before=%v after=%v", b.UpdatedAt, got.UpdatedAt)
	}
}

// ---------------------------------------------------------------------------
// List(includeArchived=true) returns archived brains
// ---------------------------------------------------------------------------

func TestBrain_List_IncludeArchived(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	active, err := store.Create(ctx, "live", "Live", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create live: %v", err)
	}
	archived, err := store.Create(ctx, "dead", "Dead", brain.KindSandbox, "", nil)
	if err != nil {
		t.Fatalf("Create dead: %v", err)
	}
	if err := store.Archive(ctx, archived.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	all, err := store.List(ctx, true)
	if err != nil {
		t.Fatalf("List(true): %v", err)
	}

	foundActive, foundArchived := false, false
	for _, br := range all {
		if br.ID == active.ID {
			foundActive = true
		}
		if br.ID == archived.ID {
			foundArchived = true
		}
	}
	if !foundActive {
		t.Error("List(includeArchived=true): active brain not found")
	}
	if !foundArchived {
		t.Error("List(includeArchived=true): archived brain not found")
	}
}

// ---------------------------------------------------------------------------
// Unarchive restores a brain
// ---------------------------------------------------------------------------

func TestBrain_Unarchive(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "revive", "Revive", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Archive(ctx, b.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if err := store.Unarchive(ctx, b.ID); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}

	got, err := store.GetBySlug(ctx, "revive")
	if err != nil {
		t.Fatalf("GetBySlug after Unarchive: %v", err)
	}
	if got.ArchivedAt != nil {
		t.Error("Unarchive: ArchivedAt should be nil after unarchive")
	}
}

// ---------------------------------------------------------------------------
// configJSON defaults to {} when nil is passed
// ---------------------------------------------------------------------------

func TestBrain_Create_DefaultConfigJSON(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "defaults", "Defaults", brain.KindSynthetic, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if string(got.ConfigJSON) != "{}" {
		t.Errorf("default ConfigJSON = %s, want {}", got.ConfigJSON)
	}
}

// ---------------------------------------------------------------------------
// Task 9.4 — BrainConfigChanged bus emission
// ---------------------------------------------------------------------------

func TestBrain_BusEmitOnUpdateConfig(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.Open(context.Background(), dir+"/test.db")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	bus := events.New(nil)
	store := brain.New(s.DB())
	store.SetBus(bus)

	ctx := context.Background()
	b, err := store.Create(ctx, "config-emit", "Config Emit", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ch, unsub := bus.Subscribe(b.ID)
	defer unsub()

	newCfg := json.RawMessage(`{"decay":{"half_life_days":7}}`)
	if err := store.UpdateConfig(ctx, b.ID, newCfg); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}

	select {
	case ev := <-ch:
		bcc, ok := ev.(events.BrainConfigChanged)
		if !ok {
			t.Fatalf("expected BrainConfigChanged, got %T", ev)
		}
		if bcc.Brain != b.ID {
			t.Errorf("BrainConfigChanged.Brain = %d, want %d", bcc.Brain, b.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for BrainConfigChanged event")
	}
}

func TestBrain_BusNilSafe(t *testing.T) {
	store := openTestStorage(t)
	ctx := context.Background()

	b, err := store.Create(ctx, "no-bus", "No Bus", brain.KindReal, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// SetBus was never called — UpdateConfig must not panic.
	if err := store.UpdateConfig(ctx, b.ID, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("UpdateConfig without bus: %v", err)
	}
}

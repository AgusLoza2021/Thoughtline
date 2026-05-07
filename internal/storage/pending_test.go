package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

func samplePendingEvent() pending.Event {
	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	return pending.Event{
		SyncID:    "sync-001",
		Project:   "enchanted-inn",
		SessionID: "sess-abc",
		EventType: "PreToolUse",
		ToolName:  "Read",
		ToolUseID: "tu-001",
		Payload:   `{"hook_event_name":"PreToolUse","session_id":"sess-abc"}`,
		Hash:      "aabbcc",
		Status:    pending.StatusPending,
		CreatedAt: ts,
		CapturedAt: ts,
	}
}

// TestInsertPending_HappyPath verifies that InsertPending stores a row with
// status=pending and promoted_memory_id == 0 (NULL).
func TestInsertPending_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	ev := samplePendingEvent()
	result, err := st.InsertPending(ctx, ev)
	if err != nil {
		t.Fatalf("InsertPending: %v", err)
	}
	if result != ActionCreated {
		t.Errorf("expected ActionCreated, got %s", result)
	}

	// Verify the row is actually there.
	var (
		status            string
		promotedMemoryID  *int64
	)
	err = st.db.QueryRowContext(ctx,
		`SELECT status, promoted_memory_id FROM pending_events WHERE sync_id = ?`, ev.SyncID,
	).Scan(&status, &promotedMemoryID)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status=pending, got %q", status)
	}
	if promotedMemoryID != nil {
		t.Errorf("expected promoted_memory_id=NULL, got %v", *promotedMemoryID)
	}
}

// TestInsertPending_Dedup verifies that inserting the same (project, event_hash)
// twice returns ActionNoop and does not create a second row.
func TestInsertPending_Dedup(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	ev := samplePendingEvent()

	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	result, err := st.InsertPending(ctx, ev) // same project + hash
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if result != ActionNoop {
		t.Errorf("expected ActionNoop for duplicate, got %s", result)
	}

	var n int
	if err := st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM pending_events WHERE project = ? AND event_hash = ?`,
		ev.Project, ev.Hash,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row after dedup insert, got %d", n)
	}
}

// TestListPending_Filters verifies pagination and status/project/since filters.
func TestListPending_Filters(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Insert 3 events: 2 pending, 1 archived, different captured_at.
	events := []pending.Event{
		{SyncID: "s1", Project: "proj-a", SessionID: "x", EventType: "PreToolUse",
			ToolName: "Read", ToolUseID: "t1", Payload: `{}`, Hash: "h1",
			Status: pending.StatusPending, CreatedAt: base, CapturedAt: base},
		{SyncID: "s2", Project: "proj-a", SessionID: "x", EventType: "PostToolUse",
			ToolName: "Write", ToolUseID: "t2", Payload: `{}`, Hash: "h2",
			Status: pending.StatusPending, CreatedAt: base.Add(time.Hour), CapturedAt: base.Add(time.Hour)},
		{SyncID: "s3", Project: "proj-a", SessionID: "x", EventType: "Stop",
			Payload: `{}`, Hash: "h3",
			Status: pending.StatusArchived, CreatedAt: base.Add(2 * time.Hour), CapturedAt: base.Add(2 * time.Hour)},
	}
	for _, ev := range events {
		if _, err := st.InsertPending(ctx, ev); err != nil {
			t.Fatalf("insert %s: %v", ev.SyncID, err)
		}
	}

	// Filter by status=pending.
	results, err := st.ListPending(ctx, ListPendingParams{
		Project: "proj-a",
		Status:  string(pending.StatusPending),
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 pending events, got %d", len(results))
	}

	// Filter by event_type.
	results, err = st.ListPending(ctx, ListPendingParams{
		Project:   "proj-a",
		EventType: "Stop",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListPending by event_type: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 Stop event, got %d", len(results))
	}

	// Filter by since (only events after base+30min).
	since := base.Add(30 * time.Minute)
	results, err = st.ListPending(ctx, ListPendingParams{
		Project: "proj-a",
		Since:   since,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListPending by since: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 events after since, got %d", len(results))
	}

	// Pagination: limit=1, offset=1.
	results, err = st.ListPending(ctx, ListPendingParams{
		Project: "proj-a",
		Limit:   1,
		Offset:  1,
	})
	if err != nil {
		t.Fatalf("ListPending with offset: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1 offset=1, got %d", len(results))
	}
}

// TestGetPendingByID_HappyPath retrieves a full event including payload.
func TestGetPendingByID_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	ev := samplePendingEvent()
	ev.Payload = `{"hook_event_name":"PreToolUse","session_id":"sess-abc","tool_name":"Read"}`
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Get the auto-assigned ID.
	var id int64
	if err := st.db.QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id = ?`, ev.SyncID,
	).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	got, err := st.GetPendingByID(ctx, id)
	if err != nil {
		t.Fatalf("GetPendingByID: %v", err)
	}
	if got.Payload != ev.Payload {
		t.Errorf("payload mismatch: got %q, want %q", got.Payload, ev.Payload)
	}
	if got.EventType != ev.EventType {
		t.Errorf("event_type mismatch: got %q, want %q", got.EventType, ev.EventType)
	}
	if got.Status != pending.StatusPending {
		t.Errorf("expected status=pending, got %q", got.Status)
	}
}

// TestGetPendingByID_NotFound returns an error for an unknown ID.
func TestGetPendingByID_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	_, err := st.GetPendingByID(ctx, 9999)
	if err == nil {
		t.Error("expected error for missing ID, got nil")
	}
}

// TestMarkPromoted_HappyPath transitions a pending row to promoted and records
// the promoted_memory_id.
func TestMarkPromoted_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	ev := samplePendingEvent()
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var id int64
	if err := st.db.QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id = ?`, ev.SyncID,
	).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	if err := st.MarkPromoted(ctx, id, 42); err != nil {
		t.Fatalf("MarkPromoted: %v", err)
	}

	var (
		status  string
		memID   int64
	)
	if err := st.db.QueryRowContext(ctx,
		`SELECT status, promoted_memory_id FROM pending_events WHERE id = ?`, id,
	).Scan(&status, &memID); err != nil {
		t.Fatalf("query after promote: %v", err)
	}
	if status != "promoted" {
		t.Errorf("expected status=promoted, got %q", status)
	}
	if memID != 42 {
		t.Errorf("expected promoted_memory_id=42, got %d", memID)
	}
}

// TestMarkPromoted_AlreadyPromoted returns an error without creating a second
// memory or changing the promoted_memory_id.
func TestMarkPromoted_AlreadyPromoted(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	ev := samplePendingEvent()
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var id int64
	if err := st.db.QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id = ?`, ev.SyncID,
	).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}
	if err := st.MarkPromoted(ctx, id, 42); err != nil {
		t.Fatalf("first promote: %v", err)
	}

	// Second call must fail.
	err := st.MarkPromoted(ctx, id, 99)
	if err == nil {
		t.Error("expected error on double-promote, got nil")
	}

	// promoted_memory_id must remain 42.
	var memID int64
	if err := st.db.QueryRowContext(ctx,
		`SELECT promoted_memory_id FROM pending_events WHERE id = ?`, id,
	).Scan(&memID); err != nil {
		t.Fatalf("query: %v", err)
	}
	if memID != 42 {
		t.Errorf("promoted_memory_id should stay 42, got %d", memID)
	}
}

// TestSweepPending_ArchivesOldRows verifies the retention sweep archives
// pending rows older than the window and leaves promoted rows untouched.
func TestSweepPending_ArchivesOldRows(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)   // 8 days ago — should be archived
	recent := now.Add(-2 * 24 * time.Hour) // 2 days ago — should stay

	// Old pending row.
	ev1 := samplePendingEvent()
	ev1.SyncID = "s-old"
	ev1.Hash = "h-old"
	ev1.CreatedAt = old
	ev1.CapturedAt = old
	if _, err := st.InsertPending(ctx, ev1); err != nil {
		t.Fatalf("insert old: %v", err)
	}

	// Recent pending row.
	ev2 := samplePendingEvent()
	ev2.SyncID = "s-recent"
	ev2.Hash = "h-recent"
	ev2.CreatedAt = recent
	ev2.CapturedAt = recent
	if _, err := st.InsertPending(ctx, ev2); err != nil {
		t.Fatalf("insert recent: %v", err)
	}

	// Promoted row (any age — must NEVER be touched).
	ev3 := samplePendingEvent()
	ev3.SyncID = "s-promoted"
	ev3.Hash = "h-promoted"
	ev3.CreatedAt = old
	ev3.CapturedAt = old
	if _, err := st.InsertPending(ctx, ev3); err != nil {
		t.Fatalf("insert promoted: %v", err)
	}
	// Find its id and mark promoted.
	var promotedID int64
	if err := st.db.QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id = ?`, ev3.SyncID,
	).Scan(&promotedID); err != nil {
		t.Fatalf("get promoted id: %v", err)
	}
	if err := st.MarkPromoted(ctx, promotedID, 1); err != nil {
		t.Fatalf("mark promoted: %v", err)
	}

	result, err := st.SweepPending(ctx, 7*24*time.Hour, 30*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SweepPending: %v", err)
	}
	if result.Archived != 1 {
		t.Errorf("expected 1 archived, got %d", result.Archived)
	}

	// Check statuses.
	rows, err := st.db.QueryContext(ctx, `SELECT sync_id, status FROM pending_events`)
	if err != nil {
		t.Fatalf("query all: %v", err)
	}
	defer rows.Close()
	statuses := map[string]string{}
	for rows.Next() {
		var syncID, status string
		if err := rows.Scan(&syncID, &status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		statuses[syncID] = status
	}

	if statuses["s-old"] != "archived" {
		t.Errorf("s-old expected archived, got %q", statuses["s-old"])
	}
	if statuses["s-recent"] != "pending" {
		t.Errorf("s-recent expected pending, got %q", statuses["s-recent"])
	}
	if statuses["s-promoted"] != "promoted" {
		t.Errorf("s-promoted expected promoted, got %q", statuses["s-promoted"])
	}
}

// TestSweepPending_HardDelete verifies that archived rows past the hard-delete
// window are deleted from the DB entirely.
func TestSweepPending_HardDelete(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	veryOld := now.Add(-35 * 24 * time.Hour) // 35 days — past hard-delete

	ev := samplePendingEvent()
	ev.SyncID = "s-very-old"
	ev.Hash = "h-very-old"
	ev.CreatedAt = veryOld
	ev.CapturedAt = veryOld
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Manually set to archived with old archived_at.
	oldArchiveTime := now.Add(-31 * 24 * time.Hour)
	if _, err := st.db.ExecContext(ctx,
		`UPDATE pending_events SET status='archived', archived_at=? WHERE sync_id=?`,
		oldArchiveTime.UnixMilli(), ev.SyncID,
	); err != nil {
		t.Fatalf("manual archive: %v", err)
	}

	result, err := st.SweepPending(ctx, 7*24*time.Hour, 30*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SweepPending: %v", err)
	}
	if result.Deleted != 1 {
		t.Errorf("expected 1 hard-deleted, got %d", result.Deleted)
	}

	var n int
	if err := st.db.QueryRowContext(ctx,
		`SELECT count(*) FROM pending_events WHERE sync_id=?`, ev.SyncID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("expected row deleted, still got %d rows", n)
	}
}

// TestSweepPending_NoEligibleRows is a no-op when all pending rows are recent.
func TestSweepPending_NoEligibleRows(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	recent := now.Add(-1 * 24 * time.Hour)

	ev := samplePendingEvent()
	ev.CreatedAt = recent
	ev.CapturedAt = recent
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("insert: %v", err)
	}

	result, err := st.SweepPending(ctx, 7*24*time.Hour, 30*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SweepPending: %v", err)
	}
	if result.Archived != 0 || result.Deleted != 0 {
		t.Errorf("expected no-op, got archived=%d deleted=%d", result.Archived, result.Deleted)
	}
}

// TestCountPending_ReturnsOnlyPendingStatus verifies that CountPending counts
// only rows with status=pending for the given project, ignoring other projects
// and other statuses (promoted, archived).
func TestCountPending_ReturnsOnlyPendingStatus(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	base := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)

	// 3 pending rows for "alpha".
	for i := 0; i < 3; i++ {
		ev := samplePendingEvent()
		ev.SyncID = fmt.Sprintf("a-pend-%d", i)
		ev.Hash = fmt.Sprintf("ha%d", i)
		ev.Project = "alpha"
		ev.CapturedAt = base
		if _, err := st.InsertPending(ctx, ev); err != nil {
			t.Fatalf("insert alpha pending %d: %v", i, err)
		}
	}

	// 2 promoted rows for "alpha" — must NOT be counted.
	for i := 0; i < 2; i++ {
		ev := samplePendingEvent()
		ev.SyncID = fmt.Sprintf("a-prom-%d", i)
		ev.Hash = fmt.Sprintf("hp%d", i)
		ev.Project = "alpha"
		ev.Status = "promoted"
		ev.CapturedAt = base
		if _, err := st.InsertPending(ctx, ev); err != nil {
			t.Fatalf("insert alpha promoted %d: %v", i, err)
		}
	}

	// 1 pending row for "other" — must NOT be counted for "alpha".
	other := samplePendingEvent()
	other.SyncID = "other-pend"
	other.Hash = "hother"
	other.Project = "other"
	other.CapturedAt = base
	if _, err := st.InsertPending(ctx, other); err != nil {
		t.Fatalf("insert other: %v", err)
	}

	n, err := st.CountPending(ctx, "alpha")
	if err != nil {
		t.Fatalf("CountPending: %v", err)
	}
	if n != 3 {
		t.Errorf("CountPending(alpha) = %d, want 3", n)
	}

	n2, err := st.CountPending(ctx, "other")
	if err != nil {
		t.Fatalf("CountPending other: %v", err)
	}
	if n2 != 1 {
		t.Errorf("CountPending(other) = %d, want 1", n2)
	}

	n3, err := st.CountPending(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("CountPending nonexistent: %v", err)
	}
	if n3 != 0 {
		t.Errorf("CountPending(nonexistent) = %d, want 0", n3)
	}
}

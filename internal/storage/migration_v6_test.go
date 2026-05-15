package storage

import (
	"context"
	"path/filepath"
	"testing"
)

// TestMigrateV6_AppliesIndex_Idempotent verifies that:
//
//   (a) A fresh database reaches schema version 6 and has idx_memories_hash.
//   (b) Opening the same database file a second time is a no-op: no error,
//       schema_version still shows exactly one row for version 6, and the
//       index is still present.
func TestMigrateV6_AppliesIndex_Idempotent(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "fresh_db_reaches_v6"},
		{name: "second_open_idempotent"},
	}

	path := filepath.Join(t.TempDir(), "v6.db")
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := Open(ctx, path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer st.Close()

			// schema_version must contain exactly one row for version 6.
			var v6Count int
			if err := st.db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM schema_version WHERE version = 6`).Scan(&v6Count); err != nil {
				t.Fatalf("query schema_version: %v", err)
			}
			if v6Count != 1 {
				t.Errorf("expected 1 row for schema_version=6, got %d", v6Count)
			}

			// idx_memories_hash must exist on the memories table.
			var idxCount int
			if err := st.db.QueryRowContext(ctx, `
				SELECT COUNT(*)
				FROM sqlite_master
				WHERE type = 'index'
				  AND tbl_name = 'memories'
				  AND name = 'idx_memories_hash'`).Scan(&idxCount); err != nil {
				t.Fatalf("query sqlite_master for index: %v", err)
			}
			if idxCount != 1 {
				t.Errorf("expected idx_memories_hash to exist (count=1), got %d", idxCount)
			}
		})
	}
}

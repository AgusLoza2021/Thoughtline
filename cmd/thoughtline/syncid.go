package main

import (
	"fmt"

	"github.com/google/uuid"
)

// newHookSyncID generates a UUIDv7 for use as a sync_id in pending_events.
func newHookSyncID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("uuidv7: %w", err)
	}
	return id.String(), nil
}

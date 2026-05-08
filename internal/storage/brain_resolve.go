package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// brainCache is a write-through cache that maps brain slug → brain_id.
// It is embedded inside Storage and is safe for concurrent use.
//
// Invalidation is explicit: callers must call InvalidateBrainCache(slug) when
// a brain is archived or deleted. There is no TTL — staleness here would
// silently route queries to an archived brain, which is a correctness bug.
type brainCache struct {
	mu sync.RWMutex
	m  map[string]int64
}

// ResolveBrainID returns the brain_id for the given active slug.
// It checks the in-memory cache first (read-lock); on a miss it queries SQLite
// and populates the cache (write-lock).
//
// Returns sql.ErrNoRows (wrapped) if no active brain with that slug exists.
func (s *Storage) ResolveBrainID(ctx context.Context, slug string) (int64, error) {
	// Fast path — cache hit.
	s.brains.mu.RLock()
	if s.brains.m != nil {
		if id, ok := s.brains.m[slug]; ok {
			s.brains.mu.RUnlock()
			return id, nil
		}
	}
	s.brains.mu.RUnlock()

	// Slow path — query DB.
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM brains WHERE slug = ? AND archived_at IS NULL`, slug,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("storage: brain not found for slug %q: %w", slug, err)
		}
		return 0, fmt.Errorf("storage: resolve brain id: %w", err)
	}

	// Populate cache.
	s.brains.mu.Lock()
	if s.brains.m == nil {
		s.brains.m = make(map[string]int64)
	}
	s.brains.m[slug] = id
	s.brains.mu.Unlock()

	return id, nil
}

// ResolveOrCreateBrainID returns the brain_id for the given slug. If no active
// brain with that slug exists it creates one with kind='real', using slug as both
// the slug and display_name. The new id is stored in the cache.
//
// This is the write path used by tl_save when an unknown project arrives via
// MCP. It must NOT be called in read-only contexts (stats, search, etc.).
func (s *Storage) ResolveOrCreateBrainID(ctx context.Context, slug string) (int64, error) {
	// Try cheap resolve first.
	id, err := s.ResolveBrainID(ctx, slug)
	if err == nil {
		return id, nil
	}
	// Any error other than "not found" is a hard failure.
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	// Create the brain.
	now := s.nowMillis().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO brains (slug, display_name, kind, description, config_json, created_at, updated_at)
		VALUES (?, ?, 'real', '', '{}', ?, ?)`,
		slug, slug, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("storage: create brain for slug %q: %w", slug, err)
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("storage: last insert id: %w", err)
	}

	// Store in cache.
	s.brains.mu.Lock()
	if s.brains.m == nil {
		s.brains.m = make(map[string]int64)
	}
	s.brains.m[slug] = newID
	s.brains.mu.Unlock()

	return newID, nil
}

// InvalidateBrainCache removes the entry for slug from the in-memory cache.
// Call this after archiving or hard-deleting a brain so the next
// ResolveBrainID hits SQLite and gets the correct (not-found) result.
func (s *Storage) InvalidateBrainCache(slug string) {
	s.brains.mu.Lock()
	delete(s.brains.m, slug)
	s.brains.mu.Unlock()
}

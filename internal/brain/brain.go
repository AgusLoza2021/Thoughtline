package brain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/events"
)

// Sentinel errors — callers use errors.Is to distinguish failure modes.
var (
	// ErrBrainNotFound is returned by GetBySlug and GetByID when no matching
	// active brain exists. GetBySlug also returns this for archived brains.
	ErrBrainNotFound = errors.New("brain: not found")

	// ErrInvalidSlug is returned when a slug fails the regex / length rules.
	ErrInvalidSlug = errors.New("brain: invalid slug")

	// ErrInvalidKind is returned when a Kind value is not one of the three
	// accepted constants.
	ErrInvalidKind = errors.New("brain: invalid kind")

	// ErrSlugTaken is returned when an INSERT fails because an active brain
	// with the same slug already exists (partial unique index violation).
	ErrSlugTaken = errors.New("brain: slug already taken by an active brain")
)

// Store wraps a *sql.DB and provides CRUD operations for Brain records.
// It is safe for concurrent use.
type Store struct {
	db  *sql.DB
	bus *events.Bus // optional; nil means no event emission
}

// New returns a Store backed by db. The caller is responsible for keeping db
// open for the lifetime of the Store.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// SetBus attaches an event bus to the store. After this call, every successful
// UpdateConfig emits a BrainConfigChanged event. Safe to call once at bootstrap.
func (s *Store) SetBus(b *events.Bus) {
	s.bus = b
}

// Create inserts a new brain row and returns the persisted Brain (id +
// timestamps populated). configJSON may be nil — it defaults to {}.
//
// Errors: ErrInvalidSlug, ErrInvalidKind, ErrSlugTaken.
func (s *Store) Create(
	ctx context.Context,
	slug, displayName string,
	kind Kind,
	description string,
	configJSON json.RawMessage,
) (Brain, error) {
	if err := ValidateSlug(slug); err != nil {
		return Brain{}, err
	}
	if !kind.Valid() {
		return Brain{}, fmt.Errorf("%w: %q", ErrInvalidKind, kind)
	}
	if len(configJSON) == 0 {
		configJSON = json.RawMessage("{}")
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	nowMS := now.UnixMilli()

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO brains (slug, display_name, kind, description, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		slug, displayName, string(kind), description, string(configJSON), nowMS, nowMS,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return Brain{}, fmt.Errorf("%w: %q", ErrSlugTaken, slug)
		}
		return Brain{}, fmt.Errorf("brain: insert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Brain{}, fmt.Errorf("brain: last insert id: %w", err)
	}

	return Brain{
		ID:          id,
		Slug:        slug,
		DisplayName: displayName,
		Kind:        kind,
		Description: description,
		ConfigJSON:  configJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetBySlug returns the active (non-archived) brain with the given slug.
// Returns ErrBrainNotFound if no active brain exists with that slug.
func (s *Store) GetBySlug(ctx context.Context, slug string) (Brain, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, slug, display_name, kind, description, config_json,
		       created_at, updated_at, archived_at
		FROM brains
		WHERE slug = ? AND archived_at IS NULL`,
		slug,
	)
	b, err := scanBrain(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Brain{}, ErrBrainNotFound
	}
	return b, err
}

// GetByID returns the brain with the given id, including archived brains.
// Returns ErrBrainNotFound if no brain with that id exists at all.
func (s *Store) GetByID(ctx context.Context, id int64) (Brain, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, slug, display_name, kind, description, config_json,
		       created_at, updated_at, archived_at
		FROM brains
		WHERE id = ?`,
		id,
	)
	b, err := scanBrain(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Brain{}, ErrBrainNotFound
	}
	return b, err
}

// List returns all brains ordered by created_at ASC.
// When includeArchived is false only active (archived_at IS NULL) brains are
// returned.
func (s *Store) List(ctx context.Context, includeArchived bool) ([]Brain, error) {
	q := `
		SELECT id, slug, display_name, kind, description, config_json,
		       created_at, updated_at, archived_at
		FROM brains`
	if !includeArchived {
		q += ` WHERE archived_at IS NULL`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("brain: list: %w", err)
	}
	defer rows.Close()

	var out []Brain
	for rows.Next() {
		b, err := scanBrainRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Archive sets archived_at to now for the brain with the given id. If the
// brain is already archived the call is a no-op (idempotent).
func (s *Store) Archive(ctx context.Context, id int64) error {
	nowMS := time.Now().UTC().UnixMilli()
	_, err := s.db.ExecContext(ctx, `
		UPDATE brains SET archived_at = ?
		WHERE id = ? AND archived_at IS NULL`,
		nowMS, id,
	)
	return err
}

// Unarchive clears archived_at for the brain with the given id.
func (s *Store) Unarchive(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE brains SET archived_at = NULL WHERE id = ?`,
		id,
	)
	return err
}

// UpdateConfig replaces config_json for the brain with the given id and bumps
// updated_at to now. Emits BrainConfigChanged if a bus is attached.
func (s *Store) UpdateConfig(ctx context.Context, id int64, configJSON json.RawMessage) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE brains SET config_json = ?, updated_at = ? WHERE id = ?`,
		string(configJSON), now.UnixMilli(), id,
	)
	if err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.BrainConfigChanged{Brain: id, At: now})
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type brainScanner interface {
	Scan(dest ...any) error
}

func scanBrain(row *sql.Row) (Brain, error) {
	return scanBrainScanner(row)
}

func scanBrainRow(rows *sql.Rows) (Brain, error) {
	return scanBrainScanner(rows)
}

func scanBrainScanner(sc brainScanner) (Brain, error) {
	var (
		b            Brain
		kind         string
		configJSON   string
		createdAtMS  int64
		updatedAtMS  int64
		archivedAtMS sql.NullInt64
	)
	if err := sc.Scan(
		&b.ID, &b.Slug, &b.DisplayName, &kind, &b.Description, &configJSON,
		&createdAtMS, &updatedAtMS, &archivedAtMS,
	); err != nil {
		return Brain{}, err
	}
	b.Kind = Kind(kind)
	b.ConfigJSON = json.RawMessage(configJSON)
	b.CreatedAt = time.UnixMilli(createdAtMS).UTC()
	b.UpdatedAt = time.UnixMilli(updatedAtMS).UTC()
	if archivedAtMS.Valid {
		t := time.UnixMilli(archivedAtMS.Int64).UTC()
		b.ArchivedAt = &t
	}
	return b, nil
}

// isUniqueConstraintError reports whether err is a SQLite UNIQUE constraint
// violation. The modernc.org/sqlite driver surfaces these as an error whose
// message contains "UNIQUE constraint failed".
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

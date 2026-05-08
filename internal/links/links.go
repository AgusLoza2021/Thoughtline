package links

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/events"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// Store wraps *sql.DB and provides CRUD + query operations for memory_links.
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
// Create emits LinkCreated and every successful Delete emits LinkDeleted.
// Safe to call once during app bootstrap. Passing nil is a no-op.
func (s *Store) SetBus(b *events.Bus) {
	s.bus = b
}

// SubgraphOptions controls what Subgraph returns.
type SubgraphOptions struct {
	// Types, when non-empty, restricts returned nodes to memories of these types.
	Types []memory.Type
	// Relations, when non-empty, restricts returned edges to links of these relations.
	Relations []Relation
	// Limit caps the number of nodes returned. Zero means default (500).
	Limit int
}

// Subgraph holds the result of a Subgraph query.
type Subgraph struct {
	Nodes []memory.Memory
	Edges []Link
}

// Create inserts a new memory_links row. Returns ErrSelfLoop, ErrInvalidRelation,
// ErrInvalidSource, ErrEndpointNotFound, ErrCrossBrainLink, or ErrLinkExists on
// validation / uniqueness failures. Weight defaults to 1.0 when passed as zero.
func (s *Store) Create(
	ctx context.Context,
	brainID, fromID, toID int64,
	relation Relation,
	source Source,
	weight float64,
	note string,
) (Link, error) {
	if !relation.Valid() {
		return Link{}, fmt.Errorf("%w: %q", ErrInvalidRelation, relation)
	}
	if !source.Valid() {
		return Link{}, fmt.Errorf("%w: %q", ErrInvalidSource, source)
	}
	if fromID == toID {
		return Link{}, ErrSelfLoop
	}
	if weight == 0 {
		weight = 1.0
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Link{}, fmt.Errorf("links: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Validate both endpoints exist and belong to brainID.
	var fromBrain, toBrain int64
	if err := tx.QueryRowContext(ctx, sqlEndpointBrainID, fromID).Scan(&fromBrain); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Link{}, fmt.Errorf("%w: from_id=%d", ErrEndpointNotFound, fromID)
		}
		return Link{}, fmt.Errorf("links: check from endpoint: %w", err)
	}
	if err := tx.QueryRowContext(ctx, sqlEndpointBrainID, toID).Scan(&toBrain); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Link{}, fmt.Errorf("%w: to_id=%d", ErrEndpointNotFound, toID)
		}
		return Link{}, fmt.Errorf("links: check to endpoint: %w", err)
	}
	if fromBrain != brainID || toBrain != brainID {
		return Link{}, ErrCrossBrainLink
	}

	nowMS := time.Now().UTC().UnixMilli()
	res, err := tx.ExecContext(ctx, sqlCreateLink,
		brainID, fromID, toID, string(relation), weight, string(source), note, nowMS,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return Link{}, fmt.Errorf("%w: (%d→%d, %s)", ErrLinkExists, fromID, toID, relation)
		}
		return Link{}, fmt.Errorf("links: insert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Link{}, fmt.Errorf("links: last insert id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Link{}, fmt.Errorf("links: commit: %w", err)
	}

	lk := Link{
		ID:        id,
		BrainID:   brainID,
		FromID:    fromID,
		ToID:      toID,
		Relation:  relation,
		Weight:    weight,
		Source:    source,
		Note:      note,
		CreatedAt: time.UnixMilli(nowMS).UTC(),
	}
	if s.bus != nil {
		s.bus.Publish(events.LinkCreated{
			Brain:    brainID,
			LinkID:   id,
			FromID:   fromID,
			ToID:     toID,
			Relation: string(relation),
			At:       lk.CreatedAt,
		})
	}
	return lk, nil
}

// GetByID returns the link with the given id, scoped to brainID.
// Returns ErrLinkNotFound if the link doesn't exist or belongs to a different brain.
func (s *Store) GetByID(ctx context.Context, brainID, linkID int64) (Link, error) {
	row := s.db.QueryRowContext(ctx, sqlGetLinkByID, linkID, brainID)
	l, err := scanLink(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrLinkNotFound
	}
	return l, err
}

// Delete removes the link with the given id, scoped to brainID.
// Returns ErrLinkNotFound if the link doesn't exist or belongs to a different brain.
func (s *Store) Delete(ctx context.Context, brainID, linkID int64) error {
	res, err := s.db.ExecContext(ctx, sqlDeleteLink, linkID, brainID)
	if err != nil {
		return fmt.Errorf("links: delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("links: rows affected: %w", err)
	}
	if n == 0 {
		return ErrLinkNotFound
	}
	if s.bus != nil {
		s.bus.Publish(events.LinkDeleted{
			Brain:  brainID,
			LinkID: linkID,
			At:     time.Now().UTC(),
		})
	}
	return nil
}

// Neighbors returns all incident links (inbound + outbound) for memID, scoped
// to brainID. When relation is non-nil, only links matching that relation are
// returned.
func (s *Store) Neighbors(ctx context.Context, brainID, memID int64, relation *Relation) ([]Link, error) {
	var rows *sql.Rows
	var err error
	if relation != nil {
		rows, err = s.db.QueryContext(ctx, sqlNeighborsFiltered, brainID, memID, memID, string(*relation))
	} else {
		rows, err = s.db.QueryContext(ctx, sqlNeighbors, brainID, memID, memID)
	}
	if err != nil {
		return nil, fmt.Errorf("links: neighbors: %w", err)
	}
	defer rows.Close()

	var out []Link
	for rows.Next() {
		l, err := scanLinkRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Subgraph returns memories (nodes) and links (edges) for brainID.
// Options may filter by memory type(s) and/or relation(s), and cap the node count.
func (s *Store) Subgraph(ctx context.Context, brainID int64, opts SubgraphOptions) (Subgraph, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 500
	}

	// Collect nodes.
	var nodes []memory.Memory
	if len(opts.Types) == 0 {
		rows, err := s.db.QueryContext(ctx, sqlSubgraphNodes+" LIMIT ?", brainID, limit)
		if err != nil {
			return Subgraph{}, fmt.Errorf("links: subgraph nodes: %w", err)
		}
		defer rows.Close()
		nodes, err = scanMemories(rows)
		if err != nil {
			return Subgraph{}, err
		}
	} else {
		// Multiple types — one query per type (types set is small ≤11).
		for _, typ := range opts.Types {
			rows, err := s.db.QueryContext(ctx, sqlSubgraphNodesTyped+" LIMIT ?", brainID, string(typ), limit)
			if err != nil {
				return Subgraph{}, fmt.Errorf("links: subgraph nodes typed: %w", err)
			}
			ms, err := scanAndCloseMemories(rows)
			if err != nil {
				return Subgraph{}, err
			}
			nodes = append(nodes, ms...)
		}
		if len(nodes) > limit {
			nodes = nodes[:limit]
		}
	}

	// Collect edges.
	var edges []Link
	if len(opts.Relations) == 0 {
		rows, err := s.db.QueryContext(ctx, sqlSubgraphEdges, brainID)
		if err != nil {
			return Subgraph{}, fmt.Errorf("links: subgraph edges: %w", err)
		}
		defer rows.Close()
		edges, err = scanLinks(rows)
		if err != nil {
			return Subgraph{}, err
		}
	} else {
		for _, rel := range opts.Relations {
			rows, err := s.db.QueryContext(ctx, sqlSubgraphEdgesFiltered, brainID, string(rel))
			if err != nil {
				return Subgraph{}, fmt.Errorf("links: subgraph edges filtered: %w", err)
			}
			ls, err := scanAndCloseLinks(rows)
			if err != nil {
				return Subgraph{}, err
			}
			edges = append(edges, ls...)
		}
	}

	return Subgraph{Nodes: nodes, Edges: edges}, nil
}

// ---------------------------------------------------------------------------
// scanners
// ---------------------------------------------------------------------------

type linkScanner interface {
	Scan(dest ...any) error
}

func scanLink(row *sql.Row) (Link, error) {
	return scanLinkScanner(row)
}

func scanLinkRows(rows *sql.Rows) (Link, error) {
	return scanLinkScanner(rows)
}

func scanLinkScanner(sc linkScanner) (Link, error) {
	var (
		l           Link
		rel         string
		src         string
		createdAtMS int64
	)
	if err := sc.Scan(
		&l.ID, &l.BrainID, &l.FromID, &l.ToID,
		&rel, &l.Weight, &src, &l.Note, &createdAtMS,
	); err != nil {
		return Link{}, err
	}
	l.Relation = Relation(rel)
	l.Source = Source(src)
	l.CreatedAt = time.UnixMilli(createdAtMS).UTC()
	return l, nil
}

func scanLinks(rows *sql.Rows) ([]Link, error) {
	var out []Link
	for rows.Next() {
		l, err := scanLinkRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanAndCloseLinks(rows *sql.Rows) ([]Link, error) {
	defer rows.Close()
	return scanLinks(rows)
}

// scanMemories scans rows already open (caller owns Close).
func scanMemories(rows *sql.Rows) ([]memory.Memory, error) {
	var out []memory.Memory
	for rows.Next() {
		m, err := scanMemoryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanAndCloseMemories(rows *sql.Rows) ([]memory.Memory, error) {
	defer rows.Close()
	return scanMemories(rows)
}

func scanMemoryRow(rows *sql.Rows) (memory.Memory, error) {
	var (
		m             memory.Memory
		topicKey      sql.NullString
		tags          sql.NullString
		deletedAtMS   sql.NullInt64
		sessionID     sql.NullString
		createdAtMS   int64
		updatedAtMS   int64
		typ           string
		scope         string
	)
	if err := rows.Scan(
		&m.ID, &m.SyncID, &m.BrainID, &m.Project, &scope, &typ,
		&topicKey, &m.Title, &m.Content,
		&tags, &m.NormalizedHash, &m.RevisionCount,
		&createdAtMS, &updatedAtMS, &deletedAtMS,
		&sessionID,
	); err != nil {
		return memory.Memory{}, fmt.Errorf("links: scan memory: %w", err)
	}
	m.Type = memory.Type(typ)
	m.Scope = memory.Scope(scope)
	if topicKey.Valid {
		m.TopicKey = topicKey.String
	}
	if tags.Valid && tags.String != "" {
		m.Tags = strings.Split(tags.String, ",")
	}
	m.CreatedAt = time.UnixMilli(createdAtMS).UTC()
	m.UpdatedAt = time.UnixMilli(updatedAtMS).UTC()
	if deletedAtMS.Valid {
		t := time.UnixMilli(deletedAtMS.Int64).UTC()
		m.DeletedAt = &t
	}
	if sessionID.Valid {
		m.SessionID = sessionID.String
	}
	return m, nil
}

// isUniqueConstraintError reports whether err is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

package links

// SQL constants — kept here so links.go stays readable.

const sqlCreateLink = `
INSERT INTO memory_links (brain_id, from_id, to_id, relation, weight, source, note, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

const sqlGetLinkByID = `
SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
FROM memory_links
WHERE id = ? AND brain_id = ?
`

const sqlDeleteLink = `
DELETE FROM memory_links
WHERE id = ? AND brain_id = ?
`

const sqlNeighbors = `
SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
FROM memory_links
WHERE brain_id = ?
  AND (from_id = ? OR to_id = ?)
`

const sqlNeighborsFiltered = `
SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
FROM memory_links
WHERE brain_id = ?
  AND (from_id = ? OR to_id = ?)
  AND relation = ?
`

const sqlEndpointBrainID = `
SELECT brain_id FROM memories WHERE id = ? AND deleted_at IS NULL
`

const sqlSubgraphNodes = `
SELECT id, sync_id, brain_id, project, scope, type, topic_key, title, content,
       tags, normalized_hash, revision_count, created_at, updated_at, deleted_at,
       session_id
FROM memories
WHERE brain_id = ? AND deleted_at IS NULL
`

const sqlSubgraphNodesTyped = `
SELECT id, sync_id, brain_id, project, scope, type, topic_key, title, content,
       tags, normalized_hash, revision_count, created_at, updated_at, deleted_at,
       session_id
FROM memories
WHERE brain_id = ? AND deleted_at IS NULL AND type = ?
`

const sqlSubgraphEdges = `
SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
FROM memory_links
WHERE brain_id = ?
`

const sqlSubgraphEdgesFiltered = `
SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
FROM memory_links
WHERE brain_id = ?
  AND relation = ?
`

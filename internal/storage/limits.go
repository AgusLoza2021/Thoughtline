package storage

import "time"

// DedupeWindow is the lookback interval used by DedupeCheck to suppress
// duplicate non-topic-keyed observations. A save whose normalized_hash
// matches an existing non-deleted, non-topic-keyed row within this window
// is treated as a duplicate and returns the existing row's ID instead of
// inserting a new row.
const DedupeWindow = 15 * time.Minute

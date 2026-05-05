package migrate

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// StructuredLogger writes one log line per event to an io.Writer in the format:
//
//	level=INFO sync_id=obs-abc123 action=created
//
// Keys are sorted for deterministic output. Both main.go (stdout + file) and
// tests that need to inspect output should use StructuredLogger; tests that
// don't care about log content should use NopLogger.
type StructuredLogger struct {
	w io.Writer
}

// NewStructuredLogger creates a StructuredLogger writing to w.
func NewStructuredLogger(w io.Writer) StructuredLogger {
	return StructuredLogger{w: w}
}

// Log writes a single key=value log line. Fields are sorted by key for
// determinism. The level and msg are always the first two tokens.
func (l StructuredLogger) Log(level, msg string, fields map[string]string) {
	var sb strings.Builder
	sb.WriteString("level=")
	sb.WriteString(level)
	sb.WriteString(" msg=")
	sb.WriteString(quoteIfSpaced(msg))

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteByte(' ')
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(quoteIfSpaced(fields[k]))
	}
	sb.WriteByte('\n')
	// Best-effort write; log failures don't abort the migration.
	_, _ = fmt.Fprint(l.w, sb.String())
}

func quoteIfSpaced(s string) string {
	if strings.ContainsAny(s, " \t\n") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

// NopLogger implements Logger by discarding every message. Use in tests that
// don't need to inspect log output, and in production contexts where no log
// destination is configured.
type NopLogger struct{}

// Log discards the message.
func (NopLogger) Log(_ string, _ string, _ map[string]string) {}

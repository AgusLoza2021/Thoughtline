// Package server wires the Model Context Protocol server.
//
// New() builds an mcp-go MCPServer pre-registered with every tl_* tool and
// returns it. The caller (cmd/thoughtline) is responsible for actually
// running the stdio transport via server.ServeStdio.
package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Config knobs for the server. Project is the fallback project identifier
// used when a tool call omits the `project` argument; in production it
// resolves to the basename of the working directory at boot.
type Config struct {
	Version        string
	DefaultProject string
}

// New constructs an MCPServer with all Thoughtline tools registered.
func New(s *storage.Storage, cfg Config) *server.MCPServer {
	srv := server.NewMCPServer(
		"thoughtline",
		cfg.Version,
		server.WithToolCapabilities(true),
		server.WithInstructions(serverInstructions),
	)

	registerTLSave(srv, s, cfg)
	registerTLSearch(srv, s, cfg)
	registerTLGetObservation(srv, s)
	registerTLContext(srv, s, cfg)
	registerTLUpdate(srv, s)
	registerTLDelete(srv, s)
	registerTLSessionStart(srv, s, cfg)
	registerTLSessionSummary(srv, s)
	return srv
}

// serverInstructions is sent once when an MCP client connects. It primes the
// model on what the server is for and how to use it.
const serverInstructions = `Thoughtline is a local-first persistent memory store designed for game-development workflows. Use the tl_* tools to save and recall project lore (design decisions, scene patterns, asset references, performance gotchas, pipeline steps, bug fixes, conventions).

Save proactively — don't wait to be asked. When making a non-trivial decision, fixing a bug, or noticing a gotcha, call tl_save with a short title, structured content (What/Why/Where/Learned), and a stable topic_key when the topic is likely to evolve.

Topic-key conventions:
  - design/<system>/<choice>          — game-design-decision
  - scene/<engine>/<pattern>          — scene-pattern
  - asset/<category>/<name>           — asset-reference
  - perf/<platform>/<area>            — perf-gotcha
  - pipeline/<source>-to-<target>/... — pipeline-step
  - script/<engine>/<concept>         — script-pattern
  - convention/<area>                 — convention
  - preference/<area>                 — preference (scope MUST be personal)`

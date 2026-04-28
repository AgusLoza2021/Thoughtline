// Package server wires the Model Context Protocol server.
//
// Responsibilities:
//
//   - Build an mcp-go server (github.com/mark3labs/mcp-go) over stdio.
//   - Register every tl_* tool with its JSON schema, description, and handler.
//   - Translate handler errors into MCP error envelopes the client can render.
//   - Own the lifecycle: graceful shutdown on SIGINT/SIGTERM, structured logs
//     to stderr (stdio is reserved for the protocol).
//
// What does NOT belong here:
//
//   - SQL or storage implementation — that lives in internal/storage.
//   - Domain types or validation — that lives in internal/memory.
//
// The split is deliberate: server is a thin adapter. Tool handlers should be
// at most a few dozen lines each, delegating to storage and returning a
// formatted MCP response.
//
// Status: skeleton. Real wiring lands in milestone M1 alongside tl_save.
package server

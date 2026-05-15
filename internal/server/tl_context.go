package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ContextResponseMaxChars is the total character budget for a tl_context
// response. formatContextResults will strip snippets and drop oldest entries
// to stay within this limit. It is defined here (server rendering policy)
// rather than in internal/memory/types.go (domain limits).
const ContextResponseMaxChars = 4000

// contextOmittedTailFmt is the tail appended when entries are dropped to fit
// the budget. ~60 chars — accounted for in the budget reservation.
const contextOmittedTailFmt = "\n... %d more memories omitted. Call tl_get_observation for details.\n"

// contextArgs is the typed shape of a tl_context call.
type contextArgs struct {
	Project string
	Limit   int
}

func registerTLContext(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_context",
			mcp.WithTitleAnnotation("Recent Memory Context"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlContextDescription),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max recent memories (default 10, hard cap 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeContextArgs(req)
			return doContext(ctx, s, cfg, args)
		},
	)
}

const tlContextDescription = `Return the most recently updated memories for the active project, ordered by updated_at DESC. Soft-deleted memories are excluded.

Use this PROACTIVELY at the start of a session, or after a context compaction, to recover what was being worked on. Returns the same per-result envelope as tl_search (Title / ID / Topic / Snippet / ...) so it can be parsed identically.`

func decodeContextArgs(req mcp.CallToolRequest) contextArgs {
	a := req.GetArguments()
	out := contextArgs{
		Project: asString(a, "project"),
		Limit:   asInt(a, "limit"),
	}
	out.Project = strings.TrimSpace(out.Project)
	return out
}

func doContext(ctx context.Context, s *storage.Storage, cfg Config, args contextArgs) (*mcp.CallToolResult, error) {
	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}
	if project == "" {
		return mcp.NewToolResultError("'project' is required (and could not be auto-detected). Pass it explicitly or restart the server in the project's working directory."), nil
	}

	// Read path: resolve brain, return empty on not-found (never auto-create on reads).
	brainID, err := s.ResolveBrainID(ctx, project)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Brain doesn't exist yet — no memories possible, return empty.
			return mcp.NewToolResultText(fmt.Sprintf("No recent memories for project %q.", project)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("resolve brain: %v", err)), nil
	}

	results, err := s.Recent(ctx, brainID, args.Limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("context lookup failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No recent memories for project %q.", project)), nil
	}

	header := fmt.Sprintf("Recent %d memorie(s) for project %q (newest first):\n\n", len(results), project)
	body := formatContextResults(results, ContextResponseMaxChars-utf8.RuneCountInString(header))
	return mcp.NewToolResultText(header + body), nil
}

// blockPair holds the two rendered forms of a single result block.
// full includes the Snippet line; head omits it (silent strip per design D14).
type blockPair struct {
	full string
	head string
}

// renderBlockPair renders both forms of a result block for the budget loop.
func renderBlockPair(r storage.SearchResult) blockPair {
	var fb strings.Builder
	writeResultBlock(&fb, r, false) // false: no BM25 score
	full := fb.String()

	// Head = full minus the "Snippet: …\n" line (always the last line).
	head := full
	if idx := strings.LastIndex(full, "\nSnippet: "); idx >= 0 {
		head = full[:idx+1] // keep the newline before "Snippet:"
	}
	return blockPair{full: full, head: head}
}

// formatContextResults applies the B1 budget algorithm to results and returns
// the formatted string. results MUST be non-empty; caller handles empty case.
//
// Algorithm (design #117 D13–D17):
//  1. Render full and head-only forms for each block.
//  2. Reserve ~80 chars for the omitted-count tail when len(results) > 1.
//  3. Greedily include blocks newest-first at full size.
//  4. When a block doesn't fit: strip snippets from oldest-included blocks
//     until there is room, then add the new block as head-only.
//  5. If even all-stripped still has no room: stop adding (oldest dropped).
//  6. Newest-entry floor: always emit blocks[0] even if it alone exceeds cap.
//  7. Append omitted-count tail when entries are dropped.
func formatContextResults(results []storage.SearchResult, maxChars int) string {
	n := len(results)

	// Reserve space for the omitted tail so it always fits if needed.
	effectiveMax := maxChars
	if n > 1 {
		// contextOmittedTailFmt at worst renders ~80 chars (3-digit count).
		effectiveMax = maxChars - 80
		if effectiveMax < 0 {
			effectiveMax = 0
		}
	}

	// Step 1 — render both forms up-front.
	blocks := make([]blockPair, n)
	for i, r := range results {
		blocks[i] = renderBlockPair(r)
	}

	// included: indices (into blocks/results) that fit, in newest-first order.
	included := make([]int, 0, n)
	stripped := make([]bool, n) // true when that block uses head-only form
	total := 0

	for nextIdx := 0; nextIdx < n; nextIdx++ {
		fullCost := utf8.RuneCountInString(blocks[nextIdx].full)
		headCost := utf8.RuneCountInString(blocks[nextIdx].head)

		// Try to fit the block at full size first.
		if total+fullCost <= effectiveMax {
			included = append(included, nextIdx)
			total += fullCost
			continue
		}

		// Try head-only for this block; make room by stripping older included blocks.
		for total+headCost > effectiveMax {
			// Find the oldest non-stripped included block.
			victim := -1
			for j := len(included) - 1; j >= 0; j-- {
				if !stripped[included[j]] {
					victim = j
					break
				}
			}
			if victim < 0 {
				break // nothing left to strip
			}
			vi := included[victim]
			saved := utf8.RuneCountInString(blocks[vi].full) - utf8.RuneCountInString(blocks[vi].head)
			total -= saved
			stripped[vi] = true
		}

		if total+headCost > effectiveMax {
			break // even all-stripped can't make room — this and subsequent are dropped
		}

		// Block enters as head-only.
		included = append(included, nextIdx)
		stripped[nextIdx] = true
		total += headCost
	}

	// Step 5 — newest-entry floor: always include blocks[0] even if over cap.
	if len(included) == 0 {
		// Force-include newest as head-only, regardless of size.
		included = append(included, 0)
		stripped[0] = true
	}

	// Step 4 — build output in newest-first order.
	var b strings.Builder
	for i, idx := range included {
		if i > 0 {
			b.WriteString("---\n")
		}
		if stripped[idx] {
			b.WriteString(blocks[idx].head)
		} else {
			b.WriteString(blocks[idx].full)
		}
	}

	omitted := n - len(included)
	if omitted > 0 {
		fmt.Fprintf(&b, contextOmittedTailFmt, omitted)
	} else {
		b.WriteString("\n---\n")
		b.WriteString("Snippets above are content previews (≤300 chars). Call tl_get_observation(id: <ID>) to read the full untruncated content of a specific entry.")
	}

	return strings.TrimRight(b.String(), "\n")
}

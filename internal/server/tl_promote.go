package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// promoteItem is a single entry in the tl_promote items array.
type promoteItem struct {
	PendingEventID int64
	Type           string
	TopicKey       string
	Title          string
	Content        string
	Scope          string
	Tags           []string
}

// promoteResult is the per-event outcome returned in the response.
type promoteResult struct {
	PendingEventID int64  `json:"pending_event_id"`
	MemoryID       int64  `json:"memory_id,omitempty"`
	SyncID         string `json:"sync_id,omitempty"`
	Status         string `json:"status"` // "promoted" | "error"
	Error          string `json:"error,omitempty"`
}

func registerTLPromote(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_promote",
			mcp.WithTitleAnnotation("Promote Pending Events"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(`Promote one or more pending hook events into typed memories.

For each item in the batch, this tool:
  1. Verifies the event exists and has status=pending.
  2. Creates a new memory using the supplied type, title, and content.
  3. Sets the event status to promoted and records the memory ID.

Each event is committed in its own transaction — failures in one event do NOT
roll back other events in the batch. Returns a result per event ID.`),
			mcp.WithArray("items",
				mcp.Required(),
				mcp.Description("Array of items to promote. Each item must have pending_event_id, type, title, and content."),
				mcp.Items(map[string]any{"type": "object"}),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return doTLPromote(ctx, s, cfg, req.GetArguments())
		},
	)
}

func doTLPromote(ctx context.Context, s *storage.Storage, cfg Config, args map[string]any) (*mcp.CallToolResult, error) {
	rawItems, ok := args["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return mcp.NewToolResultError("'items' is required and must be a non-empty array"), nil
	}

	items, err := decodePromoteItems(rawItems)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("decode items: %v", err)), nil
	}

	results := make([]promoteResult, 0, len(items))

	for _, item := range items {
		res := promoteOne(ctx, s, cfg, item)
		results = append(results, res)
	}

	// Encode results as JSON.
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("encode results: %v", err)), nil
	}

	promoted := 0
	errored := 0
	for _, r := range results {
		if r.Status == "promoted" {
			promoted++
		} else {
			errored++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "tl_promote: %d promoted, %d errors\n\n", promoted, errored)
	b.Write(jsonBytes)

	return mcp.NewToolResultText(b.String()), nil
}

// promoteOne handles a single item in its own transaction. It never returns an
// error — failures are encoded in the promoteResult.Error field.
func promoteOne(ctx context.Context, s *storage.Storage, cfg Config, item promoteItem) promoteResult {
	res := promoteResult{PendingEventID: item.PendingEventID}

	// Validate item.
	if item.PendingEventID == 0 {
		res.Status = "error"
		res.Error = "pending_event_id is required"
		return res
	}

	scope := item.Scope
	if scope == "" {
		if item.Type == string(memory.TypePreference) {
			scope = string(memory.ScopePersonal)
		} else {
			scope = string(memory.ScopeProject)
		}
	}

	// Retrieve the pending event to get project context.
	ev, err := s.GetPendingByID(ctx, item.PendingEventID)
	if errors.Is(err, storage.ErrPendingNotFound) {
		res.Status = "error"
		res.Error = "not found"
		return res
	}
	if err != nil {
		res.Status = "error"
		res.Error = fmt.Sprintf("get pending event: %v", err)
		return res
	}

	if ev.Status == "promoted" {
		res.Status = "error"
		res.Error = "already promoted"
		return res
	}
	if ev.Status != "pending" {
		res.Status = "error"
		res.Error = fmt.Sprintf("event is %s, not pending", ev.Status)
		return res
	}

	m := memory.Memory{
		Project:  ev.Project,
		Scope:    memory.Scope(scope),
		Type:     memory.Type(item.Type),
		TopicKey: item.TopicKey,
		Title:    item.Title,
		Content:  item.Content,
		Tags:     item.Tags,
	}

	if err := memory.Validate(m); err != nil {
		res.Status = "error"
		res.Error = fmt.Sprintf("invalid memory: %v", err)
		return res
	}

	// Resolve project → brainID. Pending events capture a project string at
	// hook time; tl_promote must resolve it. If no brain exists for this project,
	// return an error rather than auto-creating — the event was captured before
	// any tl_save ran for this project and creation policy is the user's call.
	brainID, brainErr := s.ResolveBrainID(ctx, ev.Project)
	if brainErr != nil {
		res.Status = "error"
		res.Error = fmt.Sprintf("no brain found for project: %s", ev.Project)
		return res
	}

	saved, _, saveErr := s.Save(ctx, brainID, m)
	if saveErr != nil {
		res.Status = "error"
		res.Error = fmt.Sprintf("save memory: %v", saveErr)
		return res
	}

	if err := s.MarkPromoted(ctx, item.PendingEventID, saved.ID); err != nil {
		// Memory was saved but mark failed — this is a partial success that
		// leaves the event still pending (retryable). Return error so the
		// caller knows to retry.
		res.Status = "error"
		res.Error = fmt.Sprintf("mark promoted: %v", err)
		return res
	}

	res.Status = "promoted"
	res.MemoryID = saved.ID
	res.SyncID = saved.SyncID
	return res
}

// decodePromoteItems converts the raw []any from the MCP arguments into
// typed promoteItem structs.
func decodePromoteItems(raw []any) ([]promoteItem, error) {
	items := make([]promoteItem, 0, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d is not an object", i)
		}
		item := promoteItem{
			PendingEventID: int64(asFloat(m, "pending_event_id")),
			Type:           asString(m, "type"),
			TopicKey:       asString(m, "topic_key"),
			Title:          asString(m, "title"),
			Content:        asString(m, "content"),
			Scope:          asString(m, "scope"),
			Tags:           asStringSlice(m, "tags"),
		}
		items = append(items, item)
	}
	return items, nil
}

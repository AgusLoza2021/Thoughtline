// tl_check.go — knowledge-check MCP tool.
//
// The tool answers one question for an agent that is about to write code:
// "which of the owner's hand-written recipes, and which rules of this project,
// apply to this change?" It is a read-only tool: a check never saves, never
// creates a brain and never mutates the corpus.
package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// checkArgs is the typed shape of a tl_check call.
type checkArgs struct {
	Task             string
	Paths            []string
	Project          string
	KnowledgeProject string
	ExcludeStacks    []string
	Limit            int
	Detail           bool
}

func registerTLCheck(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_check",
			mcp.WithTitleAnnotation("Knowledge Check"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlCheckDescription),
			mcp.WithString("task",
				mcp.Description("What you are about to build or fix, in one sentence (e.g. 'add camera shake when the player gets hit')."),
			),
			mcp.WithArray("paths",
				mcp.Description("Files the change will touch. Their names are part of the query, so 'Assets/EnemyHitFlashSystem.cs' finds the hit-flash recipes."),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("project",
				mcp.Description("Project whose saved conventions/decisions/architecture should be checked first. Defaults to the working directory project."),
			),
			mcp.WithString("knowledge_project",
				mcp.Description("Brain holding the recipe corpus. Default: the configured knowledge project (THOUGHTLINE_KNOWLEDGE_PROJECT, else '"+knowledge.DefaultSyncProject+"')."),
			),
			mcp.WithArray("exclude_stacks",
				mcp.Description("Foreign technology stacks to skip (e.g. ['svn','fiber']). Entries tagged stack:<name> are dropped and counted in the 'excluded' line."),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithNumber("limit",
				mcp.Description(fmt.Sprintf("Max recipe hits (default %d, hard cap %d). Project rules are separate: at most 3 are listed first.", knowledge.DefaultCheckLimit, knowledge.MaxCheckLimit)),
			),
			mcp.WithBoolean("detail",
				mcp.Description("Return each hit's Applies when / Validate / Limits sections. Default false: one line per hit with its id, to be read with tl_get_observation."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeCheckArgs(req)
			return doCheck(ctx, s, cfg, args)
		},
	)
}

const tlCheckDescription = `Check what the owner already wrote about this change BEFORE you write or edit code for a feature.

Call this FIRST — before the first edit of a task, not after it — with a one-sentence 'task' and the files you are about to touch. It searches the hand-written recipe corpus (Unity camera/UI/VFX/physics/editor recipes and similar) plus the current project's saved conventions, decisions and architecture.

The default answer is one line per hit: source, title, the curated keys that made it match, the memory id and the topic_key. Read a hit with tl_get_observation(id) only when its title and keys say it applies; pass detail:true when you want the Applies when / Validate / Limits sections inline instead.

Project rules are listed first on purpose: a rule saved for this project always wins over a generic recipe.

Rules for using the answer:
- If a hit's keys or sections match what you are doing, follow its procedure; read the full entry with tl_get_observation(id) when the one line is not enough.
- Validate the result as the entry says, not as you would have done it.
- Ignore hits that do not match; a curated key match is intent, not a promise of fit.

An empty answer is normal and means nothing applies: proceed. Entries from foreign stacks (SVN, Fiber, other projects) are excluded and counted on the last line.`

func decodeCheckArgs(req mcp.CallToolRequest) checkArgs {
	a := req.GetArguments()
	out := checkArgs{
		Task:             strings.TrimSpace(asString(a, "task")),
		Paths:            asStringSlice(a, "paths"),
		Project:          strings.TrimSpace(asString(a, "project")),
		KnowledgeProject: strings.TrimSpace(asString(a, "knowledge_project")),
		ExcludeStacks:    asStringSlice(a, "exclude_stacks"),
		Limit:            asInt(a, "limit"),
		Detail:           asBool(a, "detail"),
	}
	return out
}

func doCheck(ctx context.Context, s *storage.Storage, cfg Config, args checkArgs) (*mcp.CallToolResult, error) {
	if args.Task == "" && len(args.Paths) == 0 {
		return mcp.NewToolResultError("at least one of 'task' or 'paths' is required"), nil
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}
	// The knowledge project and the excluded stacks are operator config: an
	// agent that omits them must get the configured corpus, not an empty one.
	knowledgeProject := args.KnowledgeProject
	if knowledgeProject == "" {
		knowledgeProject = cfg.KnowledgeProject
	}
	exclude := args.ExcludeStacks
	if len(exclude) == 0 {
		exclude = cfg.KnowledgeExclude
	}

	res, err := knowledge.Check(ctx, s, knowledge.CheckRequest{
		Task:             args.Task,
		Paths:            args.Paths,
		Project:          project,
		KnowledgeProject: knowledgeProject,
		ExcludeStacks:    exclude,
		Limit:            args.Limit,
		MinCoverage:      cfg.KnowledgeMinCoverage,
		MinWeight:        cfg.KnowledgeMinWeight,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("knowledge check failed: %v", err)), nil
	}
	return mcp.NewToolResultText(formatCheckResult(res, args.Detail)), nil
}

// formatCheckResult renders the Markdown an agent reads.
//
// Compact is the default because a check is a pre-flight step, not a reading
// task: a line names the hit and its id, and the entry itself is one
// tl_get_observation call away. Detail is today's sectioned form, kept for the
// cases where re-asking is more expensive than the extra tokens.
func formatCheckResult(res knowledge.CheckResult, detail bool) string {
	var b strings.Builder
	if len(res.Hits) == 0 {
		// Say how close the corpus came instead of just "nothing": an agent that
		// knows the corpus was consulted stops re-asking with the same words.
		if res.Best != nil && res.Best.Coverage > 0 {
			fmt.Fprintf(&b, "No applicable knowledge for this task (best candidate had coverage %d: %s).\n",
				res.Best.Coverage, res.Best.Title)
		} else {
			b.WriteString("No applicable knowledge for this task.\n")
		}
		fmt.Fprintf(&b, "excluded %d foreign-stack entries", res.Excluded)
		return b.String()
	}

	fmt.Fprintf(&b, "Knowledge check for %q — %d hit(s). Project rules come first and override generic recipes.\n",
		res.Query, len(res.Hits))
	for _, h := range res.Hits {
		if !detail {
			b.WriteString(knowledge.FormatHitLine(h))
			b.WriteString("\n")
			continue
		}
		b.WriteString("\n")
		b.WriteString(knowledge.FormatHitDetail(h))
	}
	fmt.Fprintf(&b, "\nexcluded %d foreign-stack entries", res.Excluded)
	if !detail {
		b.WriteString("\n")
		b.WriteString(knowledge.CompactFooter)
	}
	return b.String()
}

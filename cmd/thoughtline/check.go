package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Exit codes for `thoughtline check`. The hook mode is the reason these exist:
// it must always exit 0 so a Claude Code PreToolUse hook can never block an
// edit, while the interactive form should still report a real failure.
const (
	checkExitOK    = 0
	checkExitFatal = 1
	checkExitUsage = 2
)

// hookEditTools are the Claude Code tools whose payload names a file this check
// can say something about. Anything else (Bash, Read, ...) is ignored.
var hookEditTools = map[string]bool{"Edit": true, "Write": true, "MultiEdit": true}

const (
	// hookMaxHits / hookMaxChars keep the additionalContext small enough to
	// ride along on every edit without eating the agent's prompt budget.
	hookMaxHits  = 3
	hookMaxChars = 2500
)

// checkUsage is short on purpose; the README carries the full reference.
const checkUsage = `Usage:
  thoughtline check --task "..." [--path FILE]... [--project NAME] [--limit N] [--json]
  thoughtline check --hook [--ext .cs]

Reports the hand-written recipes and project rules that apply to a change.

  --task TEXT      what you are about to build or fix
  --path FILE      file involved in the change (repeatable)
  --project NAME   project whose conventions are checked first (default: cwd basename)
  --limit N        max recipe hits (default 5, cap 10)
  --min-coverage N distinct task terms an entry must match (default 2)
  --min-weight F   minimum IDF weight of those terms (default: scaled to the
                   corpus, i.e. two terms that appear in a tenth of it)
  --json           print the query, hits and excluded count as JSON
  --hook           Claude Code PreToolUse mode: read the payload from stdin and
                   emit hookSpecificOutput JSON, or nothing at all
  --ext EXT        hook mode: only check files with this extension (default .cs)

Environment:
  THOUGHTLINE_KNOWLEDGE_PROJECT        corpus brain (default ` + knowledge.DefaultSyncProject + `)
  THOUGHTLINE_KNOWLEDGE_EXCLUDE        comma list of foreign stacks to skip (e.g. svn,fiber)
  THOUGHTLINE_KNOWLEDGE_MIN_COVERAGE   default for --min-coverage
  THOUGHTLINE_KNOWLEDGE_MIN_WEIGHT     default for --min-weight`

// repeatedString collects a flag that may be given many times (--path).
type repeatedString []string

func (r *repeatedString) String() string { return strings.Join(*r, ",") }

func (r *repeatedString) Set(v string) error {
	*r = append(*r, v)
	return nil
}

// hookStateDir resolves the directory holding per-session hook dedupe files.
// A variable so tests point it at a temp dir instead of the real one.
var hookStateDir = func() string {
	return filepath.Join(os.TempDir(), "thoughtline-check")
}

// runCheck implements `thoughtline check` and returns the process exit code.
// It does not return an error because hook mode owns an exit-0-always
// contract that main()'s error path cannot express.
func runCheck(ctx context.Context, args []string, stdin io.Reader, out, errW io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(errW)
	fs.Usage = func() { fmt.Fprintln(errW, checkUsage) }

	task := fs.String("task", "", "what you are about to build or fix")
	var paths repeatedString
	fs.Var(&paths, "path", "file involved in the change (repeatable)")
	project := fs.String("project", "", "project whose rules are checked first")
	limit := fs.Int("limit", knowledge.DefaultCheckLimit, "max recipe hits")
	minCoverage := fs.Int("min-coverage", 0, "distinct task terms an entry must match")
	minWeight := fs.Float64("min-weight", 0, "minimum IDF weight of the matched terms")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	hook := fs.Bool("hook", false, "Claude Code PreToolUse mode")
	ext := fs.String("ext", ".cs", "hook mode: file extension to check")

	if err := fs.Parse(args); err != nil {
		return checkExitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(errW, "thoughtline check: unexpected argument %q\n\n%s\n", fs.Arg(0), checkUsage)
		return checkExitUsage
	}

	req := knowledge.CheckRequest{
		Task:             *task,
		Paths:            paths,
		Project:          *project,
		KnowledgeProject: os.Getenv("THOUGHTLINE_KNOWLEDGE_PROJECT"),
		ExcludeStacks:    splitCSV(os.Getenv("THOUGHTLINE_KNOWLEDGE_EXCLUDE")),
		Limit:            *limit,
		MinCoverage:      *minCoverage,
		MinWeight:        *minWeight,
	}
	if req.MinCoverage <= 0 {
		// The flag is the per-run override; the env var is the installed default.
		req.MinCoverage = envInt("THOUGHTLINE_KNOWLEDGE_MIN_COVERAGE")
	}
	if req.MinWeight <= 0 {
		req.MinWeight = envFloat("THOUGHTLINE_KNOWLEDGE_MIN_WEIGHT")
	}
	if req.Project == "" {
		// Same fallback the MCP server uses, so a hook reports on the same
		// project the tools do.
		req.Project = resolveDefaultProject()
	}

	if *hook {
		// A hook has no task text, so its query is a bare file name: it gets at
		// least the interactive gate, even when the config lowered it.
		if req.MinCoverage < knowledge.HookMinCoverage {
			req.MinCoverage = knowledge.HookMinCoverage
		}
		return runCheckHook(ctx, req, *ext, stdin, out)
	}
	return runCheckOnce(ctx, req, *asJSON, out, errW)
}

// runCheckOnce is the interactive form: open the database, run one check and
// print it. A missing database is an empty corpus, never an error.
func runCheckOnce(ctx context.Context, req knowledge.CheckRequest, asJSON bool, out, errW io.Writer) int {
	if req.Task == "" && len(req.Paths) == 0 {
		fmt.Fprintf(errW, "thoughtline check: --task or --path is required\n\n%s\n", checkUsage)
		return checkExitUsage
	}

	st, closeStore, err := openCheckStorage()
	if err != nil {
		fmt.Fprintf(errW, "thoughtline check: %v\n", err)
		return checkExitFatal
	}
	defer closeStore()

	res, err := knowledge.Check(ctx, st, req)
	if err != nil {
		fmt.Fprintf(errW, "thoughtline check: %v\n", err)
		return checkExitFatal
	}

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintf(errW, "thoughtline check: encode result: %v\n", err)
			return checkExitFatal
		}
		return checkExitOK
	}

	fmt.Fprint(out, formatCheckCLI(res))
	return checkExitOK
}

// formatCheckCLI is the human/script view: one line per hit that carries the
// title, score and coverage, which is what tuning the thresholds reads. The
// sections are left to --json and to the MCP tool, which is what an agent
// consumes.
func formatCheckCLI(res knowledge.CheckResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "query: %s\n", res.Query)
	if len(res.Hits) == 0 {
		b.WriteString("no hits")
		if res.Best != nil && res.Best.Coverage > 0 {
			fmt.Fprintf(&b, " (best candidate had coverage %d: %s)", res.Best.Coverage, res.Best.Title)
		}
		b.WriteString("\n")
	}
	for _, h := range res.Hits {
		fmt.Fprintf(&b, "[%s] %s (%s) score=%.2f coverage=%d", h.Source, h.Title, h.TopicKey, h.Score, h.Coverage)
		if len(h.Matched) > 0 {
			fmt.Fprintf(&b, ": %s", strings.Join(h.Matched, ", "))
		}
		fmt.Fprintf(&b, " (weight %.1f)\n", h.Weight)
	}
	fmt.Fprintf(&b, "excluded %d foreign-stack entries\n", res.Excluded)
	return b.String()
}

// openCheckStorage opens the database for a read-only check. Unlike the write
// commands it refuses to create one: a check fired by an editor hook on a
// machine with no memories yet must not leave a database behind.
func openCheckStorage() (*storage.Storage, func(), error) {
	dbPath, err := resolveDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve db path: %w", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, func() {}, nil
		}
		return nil, nil, fmt.Errorf("stat db %s: %w", dbPath, err)
	}
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open storage at %s: %w", dbPath, err)
	}
	return st, func() { _ = st.Close() }, nil
}

// preToolUse is the subset of the Claude Code PreToolUse payload this hook
// needs. Everything else in the payload is ignored.
type preToolUse struct {
	SessionID string
	ToolName  string
	FilePath  string
}

func parsePreToolUse(raw []byte) (preToolUse, bool) {
	var doc struct {
		SessionID string `json:"session_id"`
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return preToolUse{}, false
	}
	p := preToolUse{
		SessionID: strings.TrimSpace(doc.SessionID),
		ToolName:  strings.TrimSpace(doc.ToolName),
		FilePath:  strings.TrimSpace(doc.ToolInput.FilePath),
	}
	if p.ToolName == "" || p.FilePath == "" {
		return preToolUse{}, false
	}
	return p, true
}

// hookOutput is the PreToolUse contract: it adds context and never decides.
// Encoding a struct (not a map) keeps the field order the documented shape.
type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// runCheckHook is the PreToolUse entry point. Every failure path — bad stdin,
// unknown tool, foreign extension, missing database, already reported — prints
// nothing and exits 0: a knowledge hint is never worth blocking an edit.
func runCheckHook(ctx context.Context, req knowledge.CheckRequest, ext string, stdin io.Reader, out io.Writer) int {
	raw, err := io.ReadAll(io.LimitReader(stdin, maxHookPayloadBytes+1))
	if err != nil || len(raw) > maxHookPayloadBytes || len(raw) == 0 {
		return checkExitOK
	}
	p, ok := parsePreToolUse(raw)
	if !ok || !hookEditTools[p.ToolName] || !hasExtension(p.FilePath, ext) {
		return checkExitOK
	}

	session := p.SessionID
	if session == "" {
		// An unidentified session still deserves one hint per file; the state
		// file just lives under a fixed name.
		session = "default"
	}
	stateDir := hookStateDir()
	if alreadyReported(stateDir, session, p.FilePath) {
		return checkExitOK
	}

	// The hook knows only the file: the path tokens are the whole query.
	req.Paths = []string{p.FilePath}
	req.Task = ""

	st, closeStore, err := openCheckStorage()
	if err != nil {
		return checkExitOK
	}
	defer closeStore()

	res, err := knowledge.Check(ctx, st, req)
	if err != nil {
		return checkExitOK
	}
	context := hookContext(p.FilePath, res.Hits)
	if context == "" {
		return checkExitOK
	}
	// Recorded before printing so a crash mid-write cannot cause a second hint
	// for the same file in the same session.
	markReported(stateDir, session, p.FilePath)

	enc := json.NewEncoder(out)
	if err := enc.Encode(hookOutput{HookSpecificOutput: hookSpecificOutput{
		HookEventName:     "PreToolUse",
		AdditionalContext: context,
	}}); err != nil {
		return checkExitOK
	}
	return checkExitOK
}

// hookContext renders the smallest useful reminder: the file, at most three
// hits, and each hit's validation step. "Applies when" and "Limits" are
// dropped because the agent is already editing — what it cannot derive on its
// own is how to prove the result right.
func hookContext(file string, hits []knowledge.CheckHit) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Thoughtline knowledge check for %s:", file)
	for i, h := range hits {
		if i == hookMaxHits {
			break
		}
		fmt.Fprintf(&b, "\n- %s (%s)", h.Title, h.TopicKey)
		if v := strings.TrimSpace(h.Validate); v != "" {
			fmt.Fprintf(&b, "\n  Validate: %s", strings.ReplaceAll(v, "\n", "\n  "))
		}
	}
	return clampRunes(strings.TrimRight(b.String(), "\n"), hookMaxChars)
}

// clampRunes truncates to max runes, marking the cut with an ellipsis.
func clampRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimRight(string(r[:max-1]), " \n") + "…"
}

// hasExtension compares extensions case-insensitively and accepts the flag with
// or without its leading dot.
func hasExtension(path, ext string) bool {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return true
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return strings.EqualFold(filepath.Ext(path), ext)
}

// sessionState is the on-disk shape of the per-session dedupe set. A sorted
// slice keeps the file deterministic and readable when someone inspects it.
type sessionState struct {
	Files []string `json:"files"`
}

func stateFile(dir, session string) string {
	return filepath.Join(dir, sanitizeSessionID(session)+".json")
}

func loadSessionState(dir, session string) map[string]bool {
	raw, err := os.ReadFile(stateFile(dir, session))
	if err != nil {
		return nil
	}
	var doc sessionState
	if err := json.Unmarshal(raw, &doc); err != nil {
		// A corrupt state file must not silence the hook for the rest of the
		// session: treat it as empty and let the next write replace it.
		return nil
	}
	files := make(map[string]bool, len(doc.Files))
	for _, f := range doc.Files {
		files[f] = true
	}
	return files
}

func alreadyReported(dir, session, path string) bool {
	return loadSessionState(dir, session)[path]
}

// markReported is best-effort: a read-only temp dir costs a duplicate hint, not
// a failed edit, so the caller ignores the error.
func markReported(dir, session, path string) {
	files := loadSessionState(dir, session)
	if files == nil {
		files = make(map[string]bool, 4)
	}
	files[path] = true

	doc := sessionState{Files: make([]string, 0, len(files))}
	for f := range files {
		doc.Files = append(doc.Files, f)
	}
	sort.Strings(doc.Files)

	raw, err := json.Marshal(doc)
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(stateFile(dir, session), raw, 0o600)
}

// sanitizeSessionID keeps a session id from escaping the state directory while
// staying stable, so the same session always maps to the same file.
func sanitizeSessionID(session string) string {
	var b strings.Builder
	b.Grow(len(session))
	for _, r := range session {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	s := b.String()
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Exit codes for `thoughtline knowledge`. They match main()'s contract: 0 for a
// completed run even when some entries were skipped, 1 for a fatal error (bad
// directory, unusable database) and 2 for a usage mistake.
const (
	knowledgeExitOK    = 0
	knowledgeExitFatal = 1
	knowledgeExitUsage = 2
)

// runKnowledge implements the `thoughtline knowledge` subcommand group and
// returns the process exit code, unlike its siblings in this package which
// return an error: a sync with skipped entries is a success with a warning, and
// main()'s error path can only express failure.
func runKnowledge(args []string) int {
	return runKnowledgeTo(args, os.Stdout, os.Stderr)
}

// runKnowledgeTo carries the writers so CLI behaviour (usage errors, summary
// line, per-entry errors) is testable without capturing process output.
func runKnowledgeTo(args []string, out, errW io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errW, knowledgeUsage)
		return knowledgeExitUsage
	}
	switch args[0] {
	case "sync":
		return runKnowledgeSync(args[1:], out, errW)
	case "list":
		return runKnowledgeList(args[1:], out, errW)
	case "help", "-h", "--help":
		fmt.Fprintln(out, knowledgeUsage)
		return knowledgeExitOK
	default:
		fmt.Fprintf(errW, "thoughtline knowledge: unknown action %q\n\n%s\n", args[0], knowledgeUsage)
		return knowledgeExitUsage
	}
}

const knowledgeUsage = `Usage:
  thoughtline knowledge sync <dir> [flags]
  thoughtline knowledge list [--project NAME] [--json]

sync imports a corpus directory (SKILL.md index + entries/*.md) as memories.
  --project NAME   brain to write to (default ` + knowledge.DefaultSyncProject + `)
  --corpus ID      corpus id used in tags and the footer (default ` + knowledge.DefaultSyncCorpus + `)
  --dry-run        report what would change without writing
  --no-prune       keep entries whose file disappeared
  --json           print the report as JSON

list prints "topic_key<TAB>title" for the active entries of the corpus project.`

func runKnowledgeSync(args []string, out, errW io.Writer) int {
	fs := flag.NewFlagSet("knowledge sync", flag.ContinueOnError)
	fs.SetOutput(errW)
	fs.Usage = func() { fmt.Fprintln(errW, knowledgeUsage) }

	project := fs.String("project", knowledge.DefaultSyncProject, "brain to write to")
	corpus := fs.String("corpus", knowledge.DefaultSyncCorpus, "corpus id used in tags")
	dryRun := fs.Bool("dry-run", false, "report without writing")
	noPrune := fs.Bool("no-prune", false, "keep entries whose file disappeared")
	asJSON := fs.Bool("json", false, "print the report as JSON")

	// The documented form puts <dir> first, but Go's flag package stops at the
	// first non-flag argument. Peel a leading directory off so both
	// `sync <dir> --json` and `sync --json <dir>` work.
	dir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return knowledgeExitUsage
	}
	count := fs.NArg()
	if dir != "" {
		count++
	}
	switch {
	case count > 1:
		fmt.Fprintf(errW, "thoughtline knowledge sync: expected one directory, got %d\n\n%s\n", count, knowledgeUsage)
		return knowledgeExitUsage
	case dir == "" && count == 1:
		dir = fs.Arg(0)
	case dir == "":
		fmt.Fprintf(errW, "thoughtline knowledge sync: missing directory\n\n%s\n", knowledgeUsage)
		return knowledgeExitUsage
	}

	scanned, err := knowledge.Scan(dir)
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}

	st, closeStore, err := openKnowledgeStorage()
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}
	defer closeStore()

	rep, err := knowledge.Sync(context.Background(), st, scanned, knowledge.SyncOptions{
		Project: *project,
		Corpus:  *corpus,
		DryRun:  *dryRun,
		Prune:   !*noPrune,
	})
	// Errors inside the report are per-entry; a non-nil error means the run
	// itself could not proceed.
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}

	for _, msg := range rep.Errors {
		fmt.Fprintf(errW, "thoughtline knowledge: %s\n", msg)
	}
	if *asJSON {
		if err := writeJSONReport(out, rep); err != nil {
			fmt.Fprintf(errW, "thoughtline knowledge: encode report: %v\n", err)
			return knowledgeExitFatal
		}
		return knowledgeExitOK
	}
	fmt.Fprintln(out, summaryLine(rep))
	return knowledgeExitOK
}

func runKnowledgeList(args []string, out, errW io.Writer) int {
	fs := flag.NewFlagSet("knowledge list", flag.ContinueOnError)
	fs.SetOutput(errW)
	fs.Usage = func() { fmt.Fprintln(errW, knowledgeUsage) }

	project := fs.String("project", knowledge.DefaultSyncProject, "brain to read")
	asJSON := fs.Bool("json", false, "print entries as JSON")
	if err := fs.Parse(args); err != nil {
		return knowledgeExitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(errW, "thoughtline knowledge list: unexpected argument %q\n\n%s\n", fs.Arg(0), knowledgeUsage)
		return knowledgeExitUsage
	}

	st, closeStore, err := openKnowledgeStorage()
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}
	defer closeStore()

	ctx := context.Background()
	brainID, err := st.ResolveBrainID(ctx, *project)
	if errors.Is(err, sql.ErrNoRows) {
		// Nothing synced yet is not an error: an empty listing is the answer.
		return knowledgeExitOK
	}
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}

	rows, err := st.ListByTopicPrefix(ctx, brainID, knowledge.TopicPrefix)
	if err != nil {
		fmt.Fprintf(errW, "thoughtline knowledge: %v\n", err)
		return knowledgeExitFatal
	}
	if *asJSON {
		if err := writeJSONEntries(out, rows); err != nil {
			fmt.Fprintf(errW, "thoughtline knowledge: encode entries: %v\n", err)
			return knowledgeExitFatal
		}
		return knowledgeExitOK
	}
	for _, m := range rows {
		fmt.Fprintf(out, "%s\t%s\n", m.TopicKey, m.Title)
	}
	return knowledgeExitOK
}

// openKnowledgeStorage opens the database through the same path resolution the
// server uses, so THOUGHTLINE_DB / THOUGHTLINE_HOME behave identically here.
func openKnowledgeStorage() (*storage.Storage, func(), error) {
	dbPath, err := resolveDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve db path: %w", err)
	}
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open storage at %s: %w", dbPath, err)
	}
	return st, func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "thoughtline: close storage: %v\n", cerr)
		}
	}, nil
}

// summaryLine is the one-line contract scripts grep for.
func summaryLine(rep knowledge.SyncReport) string {
	return fmt.Sprintf(
		"scanned=%d created=%d updated=%d unchanged=%d deleted=%d skipped=%d",
		rep.Scanned, rep.Created, rep.Updated, rep.Unchanged, rep.Deleted, rep.Skipped,
	)
}

type jsonSyncReport struct {
	Scanned   int      `json:"scanned"`
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Unchanged int      `json:"unchanged"`
	Deleted   int      `json:"deleted"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
}

func writeJSONReport(w io.Writer, rep knowledge.SyncReport) error {
	payload := jsonSyncReport{
		Scanned:   rep.Scanned,
		Created:   rep.Created,
		Updated:   rep.Updated,
		Unchanged: rep.Unchanged,
		Deleted:   rep.Deleted,
		Skipped:   rep.Skipped,
		Errors:    rep.Errors,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

type jsonEntry struct {
	TopicKey string   `json:"topic_key"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Tags     []string `json:"tags,omitempty"`
}

func writeJSONEntries(w io.Writer, rows []memory.Memory) error {
	entries := make([]jsonEntry, 0, len(rows))
	for _, m := range rows {
		entries = append(entries, jsonEntry{
			TopicKey: m.TopicKey,
			Title:    m.Title,
			Type:     string(m.Type),
			Tags:     m.Tags,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

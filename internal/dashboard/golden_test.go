package dashboard

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// updateGolden re-generates golden files when set. Run:
//   go test ./internal/dashboard/ -run "Golden" -update
var updateGolden = flag.Bool("update", false, "regenerate golden testdata files")

// ansiRE strips ANSI CSI escape sequences from lipgloss output so goldens
// are deterministic across machines/terminal color profiles. Lipgloss emits
// color codes only when a renderer detects a TTY; under `go test` it usually
// emits nothing, but stripping defensively keeps the goldens portable.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// relTimeRE matches the dynamic relTime outputs ("just now", "Nm ago",
// "Nh ago", "Nd ago", "YYYY-MM-DD"). Goldens normalize these to a fixed
// "<reltime>" placeholder so wall-clock drift doesn't break the snapshots.
var relTimeRE = regexp.MustCompile(`(just now|\d+m ago|\d+h ago|\d+d ago|\d{4}-\d{2}-\d{2})`)

// normalizeForGolden strips ANSI escapes, substitutes <reltime> for any
// dynamic relative-time string, AND normalizes line endings to LF. This is
// the single normalization point — goldens generated with -update are saved
// AFTER normalization.
//
// CRLF normalization is critical for CI: the repo has `.gitattributes *
// text=auto`, which causes Git to materialize text files with CRLF on
// Windows checkout. Without this normalization, the bytes read from
// testdata/*.golden on Windows runners contain \r\n while the rendered
// output from lipgloss uses \n, producing spurious golden mismatches.
func normalizeForGolden(s string) string {
	s = stripANSI(s)
	s = relTimeRE.ReplaceAllString(s, "<reltime>")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

// assertGolden compares got against testdata/<name>.golden, stripping ANSI
// codes from both sides. When -update is passed, got is written to the file
// instead and the test passes silently.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	got = normalizeForGolden(got)
	path := filepath.Join("testdata", name+".golden")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}

	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (re-run with -update to generate)", path, err)
	}
	want := normalizeForGolden(string(wantBytes))

	if got != want {
		t.Errorf("golden mismatch for %s — re-run with -update if change is intentional", path)
		t.Logf("--- want (len=%d)\n%s", len(want), want)
		t.Logf("+++ got  (len=%d)\n%s", len(got), got)

		// Surface first diverging line for quick triage.
		wantLines := strings.Split(want, "\n")
		gotLines := strings.Split(got, "\n")
		n := len(wantLines)
		if len(gotLines) < n {
			n = len(gotLines)
		}
		for i := 0; i < n; i++ {
			if wantLines[i] != gotLines[i] {
				t.Logf("first diff at line %d:\n  want: %q\n  got:  %q", i+1, wantLines[i], gotLines[i])
				break
			}
		}
	}
}

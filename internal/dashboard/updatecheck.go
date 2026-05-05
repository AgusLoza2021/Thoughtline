package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// openURLCmd opens a URL in the user's default browser. Best-effort: if
// the platform launcher is missing or fails, the command silently
// returns. The caller doesn't get feedback because there's nothing
// useful to surface in a TUI.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Start()
		return nil
	}
}

// updateInfoMsg carries the result of a GitHub releases lookup back into
// the Bubbletea Update loop. Empty Latest means "no info" (network
// down, rate-limited, parse error). The Cmd swallows errors silently so
// a failed check never spooks the user.
type updateInfoMsg struct {
	Latest    string // tag name, e.g. "v0.1.0"
	URL       string // html_url for humans
	Available bool   // strictly greater than current
}

const githubLatestReleaseURL = "https://api.github.com/repos/AgusLoza2021/Thoughtline/releases/latest"

// checkUpdateCmd returns a Bubbletea command that hits the GitHub
// releases API and reports back via updateInfoMsg. Runs with a tight
// timeout (3s) so a slow network never blocks the UI.
func checkUpdateCmd(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
		if err != nil {
			return updateInfoMsg{}
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "thoughtline/"+currentVersion)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return updateInfoMsg{}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return updateInfoMsg{}
		}

		var body struct {
			TagName string `json:"tag_name"`
			HTMLURL string `json:"html_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return updateInfoMsg{}
		}

		return updateInfoMsg{
			Latest:    body.TagName,
			URL:       body.HTMLURL,
			Available: isNewer(body.TagName, currentVersion),
		}
	}
}

// isNewer is a deliberately small semver comparison: it strips a leading
// "v", splits on ".", and lexicographically compares each numeric part.
// Pre-release suffixes ("-rc1") are ignored. Good enough for a "latest
// release available" pill — not a package manager.
func isNewer(latest, current string) bool {
	l := normalizeVersion(latest)
	c := normalizeVersion(current)
	if l == "" || c == "" {
		return false
	}
	if l == c {
		return false
	}

	lp := splitVersion(l)
	cp := splitVersion(c)
	for i := 0; i < len(lp) || i < len(cp); i++ {
		var lv, cv int
		if i < len(lp) {
			lv = lp[i]
		}
		if i < len(cp) {
			cv = cp[i]
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return false
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	return v
}

func splitVersion(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}

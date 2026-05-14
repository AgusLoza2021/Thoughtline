package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// helpers_test.go covers the eight helpers extracted into helpers.go (task C1).
// Each test is table-driven via t.Run.

func TestHelpers_RelTime(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"zero time renders dash", time.Time{}, "—"},
		{"just-now sub-minute", now.Add(-30 * time.Second), "just now"},
		{"minutes ago", now.Add(-15 * time.Minute), "15m ago"},
		{"hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"days ago", now.Add(-2 * 24 * time.Hour), "2d ago"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := relTime(tc.in)
			if got != tc.want {
				t.Errorf("relTime(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	t.Run("old time falls back to YYYY-MM-DD", func(t *testing.T) {
		old := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
		got := relTime(old)
		if got != "2020-01-15" {
			t.Errorf("relTime(old) = %q, want %q", got, "2020-01-15")
		}
	})
}

func TestHelpers_TruncateLeft(t *testing.T) {
	cases := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"empty within max", "", 10, ""},
		{"exact width unchanged", "abcdefghij", 10, "abcdefghij"},
		{"under width unchanged", "abc", 10, "abc"},
		{"overflow truncates with ellipsis", "abcdefghijklmnop", 8, "abcdefg…"},
		{"max under 4 clamped", "abcde", 2, "abc…"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateLeft(tc.s, tc.max)
			if got != tc.want {
				t.Errorf("truncateLeft(%q, %d) = %q, want %q", tc.s, tc.max, got, tc.want)
			}
		})
	}
}

func TestHelpers_Wrap(t *testing.T) {
	t.Run("short string unchanged", func(t *testing.T) {
		got := wrap("hello", 80)
		if got != "hello" {
			t.Errorf("wrap short = %q, want %q", got, "hello")
		}
	})

	t.Run("exact-width single line", func(t *testing.T) {
		s := "abcdefghij" // 10 chars
		got := wrap(s, 10)
		if got != s {
			t.Errorf("wrap exact width = %q, want %q", got, s)
		}
	})

	t.Run("overflow wraps at clamped width", func(t *testing.T) {
		// wrap clamps width to a minimum of 8, so a 10-char input wraps to
		// 8-char chunks: "abcdefgh\nij".
		s := "abcdefghij"
		got := wrap(s, 4)
		want := "abcdefgh\nij"
		if got != want {
			t.Errorf("wrap overflow = %q, want %q", got, want)
		}
	})

	t.Run("explicit-width wrap above clamp", func(t *testing.T) {
		// width=10 is above the clamp; 24-char string wraps to 10-char chunks.
		s := "abcdefghijklmnopqrstuvwx" // 24 chars
		got := wrap(s, 10)
		want := "abcdefghij\nklmnopqrst\nuvwx"
		if got != want {
			t.Errorf("wrap at width 10 = %q, want %q", got, want)
		}
	})

	t.Run("width below 8 clamped to 8", func(t *testing.T) {
		s := "abcdefghijabcdefghij" // 20 chars
		got := wrap(s, 1)
		// 8-wide chunks: abcdefgh\nijabcdef\nghij
		want := "abcdefgh\nijabcdef\nghij"
		if got != want {
			t.Errorf("wrap with clamp = %q, want %q", got, want)
		}
	})
}

func TestHelpers_PaneBox(t *testing.T) {
	t.Run("renders body within border", func(t *testing.T) {
		out := paneBox("hello", 20, 5, defaultPalette)
		if !strings.Contains(out, "hello") {
			t.Errorf("paneBox output must contain body, got:\n%s", out)
		}
		// Normal box border characters.
		if !strings.ContainsAny(out, "─│") {
			t.Errorf("paneBox output must contain a border, got:\n%s", out)
		}
	})
}

func TestHelpers_ClampCursor(t *testing.T) {
	cases := []struct {
		name      string
		c, n      int
		want      int
	}{
		{"in range", 2, 5, 2},
		{"negative clamped to 0", -3, 5, 0},
		{"over max clamped to n-1", 99, 5, 4},
		{"empty list → 0", 3, 0, 0},
		{"exactly at upper boundary", 4, 5, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampCursor(tc.c, tc.n)
			if got != tc.want {
				t.Errorf("clampCursor(%d,%d) = %d, want %d", tc.c, tc.n, got, tc.want)
			}
		})
	}
}

func TestHelpers_PadRight(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"shorter is padded", "ab", 5, "ab   "},
		{"exact width unchanged", "abcde", 5, "abcde"},
		{"longer unchanged", "abcdefgh", 5, "abcdefgh"},
		{"empty padded full", "", 3, "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := padRight(tc.in, tc.n)
			if got != tc.want {
				t.Errorf("padRight(%q,%d) = %q, want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}

func TestHelpers_CenterString(t *testing.T) {
	out := centerString("hi", 40, 10, defaultPalette)
	if !strings.Contains(out, "hi") {
		t.Errorf("centerString must include the text, got:\n%s", out)
	}
	// It should also include leading newlines for vertical centering.
	if !strings.HasPrefix(out, "\n") {
		t.Errorf("centerString must vertically center via leading newlines, got prefix %q", out[:1])
	}
}

func TestHelpers_FormatLastSave(t *testing.T) {
	t.Run("empty list returns empty string", func(t *testing.T) {
		got := formatLastSave(nil)
		if got != "" {
			t.Errorf("formatLastSave(nil) = %q, want empty", got)
		}
	})

	t.Run("non-empty uses first item updated_at", func(t *testing.T) {
		now := time.Now()
		items := []storage.SearchResult{
			{UpdatedAt: now.Add(-2 * time.Hour)},
			{UpdatedAt: now.Add(-1 * time.Hour)},
		}
		got := formatLastSave(items)
		if got != "2h ago" {
			t.Errorf("formatLastSave items[0]=2h ago, got %q", got)
		}
	})
}

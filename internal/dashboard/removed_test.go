package dashboard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// removed_test.go asserts the negative-assertion requirements from
// tui-removed/spec.md (Req 4–12). Each test verifies that a legacy file,
// symbol, or palette value has been removed from the codebase.
//
// These tests are intentionally written against the target post-deletion state.
// They FAIL while the legacy files exist (RED) and turn GREEN after H2.

// readPackageSource reads all non-test .go files in the dashboard package dir
// into a single string for grep assertions.
func readPackageSource(t *testing.T) string {
	t.Helper()
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	var parts []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		// Skip test files — negative assertions must target production code only.
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		parts = append(parts, string(raw))
	}
	return strings.Join(parts, "\n")
}

// TestRemoved_CubeFileAbsent asserts Req 4: internal/dashboard/cube.go must
// not exist.
func TestRemoved_CubeFileAbsent(t *testing.T) {
	_, err := os.Stat("cube.go")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("cube.go must not exist after the legacy deletion (Req 4)")
	}
}

// TestRemoved_SplashFileAbsent asserts Req 5: internal/dashboard/splash.go
// must not exist.
func TestRemoved_SplashFileAbsent(t *testing.T) {
	_, err := os.Stat("splash.go")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("splash.go must not exist after the legacy deletion (Req 5)")
	}
}

// TestRemoved_ThemesFileAbsent asserts Req 10:
// internal/dashboard/themes.go must not exist.
func TestRemoved_ThemesFileAbsent(t *testing.T) {
	_, err := os.Stat("themes.go")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("themes.go must not exist after the legacy deletion (Req 10)")
	}
}

// TestRemoved_TabsTestFileAbsent asserts Req 9:
// internal/dashboard/tabs_test.go must not exist.
func TestRemoved_TabsTestFileAbsent(t *testing.T) {
	_, err := os.Stat("tabs_test.go")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tabs_test.go must not exist after the legacy deletion (Req 9)")
	}
}

// TestRemoved_LegacyModelFilesAbsent asserts Req 11 (file-existence half):
// model.go, view.go, update.go must not exist.
func TestRemoved_LegacyModelFilesAbsent(t *testing.T) {
	for _, name := range []string{"model.go", "view.go", "update.go"} {
		_, err := os.Stat(name)
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s must not exist after the legacy deletion (Req 11)", name)
		}
	}
}

// TestRemoved_MultiThemeSymbolsAbsent asserts Req 10 (symbol half):
// ThemeBrand, ThemeZBrush, ThemeMono, ApplyTheme, nextTheme must not appear
// in any production source file in the dashboard package.
func TestRemoved_MultiThemeSymbolsAbsent(t *testing.T) {
	src := readPackageSource(t)

	forbidden := []string{
		"ThemeBrand",
		"ThemeZBrush",
		"ThemeMono",
		"ApplyTheme(",
		"nextTheme",
	}
	for _, tok := range forbidden {
		if strings.Contains(src, tok) {
			t.Errorf("production source must not contain %q after themes.go deletion (Req 10)", tok)
		}
	}
}

// TestRemoved_LegacyModelSymbolsAbsent asserts Req 11 (symbol half):
// legacyTabKey enum, func (m Model) View(), func (m Model) Update( must not
// appear in any production source file.
func TestRemoved_LegacyModelSymbolsAbsent(t *testing.T) {
	src := readPackageSource(t)

	forbidden := []string{
		"legacyTabKey",
		"func (m Model) View()",
		"func (m Model) Update(",
	}
	for _, tok := range forbidden {
		if strings.Contains(src, tok) {
			t.Errorf("production source must not contain %q after legacy Model deletion (Req 11)", tok)
		}
	}
}

// TestRemoved_ForbiddenRosePineMoonHex asserts Req 12:
// the 10 forbidden Rose-Pine-Moon hex values must not appear anywhere in
// internal/dashboard/*.go production files.
// #C4A7E7 is explicitly allowed (carry-over as Brand / BadgeMemoryType).
func TestRemoved_ForbiddenRosePineMoonHex(t *testing.T) {
	src := strings.ToLower(readPackageSource(t))

	forbidden := []string{
		"#e0def4", // Foreground
		"#6e6a86", // Muted
		"#393552", // Border
		"#a88dc9", // LogoGradient[1]
		"#9ccfd8", // LogoGradient[2] / legacy StatusOK
		"#3e8fb0", // LogoGradient[3]
		"#56949f", // LogoGradient[4]
		"#eb6f92", // legacy StatNumber / StatusErr
		"#f6c177", // legacy Cursor / StatusWarn
		"#2a273f", // legacy MenuSelectedBg
	}
	for _, hex := range forbidden {
		if strings.Contains(src, hex) {
			t.Errorf("production source must not contain Rose-Pine-Moon hex %q (Req 12)", hex)
		}
	}
}

package main

import (
	"errors"
	"strings"
	"testing"
)

// O1 — `thoughtline ui` with a removed flag returns the friendly migration
// error. Both `--flag=value` and bare `--flag` (space-separator) forms hit
// the same error. `--no-splashy` MUST NOT match `--no-splash`.
func TestCheckRemovedUIFlags(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
		wantSub string // expected substring in the error message
	}{
		{
			name:    "no removed flag — passes",
			args:    []string{"--no-update-check"},
			wantErr: false,
		},
		{
			name:    "no args — passes",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "--theme bare token",
			args:    []string{"--theme", "brand"},
			wantErr: true,
			wantSub: "--theme was removed in v0.2",
		},
		{
			name:    "--theme=value",
			args:    []string{"--theme=zbrush"},
			wantErr: true,
			wantSub: "--theme was removed in v0.2",
		},
		{
			name:    "--no-splash bare",
			args:    []string{"--no-splash"},
			wantErr: true,
			wantSub: "--no-splash was removed in v0.2",
		},
		{
			name:    "--no-splash=false",
			args:    []string{"--no-splash=false"},
			wantErr: true,
			wantSub: "--no-splash was removed in v0.2",
		},
		{
			name:    "--splash-ms bare",
			args:    []string{"--splash-ms", "2000"},
			wantErr: true,
			wantSub: "--splash-ms was removed in v0.2",
		},
		{
			name:    "--splash-ms=value",
			args:    []string{"--splash-ms=1500"},
			wantErr: true,
			wantSub: "--splash-ms was removed in v0.2",
		},
		{
			// FALSE-POSITIVE GUARD: --no-splashy must NOT match --no-splash.
			name:    "--no-splashy must not match --no-splash",
			args:    []string{"--no-splashy"},
			wantErr: false,
		},
		{
			// FALSE-POSITIVE GUARD: --themed must NOT match --theme.
			name:    "--themed must not match --theme",
			args:    []string{"--themed"},
			wantErr: false,
		},
		{
			// FALSE-POSITIVE GUARD: --no-update-check is preserved (NOT removed).
			name:    "--no-update-check preserved",
			args:    []string{"--no-update-check"},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkRemovedUIFlags(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for args=%v, got nil", tc.args)
				}
				if !errors.Is(err, errRemovedUIFlag) {
					t.Errorf("error does not match errRemovedUIFlag sentinel: %v", err)
				}
				if !strings.Contains(err.Error(), tc.wantSub) {
					t.Errorf("error message %q does not contain %q", err.Error(), tc.wantSub)
				}
				// Exact format check: must start with "thoughtline ui: " and end with "See CHANGELOG.md."
				if !strings.HasPrefix(err.Error(), "thoughtline ui: ") {
					t.Errorf("error must start with 'thoughtline ui: '; got %q", err.Error())
				}
				if !strings.HasSuffix(err.Error(), "See CHANGELOG.md.") {
					t.Errorf("error must end with 'See CHANGELOG.md.'; got %q", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for args=%v, got: %v", tc.args, err)
				}
			}
		})
	}
}

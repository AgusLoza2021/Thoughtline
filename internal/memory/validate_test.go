package memory

import (
	"errors"
	"strings"
	"testing"
)

// validMemory returns a passable Memory for tests to mutate one field at a time.
func validMemory() Memory {
	return Memory{
		Project: "enchanted-inn",
		Scope:   ScopeProject,
		Type:    TypeSceneEntry(),
		Title:   "Lock player loot UI to a 4x6 grid",
		Content: "**What**: chose a grid\n**Why**: cognitive load on mobile\n**Where**: ui/loot/grid.js\n",
	}
}

// TypeSceneEntry returns a non-preference type for the validMemory baseline.
// Picked separately so the baseline test case isn't anchored to a single type.
func TypeSceneEntry() Type { return TypeScenePattern }

func TestValidate_HappyPath(t *testing.T) {
	if err := Validate(validMemory()); err != nil {
		t.Fatalf("expected valid memory, got %v", err)
	}
}

func TestValidate_TypeRules(t *testing.T) {
	t.Run("invalid type rejected", func(t *testing.T) {
		m := validMemory()
		m.Type = "made-up-type"
		if err := Validate(m); !errors.Is(err, ErrInvalidType) {
			t.Fatalf("expected ErrInvalidType, got %v", err)
		}
	})

	t.Run("every catalogued type accepts when scope matches", func(t *testing.T) {
		for _, typ := range AllTypes() {
			m := validMemory()
			m.Type = typ
			if typ == TypePreference {
				m.Scope = ScopePersonal
			} else {
				m.Scope = ScopeProject
			}
			if err := Validate(m); err != nil {
				t.Errorf("type %q (scope %q) should be valid; got %v", typ, m.Scope, err)
			}
		}
	})
}

func TestValidate_ScopeRules(t *testing.T) {
	t.Run("invalid scope rejected", func(t *testing.T) {
		m := validMemory()
		m.Scope = "team"
		if err := Validate(m); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("expected ErrInvalidScope, got %v", err)
		}
	})

	t.Run("preference must be personal", func(t *testing.T) {
		m := validMemory()
		m.Type = TypePreference
		m.Scope = ScopeProject
		if err := Validate(m); !errors.Is(err, ErrPreferenceMustBePersonal) {
			t.Fatalf("expected ErrPreferenceMustBePersonal, got %v", err)
		}
	})

	t.Run("non-preference must be project", func(t *testing.T) {
		m := validMemory()
		m.Type = TypeBugfix
		m.Scope = ScopePersonal
		if err := Validate(m); !errors.Is(err, ErrNonPreferenceMustBeProject) {
			t.Fatalf("expected ErrNonPreferenceMustBeProject, got %v", err)
		}
	})
}

func TestValidate_RequiredFields(t *testing.T) {
	t.Run("empty project rejected", func(t *testing.T) {
		m := validMemory()
		m.Project = "   "
		if err := Validate(m); !errors.Is(err, ErrEmptyProject) {
			t.Fatalf("expected ErrEmptyProject, got %v", err)
		}
	})

	t.Run("empty title rejected", func(t *testing.T) {
		m := validMemory()
		m.Title = ""
		if err := Validate(m); !errors.Is(err, ErrEmptyTitle) {
			t.Fatalf("expected ErrEmptyTitle, got %v", err)
		}
	})

	t.Run("whitespace-only title rejected", func(t *testing.T) {
		m := validMemory()
		m.Title = "   \t\n"
		if err := Validate(m); !errors.Is(err, ErrEmptyTitle) {
			t.Fatalf("expected ErrEmptyTitle for whitespace title, got %v", err)
		}
	})

	t.Run("empty content rejected", func(t *testing.T) {
		m := validMemory()
		m.Content = ""
		if err := Validate(m); !errors.Is(err, ErrEmptyContent) {
			t.Fatalf("expected ErrEmptyContent, got %v", err)
		}
	})
}

func TestValidate_LengthLimits(t *testing.T) {
	t.Run("title at 200 chars accepted", func(t *testing.T) {
		m := validMemory()
		m.Title = strings.Repeat("a", MaxTitleChars)
		if err := Validate(m); err != nil {
			t.Fatalf("title at limit should be valid, got %v", err)
		}
	})

	t.Run("title over 200 chars rejected", func(t *testing.T) {
		m := validMemory()
		m.Title = strings.Repeat("a", MaxTitleChars+1)
		if err := Validate(m); !errors.Is(err, ErrTitleTooLong) {
			t.Fatalf("expected ErrTitleTooLong, got %v", err)
		}
	})

	t.Run("title length is in runes not bytes", func(t *testing.T) {
		m := validMemory()
		// 100 'á' = 100 runes but 200 bytes (2 bytes per char in UTF-8)
		m.Title = strings.Repeat("á", 100)
		if err := Validate(m); err != nil {
			t.Fatalf("100-rune title should be valid (runes < limit), got %v", err)
		}
	})

	t.Run("content at limit accepted", func(t *testing.T) {
		m := validMemory()
		m.Content = strings.Repeat("a", MaxContentBytes)
		if err := Validate(m); err != nil {
			t.Fatalf("content at limit should be valid, got %v", err)
		}
	})

	t.Run("content over limit rejected", func(t *testing.T) {
		m := validMemory()
		m.Content = strings.Repeat("a", MaxContentBytes+1)
		if err := Validate(m); !errors.Is(err, ErrContentTooLong) {
			t.Fatalf("expected ErrContentTooLong, got %v", err)
		}
	})
}

func TestValidate_TopicKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
		ok   bool
	}{
		{"empty topic_key allowed", "", true},
		{"simple slug", "design/inventory-grid", true},
		{"deep nesting", "perf/android/static-batching", true},
		{"underscores ok", "convention/asset_naming", true},
		{"single char too short", "a", false},
		{"uppercase rejected", "Design/Inventory", false},
		{"leading slash rejected", "/design/inventory", false},
		{"space rejected", "design inventory", false},
		{"trailing chars only after lead", "0", false}, // single char
		{"long under 129 ok", "a/" + strings.Repeat("b", 127), true},
		{"too long rejected", "a/" + strings.Repeat("b", 128), false}, // 130 chars total
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMemory()
			m.TopicKey = tc.key
			err := Validate(m)
			if tc.ok && err != nil {
				t.Fatalf("topic_key %q should be valid, got %v", tc.key, err)
			}
			if !tc.ok && !errors.Is(err, ErrInvalidTopicKey) {
				t.Fatalf("topic_key %q should fail with ErrInvalidTopicKey, got %v", tc.key, err)
			}
		})
	}
}

func TestValidate_Tags(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		ok   bool
	}{
		{"simple tag", "android", true},
		{"key:value tag", "engine:playcanvas", true},
		{"hyphen tag", "high-priority", true},
		{"underscore tag", "draft_v2", true},
		{"empty tag rejected", "", false},
		{"uppercase tag rejected", "Android", false},
		{"space tag rejected", "high priority", false},
		{"leading hyphen rejected", "-tag", false},
		{"too-long tag rejected", strings.Repeat("a", MaxTagLen+2), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMemory()
			m.Tags = []string{tc.tag}
			err := Validate(m)
			if tc.ok && err != nil {
				t.Fatalf("tag %q should be valid, got %v", tc.tag, err)
			}
			if !tc.ok && !errors.Is(err, ErrInvalidTag) {
				t.Fatalf("tag %q should fail with ErrInvalidTag, got %v", tc.tag, err)
			}
		})
	}
}

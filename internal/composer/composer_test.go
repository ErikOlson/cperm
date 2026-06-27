package composer_test

import (
	"reflect"
	"testing"

	"github.com/erikmav/cperm/internal/composer"
	"github.com/erikmav/cperm/internal/model"
	"github.com/erikmav/cperm/internal/store"
)

// newTestStore returns a store backed by a throwaway temp dir, seeded with the
// given modules.
func newTestStore(t *testing.T, mods ...*model.Module) *store.Store {
	t.Helper()
	s := &store.Store{Dir: t.TempDir()}
	for _, m := range mods {
		if err := s.Save(m); err != nil {
			t.Fatalf("saving module %q: %v", m.Name, err)
		}
	}
	return s
}

// allowMod is a convenience constructor for a module that only contributes
// allow rules and optionally declares dependencies.
func allowMod(name string, allow []string, requires ...string) *model.Module {
	return &model.Module{
		Name:        name,
		Requires:    requires,
		Permissions: model.Permissions{Allow: allow},
	}
}

func compose(t *testing.T, s *store.Store, cf *model.ComposeFile) *model.ComposedResult {
	t.Helper()
	result, err := composer.New(s).Compose(cf)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	return result
}

func TestComposeResolvesDependenciesBeforeDependents(t *testing.T) {
	s := newTestStore(t,
		allowMod("base", []string{"Read"}),
		allowMod("go", []string{"Bash(go:*)"}, "base"),
	)

	result := compose(t, s, &model.ComposeFile{Modules: []string{"go"}})

	if want := []string{"base", "go"}; !reflect.DeepEqual(result.ModulesUsed, want) {
		t.Errorf("ModulesUsed = %v, want %v", result.ModulesUsed, want)
	}
	// base's rules precede go's because base is resolved first.
	if want := []string{"Read", "Bash(go:*)"}; !reflect.DeepEqual(result.Policy.Permissions.Allow, want) {
		t.Errorf("Allow = %v, want %v", result.Policy.Permissions.Allow, want)
	}
}

func TestComposeResolvesDiamondDependencyOnce(t *testing.T) {
	s := newTestStore(t,
		allowMod("base", []string{"Read"}),
		allowMod("b", []string{"Bash(b:*)"}, "base"),
		allowMod("c", []string{"Bash(c:*)"}, "base"),
		allowMod("top", []string{"Bash(top:*)"}, "b", "c"),
	)

	result := compose(t, s, &model.ComposeFile{Modules: []string{"top"}})

	if want := []string{"base", "b", "c", "top"}; !reflect.DeepEqual(result.ModulesUsed, want) {
		t.Errorf("ModulesUsed = %v, want %v", result.ModulesUsed, want)
	}
}

func TestComposeDeduplicatesSharedRules(t *testing.T) {
	s := newTestStore(t,
		allowMod("a", []string{"Read", "Edit"}),
		allowMod("b", []string{"Read", "Write"}),
	)

	result := compose(t, s, &model.ComposeFile{Modules: []string{"a", "b"}})

	if want := []string{"Read", "Edit", "Write"}; !reflect.DeepEqual(result.Policy.Permissions.Allow, want) {
		t.Errorf("Allow = %v, want %v (order-preserving dedup)", result.Policy.Permissions.Allow, want)
	}
	if result.Deduplicated != 1 {
		t.Errorf("Deduplicated = %d, want 1", result.Deduplicated)
	}
	if result.AllowCount != 3 {
		t.Errorf("AllowCount = %d, want 3", result.AllowCount)
	}
}

func TestComposeDetectsAllowDenyConflict(t *testing.T) {
	s := newTestStore(t, &model.Module{
		Name: "x",
		Permissions: model.Permissions{
			Allow: []string{"Bash(rm:*)"},
			Deny:  []string{"Bash(rm:*)"},
		},
	})

	result := compose(t, s, &model.ComposeFile{Modules: []string{"x"}})

	if len(result.Conflicts) != 1 {
		t.Fatalf("Conflicts = %d, want 1", len(result.Conflicts))
	}
	c := result.Conflicts[0]
	if c.Rule != "Bash(rm:*)" || !c.InAllow || !c.InDeny || c.InAsk {
		t.Errorf("conflict = %+v, want rule Bash(rm:*) in allow+deny only", c)
	}
}

func TestComposeAppliesOverrideLast(t *testing.T) {
	s := newTestStore(t, allowMod("base", []string{"Read"}))

	result := compose(t, s, &model.ComposeFile{
		Modules:  []string{"base"},
		Override: &model.Permissions{Allow: []string{"Bash(atlas:*)"}},
	})

	if want := []string{"Read", "Bash(atlas:*)"}; !reflect.DeepEqual(result.Policy.Permissions.Allow, want) {
		t.Errorf("Allow = %v, want %v (override appended last)", result.Policy.Permissions.Allow, want)
	}
}

func TestComposeMergesEnvAndPassesSettingsThrough(t *testing.T) {
	s := newTestStore(t, &model.Module{
		Name: "t",
		Env:  map[string]string{"K": "V"},
	})

	result := compose(t, s, &model.ComposeFile{
		Modules:  []string{"t"},
		Settings: map[string]any{"defaultMode": "plan"},
	})

	if result.Policy.Env["K"] != "V" {
		t.Errorf("Env = %v, want K=V", result.Policy.Env)
	}
	if result.Policy.Settings["defaultMode"] != "plan" {
		t.Errorf("Settings = %v, want defaultMode=plan", result.Policy.Settings)
	}
}

func TestComposeRejectsCircularDependency(t *testing.T) {
	s := newTestStore(t,
		allowMod("a", nil, "b"),
		allowMod("b", nil, "a"),
	)

	if _, err := composer.New(s).Compose(&model.ComposeFile{Modules: []string{"a"}}); err == nil {
		t.Error("expected an error for a circular dependency")
	}
}

func TestComposeRejectsUnknownModule(t *testing.T) {
	s := newTestStore(t)

	if _, err := composer.New(s).Compose(&model.ComposeFile{Modules: []string{"ghost"}}); err == nil {
		t.Error("expected an error for an unknown module")
	}
}

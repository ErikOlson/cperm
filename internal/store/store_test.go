package store_test

import (
	"reflect"
	"testing"

	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/store"
)

func tempStore(t *testing.T) *store.Store {
	t.Helper()
	return &store.Store{Dir: t.TempDir()}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := tempStore(t)
	want := &model.Module{
		Name:        "go",
		Description: "Go toolchain",
		Version:     "0.1.0",
		Requires:    []string{"base"},
		Permissions: model.Permissions{Allow: []string{"Bash(go:*)"}},
		Env:         map[string]string{"K": "V"},
	}

	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("go")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestListReturnsSortedNames(t *testing.T) {
	s := tempStore(t)
	for _, n := range []string{"web", "base", "go"} {
		if err := s.Save(&model.Module{Name: n}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"base", "go", "web"}; !reflect.DeepEqual(got, want) {
		t.Errorf("List = %v, want %v (sorted)", got, want)
	}
}

func TestExistsAndDelete(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(&model.Module{Name: "x"}); err != nil {
		t.Fatal(err)
	}

	if !s.Exists("x") {
		t.Error("Exists(x) = false, want true")
	}
	if err := s.Delete("x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Exists("x") {
		t.Error("Exists(x) = true after Delete")
	}
	if err := s.Delete("x"); err == nil {
		t.Error("Delete of a missing module should error")
	}
}

func TestLoadMissingErrors(t *testing.T) {
	if _, err := tempStore(t).Load("ghost"); err == nil {
		t.Error("Load of a missing module should error")
	}
}

func TestInstallBuiltinsSeedsButDoesNotClobber(t *testing.T) {
	s := tempStore(t)

	// A user's customization of a built-in module must survive seeding.
	if err := s.Save(&model.Module{Name: "base", Description: "my custom base"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InstallBuiltins(); err != nil {
		t.Fatalf("InstallBuiltins: %v", err)
	}

	names, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"base", "git", "go", "node", "python"} {
		if !contains(names, want) {
			t.Errorf("builtin %q not seeded; have %v", want, names)
		}
	}

	got, err := s.Load("base")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "my custom base" {
		t.Errorf("InstallBuiltins overwrote a user module: description = %q", got.Description)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

package cmd

import (
	"reflect"
	"testing"

	"github.com/erikolson/cperm/internal/model"
)

func TestComputeDrift(t *testing.T) {
	expected := model.Permissions{
		Allow: []string{"Read", "Edit"},
		Ask:   []string{"Bash(git push:*)"},
	}
	actual := model.Permissions{
		Allow: []string{"Read", "Bash(curl:*)"},
		Deny:  []string{"Read(**/.env)"},
	}

	d := computeDrift(expected, actual)

	// Added = present in actual but not expected (manual approvals).
	if want := []string{"Bash(curl:*)"}; !reflect.DeepEqual(d.Added.Allow, want) {
		t.Errorf("Added.Allow = %v, want %v", d.Added.Allow, want)
	}
	if want := []string{"Read(**/.env)"}; !reflect.DeepEqual(d.Added.Deny, want) {
		t.Errorf("Added.Deny = %v, want %v", d.Added.Deny, want)
	}

	// Removed = present in expected but not actual.
	if want := []string{"Edit"}; !reflect.DeepEqual(d.Removed.Allow, want) {
		t.Errorf("Removed.Allow = %v, want %v", d.Removed.Allow, want)
	}
	if want := []string{"Bash(git push:*)"}; !reflect.DeepEqual(d.Removed.Ask, want) {
		t.Errorf("Removed.Ask = %v, want %v", d.Removed.Ask, want)
	}

	if d.addedCount() != 2 {
		t.Errorf("addedCount = %d, want 2", d.addedCount())
	}
	if d.removedCount() != 2 {
		t.Errorf("removedCount = %d, want 2", d.removedCount())
	}
}

func TestDiffSliceIsNonNil(t *testing.T) {
	// Empty diffs must be [] (non-nil) so the JSON report renders arrays, not null.
	got := diffSlice([]string{"a"}, []string{"a"})
	if got == nil {
		t.Fatal("diffSlice returned nil; want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("diffSlice = %v, want empty", got)
	}
}

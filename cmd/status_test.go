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

func TestUncovered(t *testing.T) {
	// Empty result must be [] (non-nil) so the JSON report renders arrays, not null.
	if got := uncovered([]string{"Read"}, []string{"Read"}); got == nil || len(got) != 0 {
		t.Fatalf("uncovered(exact match) = %v, want empty non-nil slice", got)
	}
	// A narrow rule covered by a broad one is not reported as drift.
	if got := uncovered([]string{"Bash(git add *)"}, []string{"Bash(git:*)"}); len(got) != 0 {
		t.Errorf("uncovered = %v, want [] (git add * is covered by git:*)", got)
	}
	// A genuinely novel rule is reported.
	if got := uncovered([]string{"Bash(npm test)"}, []string{"Bash(git:*)"}); len(got) != 1 {
		t.Errorf("uncovered = %v, want [Bash(npm test)]", got)
	}
}

func TestUnionPermissions(t *testing.T) {
	a := model.Permissions{Allow: []string{"Read", "Edit"}, Ask: []string{"Bash(git push:*)"}}
	b := model.Permissions{Allow: []string{"Edit", "Bash(curl:*)"}, Deny: []string{"Read(**/.env)"}}

	got := unionPermissions(a, b)

	if want := []string{"Read", "Edit", "Bash(curl:*)"}; !reflect.DeepEqual(got.Allow, want) {
		t.Errorf("Allow = %v, want %v (deduped, order-preserving)", got.Allow, want)
	}
	if want := []string{"Bash(git push:*)"}; !reflect.DeepEqual(got.Ask, want) {
		t.Errorf("Ask = %v, want %v", got.Ask, want)
	}
	if want := []string{"Read(**/.env)"}; !reflect.DeepEqual(got.Deny, want) {
		t.Errorf("Deny = %v, want %v", got.Deny, want)
	}
}

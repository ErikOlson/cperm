package importer_test

import (
	"reflect"
	"testing"

	"github.com/erikolson/cperm/internal/importer"
	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/store"
)

func TestSuggestModuleName(t *testing.T) {
	tests := []struct {
		name  string
		rules []string
		want  string
	}{
		{"bash command prefix", []string{"Bash(go:*)", "Bash(go build:*)"}, "go"},
		{"file ops", []string{"Read(./a)", "Edit(./b)"}, "files"},
		{"mcp server", []string{"mcp__github__get_issue"}, "mcp-github"},
		{"web tools", []string{"WebFetch", "WebSearch"}, "web"},
		{"no recognizable rules", nil, "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := importer.SuggestModuleName(tt.rules); got != tt.want {
				t.Errorf("SuggestModuleName(%v) = %q, want %q", tt.rules, got, tt.want)
			}
		})
	}
}

func TestAnalyzeReportsMatchesAndUnmatched(t *testing.T) {
	s := &store.Store{Dir: t.TempDir()}
	if err := s.Save(&model.Module{
		Name:        "go",
		Permissions: model.Permissions{Allow: []string{"Bash(go:*)", "Bash(gofmt:*)"}},
	}); err != nil {
		t.Fatal(err)
	}

	// One rule covered by the 'go' module, one covered by nothing.
	perms := model.Permissions{Allow: []string{"Bash(go:*)", "Bash(mystery:*)"}}

	result, err := importer.Analyze(perms, s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if result.TotalRules != 2 {
		t.Errorf("TotalRules = %d, want 2", result.TotalRules)
	}
	if result.CoveredRules != 1 {
		t.Errorf("CoveredRules = %d, want 1", result.CoveredRules)
	}
	if want := []string{"Bash(mystery:*)"}; !reflect.DeepEqual(result.UnmatchedAllow, want) {
		t.Errorf("UnmatchedAllow = %v, want %v", result.UnmatchedAllow, want)
	}

	// 'go' is a partial match: 1 of its 2 rules is present in the settings.
	if len(result.Matches) != 1 {
		t.Fatalf("Matches = %d, want 1", len(result.Matches))
	}
	if m := result.Matches[0]; m.ModuleName != "go" || m.Coverage != 0.5 {
		t.Errorf("match = %+v, want module go at 0.5 coverage", m)
	}
}

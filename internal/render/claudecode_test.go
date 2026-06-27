package render_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/render"
)

func TestRenderProducesClaudeCodeShape(t *testing.T) {
	p := model.Policy{
		Permissions: model.Permissions{
			Allow: []string{"Read", "Bash(go:*)"},
			Ask:   []string{"Bash(git push:*)"},
			Deny:  []string{"Read(**/.env)"},
		},
		Env:      map[string]string{"FOO": "1"},
		Settings: map[string]any{"defaultMode": "acceptEdits"},
	}

	data, err := render.ClaudeCode{}.Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.HasSuffix(string(data), "\n") {
		t.Error("rendered output should end with a trailing newline")
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v", err)
	}

	// Passthrough settings land at the top level, not under permissions.
	if got["defaultMode"] != "acceptEdits" {
		t.Errorf("defaultMode passthrough = %v, want acceptEdits", got["defaultMode"])
	}

	env, ok := got["env"].(map[string]any)
	if !ok || env["FOO"] != "1" {
		t.Errorf("env = %v, want {FOO:1}", got["env"])
	}

	perms, ok := got["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions missing or wrong type: %v", got["permissions"])
	}
	for _, k := range []string{"allow", "ask", "deny"} {
		if _, ok := perms[k]; !ok {
			t.Errorf("permissions missing %q", k)
		}
	}
}

func TestRenderOmitsEmptySections(t *testing.T) {
	data, err := render.ClaudeCode{}.Render(model.Policy{
		Permissions: model.Permissions{Allow: []string{"Read"}},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["env"]; ok {
		t.Error("empty env should be omitted")
	}
	perms := got["permissions"].(map[string]any)
	if _, ok := perms["deny"]; ok {
		t.Error("empty deny should be omitted")
	}
	if _, ok := perms["ask"]; ok {
		t.Error("empty ask should be omitted")
	}
}

func TestRenderParseRoundTrip(t *testing.T) {
	// Permissions and env survive a Render -> Parse round trip. Passthrough
	// settings deliberately do not: Parse only reconstructs what cperm manages.
	want := model.Policy{
		Permissions: model.Permissions{
			Allow: []string{"Read", "Bash(go:*)"},
			Ask:   []string{"Bash(git push:*)"},
			Deny:  []string{"Read(**/.env)"},
		},
		Env: map[string]string{"FOO": "1"},
	}

	r := render.ClaudeCode{}
	data, err := r.Render(want)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, err := r.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !reflect.DeepEqual(got.Permissions, want.Permissions) {
		t.Errorf("permissions round-trip mismatch:\n got %+v\nwant %+v", got.Permissions, want.Permissions)
	}
	if !reflect.DeepEqual(got.Env, want.Env) {
		t.Errorf("env round-trip mismatch: got %v want %v", got.Env, want.Env)
	}
}

func TestParseIgnoresUnrecognizedKeys(t *testing.T) {
	data := []byte(`{"permissions":{"allow":["Read"]},"hooks":{"x":1},"defaultMode":"plan"}`)
	got, err := render.ClaudeCode{}.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(got.Permissions.Allow, []string{"Read"}) {
		t.Errorf("allow = %v, want [Read]", got.Permissions.Allow)
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := (render.ClaudeCode{}).Parse([]byte("{not json")); err == nil {
		t.Error("expected an error parsing invalid JSON")
	}
}

func TestOutputPath(t *testing.T) {
	got := render.ClaudeCode{}.OutputPath("/proj")
	want := filepath.Join("/proj", ".claude", "settings.json")
	if got != want {
		t.Errorf("OutputPath = %q, want %q", got, want)
	}
}

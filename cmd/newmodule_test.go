package cmd

import "testing"

func TestModuleFromJSON(t *testing.T) {
	data := []byte(`{"name":"ignored","description":"docker stuff","permissions":{"allow":["Bash(docker:*)"]}}`)

	mod, err := moduleFromJSON(data, "docker-extras")
	if err != nil {
		t.Fatalf("moduleFromJSON: %v", err)
	}

	if mod.Name != "docker-extras" {
		t.Errorf("Name = %q, want docker-extras (command arg is authoritative)", mod.Name)
	}
	if mod.Version != "0.1.0" {
		t.Errorf("Version = %q, want default 0.1.0", mod.Version)
	}
	if len(mod.Permissions.Allow) != 1 || mod.Permissions.Allow[0] != "Bash(docker:*)" {
		t.Errorf("Allow = %v, want [Bash(docker:*)]", mod.Permissions.Allow)
	}
}

func TestModuleFromJSONKeepsExplicitVersion(t *testing.T) {
	mod, err := moduleFromJSON([]byte(`{"version":"2.0.0"}`), "x")
	if err != nil {
		t.Fatal(err)
	}
	if mod.Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0 (explicit version kept)", mod.Version)
	}
}

func TestModuleFromJSONRejectsInvalid(t *testing.T) {
	if _, err := moduleFromJSON([]byte("{bad"), "x"); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

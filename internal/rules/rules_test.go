package rules

import "testing"

func TestCovers(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Bash(git:*)", "Bash(git:*)", true},                // exact
		{"Read", "Read(**/.env)", true},                     // bare tool covers any specifier
		{"Bash(git:*)", "Bash(git add *)", true},            // command-prefix subsumption
		{"Bash(git:*)", "Bash(git commit -q -m ' *)", true}, // deeper prefix
		{"Bash(go:*)", "Bash(go build *)", true},
		{"Bash(cat:*)", "Bash(cat)", true},            // prefix equals the whole command
		{"Bash(*)", "Bash(rm -rf /)", true},           // Tool(*) covers everything
		{"Bash(git:*)", "Bash(github-cli)", false},    // word boundary: no false prefix match
		{"Bash(git:*)", "Bash(awk '{print}')", false}, // unrelated command
		{"Bash(git add *)", "Bash(git:*)", false},     // narrow does not cover broad
		{"Read(**/.env)", "Read", false},              // scoped does not cover bare
		{"Bash(x)", "Read(x)", false},                 // different tools
	}
	for _, c := range cases {
		if got := Covers(c.a, c.b); got != c.want {
			t.Errorf("Covers(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCoveredBy(t *testing.T) {
	set := []string{"Bash(git:*)", "Read"}

	if !CoveredBy("Bash(git push origin)", set) {
		t.Error("git push should be covered by Bash(git:*)")
	}
	if !CoveredBy("Read(/etc/hosts)", set) {
		t.Error("any Read should be covered by bare Read")
	}
	if CoveredBy("Bash(npm test)", set) {
		t.Error("npm should not be covered by the set")
	}
}

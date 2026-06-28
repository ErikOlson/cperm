package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikolson/cperm/internal/composer"
	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/rules"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current project's modules and detect drift from composed output",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Emit the drift report as machine-readable JSON")
}

// permTriple mirrors a permissions block for JSON output.
type permTriple struct {
	Allow []string `json:"allow"`
	Ask   []string `json:"ask"`
	Deny  []string `json:"deny"`
}

// driftDetail captures rules that diverged in either direction.
//
// Added: present in settings.json but not in the composed state — i.e. manual
// approvals that accumulated outside the modules. These are the candidates for
// promotion into modules (the bottom-up loop's signal).
//
// Removed: present in the composed state but missing from settings.json.
type driftDetail struct {
	Added   permTriple `json:"added"`
	Removed permTriple `json:"removed"`
}

func (d driftDetail) addedCount() int {
	return len(d.Added.Allow) + len(d.Added.Ask) + len(d.Added.Deny)
}

func (d driftDetail) removedCount() int {
	return len(d.Removed.Allow) + len(d.Removed.Ask) + len(d.Removed.Deny)
}

// driftReport is the machine-readable result of `cperm status --json`.
type driftReport struct {
	Project        string      `json:"project"`
	Compose        string      `json:"compose"`
	Modules        []string    `json:"modules"`
	Settings       string      `json:"settings"`
	Sources        []string    `json:"sources"`
	SettingsExists bool        `json:"settingsExists"`
	InSync         bool        `json:"inSync"`
	AddedCount     int         `json:"addedCount"`
	RemovedCount   int         `json:"removedCount"`
	Drift          driftDetail `json:"drift"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		// In JSON mode a missing compose.json is a non-zero exit so a hook can
		// cheaply tell "this isn't a cperm project" and skip.
		if statusJSON {
			return fmt.Errorf("no compose.json in %s — run 'cperm init' first", projectDir)
		}
		fmt.Println("No compose.json found. Run 'cperm init' to get started.")
		return nil
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	outputPath := getRenderer().OutputPath(projectDir)
	report := driftReport{
		Project:  projectDir,
		Compose:  composePath,
		Modules:  cf.Modules,
		Settings: outputPath,
		Sources:  []string{},
		Drift:    emptyDrift(),
	}
	if report.Modules == nil {
		report.Modules = []string{}
	}

	settingsData, err := os.ReadFile(outputPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		report.SettingsExists = false
		if statusJSON {
			return printJSON(report)
		}
		return printStatusHuman(report)
	}
	report.SettingsExists = true
	report.Sources = []string{outputPath}

	expected, err := composer.New(s).Compose(cf)
	if err != nil {
		return fmt.Errorf("composing expected state: %w", err)
	}
	actualPolicy, err := getRenderer().Parse(settingsData)
	if err != nil {
		return fmt.Errorf("parsing current settings.json: %w", err)
	}
	effective := actualPolicy.Permissions

	// Layer overlay files (e.g. settings.local.json) on top, so drift reflects
	// the effective state — including approvals written outside settings.json.
	for _, op := range getRenderer().OverlayPaths(projectDir) {
		data, rerr := os.ReadFile(op)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				continue
			}
			return rerr
		}
		overlay, perr := getRenderer().Parse(data)
		if perr != nil {
			return fmt.Errorf("parsing %s: %w", formatPath(op), perr)
		}
		effective = unionPermissions(effective, overlay.Permissions)
		report.Sources = append(report.Sources, op)
	}

	report.Drift = computeDrift(expected.Policy.Permissions, effective)
	report.AddedCount = report.Drift.addedCount()
	report.RemovedCount = report.Drift.removedCount()
	report.InSync = report.AddedCount == 0 && report.RemovedCount == 0

	if statusJSON {
		return printJSON(report)
	}
	return printStatusHuman(report)
}

// computeDrift diffs the composed (expected) permissions against the actual
// permissions read back from settings.json.
func computeDrift(expected, actual model.Permissions) driftDetail {
	return driftDetail{
		Added: permTriple{
			Allow: uncovered(actual.Allow, expected.Allow),
			Ask:   uncovered(actual.Ask, expected.Ask),
			Deny:  uncovered(actual.Deny, expected.Deny),
		},
		Removed: permTriple{
			Allow: uncovered(expected.Allow, actual.Allow),
			Ask:   uncovered(expected.Ask, actual.Ask),
			Deny:  uncovered(expected.Deny, actual.Deny),
		},
	}
}

func emptyDrift() driftDetail {
	empty := func() permTriple { return permTriple{Allow: []string{}, Ask: []string{}, Deny: []string{}} }
	return driftDetail{Added: empty(), Removed: empty()}
}

func printJSON(report driftReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printStatusHuman(report driftReport) error {
	fmt.Println(titleStyle.Render("cperm status"))
	fmt.Println()
	fmt.Printf("  Project:  %s\n", report.Project)
	fmt.Printf("  Compose:  %s (%d modules)\n", formatPath(report.Compose), len(report.Modules))
	fmt.Printf("  Modules:  %s\n", strings.Join(report.Modules, ", "))

	if !report.SettingsExists {
		fmt.Println()
		printWarn("settings.json does not exist. Run 'cperm compose' to create it.")
		return nil
	}

	fmt.Printf("  Output:   %s\n", formatPath(report.Settings))
	fmt.Println()

	if report.InSync {
		printSuccess("settings.json matches composed state — no drift detected")
		return nil
	}

	printWarn(fmt.Sprintf("Drift detected: %d added, %d removed vs composed state", report.AddedCount, report.RemovedCount))
	fmt.Println()

	d := report.Drift
	if d.addedCount() > 0 {
		fmt.Println("  Rules in settings.json but not in compose (manual additions?):")
		for _, r := range d.Added.Allow {
			fmt.Printf("    + allow: %s\n", successStyle.Render(r))
		}
		for _, r := range d.Added.Deny {
			fmt.Printf("    + deny:  %s\n", successStyle.Render(r))
		}
		for _, r := range d.Added.Ask {
			fmt.Printf("    + ask:   %s\n", successStyle.Render(r))
		}
	}

	if d.removedCount() > 0 {
		fmt.Println("  Rules in compose but missing from settings.json:")
		for _, r := range d.Removed.Allow {
			fmt.Printf("    - allow: %s\n", errorStyle.Render(r))
		}
		for _, r := range d.Removed.Deny {
			fmt.Printf("    - deny:  %s\n", errorStyle.Render(r))
		}
		for _, r := range d.Removed.Ask {
			fmt.Printf("    - ask:   %s\n", errorStyle.Render(r))
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("Run 'cperm compose' to reset to composed state"))
	fmt.Println(dimStyle.Render("Run 'cperm import' to incorporate manual additions into modules"))

	return nil
}

// unionPermissions merges two permission sets, deduplicating each array while
// preserving first-seen order.
func unionPermissions(a, b model.Permissions) model.Permissions {
	return model.Permissions{
		Allow: unionStrings(a.Allow, b.Allow),
		Ask:   unionStrings(a.Ask, b.Ask),
		Deny:  unionStrings(a.Deny, b.Deny),
	}
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := []string{}
	for _, group := range [][]string{a, b} {
		for _, s := range group {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// uncovered returns the rules in a that no rule in b covers — subsumption-aware
// (a broad Bash(git:*) covers a narrow Bash(git add *)), not just exact match.
// Returns a non-nil slice so it renders as [] rather than null in JSON.
func uncovered(a, b []string) []string {
	out := []string{}
	for _, r := range a {
		if !rules.CoveredBy(r, b) {
			out = append(out, r)
		}
	}
	return out
}

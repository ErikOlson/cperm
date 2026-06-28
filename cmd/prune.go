package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/erikolson/cperm/internal/composer"
	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/rules"
)

var pruneDryRun bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove redundant accumulated rules from local settings",
	Long: `Remove from .claude/settings.local.json any permission rule already covered
by the composed settings.json, leaving only genuinely novel rules.

Run it after promoting useful rules into modules and recomposing, to bring drift
back toward zero. Matching is subsumption-aware (a broad Bash(git:*) covers a
narrow Bash(git add *)). Non-permission content, such as the sandbox block, is
preserved untouched.`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Show what would be removed without writing")
}

func runPrune(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("no compose.json found — run 'cperm init' first: %w", err)
	}
	s, err := getStore()
	if err != nil {
		return err
	}
	expected, err := composer.New(s).Compose(cf)
	if err != nil {
		return fmt.Errorf("composing expected state: %w", err)
	}

	total := 0
	for _, path := range getRenderer().OverlayPaths(projectDir) {
		n, err := pruneOverlay(path, expected.Policy.Permissions)
		if err != nil {
			return err
		}
		total += n
	}

	switch {
	case total == 0:
		printSuccess("Nothing to prune — no redundant rules in local settings")
	case pruneDryRun:
		printWarn(fmt.Sprintf("%d redundant rule(s) would be removed (dry run — nothing written)", total))
	default:
		printSuccess(fmt.Sprintf("Pruned %d redundant rule(s) from local settings", total))
		fmt.Println(dimStyle.Render("Run 'cperm status' to see any remaining (genuinely novel) drift"))
	}
	return nil
}

// pruneOverlay removes from one overlay file every permission rule already
// covered by the composed permissions, reporting each removal. It rewrites the
// file in place (unless --dry-run), preserving any non-permission content.
func pruneOverlay(path string, covered model.Permissions) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, fmt.Errorf("parsing %s: %w", formatPath(path), err)
	}
	perms, ok := doc["permissions"].(map[string]any)
	if !ok {
		return 0, nil
	}

	rel := formatPath(path)
	removed := 0
	for _, section := range []struct {
		key     string
		covered []string
	}{
		{"allow", covered.Allow},
		{"ask", covered.Ask},
		{"deny", covered.Deny},
	} {
		arr, ok := perms[section.key].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(arr))
		for _, item := range arr {
			if rule, ok := item.(string); ok && rules.CoveredBy(rule, section.covered) {
				removed++
				fmt.Printf("  %s %-5s %s\n", dimStyle.Render("− "+rel), section.key, dimStyle.Render(rule))
				continue
			}
			kept = append(kept, item)
		}
		if len(kept) == 0 {
			delete(perms, section.key)
		} else {
			perms[section.key] = kept
		}
	}

	if removed == 0 || pruneDryRun {
		return removed, nil
	}

	if len(perms) == 0 {
		delete(doc, "permissions")
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return removed, err
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		return removed, err
	}
	return removed, nil
}

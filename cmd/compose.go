package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/composer"
	"github.com/erikmav/cperm/internal/model"
)

var composeDryRun bool

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Build .claude/settings.json from compose.json",
	RunE:  runCompose,
}

func init() {
	composeCmd.Flags().BoolVar(&composeDryRun, "dry-run", false, "Print composed output without writing")
}

func runCompose(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("no compose.json found — run 'cperm init' first: %w", err)
	}

	c := composer.New(s)
	result, err := c.Compose(cf)
	if err != nil {
		return err
	}

	if composeDryRun {
		// Build the same output structure as WriteSettings
		output := buildOutputMap(result)
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	outputPath := composer.OutputPath(projectDir)
	if err := composer.WriteSettings(outputPath, result); err != nil {
		return err
	}

	fmt.Printf("Resolved modules: %s\n", strings.Join(result.ModulesUsed, " → "))
	printSuccess(fmt.Sprintf("Composed %s (%d allow, %d deny, %d ask)",
		formatPath(outputPath), result.AllowCount, result.DenyCount, result.AskCount))

	if result.Deduplicated > 0 {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %d rules deduplicated", result.Deduplicated)))
	}

	for _, c := range result.Conflicts {
		var arrays []string
		if c.InAllow {
			arrays = append(arrays, "allow")
		}
		if c.InDeny {
			arrays = append(arrays, "deny")
		}
		if c.InAsk {
			arrays = append(arrays, "ask")
		}
		printWarn(fmt.Sprintf("%s appears in both %s", c.Rule, strings.Join(arrays, " and ")))
	}

	return nil
}

func buildOutputMap(result *model.ComposedResult) map[string]any {
	output := make(map[string]any)
	perms := make(map[string]any)
	if len(result.Settings.Permissions.Allow) > 0 {
		perms["allow"] = result.Settings.Permissions.Allow
	}
	if len(result.Settings.Permissions.Deny) > 0 {
		perms["deny"] = result.Settings.Permissions.Deny
	}
	if len(result.Settings.Permissions.Ask) > 0 {
		perms["ask"] = result.Settings.Permissions.Ask
	}
	output["permissions"] = perms
	if len(result.Settings.Env) > 0 {
		output["env"] = result.Settings.Env
	}
	for k, v := range result.Settings.Extra {
		output[k] = v
	}
	return output
}

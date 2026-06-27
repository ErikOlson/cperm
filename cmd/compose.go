package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikolson/cperm/internal/composer"
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
		data, err := getRenderer().Render(result.Policy)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	}

	outputPath, err := writeComposed(projectDir, result)
	if err != nil {
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

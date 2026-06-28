package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikolson/cperm/internal/composer"
	"github.com/erikolson/cperm/internal/importer"
	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/store"
)

var importJSON bool

func init() {
	importCmd.Flags().BoolVar(&importJSON, "json", false,
		"Output the analysis as JSON and skip all prompts")
}

var importCmd = &cobra.Command{
	Use:   "import [settings.json]",
	Short: "Decompose an existing settings.json into module candidates",
	Long: `Analyze an existing Claude Code settings.json and match its permission
rules against available modules. Unmatched rules can be saved as new
modules or project overrides.

If no file is given, defaults to .claude/settings.json in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImport,
}

func runImport(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	path := getRenderer().OutputPath(projectDir)
	if len(args) > 0 {
		path = args[0]
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}
	policy, err := getRenderer().Parse(data)
	if err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	result, err := importer.Analyze(policy.Permissions, s)
	if err != nil {
		return err
	}

	if importJSON {
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Println(titleStyle.Render("cperm import"))
	fmt.Println()
	fmt.Printf("Analyzing %s (%d rules)...\n\n", formatPath(path), result.TotalRules)

	if len(result.Matches) > 0 {
		fmt.Println(boldStyle.Render("Module matches:"))
		for _, m := range result.Matches {
			icon := "✓"
			style := successStyle
			if m.Coverage < 1.0 {
				icon = "~"
				style = warnStyle
			}
			fmt.Printf("  %s %-18s %s (%d/%d rules)\n",
				style.Render(icon),
				m.ModuleName,
				dimStyle.Render(fmt.Sprintf("%.0f%% coverage", m.Coverage*100)),
				len(m.Matched), m.Total,
			)
		}
		fmt.Println()
	}

	totalUnmatched := len(result.UnmatchedAllow) + len(result.UnmatchedDeny) + len(result.UnmatchedAsk)

	if totalUnmatched > 0 {
		fmt.Println(boldStyle.Render("Unmatched rules:"))
		for _, r := range result.UnmatchedAllow {
			fmt.Printf("  allow: %s\n", r)
		}
		for _, r := range result.UnmatchedDeny {
			fmt.Printf("  deny:  %s\n", r)
		}
		for _, r := range result.UnmatchedAsk {
			fmt.Printf("  ask:   %s\n", r)
		}
		fmt.Println()
	}

	if totalUnmatched == 0 && len(result.Matches) > 0 {
		printSuccess("All rules covered by existing modules!")
	}

	// Prompt for action
	if totalUnmatched > 0 {
		fmt.Println("What would you like to do with unmatched rules?")
		fmt.Println("  [c] Create a new module from unmatched rules")
		fmt.Println("  [o] Add as project overrides in compose.json")
		fmt.Println("  [s] Skip")
		fmt.Print("Choice [s]: ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))

		switch choice {
		case "c":
			return createModuleFromUnmatched(s, result)
		case "o":
			return addOverrides(projectDir, result)
		}
	}

	// Offer to create compose.json from matched modules
	if len(result.Matches) > 0 {
		fmt.Println()
		fmt.Print("Create compose.json from matched modules? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "n" {
			var modules []string
			for _, m := range result.Matches {
				if m.Coverage >= 0.5 { // Only include modules with >=50% coverage
					modules = append(modules, m.ModuleName)
				}
			}

			cf := &model.ComposeFile{
				Modules: modules,
			}

			composePath := composer.ComposeFilePath(projectDir)
			if err := composer.SaveComposeFile(composePath, cf); err != nil {
				return err
			}

			printSuccess(fmt.Sprintf("Wrote %s (%d modules)", formatPath(composePath), len(modules)))
		}
	}

	return nil
}

func createModuleFromUnmatched(s *store.Store, result *importer.ImportResult) error {
	allUnmatched := append(append(result.UnmatchedAllow, result.UnmatchedDeny...), result.UnmatchedAsk...)
	suggestedName := importer.SuggestModuleName(allUnmatched)

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Module name [%s]: ", suggestedName)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = suggestedName
	}

	fmt.Print("Description: ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	mod := &model.Module{
		Name:        name,
		Description: desc,
		Version:     "0.1.0",
		Permissions: model.Permissions{
			Allow: result.UnmatchedAllow,
			Deny:  result.UnmatchedDeny,
			Ask:   result.UnmatchedAsk,
		},
	}

	if err := s.Save(mod); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Created module %q with %d rules", name,
		len(result.UnmatchedAllow)+len(result.UnmatchedDeny)+len(result.UnmatchedAsk)))

	return nil
}

func addOverrides(projectDir string, result *importer.ImportResult) error {
	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		// No compose file yet — create one
		cf = &model.ComposeFile{
			Modules: []string{},
		}
	}

	if cf.Override == nil {
		cf.Override = &model.Permissions{}
	}
	cf.Override.Allow = append(cf.Override.Allow, result.UnmatchedAllow...)
	cf.Override.Deny = append(cf.Override.Deny, result.UnmatchedDeny...)
	cf.Override.Ask = append(cf.Override.Ask, result.UnmatchedAsk...)

	if err := composer.SaveComposeFile(composePath, cf); err != nil {
		return err
	}

	total := len(result.UnmatchedAllow) + len(result.UnmatchedDeny) + len(result.UnmatchedAsk)
	printSuccess(fmt.Sprintf("Added %d override rules to %s", total, formatPath(composePath)))
	return nil
}

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/composer"
	"github.com/erikmav/cperm/internal/model"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup — pick modules and create compose.json for this project",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	if _, err := os.Stat(composePath); err == nil {
		printWarn("compose.json already exists at " + formatPath(composePath))
		fmt.Print("Overwrite? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	names, err := s.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		printError("No modules available. Run 'cperm new <name>' to create one.")
		return nil
	}

	fmt.Println(titleStyle.Render("cperm init"))
	fmt.Println()
	fmt.Println("Select modules for this project. Enter numbers separated by spaces,")
	fmt.Println("or 'a' for all, or press enter to skip.")
	fmt.Println()

	for i, name := range names {
		mod, err := s.Load(name)
		if err != nil {
			continue
		}
		fmt.Printf("  %s %s  %s\n",
			boldStyle.Render(fmt.Sprintf("[%d]", i+1)),
			fmt.Sprintf("%-18s", name),
			dimStyle.Render(mod.Description),
		)
	}

	fmt.Println()
	fmt.Print("Modules: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selected []string
	if input == "a" || input == "A" {
		selected = names
	} else if input != "" {
		// Parse space-separated numbers
		for _, part := range strings.Fields(input) {
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx >= 1 && idx <= len(names) {
				selected = append(selected, names[idx-1])
			} else {
				// Maybe they typed a name directly
				for _, n := range names {
					if n == part {
						selected = append(selected, n)
						break
					}
				}
			}
		}
	}

	if len(selected) == 0 {
		printWarn("No modules selected. You can add them later with 'cperm add <module>'.")
		selected = []string{}
	}

	// Ask for default mode
	fmt.Println()
	fmt.Println("Default permission mode:")
	fmt.Println("  [1] default       — prompt for each tool on first use")
	fmt.Println("  [2] acceptEdits   — auto-approve file edits, prompt for bash")
	fmt.Println("  [3] plan          — read-only, no modifications")
	fmt.Print("Mode [1]: ")
	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)

	settings := make(map[string]any)
	switch modeInput {
	case "2":
		settings["defaultMode"] = "acceptEdits"
	case "3":
		settings["defaultMode"] = "plan"
	default:
		// "default" mode — don't set it explicitly
	}

	cf := &model.ComposeFile{
		Modules:  selected,
		Settings: settings,
	}

	if err := composer.SaveComposeFile(composePath, cf); err != nil {
		return fmt.Errorf("writing compose file: %w", err)
	}

	printSuccess(fmt.Sprintf("Wrote %s (%d modules)", formatPath(composePath), len(selected)))

	// Auto-compose
	c := composer.New(s)
	result, err := c.Compose(cf)
	if err != nil {
		return fmt.Errorf("composing: %w", err)
	}

	outputPath := composer.OutputPath(projectDir)
	if err := composer.WriteSettings(outputPath, result); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	printSuccess(fmt.Sprintf("Composed %s (%d allow, %d deny, %d ask)",
		formatPath(outputPath), result.AllowCount, result.DenyCount, result.AskCount))

	if len(result.Conflicts) > 0 {
		fmt.Println()
		printWarn(fmt.Sprintf("%d conflict(s) detected:", len(result.Conflicts)))
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
			fmt.Printf("  %s in both %s (from: %s)\n",
				c.Rule, strings.Join(arrays, " and "), strings.Join(c.Sources, ", "))
		}
	}

	return nil
}

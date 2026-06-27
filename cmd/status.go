package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/composer"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current project's modules and detect drift from composed output",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		fmt.Println("No compose.json found. Run 'cperm init' to get started.")
		return nil
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	fmt.Println(titleStyle.Render("cperm status"))
	fmt.Println()
	fmt.Printf("  Project:  %s\n", projectDir)
	fmt.Printf("  Compose:  %s (%d modules)\n", formatPath(composePath), len(cf.Modules))
	fmt.Printf("  Modules:  %s\n", strings.Join(cf.Modules, ", "))

	// Check if settings.json exists
	outputPath := getRenderer().OutputPath(projectDir)
	settingsData, err := os.ReadFile(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println()
			printWarn("settings.json does not exist. Run 'cperm compose' to create it.")
			return nil
		}
		return err
	}

	fmt.Printf("  Output:   %s\n", formatPath(outputPath))

	// Compose what it _should_ be
	c := composer.New(s)
	expected, err := c.Compose(cf)
	if err != nil {
		return fmt.Errorf("composing expected state: %w", err)
	}

	// Parse what actually exists
	actualPolicy, err := getRenderer().Parse(settingsData)
	if err != nil {
		return fmt.Errorf("parsing current settings.json: %w", err)
	}
	actual := actualPolicy.Permissions

	// Diff
	addedAllow := diffSlice(actual.Allow, expected.Policy.Permissions.Allow)
	removedAllow := diffSlice(expected.Policy.Permissions.Allow, actual.Allow)
	addedDeny := diffSlice(actual.Deny, expected.Policy.Permissions.Deny)
	removedDeny := diffSlice(expected.Policy.Permissions.Deny, actual.Deny)
	addedAsk := diffSlice(actual.Ask, expected.Policy.Permissions.Ask)
	removedAsk := diffSlice(expected.Policy.Permissions.Ask, actual.Ask)

	totalAdded := len(addedAllow) + len(addedDeny) + len(addedAsk)
	totalRemoved := len(removedAllow) + len(removedDeny) + len(removedAsk)

	fmt.Println()
	if totalAdded == 0 && totalRemoved == 0 {
		printSuccess("settings.json matches composed state — no drift detected")
		return nil
	}

	printWarn(fmt.Sprintf("Drift detected: %d added, %d removed vs composed state", totalAdded, totalRemoved))
	fmt.Println()

	if len(addedAllow) > 0 {
		fmt.Println("  Rules in settings.json but not in compose (manual additions?):")
		for _, r := range addedAllow {
			fmt.Printf("    + allow: %s\n", successStyle.Render(r))
		}
	}
	if len(addedDeny) > 0 {
		for _, r := range addedDeny {
			fmt.Printf("    + deny:  %s\n", successStyle.Render(r))
		}
	}
	if len(addedAsk) > 0 {
		for _, r := range addedAsk {
			fmt.Printf("    + ask:   %s\n", successStyle.Render(r))
		}
	}

	if len(removedAllow) > 0 {
		fmt.Println("  Rules in compose but missing from settings.json:")
		for _, r := range removedAllow {
			fmt.Printf("    - allow: %s\n", errorStyle.Render(r))
		}
	}
	if len(removedDeny) > 0 {
		for _, r := range removedDeny {
			fmt.Printf("    - deny:  %s\n", errorStyle.Render(r))
		}
	}
	if len(removedAsk) > 0 {
		for _, r := range removedAsk {
			fmt.Printf("    - ask:   %s\n", errorStyle.Render(r))
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("Run 'cperm compose' to reset to composed state"))
	fmt.Println(dimStyle.Render("Run 'cperm import' to incorporate manual additions into modules"))

	return nil
}

// diffSlice returns items in a that are not in b.
func diffSlice(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, s := range b {
		bSet[s] = true
	}
	var diff []string
	for _, s := range a {
		if !bSet[s] {
			diff = append(diff, s)
		}
	}
	return diff
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/composer"
)

var addCmd = &cobra.Command{
	Use:   "add <module>",
	Short: "Add a module to the current project's compose.json",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove <module>",
	Short: "Remove a module from the current project's compose.json",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func runAdd(cmd *cobra.Command, args []string) error {
	moduleName := args[0]

	s, err := getStore()
	if err != nil {
		return err
	}

	// Verify module exists
	if !s.Exists(moduleName) {
		return fmt.Errorf("module %q not found — run 'cperm modules' to see available modules", moduleName)
	}

	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("no compose.json found — run 'cperm init' first: %w", err)
	}

	// Check if already present
	for _, m := range cf.Modules {
		if m == moduleName {
			printWarn(fmt.Sprintf("Module %q is already in compose.json", moduleName))
			return nil
		}
	}

	cf.Modules = append(cf.Modules, moduleName)

	if err := composer.SaveComposeFile(composePath, cf); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Added %q to compose.json", moduleName))

	// Auto-recompose
	c := composer.New(s)
	result, err := c.Compose(cf)
	if err != nil {
		return fmt.Errorf("recomposing: %w", err)
	}

	outputPath := composer.OutputPath(projectDir)
	if err := composer.WriteSettings(outputPath, result); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Recomposed %s (%d allow, %d deny, %d ask)",
		formatPath(outputPath), result.AllowCount, result.DenyCount, result.AskCount))

	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	moduleName := args[0]

	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	composePath := composer.ComposeFilePath(projectDir)
	cf, err := composer.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("no compose.json found — run 'cperm init' first: %w", err)
	}

	// Find and remove
	found := false
	var updated []string
	for _, m := range cf.Modules {
		if m == moduleName {
			found = true
			continue
		}
		updated = append(updated, m)
	}

	if !found {
		return fmt.Errorf("module %q is not in compose.json", moduleName)
	}

	cf.Modules = updated

	if err := composer.SaveComposeFile(composePath, cf); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Removed %q from compose.json", moduleName))

	// Auto-recompose
	s, err := getStore()
	if err != nil {
		return err
	}

	c := composer.New(s)
	result, err := c.Compose(cf)
	if err != nil {
		return fmt.Errorf("recomposing: %w", err)
	}

	outputPath := composer.OutputPath(projectDir)
	if err := composer.WriteSettings(outputPath, result); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Recomposed %s (%d allow, %d deny, %d ask)",
		formatPath(outputPath), result.AllowCount, result.DenyCount, result.AskCount))

	return nil
}

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "List available permission modules",
	RunE:  runModulesList,
}

var modulesShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show the full contents of a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runModulesShow,
}

func init() {
	modulesCmd.AddCommand(modulesShowCmd)
}

func runModulesList(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	names, err := s.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No modules found. Run 'cperm new <name>' to create one.")
		fmt.Println(dimStyle.Render("Module store: " + s.Dir))
		return nil
	}

	fmt.Println(titleStyle.Render("Available modules"))
	fmt.Println()

	for _, name := range names {
		mod, err := s.Load(name)
		if err != nil {
			printWarn(fmt.Sprintf("  %s — error loading: %v", name, err))
			continue
		}

		allowCount := len(mod.Permissions.Allow)
		denyCount := len(mod.Permissions.Deny)
		askCount := len(mod.Permissions.Ask)

		counts := fmt.Sprintf("%d allow", allowCount)
		if denyCount > 0 {
			counts += fmt.Sprintf(", %d deny", denyCount)
		}
		if askCount > 0 {
			counts += fmt.Sprintf(", %d ask", askCount)
		}

		fmt.Printf("  %s  %s\n", boldStyle.Render(fmt.Sprintf("%-18s", name)), mod.Description)
		fmt.Printf("  %s  %s\n", fmt.Sprintf("%-18s", ""), dimStyle.Render(counts))
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("Store: " + s.Dir))
	return nil
}

func runModulesShow(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	mod, err := s.Load(args[0])
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(mod, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(titleStyle.Render(mod.Name) + "  " + dimStyle.Render(mod.Description))
	fmt.Println()
	fmt.Println(string(data))
	return nil
}

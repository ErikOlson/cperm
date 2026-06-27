package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/model"
)

var newModuleCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new permission module",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewModule,
}

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open a module in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func runNewModule(cmd *cobra.Command, args []string) error {
	name := args[0]

	s, err := getStore()
	if err != nil {
		return err
	}

	if s.Exists(name) {
		return fmt.Errorf("module %q already exists — use 'cperm edit %s' to modify it", name, name)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(titleStyle.Render("Create module: " + name))
	fmt.Println()

	fmt.Print("Description: ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	fmt.Println()
	fmt.Println("Enter permission rules, one per line. Empty line to finish each section.")

	fmt.Println()
	fmt.Println(boldStyle.Render("Allow rules") + dimStyle.Render(" (e.g., Bash(go:*), Edit, WebFetch)"))
	allow := readLines(reader)

	fmt.Println(boldStyle.Render("Deny rules") + dimStyle.Render(" (e.g., Bash(rm -rf:*), Read(**/.env))"))
	deny := readLines(reader)

	fmt.Println(boldStyle.Render("Ask rules") + dimStyle.Render(" (e.g., Bash(git push:*))"))
	ask := readLines(reader)

	fmt.Println(boldStyle.Render("Dependencies") + dimStyle.Render(" (module names, space-separated, or empty)"))
	fmt.Print(": ")
	reqLine, _ := reader.ReadString('\n')
	var requires []string
	for _, r := range strings.Fields(strings.TrimSpace(reqLine)) {
		if r != "" {
			requires = append(requires, r)
		}
	}

	mod := &model.Module{
		Name:        name,
		Description: desc,
		Version:     "0.1.0",
		Requires:    requires,
		Permissions: model.Permissions{
			Allow: allow,
			Deny:  deny,
			Ask:   ask,
		},
	}

	if err := s.Save(mod); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Created module %q at %s", name, s.ModulePath(name)))
	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	s, err := getStore()
	if err != nil {
		return err
	}

	if !s.Exists(name) {
		return fmt.Errorf("module %q not found", name)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	path := s.ModulePath(name)
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

// readLines reads lines from the reader until an empty line is entered.
func readLines(reader *bufio.Reader) []string {
	var lines []string
	for {
		fmt.Print(": ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	return lines
}

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/composer"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Output composed settings.json to stdout without writing a file",
	RunE:  runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
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

	output := buildOutputMap(result)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

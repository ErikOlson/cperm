package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/erikmav/cperm/internal/store"
)

var (
	// Styles
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

var rootCmd = &cobra.Command{
	Use:   "cperm",
	Short: "Composable Claude Code permissions",
	Long: `cperm — composable Claude Code permissions

A Nix-inspired tool for managing Claude Code permission configurations.
Define reusable permission modules, compose them per-project, and keep
your settings.json deterministic and drift-free.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(modulesCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(composeCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newModuleCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
}

// getStore returns the module store, initializing it if needed.
// On first use, it installs the built-in starter modules.
func getStore() (*store.Store, error) {
	s, err := store.DefaultStore()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	// Seed with built-in modules (skips any that already exist)
	if err := s.InstallBuiltins(); err != nil {
		return nil, fmt.Errorf("installing built-in modules: %w", err)
	}
	return s, nil
}

// getProjectDir returns the current working directory.
func getProjectDir() (string, error) {
	return os.Getwd()
}

// formatPath makes a path relative to cwd for display.
func formatPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

// printSuccess prints a success message.
func printSuccess(msg string) {
	fmt.Println(successStyle.Render("✓ " + msg))
}

// printWarn prints a warning message.
func printWarn(msg string) {
	fmt.Println(warnStyle.Render("⚠ " + msg))
}

// printError prints an error message.
func printError(msg string) {
	fmt.Println(errorStyle.Render("✗ " + msg))
}

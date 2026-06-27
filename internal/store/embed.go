package store

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/erikolson/cperm/internal/model"
)

//go:embed all:builtins
var builtinFS embed.FS

// InstallBuiltins copies embedded built-in modules to the store,
// skipping any that already exist (user's customizations take precedence).
func (s *Store) InstallBuiltins() error {
	if err := s.Init(); err != nil {
		return err
	}

	entries, err := fs.ReadDir(builtinFS, "builtins")
	if err != nil {
		return fmt.Errorf("reading embedded builtins: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := builtinFS.ReadFile("builtins/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading builtin %s: %w", entry.Name(), err)
		}

		var mod model.Module
		if err := json.Unmarshal(data, &mod); err != nil {
			return fmt.Errorf("parsing builtin %s: %w", entry.Name(), err)
		}

		if s.Exists(mod.Name) {
			continue
		}

		if err := s.Save(&mod); err != nil {
			return fmt.Errorf("installing builtin %s: %w", mod.Name, err)
		}
	}

	return nil
}

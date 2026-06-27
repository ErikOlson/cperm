package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/erikolson/cperm/internal/model"
)

const (
	defaultStoreDir = ".config/cperm/modules"
	modulesSubdir   = "modules"
)

// Store manages the module repository on disk.
type Store struct {
	Dir string
}

// DefaultStore returns a store at ~/.config/cperm/modules.
func DefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}
	dir := filepath.Join(home, defaultStoreDir)
	return &Store{Dir: dir}, nil
}

// Init ensures the store directory exists.
func (s *Store) Init() error {
	return os.MkdirAll(s.Dir, 0755)
}

// List returns all available module names, sorted alphabetically.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names, nil
}

// Load reads a module by name from the store.
func (s *Store) Load(name string) (*model.Module, error) {
	path := filepath.Join(s.Dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("module %q not found", name)
		}
		return nil, err
	}

	var mod model.Module
	if err := json.Unmarshal(data, &mod); err != nil {
		return nil, fmt.Errorf("parsing module %q: %w", name, err)
	}
	return &mod, nil
}

// Save writes a module to the store.
func (s *Store) Save(mod *model.Module) error {
	if err := s.Init(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(mod, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.Dir, mod.Name+".json")
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// Delete removes a module from the store.
func (s *Store) Delete(name string) error {
	path := filepath.Join(s.Dir, name+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("module %q not found", name)
	}
	return err
}

// LoadAll loads every module in the store, keyed by name.
func (s *Store) LoadAll() (map[string]*model.Module, error) {
	names, err := s.List()
	if err != nil {
		return nil, err
	}

	mods := make(map[string]*model.Module, len(names))
	for _, name := range names {
		mod, err := s.Load(name)
		if err != nil {
			return nil, err
		}
		mods[name] = mod
	}
	return mods, nil
}

// Exists checks if a module exists in the store.
func (s *Store) Exists(name string) bool {
	path := filepath.Join(s.Dir, name+".json")
	_, err := os.Stat(path)
	return err == nil
}

// ModulePath returns the filesystem path for a module.
func (s *Store) ModulePath(name string) string {
	return filepath.Join(s.Dir, name+".json")
}

package source

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	// Version is the newest mur.yaml major version this mur understands.
	Version  = 1
	fileName = "mur.yaml"
	wsDir    = "workspaces"
)

// Source is an attached checkout holding mur.yaml and workspaces/.
type Source struct {
	Name string
	Dir  string
}

// Init writes the skeleton of a source into dir, keeping whatever is there.
func Init(dir string) error {
	murFile := filepath.Join(dir, fileName)
	if _, err := os.Stat(murFile); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(murFile, fmt.Appendf(nil, "version: %d\n", Version), 0o600); err != nil {
			return fmt.Errorf("write mur.yaml: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Join(dir, wsDir), 0o750); err != nil {
		return fmt.Errorf("create workspaces dir: %w", err)
	}

	return nil
}

// Open checks that dir is a source this mur can read.
func Open(name, dir string) (*Source, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return nil, fmt.Errorf("source %s: %w", name, err)
	}

	var head struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("source %s: parse mur.yaml: %w", name, err)
	}

	if head.Version > Version {
		return nil, fmt.Errorf("source %s: mur.yaml version %d is newer than this mur supports (%d); upgrade mur",
			name, head.Version, Version)
	}

	return &Source{Name: name, Dir: dir}, nil
}

// WorkspaceNames lists the workspaces defined in this source, sorted.
func (s *Source) WorkspaceNames() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.Dir, wsDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("source %s: list workspaces: %w", s.Name, err)
	}

	var names []string

	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(s.Dir, wsDir, e.Name(), workspaceFile)); err == nil {
			names = append(names, e.Name())
		}
	}

	sort.Strings(names)

	return names, nil
}

func (s *Source) workspaceDir(name string) string {
	return filepath.Join(s.Dir, wsDir, name)
}

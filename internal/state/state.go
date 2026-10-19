package state

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State is what mur knows about this laptop: never shared unless pushed.
type State struct {
	Active    string            `json:"active,omitempty"     yaml:"active,omitempty"`
	Sources   []Source          `json:"sources,omitempty"    yaml:"sources,omitempty"`
	ScanRoots []string          `json:"scan_roots,omitempty" yaml:"scan_roots,omitempty"`
	Paths     map[string]string `json:"paths,omitempty"      yaml:"paths,omitempty"`
}

type Source struct {
	Name   string `json:"name"             yaml:"name"`
	Path   string `json:"path"             yaml:"path"`
	Remote string `json:"remote,omitempty" yaml:"remote,omitempty"`
}

// Load reads a state file; a missing file is an empty state.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &State{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &s, nil
}

func (s *State) Save(path string) error {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	// Write then rename, so a reader never sees a half-written file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.yaml")
	if err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write state: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	return nil
}

func (s *State) Source(name string) (Source, bool) {
	for _, src := range s.Sources {
		if src.Name == name {
			return src, true
		}
	}

	return Source{}, false
}

// SetPath records where a repo is cloned on this laptop.
func (s *State) SetPath(id, path string) {
	if s.Paths == nil {
		s.Paths = map[string]string{}
	}

	s.Paths[id] = path
}

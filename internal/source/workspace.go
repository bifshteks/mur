package source

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	workspaceFile = "workspace.yaml"
	readmeFile    = "README.md"
)

// Workspace is a named set of repos plus its README, as defined in one source.
type Workspace struct {
	Source   string
	Name     string
	Dir      string
	Summary  string
	Statuses []Status
	Repos    []Repo

	// doc is the file as parsed, so a rewrite keeps comments, key order and
	// keys this mur does not know.
	doc yaml.Node
}

type Status struct {
	Name        string `json:"name"           yaml:"name"`
	Description string `json:"description"    yaml:"description"`
	Sync        *bool  `json:"sync,omitempty" yaml:"sync,omitempty"`
}

type Repo struct {
	URL         string   `yaml:"url"`
	Description string   `yaml:"description"`
	Aliases     []string `yaml:"aliases"`
	Langs       []string `yaml:"langs"`
	Status      string   `yaml:"status"`
	Notes       string   `yaml:"notes"`
}

type workspaceYAML struct {
	Summary  string   `yaml:"summary"`
	Statuses []Status `yaml:"statuses"`
	Repos    []Repo   `yaml:"repos"`
}

// Qualified is the name that is unique across all sources.
func (w *Workspace) Qualified() string {
	return w.Source + "/" + w.Name
}

func (w *Workspace) file() string {
	return filepath.Join(w.Dir, workspaceFile)
}

func (s *Source) LoadWorkspace(name string) (*Workspace, error) {
	dir := s.workspaceDir(name)

	data, err := os.ReadFile(filepath.Join(dir, workspaceFile))
	if err != nil {
		return nil, fmt.Errorf("workspace %s/%s: %w", s.Name, name, err)
	}

	w := &Workspace{Source: s.Name, Name: name, Dir: dir}
	if err := yaml.Unmarshal(data, &w.doc); err != nil {
		return nil, fmt.Errorf("workspace %s/%s: parse: %w", s.Name, name, err)
	}

	var f workspaceYAML
	if err := w.doc.Decode(&f); err != nil && w.doc.Kind != 0 {
		return nil, fmt.Errorf("workspace %s/%s: parse: %w", s.Name, name, err)
	}

	w.Summary, w.Statuses, w.Repos = f.Summary, f.Statuses, f.Repos

	return w, nil
}

var ErrExists = errors.New("already exists")

func (s *Source) CreateWorkspace(name, summary string) (*Workspace, error) {
	dir := s.workspaceDir(name)
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("workspace %s/%s: %w", s.Name, name, ErrExists)
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create workspace dir: %w", err)
	}

	doc := mapping(scalar("summary"), scalar(summary))
	if err := writeYAML(filepath.Join(dir, workspaceFile), doc); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(dir, readmeFile), []byte("# "+name+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("write README.md: %w", err)
	}

	return s.LoadWorkspace(name)
}

// Readme is the workspace's prose description, empty when there is none.
func (w *Workspace) Readme() (string, error) {
	data, err := os.ReadFile(filepath.Join(w.Dir, readmeFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("read README.md: %w", err)
	}

	return string(data), nil
}

// ExtraFiles lists every other file in the workspace directory by absolute
// path. mur never parses them; the reader opens them on demand.
func (w *Workspace) ExtraFiles() ([]string, error) {
	files := []string{}

	err := filepath.WalkDir(w.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if p != w.Dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if d.IsDir() || p == w.file() || p == filepath.Join(w.Dir, readmeFile) {
			return nil
		}

		files = append(files, p)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace files: %w", err)
	}

	return files, nil
}

// Syncs reports whether sync touches repos with this status by default.
func (w *Workspace) Syncs(status string) bool {
	for _, s := range w.Statuses {
		if s.Name == status {
			return s.Sync == nil || *s.Sync
		}
	}

	return true
}

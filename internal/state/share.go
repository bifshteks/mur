package state

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// WithTilde is a copy for sharing: every path under home is spelled with ~, so
// it lands correctly for a different username on another laptop.
func (s *State) WithTilde(home string) *State {
	return s.mapPaths(func(p string) string {
		if p == home {
			return "~"
		}

		if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
			return "~/" + rest
		}

		return p
	})
}

// Expanded is the inverse of WithTilde, against this laptop's home.
func (s *State) Expanded(home string) *State {
	return s.mapPaths(func(p string) string {
		if p == "~" {
			return home
		}

		if rest, ok := strings.CutPrefix(p, "~/"); ok {
			return filepath.Join(home, rest)
		}

		return p
	})
}

func (s *State) mapPaths(f func(string) string) *State {
	out := &State{Active: s.Active, ScanRoots: s.ScanRoots}

	for _, src := range s.Sources {
		src.Path = f(src.Path)
		out.Sources = append(out.Sources, src)
	}

	if s.Paths != nil {
		out.Paths = make(map[string]string, len(s.Paths))
		for id, p := range s.Paths {
			out.Paths[id] = f(p)
		}
	}

	return out
}

// Merged is what a pull brings in: everything other knows that s does not.
// Local entries always win.
type Merged struct {
	Sources   []Source
	Paths     []string
	Active    string
	ScanRoots []string
}

func (s *State) Merge(other *State) Merged {
	var m Merged

	for _, src := range other.Sources {
		if _, ok := s.Source(src.Name); !ok {
			m.Sources = append(m.Sources, src)
		}
	}

	for _, id := range sortedKeys(other.Paths) {
		if _, ok := s.Paths[id]; !ok {
			s.SetPath(id, other.Paths[id])
			m.Paths = append(m.Paths, id)
		}
	}

	if s.Active == "" && other.Active != "" {
		s.Active, m.Active = other.Active, other.Active
	}

	if len(s.ScanRoots) == 0 && len(other.ScanRoots) > 0 {
		s.ScanRoots, m.ScanRoots = other.ScanRoots, other.ScanRoots
	}

	return m
}

func sortedKeys(m map[string]string) []string {
	return slices.Sorted(maps.Keys(m))
}

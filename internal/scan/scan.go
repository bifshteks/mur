package scan

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bifshteks/mur/internal/git"
	"github.com/bifshteks/mur/internal/repoid"
)

const maxDepth = 4

// Directories that never hold a clone worth adopting, and are expensive to walk.
var skipNames = []string{"node_modules", "vendor", "Library", "Applications", "target"} //nolint:gochecknoglobals // fixed list

// Clones finds the git clones under the roots and indexes them by the repo id
// of their origin. Only main clones count: a worktree or submodule (.git is a
// file) would make a second candidate for a repo that already has one.
func Clones(ctx context.Context, g git.Git, roots []string) map[string][]string {
	byID := map[string][]string{}

	for _, root := range roots {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil //nolint:nilerr // an unreadable dir is skipped, not fatal
			}

			if p != root && (strings.HasPrefix(d.Name(), ".") || slices.Contains(skipNames, d.Name())) {
				return filepath.SkipDir
			}

			if depth(root, p) > maxDepth {
				return filepath.SkipDir
			}

			gitDir, err := fs.Stat(osFS{}, filepath.Join(p, ".git"))
			if err != nil {
				return nil //nolint:nilerr // not a clone, keep walking
			}

			if gitDir.IsDir() {
				if origin := g.Origin(ctx, p); origin != "" {
					id := repoid.Normalize(origin)
					byID[id] = append(byID[id], p)
				}
			}

			return filepath.SkipDir
		})
	}

	return byID
}

func depth(root, p string) int {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." {
		return 0
	}

	return strings.Count(rel, string(filepath.Separator)) + 1
}

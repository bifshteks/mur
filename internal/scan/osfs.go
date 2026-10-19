package scan

import (
	"io/fs"
	"os"
)

// osFS lets fs.Stat take absolute paths.
type osFS struct{}

func (osFS) Open(name string) (fs.File, error) { return os.Open(name) } //nolint:gosec // paths come from the user's scan roots

func (osFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

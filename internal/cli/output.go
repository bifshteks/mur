package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func (a *app) print(v any) error {
	return encode(a.env.Stdout, v, a.json)
}

func encode(w io.Writer, v any, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		if err := enc.Encode(v); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}

		return nil
	}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}

	return nil
}

type errorOutput struct {
	Error  string `json:"error"            yaml:"error"`
	Detail any    `json:"detail,omitempty" yaml:"detail,omitempty"`
}

func (a *app) printError(e *exitError) {
	if e.detail != nil {
		_ = encode(a.env.Stderr, e.detail, a.json)

		return
	}

	_ = encode(a.env.Stderr, errorOutput{Error: e.err.Error()}, a.json)
}

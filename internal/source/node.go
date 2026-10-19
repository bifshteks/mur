package source

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Every node mur creates is block style, set explicitly, so no file it writes
// ever gains flow {…} or […] collections.

const seqTag = "!!seq"

// blockStyle turns a hand-written flow collection, and everything inside it,
// into block style before mur rewrites it.
func blockStyle(n *yaml.Node) {
	n.Style &^= yaml.FlowStyle
	for _, c := range n.Content {
		blockStyle(c)
	}
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

func mapping(kv ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: kv}
}

func writeYAML(path string, n *yaml.Node) error {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(n); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func sequence(items []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: seqTag}
	for _, it := range items {
		n.Content = append(n.Content, scalar(it))
	}

	return n
}

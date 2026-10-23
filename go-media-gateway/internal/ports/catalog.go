package ports

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// StaticCatalog is the sandbox node directory. A production adaptor would
// read heartbeats instead of the NODES environment variable.
type StaticCatalog struct {
	Nodes []domain.Node
}

func (c StaticCatalog) Healthy(context.Context) ([]domain.Node, error) {
	out := make([]domain.Node, len(c.Nodes))
	copy(out, c.Nodes)
	return out, nil
}

func ParseNodes(raw string) ([]domain.Node, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []domain.Node{
			{ID: "edge-a", Region: "us-east-1"},
			{ID: "edge-b", Region: "us-east-1"},
			{ID: "edge-c", Region: "eu-west-1"},
		}, nil
	}
	parts := strings.Split(raw, ",")
	nodes := make([]domain.Node, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		region, id, ok := strings.Cut(part, ":")
		if !ok || !validToken(region) || !validToken(id) || strings.Contains(id, ":") {
			return nil, fmt.Errorf("node %q must be region:id", part)
		}
		nodes = append(nodes, domain.Node{ID: id, Region: region})
	}
	return nodes, nil
}

func validToken(value string) bool {
	if len(value) < 1 || len(value) > 63 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-':
		default:
			return false
		}
	}
	return true
}

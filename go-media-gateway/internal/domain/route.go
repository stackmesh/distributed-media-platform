package domain

import (
	"hash/fnv"
	"sort"
)

// Node is one healthy edge process that can hold a replica.
type Node struct {
	ID     string
	Region string
}

// Policy bounds object size and how many replicas a placement needs.
type Policy struct {
	MaxBytes int64
	Replicas int
}

func (p Policy) Normalized() Policy {
	if p.MaxBytes <= 0 {
		p.MaxBytes = 64 << 20
	}
	if p.Replicas <= 0 {
		p.Replicas = 2
	}
	return p
}

// RouteInput is the pure routing key. Region is a preference, not a pin.
type RouteInput struct {
	ContentHash string
	SizeBytes   int64
	MIME        string
	Region      string
}

// Placement is the ordered replica set. Primary is Replicas[0] of the full set,
// and Replicas here are the additional copies.
type Placement struct {
	Primary  string
	Replicas []string
	Region   string
}

type scoredNode struct {
	node  Node
	home  bool
	score uint64
}

// Route picks a deterministic replica set. Nodes in the preferred region
// outrank every other region, and rendezvous hashing orders nodes inside a tier.
func Route(nodes []Node, in RouteInput, policy Policy) (Placement, error) {
	policy = policy.Normalized()
	if !ValidHash(in.ContentHash) {
		return Placement{}, ErrHash
	}
	if !AllowedMIME(in.MIME) {
		return Placement{}, ErrMIME
	}
	if in.SizeBytes <= 0 {
		return Placement{}, ErrEmpty
	}
	if in.SizeBytes > policy.MaxBytes {
		return Placement{}, ErrTooLarge
	}

	scored := make([]scoredNode, 0, len(nodes))
	for _, node := range nodes {
		if node.ID == "" || node.Region == "" {
			continue
		}
		scored = append(scored, scoredNode{
			node:  node,
			home:  node.Region == in.Region,
			score: rendezvous(node.ID, in.ContentHash),
		})
	}
	if len(scored) == 0 {
		return Placement{}, ErrNoNodes
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].home != scored[j].home {
			return scored[i].home
		}
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].node.ID < scored[j].node.ID
	})

	count := policy.Replicas
	if count > len(scored) {
		count = len(scored)
	}
	chosen := scored[:count]
	placement := Placement{
		Primary:  chosen[0].node.ID,
		Region:   chosen[0].node.Region,
		Replicas: []string{},
	}
	if count > 1 {
		placement.Replicas = make([]string, 0, count-1)
		for _, node := range chosen[1:] {
			placement.Replicas = append(placement.Replicas, node.node.ID)
		}
	}
	return placement, nil
}

func rendezvous(nodeID, key string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(nodeID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(key))
	return h.Sum64()
}

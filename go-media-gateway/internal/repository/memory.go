package repository

import (
	"context"
	"sync"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// Memory is the in-process repository used when DATABASE_URL is unset.
type Memory struct {
	mu      sync.Mutex
	byID    map[string]domain.Asset
	byKey   map[string]string
	pending []domain.Usage
}

func NewMemory() *Memory {
	return &Memory{
		byID:  map[string]domain.Asset{},
		byKey: map[string]string{},
	}
}

func (m *Memory) InsertIfAbsent(_ context.Context, asset domain.Asset, usage domain.Usage) (domain.Asset, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := asset.TenantID + "/" + asset.ContentHash
	if id, ok := m.byKey[key]; ok {
		return cloneAsset(m.byID[id]), false, nil
	}
	stored := cloneAsset(asset)
	m.byID[stored.ID] = stored
	m.byKey[key] = stored.ID
	m.pending = append(m.pending, usage)
	return cloneAsset(stored), true, nil
}

func (m *Memory) Get(_ context.Context, id string) (domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	asset, ok := m.byID[id]
	if !ok {
		return domain.Asset{}, domain.ErrNotFound
	}
	return cloneAsset(asset), nil
}

func (m *Memory) PendingUsage(_ context.Context, limit int) ([]domain.Usage, error) {
	if limit <= 0 {
		limit = 32
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit > len(m.pending) {
		limit = len(m.pending)
	}
	out := make([]domain.Usage, limit)
	copy(out, m.pending[:limit])
	return out, nil
}

func (m *Memory) MarkPublished(_ context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := make([]domain.Usage, 0, len(m.pending))
	for _, usage := range m.pending {
		if usage.EventID != eventID {
			kept = append(kept, usage)
		}
	}
	m.pending = kept
	return nil
}

func cloneAsset(in domain.Asset) domain.Asset {
	in.Replicas = append([]string(nil), in.Replicas...)
	return in
}

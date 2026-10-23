package ports

import (
	"context"
	"io"
	"time"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// AssetRepository is the outbound port for metadata and the billing outbox.
// InsertIfAbsent reports created=false when the tenant already stored that hash.
type AssetRepository interface {
	InsertIfAbsent(ctx context.Context, asset domain.Asset, usage domain.Usage) (domain.Asset, bool, error)
	Get(ctx context.Context, id string) (domain.Asset, error)
	PendingUsage(ctx context.Context, limit int) ([]domain.Usage, error)
	MarkPublished(ctx context.Context, eventID string) error
}

// BlobStore is the content-addressed byte port.
type BlobStore interface {
	Put(ctx context.Context, tenant string, body io.Reader, max int64, expectHash string) (size int64, hash string, err error)
	Remove(ctx context.Context, tenant, hash string) error
}

// NodeCatalog returns edge nodes that can accept a replica.
type NodeCatalog interface {
	Healthy(ctx context.Context) ([]domain.Node, error)
}

// UsagePublisher delivers one ledger event. Publish must be safe to retry.
type UsagePublisher interface {
	Publish(ctx context.Context, usage domain.Usage) error
	Close() error
}

// IngestService is the driving application service behind the HTTP adaptor.
type IngestService struct {
	Assets AssetRepository
	Blobs  BlobStore
	Nodes  NodeCatalog
	Policy domain.Policy
	Region string
	Now    func() time.Time
}

func (s *IngestService) policy() domain.Policy {
	return s.Policy.Normalized()
}

func (s *IngestService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Ingest stores bytes, places replicas, and enqueues one storage usage event.
func (s *IngestService) Ingest(ctx context.Context, tenant, mime, expectHash string, body io.Reader) (domain.Asset, bool, error) {
	if !domain.ValidTenant(tenant) {
		return domain.Asset{}, false, domain.ErrTenant
	}
	normalized, err := domain.NormalizeMIME(mime)
	if err != nil {
		return domain.Asset{}, false, err
	}
	expectHash = normalizeHash(expectHash)
	if expectHash != "" && !domain.ValidHash(expectHash) {
		return domain.Asset{}, false, domain.ErrHash
	}

	size, hash, err := s.Blobs.Put(ctx, tenant, body, s.policy().MaxBytes, expectHash)
	if err != nil {
		return domain.Asset{}, false, err
	}
	placement, err := s.place(ctx, hash, normalized, s.Region, size)
	if err != nil {
		_ = s.Blobs.Remove(ctx, tenant, hash)
		return domain.Asset{}, false, err
	}

	createdAt := s.now()
	asset := domain.Asset{
		ID:          domain.AssetID(tenant, hash),
		TenantID:    tenant,
		ContentHash: hash,
		SizeBytes:   size,
		MIME:        normalized,
		Region:      placement.Region,
		PrimaryNode: placement.Primary,
		Replicas:    placement.Replicas,
		CreatedAt:   createdAt,
	}
	usage := domain.Usage{
		EventID:    domain.StorageEventID(tenant, hash),
		TenantID:   tenant,
		AssetID:    asset.ID,
		Meter:      domain.MeterStorageBytes,
		Quantity:   size,
		OccurredAt: createdAt,
	}
	saved, created, err := s.Assets.InsertIfAbsent(ctx, asset, usage)
	if err != nil {
		return domain.Asset{}, false, err
	}
	return saved, created, nil
}

// Preview places an object without writing bytes or billing.
func (s *IngestService) Preview(ctx context.Context, hash, mime, region string, size int64) (domain.Placement, error) {
	normalized, err := domain.NormalizeMIME(mime)
	if err != nil {
		return domain.Placement{}, err
	}
	if region == "" {
		region = s.Region
	}
	return s.place(ctx, normalizeHash(hash), normalized, region, size)
}

func (s *IngestService) Get(ctx context.Context, id string) (domain.Asset, error) {
	if id == "" {
		return domain.Asset{}, domain.ErrNotFound
	}
	return s.Assets.Get(ctx, id)
}

func (s *IngestService) place(ctx context.Context, hash, mime, region string, size int64) (domain.Placement, error) {
	nodes, err := s.Nodes.Healthy(ctx)
	if err != nil {
		return domain.Placement{}, err
	}
	return domain.Route(nodes, domain.RouteInput{
		ContentHash: hash,
		SizeBytes:   size,
		MIME:        mime,
		Region:      region,
	}, s.policy())
}

func normalizeHash(hash string) string {
	return lowerTrim(hash)
}

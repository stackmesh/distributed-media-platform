package ports

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackmesh/go-media-gateway/internal/domain"
	"github.com/stackmesh/go-media-gateway/internal/repository"
)

func TestIngestIsIdempotentAndDrainsOnce(t *testing.T) {
	svc, repo := testService(t)
	body := []byte("frame-bytes")
	var ticks int
	svc.Now = func() time.Time {
		ticks++
		return time.Date(2026, 10, 23, 0, 0, ticks, 0, time.UTC)
	}

	first, created, err := svc.Ingest(context.Background(), "demo", "video/mp4", "", bytes.NewReader(body))
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if first.PrimaryNode == "edge-c" || first.Region != "us-east-1" {
		t.Fatalf("placement drifted off home: %#v", first)
	}

	second, created, err := svc.Ingest(context.Background(), "demo", "video/mp4", "", bytes.NewReader(body))
	if err != nil || created {
		t.Fatalf("replay created=%v err=%v", created, err)
	}
	if second.ID != first.ID || !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("replay returned a new asset: %#v %#v", first, second)
	}

	pending, err := repo.PendingUsage(context.Background(), 10)
	if err != nil || len(pending) != 1 || pending[0].Quantity != int64(len(body)) {
		t.Fatalf("outbox: %+v %v", pending, err)
	}
	if pending[0].EventID != domain.StorageEventID("demo", first.ContentHash) {
		t.Fatalf("event id: %s", pending[0].EventID)
	}

	if err := DrainOnce(context.Background(), repo, errPublisher{}, 10); err == nil {
		t.Fatal("expected publisher failure")
	}
	pending, err = repo.PendingUsage(context.Background(), 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("failed drain consumed the outbox: %+v %v", pending, err)
	}
	if err := DrainOnce(context.Background(), repo, LogPublisher{}, 10); err != nil {
		t.Fatal(err)
	}
	pending, err = repo.PendingUsage(context.Background(), 10)
	if err != nil || len(pending) != 0 {
		t.Fatalf("outbox not drained: %+v %v", pending, err)
	}
}

func TestIngestRejectsTenantAndHash(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.Ingest(context.Background(), "Demo", "video/mp4", "", bytes.NewReader([]byte("x")))
	if err != domain.ErrTenant {
		t.Fatalf("tenant: %v", err)
	}
	_, _, err = svc.Ingest(context.Background(), "demo", "video/mp4", "abcd", bytes.NewReader([]byte("x")))
	if err != domain.ErrHash {
		t.Fatalf("hash: %v", err)
	}
}

type errPublisher struct{}

func (errPublisher) Publish(context.Context, domain.Usage) error { return errors.New("down") }
func (errPublisher) Close() error                                { return nil }

func testService(t *testing.T) (*IngestService, *repository.Memory) {
	t.Helper()
	blobs, err := repository.NewDiskBlob(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewMemory()
	return &IngestService{
		Assets: repo,
		Blobs:  blobs,
		Nodes: StaticCatalog{Nodes: []domain.Node{
			{ID: "edge-a", Region: "us-east-1"},
			{ID: "edge-b", Region: "us-east-1"},
			{ID: "edge-c", Region: "eu-west-1"},
		}},
		Policy: domain.Policy{MaxBytes: 1 << 20, Replicas: 2},
		Region: "us-east-1",
	}, repo
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stackmesh/go-media-gateway/internal/domain"
	"github.com/stackmesh/go-media-gateway/internal/ports"
	"github.com/stackmesh/go-media-gateway/internal/repository"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("gateway stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := getenv("HTTP_ADDR", ":8080")
	region := getenv("REGION", "us-east-1")
	blobDir := getenv("BLOB_DIR", "./data/blobs")
	topic := getenv("KAFKA_TOPIC", "stackmesh.billing.usage.v1")
	interval, err := time.ParseDuration(getenv("OUTBOX_INTERVAL", "500ms"))
	if err != nil || interval <= 0 {
		return fmt.Errorf("OUTBOX_INTERVAL must be a positive duration")
	}
	maxBytes, err := envInt64("MAX_OBJECT_BYTES", 64<<20)
	if err != nil {
		return err
	}
	replicas, err := envInt("REPLICAS", 2)
	if err != nil {
		return err
	}
	nodes, err := ports.ParseNodes(os.Getenv("NODES"))
	if err != nil {
		return err
	}

	blobs, err := repository.NewDiskBlob(blobDir)
	if err != nil {
		return err
	}

	var (
		assets ports.AssetRepository
		ping   func(context.Context) error
		store  = "memory"
		pool   *pgxpool.Pool
	)
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pool, err = repository.Connect(ctx, dsn)
		if err != nil {
			return err
		}
		defer pool.Close()
		assets = repository.NewAssetRepo(pool)
		ping = pool.Ping
		store = "postgres"
	} else {
		slog.Warn("DATABASE_URL unset; metadata and outbox are in memory")
		assets = repository.NewMemory()
	}

	var publisher ports.UsagePublisher = ports.LogPublisher{}
	if raw := strings.TrimSpace(os.Getenv("KAFKA_BROKERS")); raw != "" {
		brokers := splitCSV(raw)
		publisher = ports.NewKafkaPublisher(brokers, topic)
		slog.Info("billing publisher", "brokers", brokers, "topic", topic)
	} else {
		slog.Warn("KAFKA_BROKERS unset; usage events are logged")
	}
	defer publisher.Close()

	service := &ports.IngestService{
		Assets: assets,
		Blobs:  blobs,
		Nodes:  ports.StaticCatalog{Nodes: nodes},
		Policy: domain.Policy{MaxBytes: maxBytes, Replicas: replicas},
		Region: region,
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           ports.NewHandler(service, ping),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("gateway listening", "addr", addr, "store", store, "region", region, "nodes", len(nodes))
		errCh <- server.ListenAndServe()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := ports.DrainOnce(ctx, assets, publisher, 32); err != nil && ctx.Err() == nil {
					slog.Error("outbox drain", "err", err)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shut)
		wg.Wait()
		return nil
	case err := <-errCh:
		stop()
		wg.Wait()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

func envInt64(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

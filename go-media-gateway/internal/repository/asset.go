package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// AssetRepo stores object metadata and the billing outbox in Postgres.
type AssetRepo struct {
	pool *pgxpool.Pool
}

func NewAssetRepo(pool *pgxpool.Pool) *AssetRepo {
	return &AssetRepo{pool: pool}
}

func (r *AssetRepo) InsertIfAbsent(ctx context.Context, asset domain.Asset, usage domain.Usage) (domain.Asset, bool, error) {
	if asset.Replicas == nil {
		asset.Replicas = []string{}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Asset{}, false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO asset (
			id, tenant_id, content_hash, size_bytes, mime, region, primary_node, replicas, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (tenant_id, content_hash) DO NOTHING
	`, asset.ID, asset.TenantID, asset.ContentHash, asset.SizeBytes, asset.MIME, asset.Region, asset.PrimaryNode, asset.Replicas, asset.CreatedAt)
	if err != nil {
		return domain.Asset{}, false, err
	}
	if tag.RowsAffected() == 0 {
		existing, err := scanAsset(tx.QueryRow(ctx, selectAsset+` WHERE tenant_id = $1 AND content_hash = $2`, asset.TenantID, asset.ContentHash))
		if err != nil {
			return domain.Asset{}, false, err
		}
		return existing, false, nil
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO usage_outbox (event_id, tenant_id, asset_id, meter, quantity, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, usage.EventID, usage.TenantID, usage.AssetID, usage.Meter, usage.Quantity, usage.OccurredAt)
	if err != nil {
		return domain.Asset{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Asset{}, false, err
	}
	return asset, true, nil
}

func (r *AssetRepo) Get(ctx context.Context, id string) (domain.Asset, error) {
	return scanAsset(r.pool.QueryRow(ctx, selectAsset+` WHERE id = $1`, id))
}

func (r *AssetRepo) PendingUsage(ctx context.Context, limit int) ([]domain.Usage, error) {
	if limit <= 0 {
		limit = 32
	}
	rows, err := r.pool.Query(ctx, `
		SELECT event_id, tenant_id, asset_id, meter, quantity, occurred_at
		FROM usage_outbox
		WHERE published_at IS NULL
		ORDER BY occurred_at, event_id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Usage, 0, limit)
	for rows.Next() {
		var usage domain.Usage
		if err := rows.Scan(&usage.EventID, &usage.TenantID, &usage.AssetID, &usage.Meter, &usage.Quantity, &usage.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, usage)
	}
	return out, rows.Err()
}

func (r *AssetRepo) MarkPublished(ctx context.Context, eventID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE usage_outbox SET published_at = now()
		WHERE event_id = $1 AND published_at IS NULL
	`, eventID)
	return err
}

const selectAsset = `
	SELECT id, tenant_id, content_hash, size_bytes, mime, region, primary_node, COALESCE(replicas, '{}'), created_at
	FROM asset`

func scanAsset(row pgx.Row) (domain.Asset, error) {
	var asset domain.Asset
	err := row.Scan(
		&asset.ID,
		&asset.TenantID,
		&asset.ContentHash,
		&asset.SizeBytes,
		&asset.MIME,
		&asset.Region,
		&asset.PrimaryNode,
		&asset.Replicas,
		&asset.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Asset{}, err
	}
	if asset.Replicas == nil {
		asset.Replicas = []string{}
	}
	return asset, nil
}

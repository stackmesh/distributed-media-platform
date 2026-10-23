package ports

import "context"

// DrainOnce publishes unpublished usage and marks each row only after the broker accepts it.
// A crash between publish and mark is safe: the ledger dedupes on event id.
func DrainOnce(ctx context.Context, repo AssetRepository, publisher UsagePublisher, limit int) error {
	if limit <= 0 {
		limit = 32
	}
	batch, err := repo.PendingUsage(ctx, limit)
	if err != nil {
		return err
	}
	for _, usage := range batch {
		if err := publisher.Publish(ctx, usage); err != nil {
			return err
		}
		if err := repo.MarkPublished(ctx, usage.EventID); err != nil {
			return err
		}
	}
	return nil
}

package cron

import (
	"bpl/metrics"
	"bpl/repository"
	"context"
	"log/slog"
	"time"
)

const (
	activeUsersWindow   = 15 * time.Minute
	activeUsersInterval = 15 * time.Second
)

// ActiveUsersMetricLoop periodically updates the gauge of unique players that
// were active within the last activeUsersWindow.
func ActiveUsersMetricLoop(ctx context.Context) {
	activityRepo := repository.NewActivityRepository()
	update := func() {
		count, err := activityRepo.CountActiveUsersSince(time.Now().Add(-activeUsersWindow))
		if err != nil {
			slog.Error("failed to count active users", "error", err)
			return
		}
		metrics.ActiveUsersGauge.Set(float64(count))
	}
	update()
	ticker := time.NewTicker(activeUsersInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update()
		}
	}
}

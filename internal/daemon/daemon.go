package daemon

import (
	"context"
	"log"
	"time"

	"github.com/fireharp/tinkershop/internal/config"
	"github.com/fireharp/tinkershop/internal/scan"
)

func Run(ctx context.Context, cfg config.Config, interval time.Duration) error {
	if interval <= 0 {
		interval = 12 * time.Hour
	}

	runOnce := func() {
		summary, err := scan.Run(ctx, cfg)
		if err != nil {
			log.Printf("scan failed: %v", err)
			return
		}
		log.Printf("scan ok: run=%d projects=%d observations=%d", summary.RunID, summary.ProjectCount, summary.ObservationCount)
	}

	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			runOnce()
		}
	}
}

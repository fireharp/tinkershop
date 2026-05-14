package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/fireharp/tinkershop/internal/blobstore"
	"github.com/fireharp/tinkershop/internal/collectors"
	"github.com/fireharp/tinkershop/internal/config"
	"github.com/fireharp/tinkershop/internal/storage"
)

type Summary struct {
	RunID            int64
	ProjectCount     int
	ObservationCount int
}

func Run(ctx context.Context, cfg config.Config) (Summary, error) {
	if err := cfg.Validate(); err != nil {
		return Summary{}, err
	}

	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		return Summary{}, err
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return Summary{}, err
	}

	runID, err := store.StartRun(ctx, time.Now().UTC())
	if err != nil {
		return Summary{}, err
	}

	blobStore := blobstore.Store{Root: cfg.BlobDir, Compression: cfg.Compression}
	meta, err := blobStore.Put(ctx, "application/json", []byte(`{"run":"started"}`))
	if err == nil {
		_ = store.UpsertBlob(ctx, storage.Blob{
			SHA256:            meta.SHA256,
			Path:              meta.Path,
			Compression:       meta.Compression,
			MediaType:         meta.MediaType,
			BytesUncompressed: meta.BytesUncompressed,
			BytesStored:       meta.BytesStored,
			CreatedAt:         meta.CreatedAt,
		})
	}

	result, err := collectors.CollectGit(ctx, cfg.Roots, cfg.Policies, runID)
	if err != nil {
		_ = store.FinishRun(ctx, runID, "error", err.Error(), time.Now().UTC())
		return Summary{}, err
	}

	for _, project := range result.Projects {
		if err := store.UpsertProject(ctx, project); err != nil {
			_ = store.FinishRun(ctx, runID, "error", err.Error(), time.Now().UTC())
			return Summary{}, err
		}
	}
	for _, observation := range result.Observations {
		if err := store.InsertObservation(ctx, observation); err != nil {
			_ = store.FinishRun(ctx, runID, "error", err.Error(), time.Now().UTC())
			return Summary{}, err
		}
	}

	summary := Summary{
		RunID:            runID,
		ProjectCount:     len(result.Projects),
		ObservationCount: len(result.Observations),
	}
	if err := store.FinishRun(ctx, runID, "ok", fmt.Sprintf("%d projects, %d observations", summary.ProjectCount, summary.ObservationCount), time.Now().UTC()); err != nil {
		return Summary{}, err
	}
	return summary, nil
}

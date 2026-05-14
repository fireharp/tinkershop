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
	Since            *time.Time
}

func Run(ctx context.Context, cfg config.Config) (Summary, error) {
	if err := cfg.Validate(); err != nil {
		return Summary{}, err
	}

	since, err := cfg.SinceTime(time.Now())
	if err != nil {
		return Summary{}, err
	}

	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = store.Close() }()

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

	result, err := collectors.CollectGit(ctx, collectors.GitOptions{
		Roots:    cfg.Roots,
		Policies: cfg.Policies,
		RunID:    runID,
		Since:    since,
	})
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
		Since:            since,
	}
	summaryText := fmt.Sprintf("%d projects, %d observations", summary.ProjectCount, summary.ObservationCount)
	if since != nil {
		summaryText = fmt.Sprintf("%s since %s", summaryText, since.In(time.Local).Format("2006-01-02"))
	}
	if err := store.FinishRun(ctx, runID, "ok", summaryText, time.Now().UTC()); err != nil {
		return Summary{}, err
	}
	return summary, nil
}

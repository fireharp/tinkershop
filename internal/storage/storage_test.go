package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreMigratesAndListsProjects(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	runID, err := store.StartRun(ctx, time.Unix(10, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	err = store.UpsertProject(ctx, Project{
		ID:             "p1",
		Path:           "/tmp/project",
		Name:           "project",
		PolicyState:    "needs_review",
		DirtyCount:     2,
		LastCommitUnix: 9,
		UpdatedAt:      time.Unix(11, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.FinishRun(ctx, runID, "ok", "1 project", time.Unix(12, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d", len(projects))
	}
	if projects[0].DirtyCount != 2 {
		t.Fatalf("dirty count = %d", projects[0].DirtyCount)
	}
}

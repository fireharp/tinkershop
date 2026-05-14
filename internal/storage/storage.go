package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type Run struct {
	ID          int64     `json:"id"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"`
	Summary     string    `json:"summary"`
}

type Project struct {
	ID             string    `json:"id"`
	Path           string    `json:"path"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"display_name"`
	RemoteURL      string    `json:"remote_url"`
	PolicyState    string    `json:"policy_state"`
	DirtyCount     int       `json:"dirty_count"`
	LastCommitUnix int64     `json:"last_commit_unix"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Observation struct {
	RunID      int64     `json:"run_id"`
	ProjectID  string    `json:"project_id"`
	Source     string    `json:"source"`
	Kind       string    `json:"kind"`
	ObservedAt time.Time `json:"observed_at"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	BlobSHA    string    `json:"blob_sha"`
	Confidence float64   `json:"confidence"`
}

type Blob struct {
	SHA256            string
	Path              string
	Compression       string
	MediaType         string
	BytesUncompressed int64
	BytesStored       int64
	CreatedAt         time.Time
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *Store) StartRun(ctx context.Context, startedAt time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`insert into runs(started_at, status) values(?, ?)`,
		startedAt.Unix(), "running",
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishRun(ctx context.Context, id int64, status, summary string, completedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`update runs set completed_at = ?, status = ?, summary = ? where id = ?`,
		completedAt.Unix(), status, summary, id,
	)
	return err
}

func (s *Store) UpsertProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `
insert into projects(id, path, name, display_name, remote_url, policy_state, dirty_count, last_commit_unix, updated_at)
values(?, ?, ?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update set
  path = excluded.path,
  name = excluded.name,
  display_name = excluded.display_name,
  remote_url = excluded.remote_url,
  policy_state = excluded.policy_state,
  dirty_count = excluded.dirty_count,
  last_commit_unix = excluded.last_commit_unix,
  updated_at = excluded.updated_at
`, p.ID, p.Path, p.Name, p.DisplayName, p.RemoteURL, p.PolicyState, p.DirtyCount, p.LastCommitUnix, p.UpdatedAt.Unix())
	return err
}

func (s *Store) InsertObservation(ctx context.Context, o Observation) error {
	_, err := s.db.ExecContext(ctx, `
insert into observations(run_id, project_id, source, kind, observed_at, title, summary, blob_sha, confidence)
values(?, ?, ?, ?, ?, ?, ?, ?, ?)
`, o.RunID, o.ProjectID, o.Source, o.Kind, o.ObservedAt.Unix(), o.Title, o.Summary, o.BlobSHA, o.Confidence)
	return err
}

func (s *Store) UpsertBlob(ctx context.Context, b Blob) error {
	_, err := s.db.ExecContext(ctx, `
insert into blobs(sha256, path, compression, media_type, bytes_uncompressed, bytes_stored, created_at)
values(?, ?, ?, ?, ?, ?, ?)
on conflict(sha256) do update set
  path = excluded.path,
  compression = excluded.compression,
  media_type = excluded.media_type,
  bytes_uncompressed = excluded.bytes_uncompressed,
  bytes_stored = excluded.bytes_stored
`, b.SHA256, b.Path, b.Compression, b.MediaType, b.BytesUncompressed, b.BytesStored, b.CreatedAt.Unix())
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `
select id, path, name, display_name, remote_url, policy_state, dirty_count, last_commit_unix, updated_at
from projects
order by updated_at desc, name asc
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var updatedAt int64
		if err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.DisplayName, &p.RemoteURL, &p.PolicyState, &p.DirtyCount, &p.LastCommitUnix, &updatedAt); err != nil {
			return nil, err
		}
		p.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) ListRuns(ctx context.Context, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
select id, started_at, coalesce(completed_at, 0), status, summary
from runs
order by id desc
limit ?
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var r Run
		var startedAt, completedAt int64
		if err := rows.Scan(&r.ID, &startedAt, &completedAt, &r.Status, &r.Summary); err != nil {
			return nil, err
		}
		r.StartedAt = time.Unix(startedAt, 0).UTC()
		if completedAt > 0 {
			r.CompletedAt = time.Unix(completedAt, 0).UTC()
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

const schema = `
create table if not exists runs (
  id integer primary key autoincrement,
  started_at integer not null,
  completed_at integer,
  status text not null,
  summary text not null default ''
);

create table if not exists projects (
  id text primary key,
  path text not null unique,
  name text not null,
  display_name text not null default '',
  remote_url text not null default '',
  policy_state text not null,
  dirty_count integer not null default 0,
  last_commit_unix integer not null default 0,
  updated_at integer not null
);

create table if not exists observations (
  id integer primary key autoincrement,
  run_id integer not null references runs(id),
  project_id text not null references projects(id),
  source text not null,
  kind text not null,
  observed_at integer not null,
  title text not null,
  summary text not null default '',
  blob_sha text not null default '',
  confidence real not null default 0
);

create table if not exists blobs (
  sha256 text primary key,
  path text not null,
  compression text not null,
  media_type text not null,
  bytes_uncompressed integer not null,
  bytes_stored integer not null,
  created_at integer not null
);
`

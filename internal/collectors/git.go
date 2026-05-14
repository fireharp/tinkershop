package collectors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fireharp/tinkershop/internal/policy"
	"github.com/fireharp/tinkershop/internal/storage"
)

type GitOptions struct {
	Roots    []string
	Policies []policy.Rule
	RunID    int64
	Since    *time.Time
}

type dirtyState struct {
	Count     int
	LastMTime int64
}

func CollectGit(ctx context.Context, opts GitOptions) (Result, error) {
	repos, err := discoverRepos(opts.Roots)
	if err != nil {
		return Result{}, err
	}

	now := time.Now().UTC()
	result := Result{}
	for _, repo := range repos {
		dirty := gitDirtyState(ctx, repo)
		lastCommit := gitLastCommit(ctx, repo)
		lastActivity := maxInt64(lastCommit, dirty.LastMTime)
		if opts.Since != nil && lastActivity < opts.Since.Unix() {
			continue
		}

		decision := policy.Evaluate(repo, opts.Policies)
		name := filepath.Base(repo)
		displayName := decision.DisplayName
		if displayName == "" {
			displayName = name
		}
		remoteURL := gitRemoteURL(ctx, repo)

		project := storage.Project{
			ID:             projectID(repo),
			Path:           repo,
			Name:           name,
			DisplayName:    displayName,
			RemoteURL:      remoteURL,
			PolicyState:    string(decision.State),
			DirtyCount:     dirty.Count,
			LastCommitUnix: lastCommit,
			UpdatedAt:      now,
		}
		result.Projects = append(result.Projects, project)
		result.Observations = append(result.Observations, storage.Observation{
			RunID:      opts.RunID,
			ProjectID:  project.ID,
			Source:     "git",
			Kind:       "repo_activity",
			ObservedAt: now,
			Title:      displayName,
			Summary:    fmt.Sprintf("dirty=%d last_commit_unix=%d last_dirty_mtime=%d", dirty.Count, lastCommit, dirty.LastMTime),
			Confidence: 0.8,
		})
	}
	return result, nil
}

func discoverRepos(roots []string) ([]string, error) {
	seen := map[string]bool{}
	var repos []string

	for _, root := range roots {
		if root == "" {
			continue
		}
		cleanRoot := filepath.Clean(root)
		err := filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil //nolint:nilerr // skip unreadable entries and keep walking
			}
			if !d.IsDir() {
				return nil
			}
			if d.Name() == ".git" {
				repo := filepath.Dir(path)
				if !seen[repo] {
					seen[repo] = true
					repos = append(repos, repo)
				}
				return filepath.SkipDir
			}
			if d.Name() == "node_modules" || d.Name() == ".venv" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return repos, nil
}

func gitDirtyState(ctx context.Context, repo string) dirtyState {
	out, err := git(ctx, repo, "status", "--porcelain", "-z")
	if err != nil {
		return dirtyState{}
	}
	if out == "" {
		return dirtyState{}
	}

	entries := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	state := dirtyState{}
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue
		}

		state.Count++
		status := entry[:2]
		relPath := entry[3:]
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			i++
		}

		fullPath := filepath.Join(repo, relPath)
		if info, err := os.Stat(fullPath); err == nil && info.ModTime().Unix() > state.LastMTime {
			state.LastMTime = info.ModTime().Unix()
		}
	}

	if state.Count > 0 && state.LastMTime == 0 {
		if info, err := os.Stat(filepath.Join(repo, ".git", "index")); err == nil {
			state.LastMTime = info.ModTime().Unix()
		}
	}

	return state
}

func gitLastCommit(ctx context.Context, repo string) int64 {
	out, err := git(ctx, repo, "log", "-1", "--format=%ct")
	if err != nil {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func gitRemoteURL(ctx context.Context, repo string) string {
	out, err := git(ctx, repo, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func git(ctx context.Context, repo string, args ...string) (string, error) {
	allArgs := append([]string{"-C", repo}, args...)
	cmd := exec.CommandContext(ctx, "git", allArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func projectID(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(path)))
	return hex.EncodeToString(sum[:])
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

package collectors

import (
	"bufio"
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

func CollectGit(ctx context.Context, roots []string, policies []policy.Rule, runID int64) (Result, error) {
	repos, err := discoverRepos(roots)
	if err != nil {
		return Result{}, err
	}

	now := time.Now().UTC()
	result := Result{}
	for _, repo := range repos {
		decision := policy.Evaluate(repo, policies)
		name := filepath.Base(repo)
		displayName := decision.DisplayName
		if displayName == "" {
			displayName = name
		}
		dirtyCount := gitDirtyCount(ctx, repo)
		lastCommit := gitLastCommit(ctx, repo)
		remoteURL := gitRemoteURL(ctx, repo)

		project := storage.Project{
			ID:             projectID(repo),
			Path:           repo,
			Name:           name,
			DisplayName:    displayName,
			RemoteURL:      remoteURL,
			PolicyState:    string(decision.State),
			DirtyCount:     dirtyCount,
			LastCommitUnix: lastCommit,
			UpdatedAt:      now,
		}
		result.Projects = append(result.Projects, project)
		result.Observations = append(result.Observations, storage.Observation{
			RunID:      runID,
			ProjectID:  project.ID,
			Source:     "git",
			Kind:       "repo_activity",
			ObservedAt: now,
			Title:      displayName,
			Summary:    fmt.Sprintf("dirty=%d last_commit_unix=%d", dirtyCount, lastCommit),
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
		err := filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
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

func gitDirtyCount(ctx context.Context, repo string) int {
	out, err := git(ctx, repo, "status", "--porcelain")
	if err != nil {
		return 0
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	count := 0
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count
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

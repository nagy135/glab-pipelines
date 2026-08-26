package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const pipelineCacheVersion = 2

type pipelineCache struct {
	Version   int        `json:"version"`
	CachedAt  time.Time  `json:"cached_at"`
	Repo      string     `json:"repo"`
	Status    string     `json:"status"`
	Limit     int        `json:"limit"`
	Pipelines []pipeline `json:"pipelines"`
}

func loadPipelineCache(repo, status string, limit int) ([]pipeline, bool) {
	path, err := pipelineCachePath(repo, status, limit)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cache pipelineCache
	if err := json.Unmarshal(data, &cache); err != nil ||
		cache.Version != pipelineCacheVersion ||
		cache.Repo != repo ||
		cache.Status != status ||
		cache.Limit != limit {
		return nil, false
	}
	normalizePipelineCommitTitles(cache.Pipelines)
	for i := range cache.Pipelines {
		sanitizePipeline(&cache.Pipelines[i])
	}
	return cache.Pipelines, len(cache.Pipelines) > 0
}

func savePipelineCache(repo, status string, limit int, pipelines []pipeline) {
	pipelines = append([]pipeline(nil), pipelines...)
	normalizePipelineCommitTitles(pipelines)
	path, err := pipelineCachePath(repo, status, limit)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(pipelineCache{
		Version:   pipelineCacheVersion,
		CachedAt:  time.Now(),
		Repo:      repo,
		Status:    status,
		Limit:     limit,
		Pipelines: pipelines,
	}, "", "  ")
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pipelines-*.tmp")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(tmpPath, path)
}

func normalizePipelineCommitTitles(pipelines []pipeline) {
	for i := range pipelines {
		if pipelines[i].CommitTitle == "" {
			pipelines[i].CommitTitle = pipelines[i].Commit.Title
		}
		if pipelines[i].Commit.Title == "" {
			pipelines[i].Commit.Title = pipelines[i].CommitTitle
		}
	}
}

func pipelineCachePath(repo, status string, limit int) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	keyRepo := repo
	if keyRepo == "" {
		if wd, err := os.Getwd(); err == nil {
			keyRepo = wd
		}
	}
	h := sha1.Sum([]byte(fmt.Sprintf("%s\x00%s\x00%d", keyRepo, status, limit)))
	return filepath.Join(cacheDir, "glab-pipelines", "pipelines", hex.EncodeToString(h[:])+".json"), nil
}

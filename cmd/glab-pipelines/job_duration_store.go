package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const jobDurationStoreVersion = 1

type jobDurationStat struct {
	Average   float64 `json:"average"`
	Count     int     `json:"count"`
	LastJobID int64   `json:"last_job_id"`
}

type jobDurationStore struct {
	Version int                        `json:"version"`
	Repo    string                     `json:"repo"`
	Jobs    map[string]jobDurationStat `json:"jobs"`
}

func loadJobDurations(repo string) map[string]jobDurationStat {
	path, repoKey, err := jobDurationStorePath(repo)
	if err != nil {
		return make(map[string]jobDurationStat)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]jobDurationStat)
	}
	var store jobDurationStore
	if err := json.Unmarshal(data, &store); err != nil || store.Version != jobDurationStoreVersion || store.Repo != repoKey || store.Jobs == nil {
		return make(map[string]jobDurationStat)
	}
	return store.Jobs
}

func saveJobDurations(repo string, jobs map[string]jobDurationStat) {
	path, repoKey, err := jobDurationStorePath(repo)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(jobDurationStore{
		Version: jobDurationStoreVersion,
		Repo:    repoKey,
		Jobs:    jobs,
	}, "", "  ")
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".job-durations-*.tmp")
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

func jobDurationStorePath(repo string) (string, string, error) {
	repoKey := repo
	if repoKey == "" {
		var err error
		repoKey, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	h := sha1.Sum([]byte(repoKey))
	path, err := dataStorePath(filepath.Join("job-durations", hex.EncodeToString(h[:])+".json"))
	return path, repoKey, err
}

func recordJobDurations(stats map[string]jobDurationStat, jobs []job) bool {
	changed := false
	for _, j := range jobs {
		if j.Name == "" || j.FinishedAt == "" || j.Duration == nil || *j.Duration <= 0 {
			continue
		}
		stat := stats[j.Name]
		if j.ID <= stat.LastJobID {
			continue
		}
		stat.Average = (stat.Average*float64(stat.Count) + *j.Duration) / float64(stat.Count+1)
		stat.Count++
		stat.LastJobID = j.ID
		stats[j.Name] = stat
		changed = true
	}
	return changed
}

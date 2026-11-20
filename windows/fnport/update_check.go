package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	releaseAPI       = "https://api.github.com/repos/kabachok-hub/matchroute/releases?per_page=20"
	releaseDownloads = "https://github.com/kabachok-hub/matchroute/releases/download/"
)

type releaseUpdate struct {
	version string
	setup   string
}

type githubRelease struct {
	Tag    string `json:"tag_name"`
	Draft  bool   `json:"draft"`
	Assets []struct {
		Name  string `json:"name"`
		URL   string `json:"browser_download_url"`
		State string `json:"state"`
	} `json:"assets"`
}

var (
	updateOnce sync.Once
	updateInfo atomic.Pointer[releaseUpdate]
)

func startUpdateChecks() {
	current := strings.TrimPrefix(version, "v")
	if _, ok := semverParts(current); !ok {
		return
	}
	updateOnce.Do(func() {
		go func() {
			time.Sleep(2 * time.Second)
			check := func() {
				next, ok := fetchReleaseUpdate(current)
				if !ok {
					return
				}
				last := updateInfo.Load()
				if last == nil && next == nil || last != nil && next != nil && *last == *next {
					return
				}
				updateInfo.Store(next)
				changes.Add(1)
			}
			check()
			ticker := time.NewTicker(4 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				check()
			}
		}()
	})
}

func fetchReleaseUpdate(current string) (update *releaseUpdate, checked bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MatchRoute/"+version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var releases []githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&releases); err != nil {
		return nil, false
	}
	return newestInstaller(releases, current), true
}

func newestInstaller(releases []githubRelease, current string) *releaseUpdate {
	var best *releaseUpdate
	for _, release := range releases {
		tag := strings.TrimPrefix(release.Tag, "matchroute-")
		remote, ok := strings.CutPrefix(tag, "v")
		if release.Draft || !ok || !newerSemver(remote, current) ||
			(best != nil && !newerSemver(remote, best.version)) {
			continue
		}
		name := "MatchRoute-v" + remote + "-windows-x64-setup.exe"
		setup := releaseDownloads + release.Tag + "/" + name
		for _, asset := range release.Assets {
			if asset.Name == name && asset.State == "uploaded" && asset.URL == setup {
				best = &releaseUpdate{version: remote, setup: setup}
				break
			}
		}
	}
	return best
}

func semverParts(s string) ([3]int, bool) {
	var result [3]int
	parts := strings.Split(s, ".")
	if len(parts) != len(result) {
		return result, false
	}
	for i, p := range parts {
		if p == "" || len(p) > 1 && p[0] == '0' {
			return result, false
		}
		for _, digit := range p {
			if digit < '0' || digit > '9' {
				return result, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return result, false
		}
		result[i] = n
	}
	return result, true
}

func newerSemver(candidate, current string) bool {
	a, ok1 := semverParts(candidate)
	b, ok2 := semverParts(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

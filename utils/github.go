package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	username      = "AnAverageBeing"
	repo          = "JavaDeezNuts"
	maxBufferSize = 2048
	cacheDuration = 10 * time.Minute
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    100,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	cache     = make(map[string]cachedResult)
	cacheLock sync.RWMutex
)

type cachedResult struct {
	content   string
	timestamp time.Time
}

// FetchFileFromGitHub fetches a file from a public GitHub repository and returns its contents as a string.
// It supports optional caching if the `cache` parameter is set to true.
func FetchFileFromGitHub(filePath string, useCache bool) (string, error) {
	cacheKey := username + "/" + repo + "/" + filePath

	if useCache {
		// Check if the file is present in the cache and not expired.
		cacheLock.RLock()
		cached, found := cache[cacheKey]
		cacheLock.RUnlock()

		if found && time.Since(cached.timestamp) < cacheDuration {
			return cached.content, nil
		}
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", username, repo, filePath)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch file, status code: %d", resp.StatusCode)
	}

	var sb strings.Builder
	buf := make([]byte, maxBufferSize)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		sb.Write(buf[:n])
	}

	content := sb.String()

	if useCache {
		// Update the cache with the fetched content and the current timestamp.
		cacheLock.Lock()
		cache[cacheKey] = cachedResult{
			content:   content,
			timestamp: time.Now(),
		}
		cacheLock.Unlock()
	}

	return content, nil
}

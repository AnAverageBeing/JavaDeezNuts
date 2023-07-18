package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	username      = "AnAverageBeing"
	repo          = "JavaDeezNuts"
	maxBufferSize = 8192
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 30 * time.Second,
	},
}

// FetchFileFromGitHub fetches a file from a public GitHub repository and returns its contents as a string.
func FetchFileFromGitHub(filePath string) (string, error) {
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

	return sb.String(), nil
}

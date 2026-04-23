package runninghub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type DownloadedFile struct {
	URL        string
	OutputType string
	Path       string
}

func (c *Client) DownloadFile(ctx context.Context, fileURL, destPath string) error {
	if strings.TrimSpace(fileURL) == "" {
		return errors.New("empty file url")
	}
	if strings.TrimSpace(destPath) == "" {
		return errors.New("empty destination path")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}
	if c != nil && c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	hc := http.DefaultClient
	if c != nil && c.httpClient != nil {
		hc = c.httpClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func (c *Client) DownloadTaskResults(ctx context.Context, task *QueryV2Response, outputDir string) ([]DownloadedFile, error) {
	if task == nil {
		return nil, errors.New("nil task response")
	}
	if strings.TrimSpace(outputDir) == "" {
		return nil, errors.New("empty output dir")
	}
	if len(task.Results) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	downloads := make([]DownloadedFile, 0, len(task.Results))
	for idx, item := range task.Results {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		name, err := taskResultFilename(task.TaskID, idx, item)
		if err != nil {
			return nil, err
		}
		destPath := filepath.Join(outputDir, name)
		if err := c.DownloadFile(ctx, item.URL, destPath); err != nil {
			return nil, err
		}
		downloads = append(downloads, DownloadedFile{
			URL:        item.URL,
			OutputType: item.OutputType,
			Path:       destPath,
		})
	}
	return downloads, nil
}

func taskResultFilename(taskID string, index int, item QueryV2ResultItem) (string, error) {
	u, err := url.Parse(item.URL)
	if err != nil {
		return "", err
	}
	ext := path.Ext(u.Path)
	if ext == "" {
		trimmed := strings.TrimSpace(item.OutputType)
		if trimmed != "" {
			ext = "." + strings.TrimPrefix(trimmed, ".")
		}
	}
	base := strings.TrimSpace(taskID)
	if base == "" {
		base = "result"
	}
	return fmt.Sprintf("%s_%02d%s", sanitizeFilename(base), index+1, ext), nil
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	out := strings.TrimSpace(replacer.Replace(s))
	if out == "" {
		return "result"
	}
	return out
}
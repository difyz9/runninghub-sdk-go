package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const defaultConfigFile = "./examples/text_to_image/config.yaml"
const defaultPayloadFile = "./examples/text_to_image/payload.example.json"
const defaultModelPathValue = "/openapi/v2/seedream-v4/text-to-image"

func main() {
	config, err := runninghub.LoadYAMLConfig[Config](defaultConfigFile)
	if err != nil {
		exitf("load config: %v", err)
	}

	if config.APIKey == "" {
		exitf("missing API key in config: %s", config.Path)
	}

	pollInterval, err := config.PollIntervalDuration()
	if err != nil {
		exitf("parse poll interval from config: %v", err)
	}
	timeout, err := config.TimeoutDuration()
	if err != nil {
		exitf("parse timeout from config: %v", err)
	}

	reqBody, err := loadPayload(config.PayloadFile)
	if err != nil {
		exitf("load payload: %v", err)
	}
	if config.WebhookURL != "" {
		reqBody["webhookUrl"] = config.WebhookURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := runninghub.New(config.APIKey)
	if err != nil {
		exitf("create client: %v", err)
	}

	runTask(ctx, client, config.ModelPath, reqBody, config.PreviewOnly, pollInterval, config.OutputDir)
}

type Config struct {
	Path         string `yaml:"-"`
	APIKey       string `yaml:"apiKey"`
	ModelPath    string `yaml:"modelPath"`
	PayloadFile  string `yaml:"payloadFile"`
	OutputDir    string `yaml:"outputDir"`
	PollInterval string `yaml:"pollInterval"`
	Timeout      string `yaml:"timeout"`
	PreviewOnly  bool   `yaml:"previewOnly"`
	WebhookURL   string `yaml:"webhookUrl"`
}

func (c *Config) SetConfigPath(path string) {
	c.Path = path
}

func (c *Config) ApplyYAMLDefaults() {
	if c.ModelPath == "" {
		c.ModelPath = defaultModelPathValue
	}
	if c.PayloadFile == "" {
		c.PayloadFile = defaultPayloadFile
	}
	if c.OutputDir == "" {
		c.OutputDir = "./examples/text_to_image/output"
	}
	if c.PollInterval == "" {
		c.PollInterval = "2s"
	}
	if c.Timeout == "" {
		c.Timeout = "5m"
	}
}

func (c *Config) PollIntervalDuration() (time.Duration, error) {
	if c == nil {
		return 0, errors.New("config cannot be nil")
	}
	if c.PollInterval == "" {
		return 2 * time.Second, nil
	}
	return time.ParseDuration(c.PollInterval)
}

func (c *Config) TimeoutDuration() (time.Duration, error) {
	if c == nil {
		return 0, errors.New("config cannot be nil")
	}
	if c.Timeout == "" {
		return 5 * time.Minute, nil
	}
	return time.ParseDuration(c.Timeout)
}

func loadPayload(payloadFile string) (map[string]any, error) {
	if payloadFile == "" {
		return nil, errors.New("payload file path cannot be empty")
	}

	raw, err := os.ReadFile(payloadFile)
	if err != nil {
		return nil, err
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func runTask(ctx context.Context, client *runninghub.Client, modelPath string, reqBody map[string]any, previewOnly bool, pollInterval time.Duration, outputDir string) {
	if previewOnly {
		price, err := client.PricePreview(ctx, modelPath, reqBody)
		if err != nil {
			exitWithSDKError(err)
		}
		printJSON("price_preview", price)
		return
	}

	task, err := client.RunStandardModel(ctx, modelPath, reqBody)
	if err != nil {
		exitWithSDKError(err)
	}
	printJSON("submit", task)

	for {
		select {
		case <-ctx.Done():
			exitf("poll timeout: %v", ctx.Err())
		case <-time.After(pollInterval):
		}

		out, err := client.QueryTaskV2(ctx, task.TaskID)
		if err != nil {
			exitWithSDKError(err)
		}

		fmt.Printf("task status: %s\n", out.Status)
		if isFinished(out.Status) {
			printJSON("result", out)
			if out.Status == "FAILED" {
				exitf("task failed: %s %s", out.ErrorCode, out.ErrorMessage)
			}
			persistOutputs(ctx, client, out, outputDir)
			return
		}
	}
}

func persistOutputs(ctx context.Context, client *runninghub.Client, out *runninghub.QueryV2Response, outputDir string) {
	if out == nil {
		return
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		exitf("create output dir: %v", err)
	}
	resultPath := filepath.Join(outputDir, "result.json")
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		exitf("marshal result.json: %v", err)
	}
	if err := os.WriteFile(resultPath, b, 0o644); err != nil {
		exitf("write result.json: %v", err)
	}
	fmt.Printf("saved result JSON: %s\n", resultPath)
	files, err := client.DownloadTaskResults(ctx, out, outputDir)
	if err != nil {
		exitf("download outputs: %v", err)
	}
	for _, file := range files {
		fmt.Printf("downloaded output: %s\n", file.Path)
	}
}

func isFinished(status string) bool {
	switch status {
	case "SUCCESS", "FAILED", "CANCELLED":
		return true
	default:
		return false
	}
}

func printJSON(label string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		exitf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s\n", label, string(b))
}

func exitWithSDKError(err error) {
	var apiErr *runninghub.APIError
	if errors.As(err, &apiErr) {
		exitf("api error: code=%d message=%s", apiErr.Code, apiErr.Message)
	}

	var httpErr *runninghub.HTTPError
	if errors.As(err, &httpErr) {
		exitf("http error: status=%d body=%s", httpErr.StatusCode, httpErr.Body)
	}

	exitf("request failed: %v", err)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
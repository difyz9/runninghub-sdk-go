package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const defaultPayloadFile = "./examples/image_to_video/payload.example.json"

func main() {
	var (
		apiKey       = flag.String("api-key", os.Getenv("RUNNINGHUB_API_KEY"), "RunningHub API key; defaults to RUNNINGHUB_API_KEY")
		modelPath    = flag.String("path", "/openapi/v2/vidu/image-to-video-q3-pro-fast", "image-to-video model path")
		payload      = flag.String("payload", "", "JSON request body string")
		payloadFile  = flag.String("payload-file", defaultPayloadFile, "path to JSON request body file")
		imageURL     = flag.String("image-url", "", "public image URL to use as imageUrl")
		uploadFile   = flag.String("upload-file", "", "local image file to upload before calling the model")
		outputDir    = flag.String("output-dir", "./examples/image_to_video/output", "directory to save result.json and downloaded outputs")
		pollInterval = flag.Duration("poll-interval", 2*time.Second, "poll interval for QueryTaskV2")
		timeout      = flag.Duration("timeout", 5*time.Minute, "overall request timeout")
		previewOnly  = flag.Bool("preview", false, "only call PricePreview and exit")
	)
	flag.Parse()

	if *apiKey == "" {
		exitf("missing API key: set -api-key or RUNNINGHUB_API_KEY")
	}
	if *imageURL != "" && *uploadFile != "" {
		exitf("use either -image-url or -upload-file, not both")
	}

	reqBody, err := loadPayload(*payload, *payloadFile)
	if err != nil {
		exitf("load payload: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := runninghub.New(*apiKey)
	if err != nil {
		exitf("create client: %v", err)
	}

	switch {
	case *uploadFile != "":
		up, err := client.UploadBinaryFile(ctx, *uploadFile)
		if err != nil {
			exitf("upload file: %v", err)
		}
		reqBody["imageUrl"] = up.DownloadURL
		fmt.Printf("uploaded file URL assigned: imageUrl=%s\n", up.DownloadURL)
	case *imageURL != "":
		reqBody["imageUrl"] = *imageURL
	}

	if err := validateImageInput(reqBody); err != nil {
		exitf(err.Error())
	}

	runTask(ctx, client, *modelPath, reqBody, *previewOnly, *pollInterval, *outputDir)
}

func loadPayload(payload, payloadFile string) (map[string]any, error) {
	if payload != "" && payloadFile != "" {
		return nil, errors.New("use either -payload or -payload-file, not both")
	}

	var raw []byte
	switch {
	case payload != "":
		raw = []byte(payload)
	case payloadFile != "":
		b, err := os.ReadFile(payloadFile)
		if err != nil {
			return nil, err
		}
		raw = b
	default:
		return map[string]any{}, nil
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateImageInput(reqBody map[string]any) error {
	v, ok := reqBody["imageUrl"]
	if !ok {
		return errors.New("missing image input: use -upload-file, -image-url, or set imageUrl in the payload")
	}
	imageURL, _ := v.(string)
	if strings.TrimSpace(imageURL) == "" || strings.Contains(imageURL, "replace-with") {
		return errors.New("invalid imageUrl: use -upload-file, -image-url, or replace the placeholder in payload.example.json")
	}
	return nil
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
				exitTaskFailure(out)
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

func exitTaskFailure(out *runninghub.QueryV2Response) {
	if out == nil {
		exitf("task failed")
	}
	msg := fmt.Sprintf("task failed: %s %s", out.ErrorCode, out.ErrorMessage)
	if out.ErrorCode == "1013" {
		msg += "\nhint: 输入文件 URL 无法被模型读取。优先使用 -upload-file，或改成 RunningHub 上传返回的 download_url。"
	}
	exitf(msg)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
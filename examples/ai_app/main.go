package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const defaultPayloadFile = "./examples/ai_app/payload.example.json"

func main() {
	var (
		apiKey       = flag.String("api-key", os.Getenv("RUNNINGHUB_API_KEY"), "RunningHub API key; defaults to RUNNINGHUB_API_KEY")
		appID        = flag.String("app-id", "", "RunningHub AI App ID")
		payload      = flag.String("payload", "", "JSON request body string")
		payloadFile  = flag.String("payload-file", defaultPayloadFile, "path to JSON request body file")
		pollInterval = flag.Duration("poll-interval", 2*time.Second, "poll interval for QueryTaskV2")
		timeout      = flag.Duration("timeout", 5*time.Minute, "overall request timeout")
	)
	flag.Parse()

	if *apiKey == "" {
		exitf("missing API key: set -api-key or RUNNINGHUB_API_KEY")
	}
	if *appID == "" {
		exitf("missing app ID: set -app-id")
	}

	req, err := loadPayload(*payload, *payloadFile)
	if err != nil {
		exitf("load payload: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := runninghub.New(*apiKey)
	if err != nil {
		exitf("create client: %v", err)
	}

	resp, err := client.RunAIApp(ctx, *appID, req)
	if err != nil {
		exitWithSDKError(err)
	}
	printJSON("submit", resp)

	for {
		select {
		case <-ctx.Done():
			exitf("poll timeout: %v", ctx.Err())
		case <-time.After(*pollInterval):
		}

		out, err := client.QueryTaskV2(ctx, resp.TaskID)
		if err != nil {
			exitWithSDKError(err)
		}

		fmt.Printf("task status: %s\n", out.Status)
		if isFinished(out.Status) {
			printJSON("result", out)
			if out.Status == "FAILED" {
				exitf("task failed: %s %s", out.ErrorCode, out.ErrorMessage)
			}
			return
		}
	}
}

func loadPayload(payload, payloadFile string) (runninghub.RunAIAppRequest, error) {
	if payload != "" && payloadFile != "" {
		return runninghub.RunAIAppRequest{}, errors.New("use either -payload or -payload-file, not both")
	}

	var raw []byte
	switch {
	case payload != "":
		raw = []byte(payload)
	case payloadFile != "":
		b, err := os.ReadFile(payloadFile)
		if err != nil {
			return runninghub.RunAIAppRequest{}, err
		}
		raw = b
	default:
		return runninghub.RunAIAppRequest{}, nil
	}

	var out runninghub.RunAIAppRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		return runninghub.RunAIAppRequest{}, err
	}
	return out, nil
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
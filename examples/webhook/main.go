package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const defaultPayloadFile = "./examples/webhook/payload.example.json"

func main() {
	var (
		apiKey           = flag.String("api-key", os.Getenv("RUNNINGHUB_API_KEY"), "RunningHub API key; defaults to RUNNINGHUB_API_KEY")
		appID            = flag.String("app-id", "", "RunningHub AI App ID; when set together with -public-webhook-url, the example will submit a task")
		publicWebhookURL = flag.String("public-webhook-url", "", "public webhook callback URL exposed to RunningHub, for example https://example.com/webhook")
		listenAddr       = flag.String("listen", ":8080", "local HTTP listen address")
		webhookPath      = flag.String("path", "/webhook", "local HTTP path for receiving webhook callbacks")
		payload          = flag.String("payload", "", "JSON request body string used when submitting an AI App task")
		payloadFile      = flag.String("payload-file", defaultPayloadFile, "path to JSON request body file used when submitting an AI App task")
		logConsole       = flag.Bool("log-console", true, "write webhook logs to stdout")
		logFile          = flag.Bool("log-file", true, "write webhook logs to a file")
		logFilePath      = flag.String("log-file-path", "./examples/webhook/webhook.log", "path to the webhook log file")
		timeout          = flag.Duration("timeout", 15*time.Minute, "time to wait for the first webhook callback in submit mode")
	)
	flag.Parse()

	logger, closeLogger, err := newLogger(*logConsole, *logFile, *logFilePath)
	if err != nil {
		exitf("configure logger: %v", err)
	}
	defer closeLogger()

	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	callbackCh := make(chan runninghub.QueryV2Response, 1)

	router := gin.New()
	router.Use(requestLogger(logger), gin.Recovery())
	router.POST(*webhookPath, func(c *gin.Context) {
		raw, err := c.GetRawData()
		if err != nil {
			logger.Printf("read webhook body failed: %v", err)
			c.JSON(400, gin.H{"error": "read webhook body failed"})
			return
		}

		var callback runninghub.QueryV2Response
		if err := json.Unmarshal(raw, &callback); err != nil {
			logger.Printf("decode webhook body failed: %v", err)
			c.JSON(400, gin.H{"error": fmt.Sprintf("decode webhook body: %v", err)})
			return
		}

		logJSON(logger, "webhook", callback)
		logger.Printf("webhook raw body: %s", string(raw))

		select {
		case callbackCh <- callback:
		default:
		}

		c.JSON(200, gin.H{"ok": true})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"error": "method not allowed"})
	})

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("listening on http://127.0.0.1%s%s", *listenAddr, *webhookPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			exitf("webhook server failed: %v", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if *appID == "" || *publicWebhookURL == "" {
		logger.Printf("receiver mode only. expose http://127.0.0.1%s%s through a public tunnel and then rerun with -app-id and -public-webhook-url.", *listenAddr, *webhookPath)
		waitForInterrupt()
		return
	}

	if *apiKey == "" {
		exitf("missing API key: set -api-key or RUNNINGHUB_API_KEY")
	}

	req, err := loadPayload(*payload, *payloadFile)
	if err != nil {
		exitf("load payload: %v", err)
	}
	req.WebhookURL = *publicWebhookURL

	client, err := runninghub.New(*apiKey)
	if err != nil {
		exitf("create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := client.RunAIApp(ctx, *appID, req)
	if err != nil {
		exitWithSDKError(err)
	}
	logJSON(logger, "submit", resp)

	select {
	case callback := <-callbackCh:
		logger.Printf("received webhook for task %s with status %s", callback.TaskID, callback.Status)
	case <-ctx.Done():
		exitf("wait webhook timeout: %v", ctx.Err())
	}
}

func newLogger(enableConsole, enableFile bool, filePath string) (*log.Logger, func(), error) {
	outputs := make([]io.Writer, 0, 2)
	closers := make([]io.Closer, 0, 1)

	if enableConsole {
		outputs = append(outputs, os.Stdout)
	}
	if enableFile {
		if filePath == "" {
			return nil, nil, errors.New("log file path cannot be empty when -log-file is enabled")
		}
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return nil, nil, err
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, file)
		closers = append(closers, file)
	}

	writer := io.Writer(io.Discard)
	if len(outputs) == 1 {
		writer = outputs[0]
	}
	if len(outputs) > 1 {
		writer = io.MultiWriter(outputs...)
	}

	cleanup := func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}

	return log.New(writer, "[webhook] ", log.LstdFlags|log.Lmicroseconds), cleanup, nil
}

func requestLogger(logger *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Printf("request method=%s path=%s status=%d ip=%s latency=%s", c.Request.Method, c.FullPath(), c.Writer.Status(), c.ClientIP(), time.Since(start).String())
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

func logJSON(logger *log.Logger, label string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		exitf("marshal %s: %v", label, err)
	}
	logger.Printf("%s:\n%s", label, string(b))
}

func waitForInterrupt() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	<-sigCh
	fmt.Println("received interrupt, shutting down")
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
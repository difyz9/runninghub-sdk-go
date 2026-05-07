package main

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/goccy/go-yaml"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const defaultConfigFile = "./config.yaml"
const defaultPayloadFile = "./payload.example.json"

func main() {
	config, err := LoadConfig(defaultConfigFile)
	if err != nil {
		exitf("load config: %v", err)
	}

	timeout, err := config.TimeoutDuration()
	if err != nil {
		exitf("parse timeout from config: %v", err)
	}

	logger, closeLogger, err := newLogger(config.Log.Console, config.Log.File, config.Log.FilePath)
	if err != nil {
		exitf("configure logger: %v", err)
	}
	defer closeLogger()

	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	callbackCh := make(chan runninghub.QueryV2Response, 1)

	router := gin.New()
	router.Use(requestLogger(logger), gin.Recovery())
	router.POST(config.WebhookPath, func(c *gin.Context) {
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
		Addr:              config.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("using config file: %s", config.Path)
		logger.Printf("listening on http://127.0.0.1%s%s", config.ListenAddr, config.WebhookPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			exitf("webhook server failed: %v", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if config.AppID == "" || config.PublicWebhookURL == "" {
		logger.Printf("receiver mode only. expose http://127.0.0.1%s%s through a public tunnel and then rerun with -app-id and -public-webhook-url.", config.ListenAddr, config.WebhookPath)
		waitForInterrupt()
		return
	}

	if config.APIKey == "" {
		exitf("missing API key: set -api-key or RUNNINGHUB_API_KEY")
	}

	req, err := loadPayload(defaultPayloadFile)
	if err != nil {
		exitf("load payload: %v", err)
	}
	req.WebhookURL = config.PublicWebhookURL

	client, err := runninghub.New(config.APIKey)
	if err != nil {
		exitf("create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.RunAIApp(ctx, config.AppID, req)
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

type Config struct {
	Path             string    `yaml:"-"`
	APIKey           string    `yaml:"apiKey"`
	AppID            string    `yaml:"appId"`
	PublicWebhookURL string    `yaml:"publicWebhookUrl"`
	ListenAddr       string    `yaml:"listenAddr"`
	WebhookPath      string    `yaml:"webhookPath"`
	Timeout          string    `yaml:"timeout"`
	Log              LogConfig `yaml:"log"`
}

type LogConfig struct {
	Console  bool   `yaml:"console"`
	File     bool   `yaml:"file"`
	FilePath string `yaml:"filePath"`
}

func NewDefaultConfig() *Config {
	return &Config{
		ListenAddr: ":8080",
		WebhookPath: "/webhook",
		Timeout:    "15m",
		Log: LogConfig{
			Console:  true,
			File:     true,
			FilePath: "./webhook.log",
		},
	}
}

func (c *Config) TimeoutDuration() (time.Duration, error) {
	if c == nil {
		return 0, errors.New("config cannot be nil")
	}
	if c.Timeout == "" {
		return 15 * time.Minute, nil
	}
	return time.ParseDuration(c.Timeout)
}

func LoadConfig(configFile string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("config file path cannot be empty")
	}

	if _, err := os.Stat(configFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		config := NewDefaultConfig()
		config.Path = configFile
		if err := SaveConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	config := NewDefaultConfig()
	if err := yaml.Unmarshal(raw, config); err != nil {
		return nil, err
	}
	config.Path = configFile
	return config, nil
}

func SaveConfig(config *Config) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}
	if config.Path == "" {
		return errors.New("config path cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(config.Path), 0o755); err != nil {
		return err
	}

	raw, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(config.Path, raw, 0o644)
}


func loadPayload(payloadFile string) (runninghub.RunAIAppRequest, error) {
	raw, err := os.ReadFile(payloadFile)
	if err != nil {
		return runninghub.RunAIAppRequest{}, err
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
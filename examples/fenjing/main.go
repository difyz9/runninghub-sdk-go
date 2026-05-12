package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

const (
	defaultWorkflowID   = "2013908081847046145"
	defaultDeepSeekURL  = "https://api.deepseek.com"
	defaultDeepSeekPath = "/chat/completions"
	defaultModel        = "deepseek-chat"
	defaultIdea         = "主角深夜误入荒废寺庙，在恐惧中逐步发现庙内异样，气氛持续升级。"
	defaultStyle        = "国漫分镜，电影感构图，悬疑惊悚，强氛围光影"
	defaultCharacters   = "主角：年轻男性，谨慎、紧张、易受惊。"
	defaultNegative     = "低质量，模糊，错误透视，人物崩坏，手部异常，额外肢体，画面拥挤，构图混乱，风格漂移"
)

var defaultPromptNodeIDs = []string{"343", "411"}

const defaultSystemPrompt = `You are a senior storyboard designer for comic and anime previsualization.
Return valid JSON only.

Create a production-ready storyboard prompt payload. The JSON schema must be:
{
  "title": "short title",
  "global_style": "one concise global style line in Chinese",
  "story_summary": "2-4 sentences summarizing the scene progression in Chinese",
  "storyboard_prompt": "multi-line Chinese storyboard prompt using Slot format",
  "negative_prompt": "one concise negative prompt in Chinese"
}

Rules:
- Return JSON only, no markdown fences.
- storyboard_prompt must use this exact structure:
  Slot 1 (缓冲帧):
  ...

  Slot 2 (剧情帧):
  ...

  Slot 3 (剧情帧):
  ...
- There must be a blank line between every Slot block.
- Include exactly 6 slots total: Slot 1 is a pure black buffer frame, Slots 2-6 are story frames.
- The storyboard_prompt must be directly usable as input for a storyboard image workflow.
- Each story frame should mention environment, subject, action, mood, and camera shot.
`

type config struct {
	APIKey         string
	DeepSeekAPIKey string
	WorkflowID     string
	PromptNodeIDs  []string
	PollInterval   time.Duration
	Timeout        time.Duration
	Idea           string
	Style          string
	Characters     string
	Model          string
	PromptOutput   string
	DownloadRoot   string
	DeepSeekBase   string
	HTTPTimeout    time.Duration
}

type promptPayload struct {
	Title            string `json:"title"`
	GlobalStyle      string `json:"global_style"`
	StorySummary     string `json:"story_summary"`
	StoryboardPrompt string `json:"storyboard_prompt"`
	NegativePrompt   string `json:"negative_prompt"`
}

type deepSeekRequest struct {
	Model          string              `json:"model"`
	ResponseFormat deepSeekRespFormat  `json:"response_format"`
	Messages       []deepSeekMessage   `json:"messages"`
	Temperature    float64             `json:"temperature"`
}

type deepSeekRespFormat struct {
	Type string `json:"type"`
}

type deepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	if err := run(); err != nil {
		exitWithSDKError(err)
	}
}

func run() error {
	repoRoot, scriptDir, err := locatePaths()
	if err != nil {
		return err
	}

	loadedEnvPaths, err := bootstrapEnv(scriptDir, repoRoot)
	if err != nil {
		return err
	}
	printLoadedEnv(loadedEnvPaths)

	config, err := loadConfig(repoRoot, scriptDir)
	if err != nil {
		return err
	}

	printSection("1. Generate Storyboard Prompt JSON")
	prompt, err := generatePrompt(config)
	if err != nil {
		return err
	}
	if err := saveJSON(config.PromptOutput, prompt); err != nil {
		return fmt.Errorf("save prompt json: %w", err)
	}
	printPromptResult(prompt, config.PromptOutput)

	client, err := runninghub.CreateClient(config.APIKey)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	promptNodeIDs, err := resolvePromptNodeIDs(ctx, client, config.WorkflowID, config.PromptNodeIDs)
	if err != nil {
		return err
	}

	modifier := runninghub.ModifyNodes()
	for _, nodeID := range promptNodeIDs {
		modifier.Set(nodeID, "prompt", prompt.StoryboardPrompt)
	}

	printSection("2. Request Preview")
	fmt.Println("sdk_method: RunningHubClient.RunWithModifier")
	fmt.Println("workflow_id:", config.WorkflowID)
	fmt.Println("node_count:", len(promptNodeIDs))
	for _, nodeID := range promptNodeIDs {
		fmt.Printf("nodeId=%s | fieldName=prompt | fieldValue=%s\n", nodeID, prompt.StoryboardPrompt)
	}

	addMetadata := true
	usePersonalQueue := false

	printSection("3. Submit Task")
	task, err := client.RunWithModifier(ctx, config.WorkflowID, modifier, &runninghub.RunOptions{
		AddMetadata:      &addMetadata,
		InstanceType:     "default",
		UsePersonalQueue: &usePersonalQueue,
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(task.TaskID) == "" {
		return errors.New("submit failed: task id is empty")
	}
	fmt.Println("task_id:", task.TaskID)
	fmt.Println("task_status:", task.Status)
	fmt.Println("client_id:", task.ClientID)
	fmt.Println("prompt_tips:", task.PromptTips)

	var lastStatus runninghub.TaskStatus
	printSection("4. Wait For Completion")
	result, err := client.WaitForCompletion(ctx, task.TaskID, &runninghub.WaitForCompletionOptions{
		PollInterval: config.PollInterval,
		Timeout:      config.Timeout,
		OnStatusChange: func(status runninghub.TaskStatus) {
			if status == lastStatus {
				return
			}
			fmt.Printf("status -> %s\n", status)
			lastStatus = status
		},
	})
	if err != nil {
		return err
	}
	if result.Status == string(runninghub.TaskStatusFailed) {
		return fmt.Errorf("task failed: %s %s", result.ErrorCode, result.ErrorMessage)
	}

	printSection("5. Results")
	for index, item := range result.Results {
		fmt.Printf("[%d] file_type=%s\n", index+1, item.OutputType)
		fmt.Printf("    node_id=%s\n", item.NodeID)
		fmt.Printf("    file_url=%s\n", item.URL)
	}

	downloadDir := filepath.Join(config.DownloadRoot, fmt.Sprintf("fenjing_%s", task.TaskID))
	printSection("6. Download Outputs")
	if err := saveJSON(filepath.Join(downloadDir, "result.json"), result); err != nil {
		return fmt.Errorf("save result json: %w", err)
	}
	downloadedFiles, err := client.DownloadTaskResults(ctx, result, downloadDir)
	if err != nil {
		return fmt.Errorf("download outputs: %w", err)
	}
	fmt.Println("download_dir:", downloadDir)
	for _, item := range downloadedFiles {
		fmt.Println("saved:", item.Path)
	}

	printSection("Done")
	fmt.Println("DeepSeek -> fenjing workflow finished successfully.")
	return nil
}

func loadConfig(repoRoot, scriptDir string) (*config, error) {
	workflowID := getEnv("RUNNINGHUB_FENJING_WORKFLOW_ID", defaultWorkflowID)
	pollInterval, err := parseDurationSecondsFirst(getEnv("RUNNINGHUB_FENJING_POLL_INTERVAL", "5"), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("parse RUNNINGHUB_FENJING_POLL_INTERVAL: %w", err)
	}
	timeout, err := parseDurationSecondsFirst(getEnv("RUNNINGHUB_FENJING_TIMEOUT", "1800"), 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("parse RUNNINGHUB_FENJING_TIMEOUT: %w", err)
	}
	httpTimeout, err := parseDurationSecondsFirst(getEnv("RUNNINGHUB_FENJING_HTTP_TIMEOUT", "60"), 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("parse RUNNINGHUB_FENJING_HTTP_TIMEOUT: %w", err)
	}

	defaultPromptOutput := filepath.Join(scriptDir, "outputs", "deepseek_fenjing_storyboard_prompt.json")
	defaultDownloadRoot := filepath.Join(scriptDir, "downloads")
	defaultDeepSeekEnv := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if defaultDeepSeekEnv == "" {
		defaultDeepSeekEnv = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}

	config := &config{
		APIKey:         strings.TrimSpace(os.Getenv("RUNNINGHUB_API_KEY")),
		DeepSeekAPIKey: defaultDeepSeekEnv,
		WorkflowID:     workflowID,
		PromptNodeIDs:  splitCSV(os.Getenv("RUNNINGHUB_FENJING_PROMPT_NODE_IDS")),
		PollInterval:   pollInterval,
		Timeout:        timeout,
		Idea:           getEnv("RUNNINGHUB_DEEPSEEK_FENJING_IDEA", defaultIdea),
		Style:          getEnv("RUNNINGHUB_DEEPSEEK_FENJING_STYLE", defaultStyle),
		Characters:     getEnv("RUNNINGHUB_DEEPSEEK_FENJING_CHARACTERS", defaultCharacters),
		Model:          getEnv("RUNNINGHUB_DEEPSEEK_FENJING_MODEL", defaultModel),
		PromptOutput:   getEnv("RUNNINGHUB_DEEPSEEK_FENJING_OUTPUT", defaultPromptOutput),
		DownloadRoot:   getEnv("RUNNINGHUB_FENJING_DOWNLOAD_ROOT", defaultDownloadRoot),
		DeepSeekBase:   getEnv("RUNNINGHUB_DEEPSEEK_BASE_URL", defaultDeepSeekURL),
		HTTPTimeout:    httpTimeout,
	}

	apiKeyFlag := flag.String("api-key", config.APIKey, "RunningHub API key; defaults to RUNNINGHUB_API_KEY")
	deepSeekKeyFlag := flag.String("deepseek-api-key", config.DeepSeekAPIKey, "DeepSeek API key; defaults to DEEPSEEK_API_KEY or OPENAI_API_KEY")
	workflowIDFlag := flag.String("workflow-id", config.WorkflowID, "RunningHub workflow ID")
	promptNodesFlag := flag.String("prompt-node-ids", strings.Join(config.PromptNodeIDs, ","), "comma-separated CR Prompt Text node IDs")
	pollIntervalFlag := flag.Duration("poll-interval", config.PollInterval, "poll interval for task status")
	timeoutFlag := flag.Duration("timeout", config.Timeout, "overall timeout for DeepSeek and RunningHub requests")
	ideaFlag := flag.String("idea", config.Idea, "high-level storyboard idea")
	styleFlag := flag.String("style", config.Style, "visual style guidance")
	charactersFlag := flag.String("characters", config.Characters, "character description")
	modelFlag := flag.String("model", config.Model, "DeepSeek model name")
	promptOutputFlag := flag.String("output", config.PromptOutput, "path to save generated storyboard prompt JSON")
	downloadRootFlag := flag.String("download-root", config.DownloadRoot, "directory for downloaded workflow outputs")
	deepSeekBaseFlag := flag.String("deepseek-base-url", config.DeepSeekBase, "DeepSeek API base URL")
	httpTimeoutFlag := flag.Duration("http-timeout", config.HTTPTimeout, "timeout for DeepSeek HTTP request")
	flag.Parse()

	config.APIKey = strings.TrimSpace(*apiKeyFlag)
	config.DeepSeekAPIKey = strings.TrimSpace(*deepSeekKeyFlag)
	config.WorkflowID = strings.TrimSpace(*workflowIDFlag)
	config.PromptNodeIDs = splitCSV(*promptNodesFlag)
	config.PollInterval = *pollIntervalFlag
	config.Timeout = *timeoutFlag
	config.Idea = strings.TrimSpace(*ideaFlag)
	config.Style = strings.TrimSpace(*styleFlag)
	config.Characters = strings.TrimSpace(*charactersFlag)
	config.Model = strings.TrimSpace(*modelFlag)
	config.PromptOutput = resolvePath(repoRoot, scriptDir, *promptOutputFlag)
	config.DownloadRoot = resolvePath(repoRoot, scriptDir, *downloadRootFlag)
	config.DeepSeekBase = strings.TrimSpace(*deepSeekBaseFlag)
	config.HTTPTimeout = *httpTimeoutFlag

	if config.APIKey == "" {
		return nil, errors.New("missing API key: set RUNNINGHUB_API_KEY or use -api-key")
	}
	if config.DeepSeekAPIKey == "" {
		return nil, errors.New("missing DeepSeek API key: set DEEPSEEK_API_KEY or OPENAI_API_KEY, or use -deepseek-api-key")
	}
	if config.WorkflowID == "" {
		return nil, errors.New("missing workflow ID")
	}
	if config.Model == "" {
		return nil, errors.New("missing DeepSeek model")
	}
	return config, nil
}

func generatePrompt(config *config) (*promptPayload, error) {
	requestBody := deepSeekRequest{
		Model:          config.Model,
		ResponseFormat: deepSeekRespFormat{Type: "json_object"},
		Messages: []deepSeekMessage{
			{Role: "system", Content: defaultSystemPrompt},
			{Role: "user", Content: buildUserPrompt(config)},
		},
		Temperature: 0.9,
	}

	body, err := marshalJSON(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal DeepSeek request: %w", err)
	}

	url := strings.TrimRight(config.DeepSeekBase, "/") + defaultDeepSeekPath
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create DeepSeek request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.DeepSeekAPIKey)
	req.Header.Set("Content-Type", "application/json")

	hc := &http.Client{Timeout: config.HTTPTimeout}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request DeepSeek: %w", err)
	}
	defer resp.Body.Close()

	var rawResponse deepSeekResponse
	if resp.StatusCode != http.StatusOK {
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return nil, fmt.Errorf("DeepSeek request failed: status=%d body=%v", resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("decode DeepSeek response: %w", err)
	}
	if len(rawResponse.Choices) == 0 {
		return nil, errors.New("DeepSeek response missing choices")
	}

	content := strings.TrimSpace(rawResponse.Choices[0].Message.Content)
	if content == "" {
		return nil, errors.New("DeepSeek response content is empty")
	}

	var prompt promptPayload
	if err := json.Unmarshal([]byte(content), &prompt); err != nil {
		return nil, fmt.Errorf("DeepSeek returned invalid JSON: %w", err)
	}
	if err := validatePromptPayload(&prompt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(prompt.NegativePrompt) == "" {
		prompt.NegativePrompt = defaultNegative
	}
	return &prompt, nil
}

func validatePromptPayload(prompt *promptPayload) error {
	if prompt == nil {
		return errors.New("nil prompt payload")
	}
	missing := make([]string, 0, 4)
	if strings.TrimSpace(prompt.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(prompt.GlobalStyle) == "" {
		missing = append(missing, "global_style")
	}
	if strings.TrimSpace(prompt.StorySummary) == "" {
		missing = append(missing, "story_summary")
	}
	if strings.TrimSpace(prompt.StoryboardPrompt) == "" {
		missing = append(missing, "storyboard_prompt")
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("DeepSeek response missing keys: %s", strings.Join(missing, ", "))
	}
	return nil
}

func resolvePromptNodeIDs(ctx context.Context, client *runninghub.Client, workflowID string, configured []string) ([]string, error) {
	if len(configured) > 0 {
		return configured, nil
	}

	workflowJSON, err := client.GetWorkflowJSON(ctx, workflowID)
	if err != nil {
		return defaultPromptNodeIDs, nil
	}

	resolved := make([]string, 0, len(defaultPromptNodeIDs))
	for nodeID, rawNode := range workflowJSON {
		nodeMap, ok := rawNode.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(toString(nodeMap["class_type"])) != "CR Prompt Text" {
			continue
		}
		inputs, ok := nodeMap["inputs"].(map[string]any)
		if !ok {
			continue
		}
		if strings.Contains(strings.TrimSpace(toString(inputs["prompt"])), "Slot 1") {
			resolved = append(resolved, nodeID)
		}
	}

	if len(resolved) == 0 {
		return defaultPromptNodeIDs, nil
	}
	sort.Strings(resolved)
	return resolved, nil
}

func locatePaths() (repoRoot string, scriptDir string, err error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", errors.New("cannot resolve current file path")
	}
	scriptDir = filepath.Dir(currentFile)
	repoRoot = scriptDir
	for {
		if _, statErr := os.Stat(filepath.Join(repoRoot, "go.mod")); statErr == nil {
			return repoRoot, scriptDir, nil
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			return "", "", errors.New("cannot locate repository root containing go.mod")
		}
		repoRoot = parent
	}
}

func bootstrapEnv(scriptDir, repoRoot string) ([]string, error) {
	paths := []string{
		filepath.Join(scriptDir, ".env"),
		filepath.Join(repoRoot, ".env"),
	}
	loaded := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if err := loadEnvFile(path); err != nil {
			return nil, fmt.Errorf("load env file %s: %w", path, err)
		}
		loaded = append(loaded, path)
	}
	return loaded, nil
}

func loadEnvFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) >= 2 {
			value = value[1 : len(value)-1]
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
			value = value[1 : len(value)-1]
		}
		if key == "" || value == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildUserPrompt(config *config) string {
	return fmt.Sprintf(
		"Generate a storyboard payload with these constraints:\n- Core story idea: %s\n- Visual style: %s\n- Characters: %s\n- Output language: Chinese\n- The storyboard_prompt must be directly usable for a RunningHub storyboard workflow\n- Keep the scene progression coherent and cinematic\n",
		config.Idea,
		config.Style,
		config.Characters,
	)
}

func saveJSON(path string, value any) error {
	b, err := marshalJSON(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func marshalJSON(value any) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func printPromptResult(prompt *promptPayload, outputPath string) {
	fmt.Println("title:", prompt.Title)
	fmt.Println()
	fmt.Println("global_style:")
	fmt.Println(prompt.GlobalStyle)
	fmt.Println()
	fmt.Println("story_summary:")
	fmt.Println(prompt.StorySummary)
	fmt.Println()
	fmt.Println("storyboard_prompt:")
	fmt.Println(prompt.StoryboardPrompt)
	fmt.Println()
	fmt.Println("negative_prompt:")
	fmt.Println(prompt.NegativePrompt)
	fmt.Println()
	fmt.Println("saved prompt json to:", outputPath)
}

func printLoadedEnv(paths []string) {
	if len(paths) == 0 {
		fmt.Println("loaded_env: <none>")
		return
	}
	fmt.Println("loaded_env:", paths[0])
	for _, path := range paths[1:] {
		fmt.Println("loaded_env_extra:", path)
	}
}

func parseDurationSecondsFirst(raw string, fallback time.Duration) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	if strings.ContainsAny(raw, "hms") {
		return time.ParseDuration(raw)
	}
	seconds, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 0, err
	}
	return seconds, nil
}

func splitCSV(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func resolvePath(repoRoot, scriptDir, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	if strings.HasPrefix(raw, "./examples/") || strings.HasPrefix(raw, "examples/") {
		return filepath.Join(repoRoot, raw)
	}
	return filepath.Join(scriptDir, raw)
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}

func printSection(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

func exitWithSDKError(err error) {
	var apiErr *runninghub.APIError
	if errors.As(err, &apiErr) {
		fmt.Fprintf(os.Stderr, "api error: code=%d message=%s\n", apiErr.Code, apiErr.Message)
		os.Exit(1)
	}

	var httpErr *runninghub.HTTPError
	if errors.As(err, &httpErr) {
		fmt.Fprintf(os.Stderr, "http error: status=%d body=%s\n", httpErr.StatusCode, httpErr.Body)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
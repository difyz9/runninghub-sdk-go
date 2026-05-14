# runninghub-sdk-go

RunningHub Go SDK。

该仓库提供一个面向 RunningHub API 的 Go 封装，覆盖当前仓库内已实现的 OpenAPI v2、ComfyUI legacy 接口、标准模型通用调用、任务查询与结果下载能力，并保留一组兼容层方法，方便从已有 Python / 历史 Go 用法迁移。

## 概览

- 当前推荐接口面：`RunWorkflow`、`RunAIApp`、`RunStandardModel`、`QueryTaskV2`、`WaitForCompletion`
- 兼容接口面：`Run`、`RunWithModifier`、`RunAIAppWithModifier`、`RunLegacyAIApp`、`RunLegacyComfyTask`
- 上传与下载：`UploadBinaryFile`、`UploadBinaryReader`、`DownloadTaskResults`
- 查询与调试：`GetAccountStatus`、`GetQueueStatus`、`ListAPIKeys`、`GetWebhookDetail`
- 低层扩展：`DoJSON` 可直接调用未封装接口

## 设计目标

- 以 `Client` 为中心，统一鉴权、超时、请求与响应处理
- 优先为稳定通用场景提供 typed wrapper
- 对标准模型 API 使用通用调用器，避免为每个模型端点维护单独 wrapper
- 保留兼容层，减少历史接入代码迁移成本

## 依赖与环境

- Go 版本：见 [go.mod](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/go.mod)
- 运行时核心通信基于 Go 标准库 `net/http`
- YAML 配置辅助能力依赖 `github.com/goccy/go-yaml`，用于 [runninghub/config_yaml.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub/config_yaml.go)

安装：

```bash
go get github.com/difyz9/runninghub-sdk-go
```

## 仓库结构

- [runninghub](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub)：SDK 主包
- [examples/ai_app_flux_krea_text_to_image](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/examples/ai_app_flux_krea_text_to_image)：当前仓库内可直接运行的示例
- [downloads](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/downloads)：示例下载结果目录

## 命名约定

当前 README 采用以下约定来区分推荐 API 与兼容 API：

- 当前接口优先使用语义明确的 v2 / canonical 名称：`RunWorkflow`、`RunAIApp`、`ListAPIKeys`、`GetQueueStatus`
- 旧文档对应接口统一使用 `RunLegacy...` 风格：`RunLegacyAIApp`、`RunLegacyComfyTaskSimple`、`RunLegacyComfyTask`
- 历史名称仍保留兼容，但新代码不建议继续作为首选入口，例如：`CreateAIAppTask`、`CreateComfyTaskAdvanced`、`APIKeyList`、`QueueStatus`、`AccountStatus`
- 等待任务结果时，默认优先使用 `WaitForCompletion`；`WaitForTask` 保留为轻量轮询方法

## 快速开始

### 1. 初始化 Client

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/difyz9/runninghub-sdk-go/runninghub"
)

func main() {
	client, err := runninghub.New(
		os.Getenv("RUNNINGHUB_API_KEY"),
		runninghub.WithUserAgent("myapp/1.0"),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queue, err := client.GetQueueStatus(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("queued=", queue.QueuedCount, "running=", queue.RunningCount)
}
```

鉴权方式：SDK 默认自动注入 `Authorization: Bearer <API_KEY>`。

### 2. AI App 提交与等待结果

```go
usePersonalQueue := false

resp, err := client.RunAIApp(ctx, "2016796569449795585", runninghub.RunAIAppRequest{
	NodeInfoList: []runninghub.AIAppNodeInfo{
		{
			NodeID:      "41",
			FieldName:   "select",
			FieldValue:  "7",
			Description: "设置比例",
		},
		{
			NodeID:      "50",
			FieldName:   "text",
			FieldValue:  "润色这段话",
			Description: "输入文本",
		},
	},
	InstanceType:     "default",
	UsePersonalQueue: &usePersonalQueue,
})
if err != nil {
	panic(err)
}

result, err := client.WaitForCompletion(ctx, resp.TaskID, &runninghub.WaitForCompletionOptions{
	PollInterval: 2 * time.Second,
})
if err != nil {
	panic(err)
}

fmt.Println(result.Status, result.Results)
```

### 3. Workflow 提交与等待结果

```go
addMetadata := true
usePersonalQueue := false

resp, err := client.RunWorkflow(ctx, "2037060865681264641", runninghub.RunWorkflowRequest{
	AddMetadata:      &addMetadata,
	NodeInfoList:     []runninghub.WorkflowNodeInfo{},
	InstanceType:     "default",
	UsePersonalQueue: &usePersonalQueue,
})
if err != nil {
	panic(err)
}

result, err := client.WaitForCompletion(ctx, resp.TaskID, nil)
if err != nil {
	panic(err)
}

fmt.Println(result.Status)
```

### 4. 标准模型 API 通用调用

```go
task, err := client.RunStandardModel(ctx, "/openapi/v2/rhart-image-n-pro-official/edit", map[string]any{
	"imageUrls":   []string{"https://example.com/input.png"},
	"prompt":      "海边沙滩变成夏日祭典现场，风格欢乐卡通，色彩缤纷。",
	"resolution":  "1k",
	"aspectRatio": "3:4",
})
if err != nil {
	panic(err)
}

result, err := client.WaitForCompletion(ctx, task.TaskID, &runninghub.WaitForCompletionOptions{
	PollInterval: 2 * time.Second,
})
if err != nil {
	panic(err)
}

fmt.Println(result.Results)
```

`RunStandardModel` 支持两种 path 形式：

- 完整路径：`/openapi/v2/vidu/image-to-video-q3-pro-fast`
- v2 相对路径：`vidu/image-to-video-q3-pro-fast`

### 5. 上传本地文件并下载结果

```go
upload, err := client.UploadBinaryFile(ctx, "./input.png")
if err != nil {
	panic(err)
}

task, err := client.RunStandardModel(ctx, "rhart-text-g-25-pro-cv/image-to-text", map[string]any{
	"prompt":   "请描述图片内容",
	"imageUrl": upload.DownloadURL,
})
if err != nil {
	panic(err)
}

result, err := client.WaitForCompletion(ctx, task.TaskID, nil)
if err != nil {
	panic(err)
}

files, err := client.DownloadTaskResults(ctx, result, "./output")
if err != nil {
	panic(err)
}

fmt.Println(files)
```

上传接口返回的 `download_url` 适用于部分模型或 AI App 入参；是否可直接作为目标字段值，以对应 RunningHub 文档为准。

## 当前仓库示例

当前仓库内可直接运行的示例为 [examples/ai_app_flux_krea_text_to_image/main.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/examples/ai_app_flux_krea_text_to_image/main.go)。

该示例展示了完整流程：

- 构造 `RunAIAppRequest`
- 提交 AI App 任务
- 使用 `WaitForCompletion` 轮询
- 下载 `results` 中返回的文件

运行前需要设置：

```bash
export RUNNINGHUB_API_KEY="你的 API Key"
```

运行示例：

```bash
go run ./examples/ai_app_flux_krea_text_to_image
```

可用参数以示例实现为准，核心参数包括：

- `-app-id`
- `-prompt`
- `-aspect-ratio`
- `-batch-size`
- `-instance-type`
- `-use-personal-queue`
- `-poll-interval`
- `-timeout`
- `-output-dir`

## 核心概念

### Client

[runninghub/client.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub/client.go) 中的 `Client` 负责：

- 维护 `baseURL`、`apiKey`、`httpClient`
- 注入认证头、User-Agent、Host Override
- 统一 JSON 请求与响应解码
- 统一 HTTP 错误与业务错误边界

### NodeModifier

[runninghub/node_modifier.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub/node_modifier.go) 提供一组用于拼装 `nodeInfoList` 的便捷方法，适合工作流和 AI App 参数替换场景。

常见用法：

```go
modifier := runninghub.
	ModifyNodes().
	Text("6", "hello world").
	Seed("3", 12345).
	Steps("3", 28)
```

### WaitForCompletion

[runninghub/compatibility.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub/compatibility.go) 中的 `WaitForCompletion` 是当前推荐等待接口，支持：

- 自定义轮询间隔
- 统一超时控制
- 状态变化回调 `OnStatusChange`

### YAML 配置辅助

[runninghub/config_yaml.go](/Users/apple/opt/difyz_0329/0509/runninghub-sdk-go/runninghub/config_yaml.go) 提供：

- `LoadYAMLConfig[T](configFile)`
- `SaveYAMLConfig(configFile, config)`

适合示例程序或业务服务加载本地 YAML 配置文件。

## API 分类索引

### OpenAPI v2

- `RunWorkflow(ctx, workflowIDOrPath, req)`
- `RunAIApp(ctx, appIDOrPath, req)`
- `RunStandardModel(ctx, path, req)`
- `QueryTaskV2(ctx, taskID)`
- `PricePreview(ctx, modelPath, req)`
- `ListAPIKeys(ctx)`
- `GetQueueStatus(ctx)`
- `GetAccountStatus(ctx)`
- `ListPublicResources(ctx, req)`

### 上传与下载

- `UploadBinaryFile(ctx, filePath)`
- `UploadBinaryReader(ctx, r, filename)`
- `GetLoraUploadURL(ctx, loraName, md5Hex)`
- `DownloadFile(ctx, fileURL, destPath)`
- `DownloadTaskResults(ctx, task, outputDir)`

### 任务等待与查询

- `WaitForCompletion(ctx, taskID, options)`
- `WaitForTask(ctx, taskID, pollInterval)`
- `WaitForQueryV2Completion(ctx, taskID, options)`

### Legacy / 兼容接口

- `Run(ctx, workflowID, options)`
- `RunWithModifier(ctx, workflowID, modifier, options)`
- `RunAIAppWithOptions(ctx, appID, options)`
- `RunAIAppWithModifier(ctx, appID, modifier, options)`
- `RunLegacyAIApp(ctx, req)`
- `RunLegacyComfyTaskSimple(ctx, req)`
- `RunLegacyComfyTask(ctx, req)`
- `GetAIAppAPICallDemo(ctx, webappID)`
- `GetWebhookDetail(ctx, taskID)`
- `RetryWebhook(ctx, webhookID, webhookURL)`
- `GetWorkflowJSONPrompt(ctx, workflowID)`
- `GetWorkflowJSON(ctx, workflowID)`
- `CancelComfyTask(ctx, taskID)`

### 兼容别名

以下方法仍可用，但不建议作为新代码的首选入口：

- `CreateClient`
- `NewClient`
- `CreateAIAppTask`
- `CreateComfyTaskSimpleWithRequest`
- `CreateComfyTaskAdvanced`
- `APIKeyList`
- `QueueStatus`
- `AccountStatus`

## 错误处理

SDK 主要暴露两类错误：

- `*runninghub.HTTPError`：HTTP 状态码不是 200
- `*runninghub.APIError`：业务层 `code != 0` 或模型响应返回错误码

示例：

```go
if err != nil {
	var apiErr *runninghub.APIError
	if errors.As(err, &apiErr) {
		fmt.Println("api error:", apiErr.Code, apiErr.Message)
	}

	var httpErr *runninghub.HTTPError
	if errors.As(err, &httpErr) {
		fmt.Println("http error:", httpErr.StatusCode, httpErr.Body)
	}
}
```

## Client Options

创建 `Client` 时可传入如下 Option：

- `WithBaseURL(raw)`：自定义服务地址
- `WithHTTPClient(hc)`：自定义 `http.Client`
- `WithHostOverride(host)`：设置 `req.Host`
- `WithUserAgent(ua)`：设置 User-Agent
- `WithMaxBodyBytes(n)`：限制响应体读取大小，默认 10 MiB

示例：

```go
hc := &http.Client{Timeout: 120 * time.Second}

client, err := runninghub.New(
	apiKey,
	runninghub.WithHTTPClient(hc),
	runninghub.WithUserAgent("my-service/1.0"),
)
```

## 调用未封装接口

如果目标接口还没有 typed wrapper，可以直接使用 `DoJSON`：

```go
var out any

err := client.DoJSON(
	ctx,
	http.MethodPost,
	"/openapi/v2/some/endpoint",
	nil,
	nil,
	map[string]any{"k": "v"},
	&out,
)
```

这是扩展新接口时最稳妥的过渡方式；如果某个接口会长期使用，建议再补充 typed wrapper 和测试。

## FAQ

### 为什么标准模型 API 返回的是 task，而不是最终结果？

RunningHub 的标准模型接口通常是异步任务模型：提交时返回 `taskId`、`status` 等字段，最终结果需要通过 `QueryTaskV2` 或 `WaitForCompletion` 获取。

### `WaitForCompletion` 和 `WaitForTask` 应该怎么选？

- `WaitForCompletion`：推荐默认使用，支持超时与状态回调
- `WaitForTask`：适合极简脚本或你只想给出轮询间隔时使用

### `Client` 是否可以并发复用？

可以。`Client` 初始化后主要持有只读配置和底层 `http.Client`，通常适合作为服务级单例复用。

### 上传返回的 URL 有效期多久？

该时效由 RunningHub 平台规则决定。当前文档与历史说明通常将上传后的 `download_url` 视为短时有效资源，结果文件也建议在任务完成后尽快下载或转存。

## 测试

```bash
go test ./...
```

## 版本与兼容性说明

- 当前 README 以仓库现状为准，而不是以外部文档目录为准
- 标准模型 API 采用通用调用器封装，因此模型端点扩展速度快，但具体请求体字段仍应以 RunningHub 官方文档为准
- legacy 接口主要用于兼容已有调用方式，不建议新项目优先采用

## License

仓库当前未附带独立 License 文件时，请以仓库实际许可声明为准；如需明确分发许可，建议补充标准 License 文件。

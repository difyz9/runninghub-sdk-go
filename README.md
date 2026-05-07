# runninghub-sdk-go 

一个纯标准库（`net/http`）实现的 RunningHub Go SDK，用于：

- RunningHub OpenAPI v2（上传、队列、apikey、资源等）
- ComfyUI 工作流相关接口（简易任务、取消、工作流 JSON）
- **标准模型 API（大量模型端点）**：提供通用调用器，避免为每个模型单独写 wrapper
- **模型 API 价格预览**：与标准模型 API 路径/入参保持一致

> 鉴权方式：默认自动注入 `Authorization: Bearer <API_KEY>`。

## 特性

- 仅依赖 Go 标准库（无第三方依赖）
- `Client` + Options 模式，便于接入现有项目（自定义 baseURL/HTTPClient/UA/响应体大小）
- Typed wrappers 覆盖 RunningHub 常用接口
- **标准模型 API** 用通用调用器覆盖大量模型端点（只需要拿到文档里的 path 和请求体）
- 统一的错误类型：`HTTPError`（HTTP 非 200）与 `APIError`（业务 code != 0）

## 安装

```bash
go get github.com/difyz9/runninghub-sdk-go
```

## 目录结构

- `runninghub/`：SDK 主包（推荐直接使用）

## 快速开始

### 0) 可直接运行的完整案例

仓库内已经提供可直接执行的示例程序：

- `examples/workflow/main.go`
- `examples/workflow/payload.example.json`
- `examples/ai_app/main.go`
- `examples/ai_app/payload.example.json`
- `examples/text_to_image/main.go`
- `examples/text_to_image/payload.example.json`
- `examples/image_edit/main.go`
- `examples/image_edit/payload.example.json`
- `examples/image_to_video/main.go`
- `examples/image_to_video/payload.example.json`
- `examples/text_to_video/main.go`
- `examples/text_to_video/payload.example.json`
- `examples/webhook/main.go`
- `examples/webhook/payload.example.json`

先设置 API Key：

```bash
# PowerShell
$env:RUNNINGHUB_API_KEY="你的 API Key"

# bash
export RUNNINGHUB_API_KEY="你的 API Key"
```

标准模型 API 目前仅支持企业级-共享 API Key。

#### 示例 A：工作流

工作流示例会提交 `/openapi/v2/run/workflow/{workflowId}`，随后使用 `/openapi/v2/query` 轮询结果。

```bash
go run ./examples/workflow \
	-workflow-id 2037060865681264641 \
	-payload-file ./examples/workflow/payload.example.json
```

如果你需要直接传入 JSON：

```bash
go run ./examples/workflow \
	-workflow-id 2037060865681264641 \
	-payload '{"addMetadata":true,"nodeInfoList":[],"instanceType":"default","usePersonalQueue":false}'
```

#### 示例 B：AI App

AI App 示例会提交 `/openapi/v2/run/ai-app/{appId}`，随后使用 `/openapi/v2/query` 轮询结果。

```bash
go run ./examples/ai_app \
	-app-id 2016796569449795585 \
	-payload-file ./examples/ai_app/payload.example.json
```

如果你需要直接传入 JSON：

```bash
go run ./examples/ai_app \
	-app-id 2016796569449795585 \
	-payload '{"nodeInfoList":[{"nodeId":"50","fieldName":"text","fieldValue":"润色这段话"}]}'
```

#### 示例 C：文生图

文生图示例默认使用 `/openapi/v2/seedream-v4/text-to-image`。如果你想切换到别的文生图模型，再用 `-path` 或 `RUNNINGHUB_TEXT_TO_IMAGE_PATH` 覆盖：

```bash
go run ./examples/text_to_image \
	-payload-file ./examples/text_to_image/payload.example.json
```

任务成功后，示例会把 `result.json` 和下载后的图片保存到 `./examples/text_to_image/output/`。

如果你想覆盖默认模型路径：

```bash
go run ./examples/text_to_image \
	-path /openapi/v2/your-text-to-image-model
```

或者：

```bash
# PowerShell
$env:RUNNINGHUB_TEXT_TO_IMAGE_PATH="/openapi/v2/your-text-to-image-model"

# bash
export RUNNINGHUB_TEXT_TO_IMAGE_PATH="/openapi/v2/your-text-to-image-model"
```

#### 示例 D：图片编辑

图片编辑示例默认使用 `/openapi/v2/rhart-image-n-pro-official/edit`。这个场景既可以直接传公网 URL，也可以先上传本地图片，再把 `download_url` 自动写进 `imageUrls[0]`。

```bash
go run ./examples/image_edit \
	-payload-file ./examples/image_edit/payload.example.json \
	-upload-file ./input.png
```

任务成功后，示例会把 `result.json` 和下载后的图片保存到 `./examples/image_edit/output/`。

如果你已经有公网可访问的图片 URL，也可以直接传：

```bash
go run ./examples/image_edit \
	-payload-file ./examples/image_edit/payload.example.json \
	-image-url https://your-public-image-url
```

#### 示例 E：图生视频

这个场景最容易踩坑的是 `imageUrl` 不可访问。优先用 `-upload-file`，让示例先上传本地图，再自动把 `download_url` 写进请求体。

```bash
go run ./examples/image_to_video \
	-path /openapi/v2/vidu/image-to-video-q3-pro-fast \
	-payload-file ./examples/image_to_video/payload.example.json \
	-upload-file ./input.png
```

任务成功后，示例会把 `result.json` 和下载后的视频保存到 `./examples/image_to_video/output/`。

#### 示例 F：文生视频

文生视频示例默认使用 `/openapi/v2/seedance-v1.5-pro/text-to-video`。如果你想切换到别的文生视频模型，再用 `-path` 或 `RUNNINGHUB_TEXT_TO_VIDEO_PATH` 覆盖：

默认 payload 已包含 `duration`、`aspectRatio`、`resolution`、`generateAudio` 这些常见字段；其中 `generateAudio` 需要传字符串值，例如 `"false"`。

```bash
go run ./examples/text_to_video \
	-payload-file ./examples/text_to_video/payload.example.json
```

任务成功后，示例会把 `result.json` 和下载后的视频保存到 `./examples/text_to_video/output/`。

如果你想覆盖默认模型路径：

```bash
go run ./examples/text_to_video \
	-path /openapi/v2/your-text-to-video-model
```

或者：

```bash
# PowerShell
$env:RUNNINGHUB_TEXT_TO_VIDEO_PATH="/openapi/v2/your-text-to-video-model"

# bash
export RUNNINGHUB_TEXT_TO_VIDEO_PATH="/openapi/v2/your-text-to-video-model"
```

#### 示例 G：任意场景先预览价格，不提交任务

```bash
go run ./examples/text_to_video \
	-payload-file ./examples/text_to_video/payload.example.json \
	-preview
```

#### 示例 H：图生视频手动传可访问 URL

如果你已经有公网可访问的图片 URL，也可以直接传 `-image-url`：

```bash
go run ./examples/image_to_video \
	-path /openapi/v2/vidu/image-to-video-q3-pro-fast \
	-payload-file ./examples/image_to_video/payload.example.json \
	-image-url https://your-public-image-url
```

#### 示例 I：Webhook 回调

这个示例使用 Gin v1.12.0 实现，包含两部分：

- 本地启动一个 HTTP 服务接收 RunningHub 的 webhook 回调
- 当 `config.yaml` 中同时提供 `appId` 和 `publicWebhookUrl` 时，示例会提交 AI App 任务，并等待回调到达
- 日志可分别控制是否输出到终端和文件
- 支持从 `yaml` 配置文件加载参数；配置文件不存在时会自动生成默认值

先只启动本地接收服务。仓库已经提供了可直接修改的 `config.yaml` 模板：

```bash
cd ./examples/webhook && go run .
```

然后直接编辑 `config.yaml`，填入你自己的 `apiKey`、`appId`、`publicWebhookUrl` 等字段。如果你使用 `ngrok`、`cloudflared tunnel` 或其他方式把本地 `/webhook` 暴露成公网地址，再把这个公网地址写进配置文件：

```yaml
apiKey: "your-runninghub-api-key"
appId: "2016796569449795585"
publicWebhookUrl: "https://your-domain.example/webhook"
listenAddr: ":8080"
webhookPath: "/webhook"
timeout: "15m"
log:
  console: true
  file: true
  filePath: "./webhook.log"
```

如果你想只写文件、不打印终端日志，直接改配置：

```yaml
log:
  console: false
  file: true
  filePath: "./webhook.log"
```

收到回调后，示例会把回调 JSON 和请求日志写到配置的输出目标。回调体结构与 `/openapi/v2/query` 返回的任务结果一致，可以直接复用 SDK 中的 `QueryV2Response`。

默认生成的 `config.yaml` 里会包含监听地址、回调路径和日志输出。提交任务时会固定读取 [examples/webhook/payload.example.json](examples/webhook/payload.example.json)。

各独立示例支持的主要参数：

- `-api-key`：也可不传，默认读取环境变量 `RUNNINGHUB_API_KEY`
- `-path`：标准模型接口路径，支持完整路径或 v2 相对路径
- `-payload`：直接传 JSON 字符串
- `-payload-file`：从 JSON 文件读取请求体
- `-upload-file`：图生视频、图片编辑示例支持，先上传本地图片
- `-image-url`：图生视频、图片编辑示例支持，直接传公网可访问图片 URL
- `-output-dir`：结果 JSON 和下载后的图片/视频保存目录
- `-preview`：只调用价格预览接口
- `-poll-interval`：轮询间隔，默认 2 秒
- `-timeout`：整体超时，默认 5 分钟
- `RUNNINGHUB_TEXT_TO_IMAGE_PATH`：可选，用来覆盖文生图示例默认路径 `/openapi/v2/seedream-v4/text-to-image`
- `RUNNINGHUB_TEXT_TO_VIDEO_PATH`：可选，用来覆盖文生视频示例默认路径 `/openapi/v2/seedance-v1.5-pro/text-to-video`

### 1) 初始化 Client

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
	apiKey := os.Getenv("RUNNINGHUB_API_KEY")
	c, err := runninghub.New(apiKey)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 示例：查询队列状态
	q, err := c.QueueStatus(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("queued=", q.QueuedCount, "running=", q.RunningCount)
}
```

### 2) 标准模型 API：通用调用 + 轮询结果

标准模型 API 的共同模式通常是：

- `POST /openapi/v2/<模型路径>` 提交任务
- 返回任务对象（含 `taskId/status/...`）
- 用 `POST /openapi/v2/query` 轮询查询结果

SDK 提供通用方法：

- `RunStandardModel(ctx, path, req)`：提交任意标准模型任务
- `WaitForTask(ctx, taskID, pollInterval)`：轮询直到任务进入终态
- `QueryTaskV2(ctx, taskID)`：查询任务结果

```go
// 这里只演示请求体形状；具体字段以 RunningHub 文档为准。
task, err := c.RunStandardModel(ctx, "/openapi/v2/rhart-image-n-pro-official/edit", map[string]any{
	"imageUrls": []string{"https://example.com/image.png"},
	"prompt":    "海边沙滩变成夏日祭典现场，风格欢乐卡通，色彩缤纷。",
	"resolution": "1k",
	"aspectRatio": "3:4",
})
if err != nil {
	panic(err)
}

out, err := c.WaitForTask(ctx, task.TaskID, 2*time.Second)
if err != nil {
	panic(err)
}
if out.Status == "FAILED" {
	panic(fmt.Errorf("task failed: %s %s", out.ErrorCode, out.ErrorMessage))
}
fmt.Println("results=", out.Results)
```

对于图片编辑这类依赖输入图片的模型，`imageUrls` 支持三种方式：

- 直接传公网可访问 URL
- 直接传 Base64 Data URI
- 先调用 `UploadBinaryFile` / `UploadBinaryReader` 上传本地文件，再把返回的 `download_url` 填进请求体

### 2.1) 工作流：提交任务 + 查询结果

SDK 提供：

- `RunWorkflow(ctx, workflowID, req)`：提交工作流任务
- `WaitForTask(ctx, taskID, pollInterval)`：轮询直到任务进入终态
- `QueryTaskV2(ctx, taskID)`：查询任务结果

```go
addMetadata := true
usePersonalQueue := false

resp, err := c.RunWorkflow(ctx, "2037060865681264641", runninghub.RunWorkflowRequest{
	AddMetadata:      &addMetadata,
	NodeInfoList:     []runninghub.WorkflowNodeInfo{},
	InstanceType:     "default",
	UsePersonalQueue: &usePersonalQueue,
})
if err != nil {
	panic(err)
}

out, err := c.WaitForTask(ctx, resp.TaskID, 2*time.Second)
if err != nil {
	panic(err)
}
if out.Status == "FAILED" {
	panic(fmt.Errorf("task failed: %s %s", out.ErrorCode, out.ErrorMessage))
}
fmt.Println("results=", out.Results)
```

`WorkflowNodeInfo` 复用了通用节点映射结构，`FieldValue` 使用 `any`，因此除了字符串，也可以传布尔、数字、数组、对象，以及文件 URL 或 Base64 Data URI。

### 2.2) AI App：提交任务 + 查询结果

SDK 提供：

- `RunAIApp(ctx, appID, req)`：提交 AI App 任务
- `QueryTaskV2(ctx, taskID)`：查询任务结果

```go
usePersonalQueue := false

resp, err := c.RunAIApp(ctx, "2016796569449795585", runninghub.RunAIAppRequest{
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

for {
	out, err := c.QueryTaskV2(ctx, resp.TaskID)
	if err != nil {
		panic(err)
	}
	if out.Status == "SUCCESS" {
		fmt.Println("results=", out.Results)
		break
	}
	if out.Status == "FAILED" {
		panic(fmt.Errorf("task failed: %s %s", out.ErrorCode, out.ErrorMessage))
	}
	time.Sleep(2 * time.Second)
}
```

### 2.3) 端到端示例：上传本地文件 → 标准模型调用 → 查询结果

很多标准模型的输入需要一个可访问的 `imageUrl` / `videoUrl` / `audioUrl`。
通常你可以先把本地文件上传到 RunningHub，再把返回的 URL 作为后续模型入参（**具体以对应模型文档的字段为准**）。

```go
up, err := c.UploadBinaryFile(ctx, "./input.png")
if err != nil {
	panic(err)
}

// 注意：不同模型文档里字段名可能是 imageUrl / videoUrl / audioUrl。
// up.DownloadURL 是否可直接作为模型输入 URL，以模型文档为准。
task, err := c.RunStandardModel(ctx, "rhart-text-g-25-pro-cv/image-to-text", map[string]any{
	"prompt":   "请描述图片内容",
	"imageUrl": up.DownloadURL,
})
if err != nil {
	panic(err)
}

out, err := c.QueryTaskV2(ctx, task.TaskID)
if err != nil {
	panic(err)
}
fmt.Println(out.Status, out.Results)

// 如果某个 AI App 节点字段需要文件 URL，也可以把上传返回的 download_url
// 放进 nodeInfoList。
_, _ = c.RunAIApp(ctx, "2016796569449795585", runninghub.RunAIAppRequest{
	NodeInfoList: []runninghub.AIAppNodeInfo{
		{
			NodeID:     "12",
			FieldName:  "imageUrl",
			FieldValue: up.DownloadURL,
		},
	},
})
```

### 2.4) 文件上传：公共 URL / Base64 / RH 上传接口

资源字段通常支持三种方式，具体字段名以对应模型或 AI App 工作流定义为准：

- 直接传公共 URL，例如 `https://example.com/image.png`
- 直接传 Base64 Data URI，例如 `data:image/png;base64,...`
- 先调用 `UploadBinaryFile` 或 `UploadBinaryReader` 上传本地文件，再使用返回的 `download_url`

```go
up, err := c.UploadBinaryFile(ctx, "./image.png")
if err != nil {
	panic(err)
}

req := runninghub.RunAIAppRequest{
	NodeInfoList: []runninghub.AIAppNodeInfo{
		{
			NodeID:     "12",
			FieldName:  "imageUrl",
			FieldValue: up.DownloadURL,
		},
	},
}

_, err = c.RunAIApp(ctx, "2016796569449795585", req)
if err != nil {
	panic(err)
}
```

上传接口返回的 `download_url` 有效期为 1 天；任务结果里的 `results[].url` 有效期为 24 小时，建议任务完成后尽快下载或转存。

### 3) 标准模型 API：价格预览

RunningHub 提供 `POST /openapi/v2/price-preview/**` 用于预估价格：

- URL 路径与对应的模型 API 路径保持一致
- 请求体与对应的模型 API 请求体保持一致

SDK 提供：`PricePreview(ctx, modelPath, req)`。

```go
price, err := c.PricePreview(ctx, "/openapi/v2/vidu/image-to-video-q3-pro-fast", map[string]any{
	"prompt":   "...",
	"imageUrl": "https://...",
	"duration": "5",
})
if err != nil {
	panic(err)
}
fmt.Println(price.PriceText, price.EstimatedPrice, price.Currency)
```

## 已封装的接口（Typed Methods）

> 下面是当前仓库代码中已实现的封装方法清单（持续扩展中）。

### OpenAPI v2

- 上传（二进制 multipart）

  - `UploadBinaryFile(ctx, filePath)` → `POST /openapi/v2/media/upload/binary`
  - `UploadBinaryReader(ctx, r, filename)` → 同上

- 任务查询

  - `QueryTaskV2(ctx, taskID)` → `POST /openapi/v2/query`（注意：该接口返回 task 对象本身，不是 envelope）
	- `RunWorkflow(ctx, workflowID, req)` → `POST /openapi/v2/run/workflow/{workflowID}`
	- `RunAIApp(ctx, appID, req)` → `POST /openapi/v2/run/ai-app/{appID}`

- 账户 / 队列 / apikey

  - `APIKeyList(ctx)` → `GET /openapi/v2/api-key/list`
  - `QueueStatus(ctx)` → `GET /openapi/v2/queue/status`
  - `AccountStatus(ctx)` → `POST /uc/openapi/accountStatus`（body 字段名为 `apikey`）

- 公共资源

  - `ListPublicResources(ctx, req)` → `POST /openapi/v2/resource/list`

### ComfyUI 工作流

- `CreateComfyTaskSimple(ctx, workflowID, addMetadata)` → `POST /task/openapi/create`
- `CancelComfyTask(ctx, taskID)` → `POST /task/openapi/cancel`
- `GetWorkflowJSONPrompt(ctx, workflowID)` → `POST /api/openapi/getJsonApiFormat`（返回 prompt JSON 字符串）
- `GetWorkflowJSON(ctx, workflowID)` → 解析 prompt 为 `map[string]any`

### 标准模型 API（通用）

- `RunStandardModel(ctx, path, req)`：调用任意 `POST /openapi/v2/...` 模型端点
- `PricePreview(ctx, modelPath, req)`：调用 `/openapi/v2/price-preview/...`
- `WaitForTask(ctx, taskID, pollInterval)`：轮询 `QueryTaskV2` 直到 `SUCCESS`、`FAILED` 或 `CANCELLED`
- `DownloadFile(ctx, fileURL, destPath)`：把图片/视频 URL 下载到本地文件
- `DownloadTaskResults(ctx, task, outputDir)`：批量下载任务结果里的所有 URL 到本地目录

## 错误处理

SDK 在两类情况下会返回错误：

1) HTTP 层错误（非 200）会返回 `*runninghub.HTTPError`（包含 status code 与 body）
2) RunningHub 业务层错误（响应 `code != 0`）会返回 `*runninghub.APIError`

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

## FAQ / 注意事项

### 1) `RunStandardModel` 的 path 怎么写？

两种写法都支持：

- 完整路径：`/openapi/v2/vidu/image-to-video-q3-pro-fast`
- v2 相对路径：`vidu/image-to-video-q3-pro-fast`

标准模型文档页面里 `paths:` 下的路径就是你要传的 path。

### 2) 为什么标准模型调用返回的是 task，而不是最终结果？

标准模型 API 通常是异步任务：提交后返回 `taskId` + `status`。
SDK 复用 RunningHub 官方的查询接口 `QueryTaskV2` 获取最终 `results`。

### 3) 并发安全吗？

`Client` 在创建后不再修改内部配置，通常可以并发复用（底层使用 `http.Client`）。
建议把 `Client` 作为全局单例注入你的服务中。

### 4) 超时 / 大响应怎么办？

- 使用 `context.WithTimeout` 控制单次请求超时
- 用 `WithHTTPClient` 配置更长的 `http.Client.Timeout`
- 返回 body 默认最多读取 10MiB，可用 `WithMaxBodyBytes` 调整

## 配置项（Options）

创建 Client 时可传入 Option：

- `WithBaseURL(raw)`：自定义 base URL（默认 `https://www.runninghub.cn`）
- `WithHTTPClient(hc)`：自定义 `*http.Client`（超时、代理等）
- `WithHostOverride(host)`：设置 `req.Host`（少数场景用）
- `WithUserAgent(ua)`：设置 User-Agent
- `WithMaxBodyBytes(n)`：限制响应体读取大小（默认 10MiB）

示例：

```go
hc := &http.Client{Timeout: 120 * time.Second}
c, err := runninghub.New(apiKey,
	runninghub.WithHTTPClient(hc),
	runninghub.WithUserAgent("myapp/1.0"),
)
```

## 低层调用（适合未封装的 API）

如果你要调用尚未写 typed wrapper 的接口，可以直接用：

```go
var out any
err := c.DoJSON(ctx, http.MethodPost, "/openapi/v2/some/endpoint", nil, nil, map[string]any{"k": "v"}, &out)
```

## 测试

```bash
go test ./...
```

## 版本与兼容性说明

- 只依赖 Go 标准库，无第三方依赖。
- SDK 会优先做“可用性封装”：保证常用接口可直接调用；标准模型 API 通过通用调用器覆盖大量端点。

## License

仓库未附带额外 License 文件时，默认遵循仓库当前的许可声明（如需补充 License，请在仓库内添加）。


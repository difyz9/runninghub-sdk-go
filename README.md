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
- `QueryTaskV2(ctx, taskID)`：查询任务结果

```go
// 这里只演示请求体形状；具体字段以 RunningHub 文档为准。
task, err := c.RunStandardModel(ctx, "/openapi/v2/vidu/image-to-video-q3-pro-fast", map[string]any{
	"prompt":   "女孩缓缓拿起茶杯喝茶...",
	"imageUrl": "https://www.runninghub.cn/view?filename=...",
	"duration": "5",
	"resolution": "720p",
	"audio": true,
})
if err != nil {
	panic(err)
}

// 轮询（示例：简单 sleep 轮询；你也可以用更稳健的 backoff）
for {
	out, err := c.QueryTaskV2(ctx, task.TaskID)
	if err != nil {
		panic(err)
	}
	if out.Status == "SUCCESS" {
		// out.Results: [{url, outputType, text?}, ...]
		fmt.Println("results=", out.Results)
		break
	}
	if out.Status == "FAILED" {
		panic(fmt.Errorf("task failed: %s %s", out.ErrorCode, out.ErrorMessage))
	}
	time.Sleep(2 * time.Second)
}
```

### 2.1) 端到端示例：上传本地文件 → 标准模型调用 → 查询结果

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
```

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


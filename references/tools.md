# Tool 创建完整指南

Tool 是 Agent 可以调用的外部能力单元。本文档介绍如何创建和使用 Tool。

## Tool 接口

```go
// 基础接口
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}

// 可调用 Tool
type InvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}

// 流式 Tool
type StreamableTool interface {
    BaseTool
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}

// 增强型 Tool（支持多模态）
type EnhancedInvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, toolArgument *schema.ToolArgument, opts ...Option) (*schema.ToolResult, error)
}
```

## 创建方式

### 1. InferTool（推荐）

从函数签名自动推断参数约束：

```go
import "github.com/cloudwego/eino/components/tool/utils"

type WeatherInput struct {
    City    string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
    Country string `json:"country" jsonschema_description:"国家代码，如 CN、US"`
}

weatherTool, err := utils.InferTool(
    "get_weather",
    "获取指定城市的天气信息",
    func(ctx context.Context, input *WeatherInput) (string, error) {
        return fmt.Sprintf("%s, %s 的天气：晴，25°C", input.City, input.Country), nil
    },
)
```

### 2. NewTool

手动指定 ToolInfo：

```go
tool := utils.NewTool(&schema.ToolInfo{
    Name: "calculate",
    Desc: "执行数学计算",
    ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
        "expression": {
            Type:     schema.String,
            Desc:     "数学表达式，如 1+2*3",
            Required: true,
        },
    }),
}, func(ctx context.Context, input *struct {
    Expression string `json:"expression"`
}) (string, error) {
    // 计算逻辑
    return result, nil
})
```

### 3. InferOptionableTool

支持运行时 Option：

```go
type SearchOption struct {
    MaxResults int
    Region     string
}

func WithMaxResults(n int) tool.Option {
    return tool.WrapImplSpecificOptFn(func(o *SearchOption) {
        o.MaxResults = n
    })
}

func WithRegion(r string) tool.Option {
    return tool.WrapImplSpecificOptFn(func(o *SearchOption) {
        o.Region = r
    })
}

searchTool, _ := utils.InferOptionableTool(
    "web_search",
    "搜索网页内容",
    func(ctx context.Context, input *SearchInput, opts ...tool.Option) (string, error) {
        // 默认配置
        opt := tool.GetImplSpecificOptions(&SearchOption{
            MaxResults: 10,
            Region:     "US",
        }, opts...)
        
        // 使用 opt.MaxResults 和 opt.Region
        return search(input.Query, opt.MaxResults, opt.Region), nil
    },
)

// 调用时传入 Option
result, _ := searchTool.InvokableRun(ctx, `{"query": "golang"}`,
    WithMaxResults(20),
    WithRegion("CN"),
)
```

### 4. 流式 Tool

```go
streamTool, _ := utils.InferStreamTool(
    "stream_search",
    "流式搜索",
    func(ctx context.Context, input *SearchInput) (*schema.StreamReader[string], error) {
        results := []string{"result 1", "result 2", "result 3"}
        return schema.StreamReaderFromArray(results), nil
    },
)
```

### 5. 增强型 Tool（多模态）

返回图片、文件等内容：

```go
imageTool, _ := utils.InferEnhancedTool(
    "image_search",
    "搜索图片",
    func(ctx context.Context, input *ImageSearchInput) (*schema.ToolResult, error) {
        imageURL := "https://example.com/image.png"
        
        return &schema.ToolResult{
            Parts: []schema.ToolOutputPart{
                {Type: schema.ToolPartTypeText, Text: "找到以下图片："},
                {
                    Type: schema.ToolPartTypeImage,
                    Image: &schema.ToolOutputImage{
                        MessagePartCommon: schema.MessagePartCommon{
                            URL: &imageURL,
                        },
                    },
                },
            },
        }, nil
    },
)
```

### 6. 直接实现接口

```go
type CustomTool struct{}

func (t *CustomTool) Info(_ context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "custom_tool",
        Desc: "自定义工具",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "param1": {Type: schema.String, Required: true},
        }),
    }, nil
}

func (t *CustomTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
    var input struct {
        Param1 string `json:"param1"`
    }
    if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
        return "", err
    }
    // 处理逻辑
    return "result", nil
}
```

## 参数约束

### 使用 jsonschema Tag

```go
type Input struct {
    // 字段名从 json tag 获取
    Name string `json:"name" jsonschema:"required" jsonschema_description:"用户姓名"`
    
    // 枚举值
    Gender string `json:"gender" jsonschema:"enum=male,enum=female,enum=other"`
    
    // 数值范围（通过 description 说明）
    Age int `json:"age" jsonschema_description:"年龄，范围 0-150"`
    
    // 数组
    Tags []string `json:"tags" jsonschema_description:"标签列表"`
    
    // 嵌套对象
    Address Address `json:"address" jsonschema_description:"地址信息"`
}

type Address struct {
    City    string `json:"city" jsonschema:"required"`
    Country string `json:"country" jsonschema:"required"`
}
```

### 使用 ParameterInfo

```go
ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
    "name": {
        Type:     schema.String,
        Desc:     "用户姓名",
        Required: true,
    },
    "gender": {
        Type: schema.String,
        Enum: []string{"male", "female"},
    },
    "age": {
        Type: schema.Integer,
        Desc: "年龄",
    },
    "tags": {
        Type: schema.Array,
        ElemInfo: &schema.ParameterInfo{
            Type: schema.String,
        },
    },
})
```

## Tool 调用

### 在 ReAct Agent 中

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{tool1, tool2},
    },
})
```

### 在 ChatModelAgent 中

```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{tool1, tool2},
        },
    },
})
```

### 直接调用

```go
result, err := tool.InvokableRun(ctx, `{"param": "value"}`)
```

## MCP Tool

从 MCP 服务器获取 Tool：

```go
import (
    "github.com/mark3labs/mcp-go/client"
    mcp "github.com/cloudwego/eino-ext/components/tool/mcp"
)

func getMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
    cli, err := client.NewSSEMCPClient("http://localhost:12345/sse")
    if err != nil {
        return nil, err
    }
    
    if err := cli.Start(ctx); err != nil {
        return nil, err
    }
    
    return mcp.GetTools(ctx, &mcp.Config{Cli: cli})
}
```

## eino-ext 内置 Tool

eino-ext 提供了多种开箱即用的 Tool：

```go
import (
    "github.com/cloudwego/eino-ext/components/tool/googlesearch"
    "github.com/cloudwego/eino-ext/components/tool/duckduckgo"
    "github.com/cloudwego/eino-ext/components/tool/wikipedia"
    "github.com/cloudwego/eino-ext/components/tool/httprequest"
)

// Google 搜索
googleTool, _ := googlesearch.NewTool(ctx, &googlesearch.Config{
    APIKey:  os.Getenv("GOOGLE_API_KEY"),
    EngineID: os.Getenv("GOOGLE_ENGINE_ID"),
})

// DuckDuckGo 搜索
ddgTool, _ := duckduckgo.NewTool(ctx)

// Wikipedia
wikiTool, _ := wikipedia.NewTool(ctx)

// HTTP Request
httpTool, _ := httprequest.NewTool(ctx)
```

## Tool 包装器

### 错误处理包装器

```go
errorRemover, _ := errorremover.NewErrorRemoverMiddleware(tool)
```

### JSON 修复包装器

```go
jsonFixer, _ := jsonfix.NewJSONFixMiddleware(tool)
```

### 审批包装器

```go
approvalTool := adk.NewApprovalTool(ctx, &adk.ApprovalToolConfig{
    BaseTool:         dangerousTool,
    MessageTemplate:  "即将执行操作：{{operation}}，是否确认？",
    ApprovalRequired: true,
})
```

## 最佳实践

1. **清晰的描述**：Tool 名称和描述要准确，便于 LLM 理解何时调用
2. **参数文档化**：使用 jsonschema_description 详细描述每个参数
3. **错误处理**：返回有意义的错误信息
4. **幂等性**：Tool 调用应尽量幂等
5. **超时控制**：长时间操作需要 context 超时控制

```go
func myTool(ctx context.Context, input *Input) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // 执行操作...
}
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/schema"
)

func main() {
    // 定义输入
    type SearchInput struct {
        Query   string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
        Max     int    `json:"max" jsonschema_description:"最大结果数，默认 10"`
        Lang    string `json:"lang" jsonschema:"enum=zh,enum=en,enum=ja" jsonschema_description:"语言"`
    }
    
    // 定义 Option
    type SearchOption struct {
        Timeout int
    }
    
    timeoutOpt := func(t int) tool.Option {
        return tool.WrapImplSpecificOptFn(func(o *SearchOption) {
            o.Timeout = t
        })
    }
    
    // 创建 Tool
    searchTool, _ := utils.InferOptionableTool(
        "web_search",
        "搜索网页内容",
        func(ctx context.Context, input *SearchInput, opts ...tool.Option) (string, error) {
            opt := tool.GetImplSpecificOptions(&SearchOption{
                Timeout: 10,
            }, opts...)
            
            if input.Max == 0 {
                input.Max = 10
            }
            
            // 模拟搜索
            return fmt.Sprintf("搜索 '%s' (语言: %s, 最大: %d, 超时: %ds)",
                input.Query, input.Lang, input.Max, opt.Timeout), nil
        },
    )
    
    // 调用
    ctx := context.Background()
    result, _ := searchTool.InvokableRun(ctx, 
        `{"query": "golang tutorial", "max": 20, "lang": "en"}`,
        timeoutOpt(30),
    )
    fmt.Println(result)
}
```
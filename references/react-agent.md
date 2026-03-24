# ReAct Agent 完整指南

ReAct Agent 是 Eino 实现的 ReAct (Reasoning + Acting) 逻辑的智能体框架。

## 核心概念

ReAct Agent 底层使用 `compose.Graph` 编排，包含两个核心节点：
- **ChatModel**：推理节点，决定是否调用工具
- **Tools**：执行节点，调用工具并返回结果

```
用户输入 → ChatModel → 判断 → Tools → ChatModel → ... → 最终响应
```

## 初始化

### 必要依赖

```bash
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/openai@latest
```

### 基础配置

```go
import (
    "context"
    
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func createReactAgent(ctx context.Context) (*react.Agent, error) {
    // 1. 创建 ChatModel
    model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 创建 Tools
    tools := []tool.BaseTool{weatherTool, calculatorTool}
    
    // 3. 创建 Agent
    return react.NewAgent(ctx, &react.AgentConfig{
        ToolCallingModel: model,
        ToolsConfig: compose.ToolsNodeConfig{
            Tools: tools,
        },
    })
}
```

## 配置选项

### MessageModifier

在每次调用 ChatModel 前修改消息，常用于添加 system prompt：

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: toolsConfig,
    MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
        res := make([]*schema.Message, 0, len(input)+1)
        res = append(res, schema.SystemMessage("你是一个专业的助手。"))
        res = append(res, input...)
        return res
    },
})
```

### MessageRewriter

修改并持久化历史消息，适合上下文压缩：

```go
MessageRewriter: func(ctx context.Context, input []*schema.Message) []*schema.Message {
    // 压缩历史消息，保留最近 10 条
    if len(input) > 10 {
        return input[len(input)-10:]
    }
    return input
},
```

### MaxStep

设置最大运行步数：
- 1 个循环 = ChatModel + Tools = 2 步
- 默认值 = 12（最多 5 次 tool 调用）
- 想运行 N 个循环，设置 MaxStep = 2N

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    // ...
    MaxStep: 20, // 最多 10 个循环
})
```

### ToolReturnDirectly

指定某些 Tool 执行后直接返回结果：

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    // ...
    ToolReturnDirectly: map[string]struct{}{
        "get_weather": {},
    },
})
```

### StreamToolCallChecker

自定义判断流式输出是否包含 ToolCall：

```go
toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
    defer sr.Close()
    for {
        msg, err := sr.Recv()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            return false, err
        }
        if len(msg.ToolCalls) > 0 {
            return true, nil
        }
    }
    return false, nil
}

agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    // ...
    StreamToolCallChecker: toolCallChecker,
})
```

## 调用方式

### Generate（同步）

```go
msg, err := agent.Generate(ctx, []*schema.Message{
    schema.UserMessage("北京天气怎么样？"),
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(msg.Content)
```

### Stream（流式）

```go
sr, err := agent.Stream(ctx, []*schema.Message{
    schema.UserMessage("北京天气怎么样？"),
})
if err != nil {
    log.Fatal(err)
}
defer sr.Close()

for {
    msg, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(msg.Content)
}
```

## 运行时选项

### 动态修改 Model 配置

```go
msg, _ := agent.Generate(ctx, messages,
    react.WithChatModelOptions(model.WithTemperature(0.7)),
)
```

### 动态修改 Tool 列表

```go
msg, _ := agent.Generate(ctx, messages,
    react.WithToolList(newTool1, newTool2),
    react.WithChatModelOptions(model.WithTools(toolInfos...)),
)
```

## Callback 回调

```go
import (
    template "github.com/cloudwego/eino/utils/callbacks"
)

callback := react.BuildAgentCallback(
    &template.ModelCallbackHandler{
        OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *callbacks.CallbackOutput) context.Context {
            fmt.Println("Model finished")
            return ctx
        },
    },
    &template.ToolCallbackHandler{
        OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *callbacks.CallbackInput) context.Context {
            fmt.Printf("Tool called: %s\n", input)
            return ctx
        },
    },
)

agent.Generate(ctx, messages, react.WithCallbacks(callback))
```

## 嵌入其他 Graph

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{...})

chain := compose.NewChain[[]*schema.Message, string]()
agentLambda, _ := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)

chain.
    AppendLambda(agentLambda).
    AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (string, error) {
        return input.Content, nil
    }))

r, _ := chain.Compile(ctx)
result, _ := r.Invoke(ctx, []*schema.Message{schema.UserMessage("hello")})
```

## 完整示例

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

type WeatherInput struct {
    City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

func main() {
    ctx := context.Background()
    
    model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    
    weatherTool, _ := utils.InferTool(
        "get_weather",
        "获取指定城市的天气",
        func(ctx context.Context, input *WeatherInput) (string, error) {
            return fmt.Sprintf("%s 今天天气晴朗，温度 25°C", input.City), nil
        },
    )
    
    agent, _ := react.NewAgent(ctx, &react.AgentConfig{
        ToolCallingModel: model,
        ToolsConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{weatherTool},
        },
        MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
            res := make([]*schema.Message, 0, len(input)+1)
            res = append(res, schema.SystemMessage("你是一个天气助手。"))
            res = append(res, input...)
            return res
        },
        MaxStep: 20,
    })
    
    sr, _ := agent.Stream(ctx, []*schema.Message{
        schema.UserMessage("北京今天天气怎么样？"),
    })
    defer sr.Close()
    
    for {
        msg, err := sr.Recv()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            panic(err)
        }
        fmt.Print(msg.Content)
    }
}
```
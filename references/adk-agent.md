# ADK Agent 详细说明

ADK (Agent Development Kit) 是 Eino 提供的 Agent 开发套件，支持 ChatModel Agent、Workflow Agents、多 Agent 协作等。

## 安装

```bash
go get github.com/cloudwego/eino@latest
```

ADK 从 eino v0.5.0 开始提供。

## Agent 类型

### 1. ChatModelAgent

基于 LLM 的智能 Agent，支持 Tool 调用、多 Agent 协作。

```go
import "github.com/cloudwego/eino/adk"

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "assistant",
    Description: "A helpful assistant",
    Instruction: "You are a helpful assistant.",
    Model:       model,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{weatherTool},
        },
    },
})
```

### 2. Workflow Agents

#### SequentialAgent

按顺序依次执行子 Agents：

```go
agent, _ := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
    Name:        "research_workflow",
    Description: "Research and write report",
    SubAgents:   []adk.Agent{plannerAgent, writerAgent},
})
```

#### ParallelAgent

并发执行多个子 Agents：

```go
agent, _ := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
    Name:        "data_collection",
    Description: "Collect data from multiple sources",
    SubAgents:   []adk.Agent{source1Agent, source2Agent, source3Agent},
})
```

#### LoopAgent

循环执行子 Agents 直到满足条件：

```go
agent, _ := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
    Name:          "reflection_loop",
    Description:   "Iterate and improve",
    SubAgents:     []adk.Agent{mainAgent, criticAgent},
    MaxIterations: 5,
})
```

### 3. 内置 Multi-Agent 模式

#### Supervisor

层级协调模式：

```go
import "github.com/cloudwego/eino/adk/prebuilt"

supervisor, _ := prebuilt.NewSupervisor(ctx, &prebuilt.SupervisorConfig{
    Supervisor: routerAgent,
    SubAgents:  []adk.Agent{researchAgent, mathAgent},
})
```

#### Plan-Execute-Replan

计划-执行-重规划模式：

```go
planExecute, _ := prebuilt.NewPlanExecuteReplan(ctx, &prebuilt.PlanExecuteReplanConfig{
    Planner:  plannerAgent,
    Executor: executorAgent,
    Replanner: replannerAgent,
})
```

## Runner 运行

### 基本运行

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})

iter := runner.Query(ctx, "What's the weather in Beijing?")

for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    
    if event.Err != nil {
        log.Printf("Error: %v", event.Err)
        continue
    }
    
    if event.Action != nil {
        // 处理 Action（Transfer、Interrupt 等）
        fmt.Printf("Action: %+v\n", event.Action)
    } else {
        // 处理输出
        fmt.Printf("Output: %+v\n", event.Output)
    }
}
```

### 事件处理

```go
for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    
    switch {
    case event.Action != nil:
        // Action 事件
        if event.Action.TransferToAgent != nil {
            fmt.Printf("Transfer to: %s\n", event.Action.TransferToAgent.DestAgentName)
        }
        if event.Action.Interrupt != nil {
            // 处理中断
        }
        
    case event.Output != nil:
        // 输出事件
        if msg, err := event.Output.MessageOutput.GetMessage(); err == nil {
            fmt.Println(msg.Content)
        }
    }
}
```

## 多 Agent 协作

### Transfer 机制

Agent 可以将任务 Transfer 给子 Agent：

```go
// 设置子 Agents
agent, _ := adk.SetSubAgents(ctx, parentAgent, []adk.Agent{
    childAgent1,
    childAgent2,
})
```

ChatModelAgent 会自动在 system prompt 中添加可用的子 Agent 列表。

### AgentAsTool

将 Agent 封装为 Tool：

```go
agentTool := adk.NewAgentTool(ctx, specialistAgent)

// 然后可以将 agentTool 作为普通 Tool 使用
mainAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{agentTool},
        },
    },
})
```

## 中断与恢复 (HITL)

### 中断执行

```go
// 配置需要中断的工具
approvalTool, _ := adk.NewApprovalTool(ctx, &adk.ApprovalToolConfig{
    BaseTool:   dangerousTool,
    MessageTemplate: "即将执行敏感操作，是否继续？",
})

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{approvalTool},
        },
    },
})
```

### 恢复执行

```go
// 第一次运行，遇到中断
iter := runner.Query(ctx, "Delete all files")

// 检测到中断
event, _ := iter.Next()
if event.Action.Interrupt != nil {
    checkpoint := event.Action.Interrupt.CheckPoint
    
    // 用户确认后恢复
    iter = runner.Resume(ctx, checkpoint, adk.WithApproval("approved"))
}
```

## SessionValues

跨 Agent 共享数据：

```go
// 注入初始值
iter := runner.Query(ctx, query,
    adk.WithSessionValues(map[string]any{
        "user_id": "123",
        "context": someData,
    }),
)

// 在 Agent 内部读取
func myAgentFunc(ctx context.Context, input *Input) (*Output, error) {
    userID, ok := adk.GetSessionValue(ctx, "user_id")
    // ...
}
```

## Middleware

ADK 提供内置中间件：

### Summarization

压缩对话历史：

```go
import "github.com/cloudwego/eino/adk/middlewares/summarization"

summarizer, _ := summarization.NewSummarizationMiddleware(&summarization.Config{
    Model:       model,
    MaxTokens:   4000,
    TriggerThreshold: 3000,
})

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    Middlewares: []adk.Middleware{summarizer},
})
```

### FileSystem

文件系统操作：

```go
import "github.com/cloudwego/eino/adk/middlewares/filesystem"

fs, _ := filesystem.NewFileSystemMiddleware(&filesystem.Config{
    RootDir: "/tmp/agent_files",
})

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    Middlewares: []adk.Middleware{fs},
})
```

### Skill

动态加载技能：

```go
import "github.com/cloudwego/eino/adk/middlewares/skill"

skillLoader, _ := skill.NewSkillMiddleware(&skill.Config{
    SkillDir: "./skills",
})

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    Middlewares: []adk.Middleware{skillLoader},
})
```

## Callback

```go
import "github.com/cloudwego/eino/adk/callback"

cb := callback.NewAgentCallback(&callback.Config{
    OnAgentStart: func(ctx context.Context, info *AgentRunInfo) context.Context {
        fmt.Printf("Agent %s started\n", info.Name)
        return ctx
    },
    OnAgentEnd: func(ctx context.Context, info *AgentRunInfo, output *AgentOutput) context.Context {
        fmt.Printf("Agent %s finished\n", info.Name)
        return ctx
    },
})

runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:    agent,
    Callback: cb,
})
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()
    
    // 创建模型
    model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    
    // 创建工具
    weatherTool, _ := utils.InferTool(
        "get_weather",
        "Get weather for a city",
        func(ctx context.Context, input *struct {
            City string `json:"city" jsonschema:"required"`
        }) (string, error) {
            return fmt.Sprintf("Weather in %s: Sunny, 25°C", input.City), nil
        },
    )
    
    // 创建 Agent
    agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "weather_agent",
        Description: "A weather assistant",
        Instruction: "You help users get weather information.",
        Model:       model,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{weatherTool},
            },
        },
    })
    
    // 创建 Runner
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           agent,
        EnableStreaming: true,
    })
    
    // 运行
    iter := runner.Query(ctx, "What's the weather in Beijing?")
    
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        
        if event.Err != nil {
            fmt.Printf("Error: %v\n", event.Err)
            continue
        }
        
        if msg, err := adk.GetMessage(event); err == nil {
            fmt.Println(msg.Content)
        }
    }
}
```
# 多 Agent 协作模式

Eino ADK 提供多种多 Agent 协作模式，支持灵活的智能体编排。

## 协作原语

### Transfer vs ToolCall

| 方式 | 描述 | 使用场景 |
|------|------|----------|
| Transfer | 任务移交，父 Agent 退出 | 层级路由，任务转交 |
| ToolCall (AgentAsTool) | 调用子 Agent 并等待结果 | 需要子 Agent 结果继续处理 |

### 上下文策略

| 策略 | 描述 |
|------|------|
| History | 继承上游 Agent 的完整对话 |
| 全新任务 | 使用新的任务描述作为输入 |

## 基础模式

### 1. Supervisor 模式

Supervisor 作为协调者，决定任务分发给哪个子 Agent。

```
┌─────────────┐
│  Supervisor │
└─────┬───────┘
      │
  ┌───┴───┬───────┐
  ▼       ▼       ▼
Agent1  Agent2  Agent3
```

```go
package main

import (
    "context"
    
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/prebuilt"
)

func buildSupervisor(ctx context.Context) (adk.Agent, error) {
    // 创建子 Agents
    researchAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "researcher",
        Description: "Search and gather information",
        Instruction: "You are a research specialist. Search and summarize information.",
        Model:       model,
    })
    
    mathAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "mathematician",
        Description: "Perform mathematical calculations",
        Instruction: "You are a math specialist. Solve mathematical problems.",
        Model:       model,
    })
    
    // 创建 Supervisor
    supervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "supervisor",
        Description: "Route tasks to specialists",
        Instruction: "You are a supervisor. Route tasks to appropriate specialists.",
        Model:       model,
    })
    
    // 组装
    return prebuilt.NewSupervisor(ctx, &prebuilt.SupervisorConfig{
        Supervisor: supervisor,
        SubAgents:  []adk.Agent{researchAgent, mathAgent},
    })
}
```

### 2. Plan-Execute-Replan 模式

适合复杂任务规划。

```
┌─────────┐     ┌──────────┐     ┌───────────┐
│ Planner │ ──► │ Executor │ ──► │ Replanner │
└─────────┘     └──────────┘     └───────────┘
                      │                │
                      └────────────────┘
                        (循环直到完成)
```

```go
func buildPlanExecute(ctx context.Context) (adk.Agent, error) {
    planner, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "planner",
        Description: "Create execution plans",
        Instruction: "Break down complex tasks into steps.",
        Model:       model,
    })
    
    executor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "executor",
        Description: "Execute plan steps",
        Instruction: "Execute each step of the plan.",
        Model:       model,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{searchTool, apiTool},
            },
        },
    })
    
    replanner, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "replanner",
        Description: "Evaluate and replan",
        Instruction: "Evaluate results and decide if replanning is needed.",
        Model:       model,
    })
    
    return prebuilt.NewPlanExecuteReplan(ctx, &prebuilt.PlanExecuteReplanConfig{
        Planner:    planner,
        Executor:   executor,
        Replanner:  replanner,
    })
}
```

### 3. 层级 Supervisor

多层 Supervisor 嵌套。

```go
func buildLayeredSupervisor(ctx context.Context) (adk.Agent, error) {
    // 底层工具 Agents
    addAgent := createMathAgent("add", "Addition")
    mulAgent := createMathAgent("multiply", "Multiplication")
    divAgent := createMathAgent("divide", "Division")
    
    // 中层 Math Supervisor
    mathSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "math_supervisor",
        Description: "Coordinate math operations",
        Instruction: "You coordinate math operations.",
        Model:       model,
    })
    
    mathLayer, _ := prebuilt.NewSupervisor(ctx, &prebuilt.SupervisorConfig{
        Supervisor: mathSupervisor,
        SubAgents:  []adk.Agent{addAgent, mulAgent, divAgent},
    })
    
    // 顶层 Supervisor
    topSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "top_supervisor",
        Description: "Route to research or math",
        Instruction: "You are the top-level supervisor.",
        Model:       model,
    })
    
    researchAgent := createResearchAgent()
    
    return prebuilt.NewSupervisor(ctx, &prebuilt.SupervisorConfig{
        Supervisor: topSupervisor,
        SubAgents:  []adk.Agent{researchAgent, mathLayer},
    })
}
```

### 4. Sequential Workflow

按顺序执行多个 Agents。

```go
func buildSequential(ctx context.Context) (adk.Agent, error) {
    planner := createPlannerAgent()
    researcher := createResearcherAgent()
    writer := createWriterAgent()
    reviewer := createReviewerAgent()
    
    return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
        Name:        "report_workflow",
        Description: "Create research report",
        SubAgents:   []adk.Agent{planner, researcher, writer, reviewer},
    })
}
```

### 5. Parallel Workflow

并发执行多个 Agents。

```go
func buildParallel(ctx context.Context) (adk.Agent, error) {
    newsAgent := createNewsSearchAgent()
    socialAgent := createSocialMediaAgent()
    webAgent := createWebSearchAgent()
    
    return adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
        Name:        "info_collection",
        Description: "Collect info from multiple sources",
        SubAgents:   []adk.Agent{newsAgent, socialAgent, webAgent},
    })
}
```

### 6. Loop Workflow

循环执行直到满足条件。

```go
func buildLoop(ctx context.Context) (adk.Agent, error) {
    coder := createCoderAgent()
    reviewer := createCodeReviewerAgent()
    
    return adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
        Name:          "code_improvement",
        Description:   "Iteratively improve code",
        SubAgents:     []adk.Agent{coder, reviewer},
        MaxIterations: 5,
    })
}
```

### 7. AgentAsTool

将 Agent 封装为 Tool 调用。

```go
func buildAgentAsTool(ctx context.Context) (adk.Agent, error) {
    // 专家 Agent 作为 Tool
    weatherExpert, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "weather_expert",
        Description: "Weather analysis expert",
        Instruction: "Analyze weather patterns and provide forecasts.",
        Model:       model,
    })
    
    weatherTool := adk.NewAgentTool(ctx, weatherExpert)
    
    // 主 Agent
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "assistant",
        Description: "General assistant",
        Instruction: "Help users with various tasks including weather.",
        Model:       model,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{weatherTool},
            },
        },
    })
}
```

## 上下文传递

### History

默认情况下，子 Agent 继承父 Agent 的对话历史。

### SessionValues

跨 Agent 共享数据：

```go
// 设置
adk.AddSessionValue(ctx, "user_preference", "detailed")

// 获取
pref, ok := adk.GetSessionValue(ctx, "user_preference")
```

### HistoryRewriter

自定义历史消息处理：

```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
}, adk.WithHistoryRewriter(func(ctx context.Context, entries []*adk.HistoryEntry) ([]adk.Message, error) {
    // 压缩或过滤历史
    if len(entries) > 10 {
        entries = entries[len(entries)-10:]
    }
    return convertEntriesToMessages(entries), nil
}))
```

## 确定性 Transfer

静态配置 Agent 跳转：

```go
// 子 Agent 执行完毕后固定跳转
wrappedAgent := adk.AgentWithDeterministicTransferTo(ctx, &adk.DeterministicTransferConfig{
    Agent:        subAgent,
    ToAgentNames: []string{"supervisor"},
})
```

## 完整示例：旅行规划系统

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/prebuilt"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
)

func main() {
    ctx := context.Background()
    
    // 工具定义
    flightTool, _ := utils.InferTool("search_flights", "Search flights",
        func(ctx context.Context, input *struct {
            From string `json:"from"`
            To   string `json:"to"`
        }) (string, error) {
            return fmt.Sprintf("Flights from %s to %s found", input.From, input.To), nil
        })
    
    hotelTool, _ := utils.InferTool("search_hotels", "Search hotels",
        func(ctx context.Context, input *struct {
            City string `json:"city"`
        }) (string, error) {
            return fmt.Sprintf("Hotels in %s found", input.City), nil
        })
    
    // 执行 Agent
    executor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "executor",
        Description: "Execute travel search tasks",
        Instruction: "Search for flights and hotels as needed.",
        Model:       model,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{flightTool, hotelTool},
            },
        },
    })
    
    // 规划 Agent
    planner, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "planner",
        Description: "Create travel plans",
        Instruction: "Break down travel requests into search tasks.",
        Model:       model,
    })
    
    // 重规划 Agent
    replanner, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "replanner",
        Description: "Evaluate and adjust plans",
        Instruction: "Evaluate results and replan if needed.",
        Model:       model,
    })
    
    // 组装 Plan-Execute-Replan
    agent, _ := prebuilt.NewPlanExecuteReplan(ctx, &prebuilt.PlanExecuteReplanConfig{
        Planner:   planner,
        Executor:  executor,
        Replanner: replanner,
    })
    
    // 运行
    runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
    iter := runner.Query(ctx, "Plan a 3-day trip to Beijing from Shanghai")
    
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
# Chain/Graph 编排指南

Eino 提供三种编排方式：Chain、Graph、Workflow，用于组合多个组件完成复杂任务。

## 编排方式对比

| 方式 | 特点 | 使用场景 |
|------|------|----------|
| Chain | 线性链式，只能向前推进 | 简单线性流程 |
| Graph | 有向图，支持分支和循环 | 复杂分支、状态管理 |
| Workflow | 有向无环图，支持字段映射 | 复杂数据流转 |

## Chain 编排

### 基本使用

```go
import "github.com/cloudwego/eino/compose"

chain, err := compose.NewChain[map[string]any, *schema.Message]().
    AppendChatTemplate(prompt).
    AppendChatModel(model).
    Compile(ctx)

result, err := chain.Invoke(ctx, map[string]any{
    "query": "Hello",
})
```

### 添加节点类型

```go
chain := compose.NewChain[Input, Output]().
    AppendLambda(myLambda).                    // 自定义函数
    AppendChatTemplate(template).              // Prompt 模板
    AppendChatModel(model).                    // ChatModel
    AppendToolsNode(toolsNode).                // Tools 节点
    AppendRetriever(retriever).                // 检索器
    AppendPassthrough().                       // 透传
    AppendParallel(parallelNodes...).          // 并行节点
```

### Lambda 节点

```go
// 可调用 Lambda
lambda := compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return "processed: " + input, nil
})

// 流式 Lambda
streamLambda := compose.StreamableLambda(
    func(ctx context.Context, input string) (string, error) {
        return "result", nil
    },
    func(ctx context.Context, input *schema.StreamReader[string]) (*schema.StreamReader[string], error) {
        return input, nil // 透传
    },
)

chain := compose.NewChain[string, string]().
    AppendLambda(lambda).
    Compile(ctx)
```

### 并行节点

```go
chain := compose.NewChain[Input, []string]().
    AppendParallel(
        compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
            return "path1: " + input, nil
        }),
        compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
            return "path2: " + input, nil
        }),
    ).
    Compile(ctx)

result, _ := chain.Invoke(ctx, "test") // []string{"path1: test", "path2: test"}
```

## Graph 编排

### 基本使用

```go
graph := compose.NewGraph[map[string]any, *schema.Message]()

// 添加节点
_ = graph.AddChatTemplateNode("template", chatTpl)
_ = graph.AddChatModelNode("model", chatModel)
_ = graph.AddLambdaNode("converter", lambda)

// 添加边
_ = graph.AddEdge(compose.START, "template")
_ = graph.AddEdge("template", "model")
_ = graph.AddEdge("model", "converter")
_ = graph.AddEdge("converter", compose.END)

// 编译
compiled, _ := graph.Compile(ctx)
result, _ := compiled.Invoke(ctx, input)
```

### 分支

```go
func branchFunc(ctx context.Context, input *schema.Message) (string, error) {
    // 返回下一个节点名称
    if strings.Contains(input.Content, "weather") {
        return "weather_node", nil
    }
    return "general_node", nil
}

_ = graph.AddBranch("model", compose.NewGraphBranch(branchFunc, map[string]bool{
    "weather_node": true,
    "general_node": true,
}))
```

### 条件边

```go
_ = graph.AddConditionalEdge("model", func(ctx context.Context, msg *schema.Message) (string, error) {
    if len(msg.ToolCalls) > 0 {
        return "tools", nil
    }
    return compose.END, nil
})
```

### 循环

```go
// 从 tools 回到 model 形成循环
_ = graph.AddEdge("tools", "model")
```

### State Graph

带状态的 Graph：

```go
type MyState struct {
    Messages []*schema.Message
    Count    int
}

graph := compose.NewGraph[Input, Output](
    compose.WithStateType(&MyState{}),
)

// 使用 StatePreHandler 在节点执行前处理状态
_ = graph.AddLambdaNode("node1", lambda,
    compose.WithStatePreHandler(func(ctx context.Context, state *MyState, input Input) (Input, error) {
        state.Count++
        return input, nil
    }),
)
```

## Workflow 编排

### 基本使用

```go
import "github.com/cloudwego/eino/compose"

workflow := compose.NewWorkflow[map[string]any, map[string]any]()

// 添加节点
_ = workflow.AddLambdaNode("step1", lambda1)
_ = workflow.AddLambdaNode("step2", lambda2)
_ = workflow.AddLambdaNode("step3", lambda3)

// 设置字段映射
_ = workflow.SetInputKey("input")
_ = workflow.SetOutputKey("result")

// 连接节点
_ = workflow.AddEdge(compose.START, "step1")
_ = workflow.AddEdge("step1", "step2")
_ = workflow.AddEdge("step2", "step3")
_ = workflow.AddEdge("step3", compose.END)

compiled, _ := workflow.Compile(ctx)
result, _ := compiled.Invoke(ctx, map[string]any{"input": "data"})
```

### 字段映射

```go
// step1 的 output1 映射到 step2 的 input
_ = workflow.AddFieldMapping("step1", "step2", map[string]string{
    "output1": "input",
})

// 多字段映射
_ = workflow.AddFieldMapping("step1", "step2", map[string]string{
    "output1": "input1",
    "output2": "input2",
})
```

### 静态值

```go
_ = workflow.AddStaticValue("step1", "config", map[string]any{
    "mode": "production",
    "debug": false,
})
```

### 控制流分支

```go
// 条件分支
_ = workflow.AddControlBranch("router", func(ctx context.Context, input map[string]any) ([]string, error) {
    if input["type"] == "A" {
        return []string{"pathA"}, nil
    }
    return []string{"pathB"}, nil
})
```

## 流处理

### 流范式

| 方法 | 输入 | 输出 |
|------|------|------|
| Invoke | 非流 I | 非流 O |
| Stream | 非流 I | StreamReader[O] |
| Collect | StreamReader[I] | 非流 O |
| Transform | StreamReader[I] | StreamReader[O] |

### 流式调用

```go
// Chain 流式
sr, _ := chain.Stream(ctx, input)
defer sr.Close()

for {
    msg, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    fmt.Print(msg.Content)
}

// Graph 流式
sr, _ := graph.Stream(ctx, input)
```

### 流转换

```go
transformLambda := compose.TransformableLambda(
    nil, // Invoke
    func(ctx context.Context, sr *schema.StreamReader[string]) (*schema.StreamReader[string], error) {
        // 转换流内容
        return schema.StreamReaderMap(sr, func(s string) string {
            return "prefix: " + s
        }), nil
    },
)
```

## CallOption

### 全局选项

```go
result, _ := chain.Invoke(ctx, input,
    compose.WithCallbacks(handler),
    compose.WithTimeout(30*time.Second),
)
```

### 组件类型选项

```go
result, _ := chain.Invoke(ctx, input,
    compose.WithChatModelOption(model.WithTemperature(0.7)),
    compose.WithToolsNodeOption(tool.WithMaxConcurrency(5)),
)
```

### 节点级别选项

```go
_ = graph.AddChatModelNode("model", model,
    compose.WithNodeName("llm_node"),
    compose.WithNodeCallbacks(nodeHandler),
)
```

## Callback

### 回调类型

- OnStart
- OnEnd
- OnError
- OnStartWithStreamInput
- OnEndWithStreamOutput

### 使用回调

```go
import "github.com/cloudwego/eino/callbacks"

handler := callbacks.NewHandlerBuilder().
    OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
        fmt.Printf("Start: %s\n", info.Name)
        return ctx
    }).
    OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
        fmt.Printf("End: %s\n", info.Name)
        return ctx
    }).
    Build()

result, _ := chain.Invoke(ctx, input, compose.WithCallbacks(handler))
```

## 完整示例：RAG Chain

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()
    
    // 创建 RAG Chain
    chain, _ := compose.NewChain[map[string]any, *schema.Message]().
        // 1. 检索相关文档
        AppendRetriever(retriever).
        // 2. 合并检索结果到 prompt
        AppendLambda(compose.InvokableLambda(
            func(ctx context.Context, docs []*schema.Document) (string, error) {
                context := ""
                for _, doc := range docs {
                    context += doc.Content + "\n"
                }
                return context, nil
            },
        )).
        // 3. 使用 ChatModel 生成答案
        AppendChatModel(model).
        Compile(ctx)
    
    // 调用
    result, _ := chain.Invoke(ctx, map[string]any{
        "query": "什么是 Eino 框架？",
    })
    
    fmt.Println(result.Content)
}
```

## 完整示例：带分支的 Graph

```go
package main

import (
    "context"
    "strings"
    
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()
    
    graph := compose.NewGraph[map[string]any, *schema.Message]()
    
    // 添加节点
    _ = graph.AddChatTemplateNode("prompt", promptTpl)
    _ = graph.AddChatModelNode("model", model)
    _ = graph.AddToolsNode("tools", toolsNode)
    _ = graph.AddLambdaNode("finish", compose.InvokableLambda(
        func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
            return msg, nil
        },
    ))
    
    // 添加边
    _ = graph.AddEdge(compose.START, "prompt")
    _ = graph.AddEdge("prompt", "model")
    
    // 条件分支：判断是否需要调用工具
    _ = graph.AddBranch("model", compose.NewGraphBranch(
        func(ctx context.Context, msg *schema.Message) (string, error) {
            if len(msg.ToolCalls) > 0 {
                return "tools", nil
            }
            return "finish", nil
        },
        map[string]bool{"tools": true, "finish": true},
    ))
    
    // 工具调用后回到 model
    _ = graph.AddEdge("tools", "model")
    _ = graph.AddEdge("finish", compose.END)
    
    // 编译运行
    compiled, _ := graph.Compile(ctx)
    result, _ := compiled.Invoke(ctx, map[string]any{"query": "Hello"})
}
```
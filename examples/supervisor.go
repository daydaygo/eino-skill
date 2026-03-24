package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func main() {
	ctx := context.Background()

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})

	// 创建计算工具
	addTool, _ := utils.InferTool(
		"add",
		"Add two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required"`
			B int `json:"b" jsonschema:"required"`
		}) (string, error) {
			return fmt.Sprintf("%d + %d = %d", input.A, input.B, input.A+input.B), nil
		},
	)

	multiplyTool, _ := utils.InferTool(
		"multiply",
		"Multiply two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required"`
			B int `json:"b" jsonschema:"required"`
		}) (string, error) {
			return fmt.Sprintf("%d * %d = %d", input.A, input.B, input.A*input.B), nil
		},
	)

	// 创建子 Agents
	mathAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_agent",
		Description: "Perform mathematical calculations",
		Instruction: "You are a math specialist. Perform calculations using the available tools.",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{addTool, multiplyTool},
			},
		},
	})

	researchAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "Search and gather information",
		Instruction: "You are a research specialist. Help users find information.",
		Model:       model,
	})

	// 创建 Supervisor
	supervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "Route tasks to specialists",
		Instruction: "You are a supervisor. Route tasks to the appropriate specialist agent.",
		Model:       model,
	})

	// 组装 Supervisor 模式
	agent, _ := prebuilt.NewSupervisor(ctx, &prebuilt.SupervisorConfig{
		Supervisor: supervisor,
		SubAgents:  []adk.Agent{mathAgent, researchAgent},
	})

	// 运行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	iter := runner.Query(ctx, "What is 15 + 27? And what is 8 * 9?")

	fmt.Println("=== Supervisor Response ===")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("[Transfer] -> %s\n", event.Action.TransferToAgent.DestAgentName)
		}

		if msg, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}
	fmt.Println("\n=== End ===")
}

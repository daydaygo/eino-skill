# Eino Skill

Eino 框架 AI 编码助手 - 帮助开发者使用 CloudWeGo Eino 框架构建 LLM 应用。

## 简介

[Eino['aino]](https://github.com/cloudwego/eino) 是 CloudWeGo 开源的 Go 语言大模型应用开发框架。本 Skill 为 AI 编码助手提供 Eino 框架的开发指导，包括：

- **组件使用**：ChatModel、Tool、Retriever、Embedding 等
- **Agent 开发**：ReAct Agent、ChatModelAgent、多 Agent 协作
- **编排能力**：Chain、Graph、Workflow
- **流式处理**：Stream、StreamReader
- **人机协作**：Interrupt、Resume、Checkpoint

## 安装

### 方式一：作为 AI 编码助手 Skill 使用

将本项目放置在 AI 编码助手的 skills 目录下：

```bash
# Claude Code
cp -r eino-skill ~/.claude/skills/

# OpenCode
cp -r eino-skill ~/.agents/skills/
```

### 方式二：作为参考文档使用

直接浏览 `references/` 目录下的文档获取开发指导。

## 目录结构

```
eino-skill/
├── SKILL.md              # Skill 主文件（AI 编码助手加载）
├── README.md             # 本文件
├── examples/             # 示例代码
│   ├── .env.example      # 环境变量配置
│   ├── chatmodel-agent-basic.go   # ChatModelAgent 基础示例
│   ├── react-agent-basic.go       # ReAct Agent 基础示例
│   └── supervisor.go              # Supervisor 多 Agent 示例
└── references/           # 详细参考文档
    ├── react-agent.md    # ReAct Agent 完整指南
    ├── adk-agent.md      # ADK Agent 详细说明
    ├── multi-agent.md    # 多 Agent 协作模式
    ├── tools.md          # Tool 创建完整指南
    └── orchestration.md  # Chain/Graph 编排
```

## 快速开始

### 环境配置

```bash
# 复制环境变量模板
cp examples/.env.example .env

# 编辑配置
vim .env
```

配置说明：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `OPENAI_BASE_URL` | API 地址 | `https://api.openai.com/v1` |
| `OPENAI_API_KEY` | API 密钥 | 无 |
| `OPENAI_MODEL` | 模型名称 | 无 |

### 依赖安装

```bash
# 核心库
go get github.com/cloudwego/eino@latest

# OpenAI 模型支持
go get github.com/cloudwego/eino-ext/components/model/openai@latest

# 其他模型支持
go get github.com/cloudwego/eino-ext/components/model/ark@latest      # 字节火山引擎
go get github.com/cloudwego/eino-ext/components/model/ollama@latest   # Ollama 本地模型
```

### 示例运行

```bash
# 运行 ReAct Agent 示例
cd examples
go run react-agent-basic.go

# 运行 ChatModelAgent 示例
go run chatmodel-agent-basic.go

# 运行 Supervisor 示例
go run supervisor.go
```

## 核心概念

### 选择 Agent 类型

| 场景 | 推荐 | 说明 |
|------|------|------|
| 简单 Tool 调用 | ReAct Agent | 基础 ReAct 循环，适合大多数场景 |
| 需要 Runner/事件流 | ChatModelAgent | ADK 提供 Runner、事件流、状态管理 |
| 多 Agent 协作 | Supervisor | 层级协调，子 Agent 执行后回到 Supervisor |
| 复杂任务规划 | Plan-Execute | 计划-执行-重规划循环 |
| 线性工作流 | SequentialAgent | 按顺序执行多个 Agent |
| 并行任务 | ParallelAgent | 并发执行多个 Agent |

### ReAct Agent vs ChatModelAgent

**ReAct Agent** (`flow/agent/react`):
- 基于 Graph 编排
- 直接调用 Generate/Stream
- 适合简单场景

**ChatModelAgent** (`adk`):
- ADK 框架提供
- 通过 Runner 运行
- 支持事件流、状态管理、中断恢复
- 支持多 Agent 协作

## 示例对照表

本 Skill 示例与 [eino-examples](https://github.com/cloudwego/eino-examples) 官方仓库对照：

| eino-skill 示例 | eino-examples 对应 | 说明 |
|-----------------|-------------------|------|
| react-agent-basic.go | [flow/agent/react](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react) | ReAct Agent 基础用法 |
| chatmodel-agent-basic.go | [adk/intro/chatmodel](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/chatmodel) | ChatModelAgent 基础用法 |
| supervisor.go | [adk/multiagent/supervisor](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/supervisor) | Supervisor 多 Agent 协作 |

## 参考文档

按需阅读 `references/` 目录下的详细文档：

- **react-agent.md** - ReAct Agent 配置选项、调用方式、运行时选项
- **adk-agent.md** - ADK Agent 类型、Runner 运行、中断恢复、Middleware
- **multi-agent.md** - Supervisor、Plan-Execute-Replan、层级嵌套、AgentAsTool
- **tools.md** - Tool 创建方式、参数定义、错误处理
- **orchestration.md** - Chain/Graph/Workflow 编排详解

## 相关链接

- **Eino 核心**: https://github.com/cloudwego/eino
- **Eino 扩展**: https://github.com/cloudwego/eino-ext
- **示例代码**: https://github.com/cloudwego/eino-examples
- **官方文档**: https://www.cloudwego.io/zh/docs/eino/
- **API 文档**: https://pkg.go.dev/github.com/cloudwego/eino

## 许可证

Apache 2.0
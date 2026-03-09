# StudyClaw

StudyClaw 是一个面向家庭学习场景的任务协同项目：家长粘贴老师原始作业文本，系统把它拆成可执行的原子任务，孩子在 Pad 端按自己的节奏完成，后端同步保存进度并生成周度观察。

自 `2026-03-09` 起，运行态后端已经收敛为 `Go` 单体后端。`Flutter` 继续负责移动端，`React + Vite` 继续负责家长端。原先的 Python `agent-core` 已完成迁移并从仓库中移除。

## 当前版本

- 版本号: `v0.1.1`
- 状态: `MVP / 联调基线版`
- 当前真实后端: `apps/api-server`

## 当前能力

- 家长端输入学校群原始文本并调用 `POST /api/v1/tasks/parse`
- 后端执行 `LLM 优先 + 规则兜底` 的任务拆解
- 家长确认后创建 `学科 -> 作业分组 -> 原子任务`
- 任务写入 Markdown 工作区并支持按天查询
- Pad 端支持单个任务、分组、学科、全部任务的完成同步
- 周报接口支持基于近 7 天任务数据生成摘要

## 仓库结构

```text
studyclaw/
├── apps/
│   ├── api-server/      # Go 后端，唯一运行态后端
│   ├── parent-web/      # React + Vite 家长端
│   ├── pad-app/         # Flutter 孩子端 / Pad 端
├── data/                # Markdown 工作区，运行后自动生成
├── docs/
├── scripts/
└── .env.example
```

`apps/api-server` 采用 Go 常见大项目目录组织：

- `cmd/studyclaw-server`: 官方启动入口
- `internal/app`: 依赖装配
- `internal/interfaces/http`: HTTP 路由与 Handler
- `internal/modules/taskboard`: 任务域、应用服务、Markdown 仓储
- `internal/modules/agent/taskparse`: 作业解析 Agent 模块
- `internal/modules/agent/weeklyinsights`: 周报 Agent 模块
- `internal/platform/llm`: OpenAI 兼容 LLM 客户端，已适配 Ark Base URL
- `internal/shared/agentic`: Agentic pattern 元数据

## 运行前提

- `Go 1.25+`
- `Node.js 20+`
- `npm 10+`
- `Flutter 3.24+`
- `Docker` 可选

说明：当前运行链路不再要求 Python。

## 安全配置

推荐先生成仓库外的私有运行时配置文件：

```bash
bash scripts/init_private_runtime_env.sh
```

默认生成路径：

```text
~/.config/studyclaw/runtime.env
```

如果接入字节火山 Ark，请在私有 `runtime.env` 中配置：

```env
API_PORT=8080
LLM_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
LLM_API_KEY=你的 Ark Key
LLM_MODEL_NAME=你的模型名
LLM_PARSER_MODEL_NAME=可选，专用于作业解析
LLM_WEEKLY_MODEL_NAME=可选，专用于周报分析
STUDYCLAW_DATA_DIR=./data
```

运行时配置优先级：

1. 进程环境变量
2. 仓库外 `runtime.env`
3. 仓库根目录 `.env`

安全原则：

- 真实密钥只放仓库外 `runtime.env`
- 浏览器端和 Flutter 端不持有后端密钥
- Git 仓库中的 `.env.example` 只保留示例，不放真实账号密码

## 本地启动

### 0. 先做环境预检

```bash
bash scripts/preflight_local_env.sh
```

如果预检失败，先修环境，不要直接进入联调。

### 1. 可选：启动 Redis

```bash
docker compose up -d redis
```

说明：

- 当前演示链路默认不依赖 Redis
- 若后续启用 Redis 或相关缓存能力，再启动这一项

### 2. 启动 Go 后端

```bash
cd apps/api-server
go run ./cmd/studyclaw-server
```

健康检查：

```bash
curl http://localhost:8080/ping
```

预期返回：

```json
{"message":"pong"}
```

### 3. 启动家长端

```bash
cd apps/parent-web
npm install
npm run dev -- --host 0.0.0.0
```

默认地址：`http://localhost:5173`

### 4. 启动 Pad 端

```bash
cd apps/pad-app
flutter pub get
flutter run --dart-define=API_BASE_URL=http://localhost:8080 -d chrome
```

如果是真机联调，把 `localhost` 改成 Mac 的局域网 IP。

### 5. 最小 smoke 检查

在 Go 后端已启动的前提下执行：

```bash
bash scripts/smoke_local_stack.sh
```

默认会检查：

- 运行时密钥文件是否仍与仓库分离
- Go 后端 `/ping`
- 最小任务板 API 返回
- Parent Web 构建
- Pad Web 构建

## 如何添加某一天的任务

### 方案一：手动新增一条任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "subject": "数学",
    "group_title": "口算本",
    "content": "完成第12页",
    "assigned_date": "2026-03-10"
  }'
```

### 方案二：先解析，再人工确认后写入

先解析但不自动写入：

```bash
curl -X POST http://localhost:8080/api/v1/tasks/parse \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "assigned_date": "2026-03-10",
    "auto_create": false,
    "raw_text": "数学：1、校本P16～17\n2、练习册P14～15\n\n英语：1、背默M1U2单词\n2、预习课文"
  }'
```

把返回结果里的 `tasks` 数组确认后再提交：

```bash
curl -X POST http://localhost:8080/api/v1/tasks/confirm \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "assigned_date": "2026-03-10",
    "tasks": [
      {
        "subject": "数学",
        "group_title": "校本P16～17",
        "title": "校本P16～17"
      },
      {
        "subject": "数学",
        "group_title": "练习册P14～15",
        "title": "练习册P14～15"
      }
    ]
  }'
```

### 方案三：解析后直接写入某一天

```bash
curl -X POST http://localhost:8080/api/v1/tasks/parse \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "assigned_date": "2026-03-10",
    "auto_create": true,
    "raw_text": "数学：1、校本P16～17\n2、练习册P14～15\n\n英语：1、背默M1U2单词\n2、预习课文"
  }'
```

### 查看某一天任务板

```bash
curl "http://localhost:8080/api/v1/tasks?family_id=306&user_id=1&date=2026-03-10"
```

Markdown 文件默认位置：

```text
data/workspaces/family_306/user_1/2026-03-10.md
```

## Agentic 设计选型

当前实现严格采用“先确定性逻辑，再有限使用 LLM”的方式，而不是多智能体堆叠。

- `taskparse`: `custom logic pattern` 为主，辅以 `single-agent system` 和 `human-in-the-loop pattern`
- `weeklyinsights`: `single-agent system` 为主，辅以 `custom logic pattern`
- 任务板读写与状态同步: 纯确定性服务，不使用 Agent

参考设计指南：

- `https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system`

## 验证命令

### Go 后端

```bash
cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" go test ./...
```

### 家长端

```bash
cd apps/parent-web
npm run test
npm run build
```

### Pad 端

```bash
cd apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

## 文档索引

- 架构说明: [docs/02_ARCHITECTURE.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/02_ARCHITECTURE.md)
- Agentic 设计: [docs/04_AGENTIC_DESIGN.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/04_AGENTIC_DESIGN.md)
- 运行手册: [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- 安全说明: [docs/07_SECURITY.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/07_SECURITY.md)
- 并行开发分组: [docs/08_PARALLEL_WORKSTREAMS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/08_PARALLEL_WORKSTREAMS.md)
- 版本计划: [docs/03_ROADMAP.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/03_ROADMAP.md)

# StudyClaw

StudyClaw 是一个面向上班族家庭的儿童学习 AI 项目。当前仓库已经交付 `v0.1.0` MVP：家长把学校群原始作业粘贴进来，Agent Core 结合 LLM 做解析，家长确认后生成按学科和作业分组的原子任务，孩子可以在 Pad 端自主选择任务并同步完成进度。

## 当前版本

- 版本号: `v0.1.0`
- 发布日期: `2026-03-09`
- 当前定位: `MVP / 内部演示版`

## 当前已交付能力

- 家长端粘贴学校群原始任务文本，支持 AI 解析和确认创建
- Agent Core 采用 `LLM 优先 + 规则兜底` 的混合解析
- 任务以 `学科 -> 作业分组 -> 原子任务` 的结构存储到 Markdown 工作区
- Pad 端支持单个任务、作业分组、学科、全部任务的完成同步
- Pad 端默认让孩子自主选择未完成任务，系统只跟踪进度，不强制排序
- 周报分析接口和基础积分接口已预留

## 仓库结构

```text
studyclaw/
├── apps/
│   ├── agent-core/      # Python FastAPI，负责作业解析和周报分析
│   ├── api-server/      # Go Gin，负责任务创建、任务板和状态同步
│   ├── parent-web/      # React + Vite，家长任务输入与确认台
│   └── pad-app/         # Flutter，孩子任务同步台
├── data/                # 本地 Markdown 工作区，运行后自动生成
├── docs/
│   ├── 01_PRD.md
│   ├── 02_ARCHITECTURE.md
│   ├── 03_ROADMAP.md
│   ├── 04_AGENTIC_DESIGN.md
│   ├── 05_HEALTH_&_PSYCHOLOGY.md
│   └── 06_RUNBOOK.md
├── CHANGELOG.md
├── docker-compose.yml
└── .env.example
```

## 本地运行前提

- macOS 或 Linux
- `Go 1.25+`
- `Python 3.10+`
- `Node.js 20+`
- `Flutter 3.24+`
- `Docker` 与 `Docker Compose`

## 快速开始

### 1. 准备环境变量

在仓库根目录执行：

```bash
cp .env.example .env
```

如果你要启用真实 LLM，请在 `.env` 中填写：

```env
LLM_API_KEY=你的真实 Key
LLM_BASE_URL=https://api.openai.com/v1
```

如果不填，系统仍可运行，但 `agent-core` 会自动走规则兜底解析。

### 2. 启动基础依赖

当前 `v0.1.0` 只需要 Redis：

```bash
docker compose up -d redis
```

### 3. 启动 Agent Core

```bash
cd apps/agent-core
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python3 main.py
```

默认地址：

- `http://localhost:8000/ping`
- `http://localhost:8000/api/v1/internal/parse`

### 4. 启动 API Server

```bash
cd apps/api-server
go run .
```

默认地址：

- `http://localhost:8080/ping`
- `http://localhost:8080/api/v1/tasks`

API Server 会读取根目录 `.env`，并把任务写入 `data/workspaces/`。

### 5. 启动家长端

```bash
cd apps/parent-web
npm install
npm run dev -- --host 0.0.0.0
```

默认访问地址：

- `http://localhost:5173`

说明：

- 默认 API 地址是 `http://localhost:8080`
- API Server 已支持浏览器跨域联调

### 6. 启动 Pad 端

```bash
cd apps/pad-app
flutter pub get
flutter run --dart-define=API_BASE_URL=http://localhost:8080
```

真机或局域网环境请把 `localhost` 改成 Mac 所在机器的局域网 IP。

## 演示流程

建议直接使用 `2026-03-06` 的演示任务：

```text
数学3.6：
1、校本P14～15
2、练习册P12～13

英：
1. 背默M1U1知识梳理单小作文
2. 部分学生继续订正1号本
3. 预习M1U2
（1）书本上标注好“黄页”出现单词的音标
（2）抄写单词（今天默写全对，可免抄）
（3）沪学习听录音跟读

语文：
1. 背作文
2. 练习卷
```

对应流程：

1. 在家长端粘贴原文并执行 AI 解析
2. 审核并确认创建任务
3. 打开 Pad 端，加载 `2026-03-06` 的任务板
4. 孩子按自己的节奏勾选完成任务
5. 家长端和 Markdown 文件可同步看到进度变化

## Markdown 数据位置

任务文件默认写入：

```text
data/workspaces/family_<family_id>/user_<user_id>/<date>.md
```

例如：

```text
data/workspaces/family_306/user_1/2026-03-06.md
```

## 验证命令

### Agent Core

```bash
cd apps/agent-core
python3 -m unittest discover -s tests
python3 -m py_compile main.py api/routes.py services/llm_parser.py services/weekly_analyst.py tests/test_llm_parser.py
```

### API Server

```bash
cd apps/api-server
GOCACHE=../.gocache GOMODCACHE=../.modcache go test ./...
```

### Parent Web

```bash
cd apps/parent-web
npm run build
```

### Pad App

```bash
cd apps/pad-app
flutter analyze
flutter test
```

## 文档索引

- 架构说明: [docs/02_ARCHITECTURE.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/02_ARCHITECTURE.md)
- 迭代计划: [docs/03_ROADMAP.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/03_ROADMAP.md)
- 本地运行手册: [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- 版本说明: [CHANGELOG.md](/Users/admin/Documents/WORK/ai/studyclaw/CHANGELOG.md)

# StudyClaw 本地运行手册

本文档描述 `2026-03-12` 交付基线下的真实运行方式。当前正式运行形态是 `Go API + React Parent Web + Flutter Pad` 三端协同，不再需要额外的 Python 后端进程。

## 1. 前置条件

- `Go 1.25+`
- `Node.js 20+`
- `npm 10+`
- `Flutter 3.24+`
- `Docker` 可选

## 2. 环境预检

推荐先执行：

```bash
bash scripts/preflight_local_env.sh
```

预检会检查：

- Go / Node / npm / Flutter / Docker
- Docker Compose 与 daemon
- 私有 `runtime.env` 是否存在
- 关键目录是否齐全
- 仓库中是否误跟踪了运行时密钥文件

若失败，先修环境，再继续。

## 3. 运行时配置

推荐先创建私有配置文件：

```bash
bash scripts/init_private_runtime_env.sh
```

默认路径：

```text
~/.config/studyclaw/runtime.env
```

示例：

```env
API_PORT=8080
LLM_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
LLM_API_KEY=your-llm-api-key
LLM_MODEL_NAME=your-llm-model
LLM_PARSER_MODEL_NAME=
LLM_GRADER_MODEL_NAME=
LLM_WEEKLY_MODEL_NAME=
LLM_HTTP_TIMEOUT_SECONDS=90
STUDYCLAW_DATA_DIR=./data
STUDYCLAW_LOG_DIR=./data/logs
```

加载顺序：

1. 进程环境变量
2. 私有 `runtime.env`
3. 仓库根目录 `.env`

注意：

- 真实密钥只放仓库外 `runtime.env`
- 不配置 `LLM_API_KEY` 也能启动；任务解析会自动回退到规则模式

## 4. 标准三端启动顺序

### 4.1 启动 API

```bash
cd apps/api-server
API_PORT=38080 go run ./cmd/studyclaw-server
```

健康检查：

```bash
curl http://127.0.0.1:38080/ping
```

预期：

```json
{"message":"pong"}
```

### 4.2 启动 Parent Web

```bash
cd apps/parent-web
VITE_API_BASE_URL=http://127.0.0.1:38080 npm run dev -- --host 127.0.0.1 --port 5173
```

地址：`http://127.0.0.1:5173`

### 4.3 启动 Pad Web

```bash
cd apps/pad-app
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

地址：`http://127.0.0.1:55771`

真机联调请把 `127.0.0.1` 替换成宿主机局域网 IP。

## 5. 交付前标准联调路径

推荐固定使用以下数据，避免“今天 / 明天”混淆：

- `family_id=306`
- `user_id / child_id=1`
- `assigned_date=2026-03-12`

### 5.1 最短交付验证

```bash
bash scripts/preflight_local_env.sh
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 \
STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 \
bash scripts/demo_local_stack.sh
```

### 5.2 页面存活检查

```bash
curl http://127.0.0.1:5173/
curl http://127.0.0.1:55771/
```

### 5.3 家长发布作业

解析：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/tasks/parse \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "assigned_date": "2026-03-12",
    "auto_create": false,
    "raw_text": "数学：1、校本P16-17\n2、练习册P14-15\n\n英语：1、背默M1U2单词\n2、预习课文"
  }'
```

确认写入：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/tasks/confirm \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "assigned_date": "2026-03-12",
    "tasks": [
      { "subject": "数学", "group_title": "校本P16-17", "title": "校本P16-17" },
      { "subject": "数学", "group_title": "练习册P14-15", "title": "练习册P14-15" },
      { "subject": "英语", "group_title": "背默M1U2单词", "title": "背默M1U2单词" },
      { "subject": "英语", "group_title": "预习课文", "title": "预习课文" }
    ]
  }'
```

### 5.4 孩子端读取与完成任务

读取任务板：

```bash
curl "http://127.0.0.1:38080/api/v1/tasks?family_id=306&user_id=1&date=2026-03-12"
```

勾选一个任务完成：

```bash
curl -X PATCH http://127.0.0.1:38080/api/v1/tasks/status/item \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "assignee_id": 1,
    "task_id": 1,
    "completed": true,
    "assigned_date": "2026-03-12"
  }'
```

### 5.5 家长端查看反馈和积分

```bash
curl "http://127.0.0.1:38080/api/v1/stats/daily?family_id=306&user_id=1&date=2026-03-12"
curl "http://127.0.0.1:38080/api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03"
```

创建人工奖励：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/points/ledger \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "user_id": 1,
    "delta": 2,
    "source_type": "parent_reward",
    "occurred_on": "2026-03-12",
    "note": "主动完成额外练习"
  }'
```

### 5.6 词单与听写会话

解析词单：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/word-lists/parse \
  -H 'Content-Type: application/json' \
  -d '{"raw_text":"apple 苹果\norange 橙子\nbanana 香蕉"}'
```

保存词单：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/word-lists \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "child_id": 1,
    "assigned_date": "2026-03-12",
    "title": "英语默写 Day 1",
    "language": "en",
    "items": [
      { "text": "apple", "meaning": "苹果" },
      { "text": "orange", "meaning": "橙子" },
      { "text": "banana", "meaning": "香蕉" }
    ]
  }'
```

启动会话：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/dictation-sessions/start \
  -H 'Content-Type: application/json' \
  -d '{
    "family_id": 306,
    "child_id": 1,
    "assigned_date": "2026-03-12"
  }'
```

## 6. 当前交付文档入口

- 发布前检查：`docs/13_RELEASE_CHECKLIST.md`
- 第一阶段演示清单：`docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- 交付就绪审计：`docs/17_DELIVERY_READINESS.md`
- 交付验收用例：`docs/19_DELIVERY_UAT_CASES.md`

## 7. 标准验证命令

### 7.1 Go 后端

```bash
cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" go test ./... -count=1
```

### 7.2 Parent Web

```bash
cd apps/parent-web
npm test
npm run build
```

### 7.3 Pad App

```bash
cd apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

## 8. 常见问题

### 8.1 `tasks/parse` 返回规则兜底

优先检查：

- `LLM_API_KEY` 是否配置在私有 `runtime.env`
- `LLM_MODEL_NAME` 或 `LLM_PARSER_MODEL_NAME` 是否配置
- `LLM_BASE_URL` 是否可达

### 8.2 Pad 端能打开但请求失败

优先检查：

- Go 后端是否监听在 `38080`
- Flutter 启动参数里的 `API_BASE_URL` 是否正确
- 真机联调时是否误用了 `localhost`

### 8.3 为什么 release 前还不能 push

优先检查：

- `git status --short` 里是否还有未计划提交内容
- 是否夹带 `.gopath/`、`build/`、`dist/`、`.dart_tool/` 等产物目录
- 是否已经按 `docs/13_RELEASE_CHECKLIST.md` 和 `docs/19_DELIVERY_UAT_CASES.md` 完成验证

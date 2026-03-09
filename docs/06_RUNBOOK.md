# StudyClaw 本地运行手册

本文档描述 `2026-03-09` 之后的当前真实运行方式：后端只启动 Go 服务，不再要求额外的 Python 后端进程。

## 1. 前置条件

- `Go 1.25+`
- `Node.js 20+`
- `npm 10+`
- `Flutter 3.24+`
- `Docker` 可选

## 2. 快速预检

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

若失败，先修环境，再继续后续步骤。

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
LLM_API_KEY=your-ark-api-key
LLM_MODEL_NAME=your-ark-model-name
LLM_PARSER_MODEL_NAME=
LLM_WEEKLY_MODEL_NAME=
STUDYCLAW_DATA_DIR=./data
```

加载顺序：

1. 进程环境变量
2. 私有 `runtime.env`
3. 仓库根目录 `.env`

注意：

- 真实密钥只放仓库外 `runtime.env`
- 不配置 `LLM_API_KEY` 也能启动，系统会回退到规则解析和 mock 周报

## 4. 启动顺序

### 4.1 可选：启动 Redis

```bash
docker compose up -d redis
```

说明：

- 当前演示链路不依赖 Redis
- 若后续引入缓存或 Redis 相关能力，再执行这一项

### 4.2 启动 Go 后端

```bash
cd apps/api-server
go run ./cmd/studyclaw-server
```

健康检查：

```bash
curl http://localhost:8080/ping
```

预期：

```json
{"message":"pong"}
```

### 4.3 启动 Parent Web

```bash
cd apps/parent-web
npm install
npm run dev -- --host 0.0.0.0
```

默认地址：`http://localhost:5173`

### 4.4 启动 Pad App

```bash
cd apps/pad-app
flutter pub get
flutter run --dart-define=API_BASE_URL=http://localhost:8080 -d chrome
```

真机请把 `localhost` 替换成宿主机局域网 IP。

## 5. 最短联调路径

在本地最短建议按这 5 步执行：

1. `bash scripts/preflight_local_env.sh`
2. `bash scripts/init_private_runtime_env.sh`
3. 启动 Go 后端
4. `bash scripts/smoke_local_stack.sh`
5. 若后续需要 Redis，再执行 `docker compose up -d redis`

说明：

- `smoke_local_stack.sh` 默认要求本地 `http://localhost:8080` 已有运行中的 Go 后端
- 如后端不是这个地址，可用环境变量覆盖：

```bash
STUDYCLAW_SMOKE_API_BASE_URL=http://your-host:8080 bash scripts/smoke_local_stack.sh
```

## 6. 常用联调方式

### 6.1 手动给某一天新增一条任务

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

### 6.2 先解析后确认

解析：

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

确认写入：

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

### 6.3 直接解析并写入某一天

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

### 6.4 查询某一天任务板

```bash
curl "http://localhost:8080/api/v1/tasks?family_id=306&user_id=1&date=2026-03-10"
```

默认 Markdown 位置：

```text
data/workspaces/family_306/user_1/2026-03-10.md
```

## 7. 验证命令

### 7.1 一键 smoke

```bash
bash scripts/smoke_local_stack.sh
```

### 7.2 Go 后端

```bash
cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" go test ./...
```

### 7.3 Parent Web

```bash
cd apps/parent-web
npm run test
npm run build
```

### 7.4 Pad App

```bash
cd apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

## 8. 常见问题

### 8.1 `tasks/parse` 返回规则兜底

优先检查：

- `LLM_API_KEY` 是否配置在私有 `runtime.env`
- `LLM_MODEL_NAME` 或 `LLM_PARSER_MODEL_NAME` 是否配置
- `LLM_BASE_URL` 是否是 `https://ark.cn-beijing.volces.com/api/v3`

### 8.2 Pad 端能打开但请求失败

优先检查：

- Go 后端是否监听在 `8080`
- Flutter 启动参数里的 `API_BASE_URL` 是否正确
- 真机联调时是否误用了 `localhost`

### 8.3 看不到任务文件

优先检查：

- `family_id`、`assignee_id`、`assigned_date` 是否一致
- `STUDYCLAW_DATA_DIR` 是否被改到别的目录
- 是否真的调用了 `confirm` 或 `parse + auto_create`

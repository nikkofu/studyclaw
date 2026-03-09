# StudyClaw 本地运行手册

本文档对应 `v0.1.0`，目标是从零启动一个可演示的本地环境。

## 1. 运行前准备

### 1.1 依赖版本

- `Go 1.25+`
- `Python 3.10+`
- `Node.js 20+`
- `npm 10+`
- `Flutter 3.24+`
- `Docker`

### 1.2 环境变量

在仓库根目录执行：

```bash
cp .env.example .env
```

关键变量：

```env
API_PORT=8080
AGENT_PORT=8000
AGENT_CORE_URL=http://localhost:8000
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-your-key
STUDYCLAW_DATA_DIR=./data
```

说明：

- 没有真实 `LLM_API_KEY` 也能跑，只是会退回规则解析
- `STUDYCLAW_DATA_DIR` 不填时默认写入仓库下的 `data/`

## 2. 启动顺序

建议固定按下面顺序启动。

### 2.1 启动 Redis

在仓库根目录：

```bash
docker compose up -d redis
docker compose ps
```

预期：

- `studyclaw_redis` 状态为 `running`

### 2.2 启动 Agent Core

```bash
cd apps/agent-core
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python3 main.py
```

验证：

```bash
curl http://localhost:8000/ping
```

预期返回：

```json
{"message":"Agent Core is alive"}
```

### 2.3 启动 API Server

新开一个终端：

```bash
cd apps/api-server
go run .
```

验证：

```bash
curl http://localhost:8080/ping
```

预期返回：

```json
{"message":"pong"}
```

### 2.4 启动 Parent Web

新开一个终端：

```bash
cd apps/parent-web
npm install
npm run dev -- --host 0.0.0.0
```

访问：

- `http://localhost:5173`

说明：

- 默认 API 地址已指向 `http://localhost:8080`
- API Server 已支持跨域访问

### 2.5 启动 Pad App

新开一个终端：

```bash
cd apps/pad-app
flutter pub get
flutter run --dart-define=API_BASE_URL=http://localhost:8080
```

说明：

- 模拟器可直接用 `localhost`
- 真机请改成 Mac 局域网 IP，例如 `http://192.168.1.10:8080`

## 3. 演示数据

推荐使用这段任务文本：

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

## 4. 演示步骤

### 4.1 家长端创建任务

1. 打开 `http://localhost:5173`
2. 粘贴上面的任务文本
3. 点击 AI 解析
4. 审核低置信度任务
5. 确认创建

### 4.2 Pad 端同步任务

1. 打开 Pad App
2. 保持默认：
   - `family_id=306`
   - `user_id=1`
   - `date=2026-03-06`
3. 点击加载任务板
4. 孩子选择任意任务开始勾选

### 4.3 Markdown 结果核对

创建后文件一般位于：

```text
data/workspaces/family_306/user_1/2026-03-06.md
```

全部完成后的内容示例：

```md
# 2026年03月06日 - 今日成长轨迹

## 🎯 任务清单

### 数学

#### 校本P14～15
- [x] 校本P14～15

#### 练习册P12～13
- [x] 练习册P12～13

### 英语

#### 背默M1U1知识梳理单小作文
- [x] 背默M1U1知识梳理单小作文

#### 部分学生继续订正1号本
- [x] 部分学生继续订正1号本

#### 预习M1U2
- [x] 书本上标注好“黄页”出现单词的音标
- [x] 抄写单词（今天默写全对，可免抄）
- [x] 沪学习听录音跟读

### 语文

#### 背作文
- [x] 背作文

#### 练习卷
- [x] 练习卷
```

## 5. 验证命令

### 5.1 Agent Core

```bash
cd apps/agent-core
python3 -m unittest discover -s tests
python3 -m py_compile main.py api/routes.py services/llm_parser.py services/weekly_analyst.py tests/test_llm_parser.py
```

### 5.2 API Server

```bash
cd apps/api-server
GOCACHE=../.gocache GOMODCACHE=../.modcache go test ./...
```

### 5.3 Parent Web

```bash
cd apps/parent-web
npm run build
```

### 5.4 Pad App

```bash
cd apps/pad-app
flutter analyze
flutter test
```

## 6. 常见问题

### 6.1 Parent Web 能打开但请求失败

先确认：

- `api-server` 是否启动在 `8080`
- 家长端页面里的 API 地址是否正确
- 真机或局域网调试是否误用了 `localhost`

### 6.2 没有真实 LLM Key

这是允许的。系统会继续运行，只是解析质量依赖规则兜底。

### 6.3 看不到任务文件

检查：

- 是否真正点击了“确认创建”
- `STUDYCLAW_DATA_DIR` 是否被改到了其他目录
- `family_id`、`user_id`、`date` 是否和查询时一致

# StudyClaw 本地运行手册

本文档描述 `2026-03-14` 的 `v0.3.5` 交付基线下的真实运行方式。当前正式运行形态是 `Go API + React Parent Web + Flutter Pad` 三端协同，不再需要额外的 Python 后端进程。

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

说明：

- 家长端当前主干已改为移动优先 H5 工位，桌面浏览器打开时也会使用手机单列宽度和底部固定导航。
- 家长端当前主干已把复杂功能拆成“四大主屏 + 子页面菜单”，发布、反馈、积分、单词都按 App 式切页组织，不再是长单页。
- 点击“去录入原文”会直接切到 `原文` 子页面，不再被空状态逻辑错误带回 `范围`。
- 背诵 / 朗读类任务在解析和审核阶段会自动带出学习素材字段；家长手动输入优先，缺失时先从老师原文抽取，再由 LLM 补全剩余空缺。
- 审核卡会标出学习素材来源：`手动录入 / 老师原文 / LLM 补全`；确认发布后，`reference_source` 会继续保留到任务板与 Pad 端。
- 真机访问时优先使用手机浏览器；如果用桌面浏览器联调，不需要再手动缩到很窄才能看到移动布局。

### 4.3 启动 Pad Web

```bash
cd apps/pad-app
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

地址：`http://127.0.0.1:55771`

真机联调请把 `127.0.0.1` 替换成宿主机局域网 IP。

Pad 语音助手说明：

- 任务板和听写页签共用 `/api/v1/voice-commands/resolve` 做语音意图解析。
- Web 首次使用语音助手时，浏览器会请求麦克风权限，需要点允许。
- 当前推荐使用 Chrome / Edge；如果浏览器不支持语音识别，Pad 会保留按钮但给出明确失败提示。
- `v0.3.5` 已支持短口令、长段朗读 / 背诵和陪伴式持续监听三种语音场景。
- `v0.3.5` 已修复 Web/STT 场景下“开始说话”后的启动判定与 `done / notListening` 收尾逻辑，并补上背诵分析主链。
- 当前主干进一步补强了连续监听对 `error_no_match`、短时静音等可恢复错误的自动续听；只要没有主动点“结束说话”，会优先保持陪伴式会话。
- 当前主干还把语音工作台拆成两层：`实时记录` 优先保留识别器捕获到的真实停顿分段；`背诵对照` 再按标准原文逐句分析，避免为了对照而覆盖掉真实开口节奏。
- 当天未配置词单时，Pad 不再直接显示 `404 / TaskApiException`，而是提示“等家长补充词单后再来默写”。
- 成长小鼓励支持自动语音播报、手动重播和自动播报开关；Pad 真机 / 平板也能走统一 TTS。
- 任务完成和听写关键节点会展示正向鼓励文案，用于孩子端即时反馈。

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

背诵 / 朗读学习素材自动补全：

- 如果家长在审核阶段已手动填写 `reference_title / reference_author / reference_text`，系统直接保留人工输入。
- 如果家长未填写，解析器会优先从老师原文中自动抽取古诗词 / 课文标题、作者和正文。
- 如果老师原文里只有“背诵《xxx》”而没有正文，且配置了可用 LLM，系统会只补全文本缺口，不覆盖家长已填内容。
- `reference_source` 会按 `manual / extracted / llm` 记录来源；家长端审核卡上对应显示为 `手动录入 / 老师原文 / LLM 补全`。
- 背诵任务默认 `hide_reference_from_child=true`，Pad 不直接展示标准原文，但可用来做背诵分析。
- 背诵分析规则兜底已支持“短前导语 + noisy transcript”场景：会先在前段窗口里识别标题/作者，再按参考原文的行句形态对整段 transcript 做切分和逐句比对。

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

语音完成同类动作：

- 在 Pad 任务板可以直接说“数学订正好了”“一课一练做完了”“全部都好了”。
- Pad 会先做 STT，再把 transcript 和当前页面上下文发给 `/api/v1/voice-commands/resolve`，再执行对应按钮动作。
- 单任务、分组、学科或全部完成后，Pad 会展示“成长小鼓励”卡片，并显示当前完成进度。

背诵分析：

```bash
curl -X POST http://127.0.0.1:38080/api/v1/recitation/analyze \
  -H 'Content-Type: application/json' \
  -d '{
    "scene": "recitation",
    "locale": "zh-CN",
    "transcript": "江畔独步寻花糖杜甫黄思塔前江水东春光缆会以微风",
    "reference_text": "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？"
  }'
```

预期：

- 返回 `reference_title`、`reference_author`、`completion_ratio`
- 返回逐句 `matched_lines`
- 能给出 `needs_retry`、`summary`、`suggestion`

### 5.5 家长端查看反馈和积分

```bash
curl "http://127.0.0.1:38080/api/v1/stats/daily?family_id=306&user_id=1&date=2026-03-12"
curl "http://127.0.0.1:38080/api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03"
```

家长端移动工位建议操作顺序：

1. 先用底部“发布”进入当日布置主线。
2. 在发布主屏内部继续切 `范围 / 原文 / 审核 / 发布 / 拆分 / 任务 / 摘要 / 任务板` 子页，完成当天布置。
3. 发布后切到“反馈”，再在 `日报 / 听写 / 趋势` 子页中查看问题与结果。
4. 需要奖惩时切到“积分”，在 `记一笔 / 看流水` 子页间切换，当天流水会和表单放在同一工位。
5. 需要给 Pad 更新默写内容时切到“单词”，在 `新建清单 / 已有清单` 子页间切换并直接编辑当前日期清单。

验收补充：

- 子页切换应带明显左右滑入滑出动效。
- 发布主屏的子菜单应在手机视口下保持可见和可点击，不需要滚回顶部重新找入口。
- 不能依赖横向拖动才能看到完整发布链路、预览、审核队列或积分原因建议。

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

听写语音指令示例：

- “好了” / “下一个” -> 下一词
- “重播” -> 重播当前词
- “上一个” -> 返回上一词

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

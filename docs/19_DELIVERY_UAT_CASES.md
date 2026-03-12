# StudyClaw v0.2.0 交付验收用例

本文档把当前阶段的交付验收步骤固定下来，作为 API、Parent Web、Pad 三端一起跑的正式基线。

## 1. 验收基线

- 验收日期：`2026-03-12`
- 版本：`v0.2.0`
- 固定数据：
  - `family_id=306`
  - `user_id / child_id=1`
  - `assigned_date=2026-03-12`
- 联调端口：
  - API：`http://127.0.0.1:38080`
  - Parent Web：`http://127.0.0.1:5173`
  - Pad Web：`http://127.0.0.1:55771`

## 2. 启动命令

```bash
# API
cd apps/api-server
API_PORT=38080 go run ./cmd/studyclaw-server

# Parent Web
cd apps/parent-web
VITE_API_BASE_URL=http://127.0.0.1:38080 npm run dev -- --host 127.0.0.1 --port 5173

# Pad Web
cd apps/pad-app
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

## 3. 用例清单

| 编号 | 范围 | 操作 / 命令 | 预期结果 |
| --- | --- | --- | --- |
| `UAT-01` | 环境 | `bash scripts/check_no_tracked_runtime_env.sh` | 没有被跟踪的运行时密钥文件 |
| `UAT-02` | 环境 | `bash scripts/preflight_local_env.sh` | Go / Node / npm / Flutter / 目录检查通过 |
| `UAT-02A` | 环境 | `bash scripts/check_release_scope.sh` | 只允许存在本次 release 范围内的改动；若失败，必须先清理噪音路径 |
| `UAT-03` | API | `curl http://127.0.0.1:38080/ping` | 返回 `{"message":"pong"}` |
| `UAT-04` | Parent Web | `curl http://127.0.0.1:5173/` | 返回 Parent Web HTML |
| `UAT-05` | Pad Web | `curl http://127.0.0.1:55771/` | 返回 Pad Web HTML |
| `UAT-06` | 家长发布 | `POST /api/v1/tasks/parse` | 成功解析 4 条任务，返回 `rule_fallback` 或 LLM 结果 |
| `UAT-07` | 家长发布 | `POST /api/v1/tasks/confirm` | 成功写入当天任务，`created_count=4` |
| `UAT-08` | 孩子读取 | `GET /api/v1/tasks?family_id=306&user_id=1&date=2026-03-12` | 返回 4 条任务和正确 summary |
| `UAT-09` | 孩子完成 | `PATCH /api/v1/tasks/status/item` | `updated_count=1`，summary 从 `0/4` 变为 `1/4` |
| `UAT-10` | 家长反馈 | `GET /api/v1/stats/daily` | 返回 `completed_tasks=1`、`auto_points=1` |
| `UAT-11` | 积分 | `POST /api/v1/points/ledger` | 成功写入一条人工奖励 |
| `UAT-12` | 积分 | `GET /api/v1/points/ledger` 与 `GET /api/v1/points/balance` | 返回自动积分 `1` + 人工积分 `2`，余额 `3` |
| `UAT-13` | 词单 | `POST /api/v1/word-lists/parse` | 返回结构化词项 |
| `UAT-14` | 词单 | `POST /api/v1/word-lists` | 成功保存 `wordlist_...` |
| `UAT-15` | 听写会话 | `POST /api/v1/dictation-sessions/start` | 返回 `session_...` 和当前单词 |
| `UAT-16` | 听写会话 | `POST /next`、`POST /replay`、`GET /api/v1/dictation-sessions` | 会话索引推进、重播可用、列表可查询 |
| `UAT-17` | 月统计 | `GET /api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03` | 返回任务、积分、词单、会话的月聚合 |
| `UAT-18` | 自动化 | `go test ./... -count=1` | 全部通过 |
| `UAT-19` | 自动化 | `npm test`、`npm run build` | 全部通过 |
| `UAT-20` | 自动化 | `flutter analyze`、`flutter test`、`flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080` | 全部通过，允许 wasm dry-run warning |
| `UAT-21` | 一键校验 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh` | smoke 全部通过 |
| `UAT-22` | 一键演示 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh` | 输出 demo walkthrough 且内部 smoke 通过 |

## 4. 推荐手工验收故事

1. 在 Parent Web 选择 `family_id=306`、`assignee_id=1`、`assigned_date=2026-03-12`。
2. 粘贴老师原文，点击“AI 解析任务”。
3. 审核草稿后确认发布。
4. 切到 Pad，使用同一天加载任务板。
5. 勾选 1 条任务完成。
6. 回到 Parent Web 刷新反馈，确认完成率同步更新。
7. 在 Parent Web 新增一条积分奖励，确认余额同步变化。
8. 在 Parent Web 创建 3 个词的词单。
9. 切到 Pad 启动听写，推进到下一词并重播。
10. 回到 Parent Web 查看月统计，确认任务、积分、词单和会话都被统计到。

## 5. 当前已验证结果

`2026-03-12` 已执行并通过的结果摘要：

- Parent Web 与 Pad Web 页面都能正常返回 HTML。
- API 成功完成 `parse -> confirm -> list -> status update -> daily/monthly stats`。
- 任务完成后自动积分 `+1`，家长人工奖励 `+2`，余额正确汇总为 `3`。
- 词单 `wordlist_000001` 保存成功。
- 听写会话 `session_000002` 成功执行 `start -> next -> replay`。
- `smoke_local_stack.sh` 与 `demo_local_stack.sh` 均通过。

## 6. GitHub 同步门槛

在正式开始下一阶段前，必须同时满足：

1. `git fetch origin` 已执行。
2. `git status --short` 中只剩计划提交的文件。
3. 不把 `.gopath/`、`build/`、`dist/`、`.dart_tool/`、运行时密钥文件带进 commit。
4. 根 README、运行手册、用户手册、release checklist、delivery readiness、UAT cases 已同步。
5. 版本声明已经对齐到 `v0.2.0`。
6. 自动化验证和三端联调结果已附在 release commit 或 PR 描述中。

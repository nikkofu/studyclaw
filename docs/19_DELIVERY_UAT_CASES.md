# StudyClaw v0.4.0 交付验收用例

本文档把当前阶段的交付验收步骤固定下来，作为 API、Parent Web、Pad 三端一起跑的正式基线。

## 1. 验收基线

- 验收日期：`2026-03-14`
- 版本：`v0.4.0`
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
| `UAT-07A` | 家长端 H5 | 用手机视口打开 Parent Web 并切换 `发布 / 反馈 / 积分 / 单词` | 页面表现为手机 H5 工位，底部导航始终可用，不是 PC 多栏长页面 |
| `UAT-07B` | 家长端 H5 | 在发布主屏切换 `范围 / 原文 / 审核 / 发布 / 拆分 / 任务 / 摘要 / 任务板` | 发布子页可以切换，默认不需要横向拖动才能看到完整入口 |
| `UAT-07C` | 家长端 H5 | 在 `范围` 页点击“去录入原文” | 立即切到 `原文` 子页面，并看到原文输入框，而不是被带回 `范围` |
| `UAT-07D` | 学习素材 | 用包含古诗词正文的老师原文调用 `POST /api/v1/tasks/parse` | 解析结果里包含 `reference_title`、`reference_author`、`reference_text`、`hide_reference_from_child=true`、`analysis_mode=classical_poem` |
| `UAT-07E` | 学习素材 | 发布草稿后检查 `daily_assignment_draft.task_items` | 草稿中仍保留学习素材元数据，不会在 parse -> draft 阶段丢失 |
| `UAT-07F` | 学习素材 | 以解析结果直接调用 `POST /api/v1/tasks/confirm` | 任务板中仍保留学习素材元数据，不会在 confirm 阶段丢失 |
| `UAT-07G` | 学习素材来源 | 在家长端审核卡分别测试手动录入、老师原文抽取、LLM 补全三种场景 | 审核卡显示 `手动录入 / 老师原文 / LLM 补全`；API / draft / task board 中 `reference_source` 分别为 `manual / extracted / llm` |
| `UAT-08` | 孩子读取 | `GET /api/v1/tasks?family_id=306&user_id=1&date=2026-03-12` | 返回 4 条任务和正确 summary |
| `UAT-09` | 孩子完成 | `PATCH /api/v1/tasks/status/item` | `updated_count=1`，summary 从 `0/4` 变为 `1/4` |
| `UAT-09A` | 鼓励反馈 | 在 Pad 勾选一个包含“订正 / 默写 / 复习”等关键词的任务 | 页面出现即时鼓励，如“这一步不轻松，你还是认真拿下了。” |
| `UAT-10` | 家长反馈 | `GET /api/v1/stats/daily` | 返回 `completed_tasks=1`、`auto_points=1`、非空 `encouragement` |
| `UAT-11` | 语音任务完成 | 在 Pad 任务板说“数学订正好了” | Pad 调用 `/api/v1/voice-commands/resolve` 并执行对应完成动作 |
| `UAT-11A` | 语音启动 | 在 Pad 点击“开始说话” | 不出现 `type 'Null' is not a bool in boolean expression`，可以正常进入监听或返回明确语音失败提示 |
| `UAT-11B` | 长段语音 | 在 Pad 朗读 / 背诵任务中开始说话，停顿后继续，再手动结束 | 监听在人工结束前保持可用，期间 transcript 会分段记录而不是几秒内自动失败 |
| `UAT-11B1` | 陪伴续听 | 在 Pad 点击“开始说话”后，先静音数秒或触发一次 `error_no_match`，随后继续说正文 | 会话不会因可恢复错误直接退出，而是自动续听；直到人工点击“结束说话”才收尾 |
| `UAT-11B2` | 真实停顿保留 | 在 Pad 背诵一段内容，中间自然停顿 2-3 次后结束 | `实时记录` 区块优先保留真实停顿分段；若有 `背诵对照`，则在下方单独按标准原文逐句展示，不覆盖上方真实记录 |
| `UAT-11C` | 背诵分析 | `POST /api/v1/recitation/analyze` | 返回标题 / 作者 / 完成度 / `matched_lines` / `needs_retry` / 建议 |
| `UAT-11D` | 背诵前导语 | 在孩子先说“我来背”再进入古诗正文的场景调用 `/api/v1/recitation/analyze` | 仍能正确识别标题 / 作者，并按参考原文句形把 transcript 切成对应句段 |
| `UAT-12` | 积分 | `POST /api/v1/points/ledger` | 成功写入一条人工奖励 |
| `UAT-13` | 积分 | `GET /api/v1/points/ledger` 与 `GET /api/v1/points/balance` | 返回自动积分 `1` + 人工积分 `2`，余额 `3` |
| `UAT-14` | 词单 | `POST /api/v1/word-lists/parse` | 返回结构化词项 |
| `UAT-15` | 词单 | `POST /api/v1/word-lists` | 成功保存 `wordlist_...` |
| `UAT-16` | 听写会话 | `POST /api/v1/dictation-sessions/start` | 返回 `session_...` 和当前单词 |
| `UAT-16A` | 听写等待态 | 在后台不提供当天词单时触发 Pad 同步 | 不展示 `404 / TaskApiException`，而是显示“默写词单还没准备好 / 等家长补充词单后再来默写” |
| `UAT-17` | 听写语音推进 | 在 Pad 听写页签说“好了”或“Next” | Pad 调用 `/api/v1/voice-commands/resolve` 并切到下一词 |
| `UAT-18` | 听写反馈 | 在 Pad 完成交卷并等待 AI 批改 | 页面出现正向鼓励式提示，不是纯技术型系统消息 |
| `UAT-18A` | 鼓励播报 | 在 Pad 完成任务并出现“成长小鼓励”卡片 | 鼓励支持自动播报，且可点击“重播鼓励”再次播报 |
| `UAT-18B` | 鼓励开关 | 在 Pad 点击“自动播报开 / 关” | 自动播报状态可切换；关闭后保留文字，不再自动出声 |
| `UAT-19` | 月统计 | `GET /api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03` | 返回任务、积分、词单、会话的月聚合 |
| `UAT-20` | 自动化 | `go test ./... -count=1` | 全部通过 |
| `UAT-21` | 自动化 | `npm test`、`npm run build` | 全部通过 |
| `UAT-22` | 自动化 | `flutter analyze`、`flutter test --no-pub`、`flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080` | 全部通过，允许 wasm dry-run warning |
| `UAT-23` | 一键校验 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh` | smoke 全部通过 |
| `UAT-24` | 一键演示 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh` | 输出 demo walkthrough 且内部 smoke 通过 |

## 4. 推荐手工验收故事

1. 在 Parent Web 选择 `family_id=306`、`assignee_id=1`、`assigned_date=2026-03-12`。
2. 粘贴老师原文，点击“AI 解析任务”。
3. 审核草稿后确认发布。
4. 切到 Pad，使用同一天加载任务板。
5. 勾选 1 条任务完成，并确认出现即时鼓励。
6. 在 Pad 任务板说“数学订正好了”，确认语音完成动作。
7. 回到 Parent Web 刷新反馈，确认完成率同步更新。
8. 在 Parent Web 新增一条积分奖励，确认余额同步变化。
9. 在 Parent Web 创建 3 个词的词单。
10. 切到 Pad 启动听写，说“好了”推进到下一词并重播。
11. 交卷并确认 AI 批改反馈使用鼓励式语气。
12. 回到 Parent Web 查看月统计，确认任务、积分、词单和会话都被统计到。

## 5. 当前已验证结果

`2026-03-14` 已执行并通过的结果摘要：

- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh` 已通过。
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh` 已通过。
- Parent Web 与 Pad Web 页面都能正常返回 HTML。
- Parent Web 点击“去录入原文”后可直接进入 `原文` 子页面并看到输入框。
- 背诵类任务在解析、草稿和确认发布后都能保留学习素材元数据。
- API 成功完成 `parse -> confirm -> list -> status update -> daily/monthly stats`。
- `POST /api/v1/recitation/analyze` 可以返回标题、完成度和逐句匹配。
- 任务完成后自动积分 `+1`，家长人工奖励 `+2`，余额正确汇总为 `3`。
- 词单 `wordlist_000001` 保存成功。
- 听写会话 `session_000002` 成功执行 `start -> next -> replay`。
- Pad 在缺少当天词单时会进入等待家长补充的友好状态，不再直接暴露原始异常文本。
- 成长小鼓励支持自动播报、手动重播和自动播报开关，相关 widget / controller 回归通过。

## 6. GitHub 同步门槛

在正式开始下一阶段前，必须同时满足：

1. `git fetch origin` 已执行。
2. `git status --short` 中只剩计划提交的文件。
3. 不把 `.gopath/`、`build/`、`dist/`、`.dart_tool/`、运行时密钥文件带进 commit。
4. 根 README、运行手册、用户手册、release checklist、delivery readiness、UAT cases 已同步。
5. 版本声明已经对齐到 `v0.4.0`。
6. 自动化验证和三端联调结果已附在 release commit 或 PR 描述中。

# Changelog

## [Unreleased]

## [0.3.2] - 2026-03-13

### Fixed

- **Parent Web 原文录入入口修复**
  - 修复发布主路径点击“去录入原文”后被错误回退到 `范围` 页的问题，保证家长能直接进入 `原文` 子页面看到输入框并继续录入。
  - 修复无草稿 / 无已发布任务的空状态回退逻辑，避免正常录入流程被误判成必须回到起始页。
- **Pad 语音指令启动与收尾修复**
  - 修复孩子端点击“开始说话”后，`speech_to_text` 的 `listen()` 返回值被误当成 `bool` 判断，导致 `type 'Null' is not a bool in boolean expression` 的崩溃。
  - 修复 Web/STT 场景下只收到中间识别结果、随后收到 `done / notListening` 时被误判失败的问题，保证“好了 / 下一个 / 继续 / 数学订正好了”等口语指令能正常收尾。

### Verified

- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter test --no-pub`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`

## [0.3.1] - 2026-03-12

### Changed

- **Parent Web 移动 H5 重构**
  - 家长端主界面改为移动优先的单列工位，桌面浏览器访问时也按手机宽度呈现，不再保持 PC 仪表盘布局。
  - 发布、反馈、积分、单词四个功能区改为“当前主屏置顶 + 固定底部导航”的 H5 流程，适合家长碎片时间快速切换。
  - 家长端顶部摘要进一步压缩成短头部，只保留当前日期、当前主屏和关键摘要，避免手机首屏被大卡片占满。
  - 发布、反馈、积分、单词进一步拆成二级子页面，采用菜单切换 + 左右滑入滑出动效，避免在单页里长距离上下查找复杂功能。
  - 发布主路径与发布子页面联动，点击“录入原文 / 审核草稿 / 发布完成”可直接切到对应 H5 子页。
  - 发布主屏继续细分为 `范围 / 原文 / 审核 / 发布 / 拆分 / 任务 / 摘要 / 任务板`，并增加粘性子菜单，保证复杂发布流程始终可切换。
  - 发布底部动作条改为只在“审核 / 发布完成”阶段显示，避免在录入阶段和固定底部导航叠压占用手机视口。
- 新增家长端移动 H5 专项操作手册，补充手机端工作流与日常使用建议。

### Verified

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test --no-pub`
- `cd apps/pad-app && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

## [0.3.0] - 2026-03-12

### Added

- **Pad 语音助手闭环**
  - 新增 `/api/v1/voice-commands/resolve` 的 Pad 端联动能力，可在任务板和听写场景中通过 STT + 规则/LLM 推理执行“好了”“下一个”“数学订正好了”等自然语音指令。
  - Pad 端增加语音助手 UI、听写场景与任务板场景的动作映射与执行反馈。
- **孩子端正向鼓励**
  - 新增任务完成即时鼓励逻辑，按单任务、作业分组、学科分组、全部完成四种场景给出成长型反馈。
  - Pad 任务页新增“成长小鼓励”卡片，真正展示后端 `dailyStats.encouragement`。
  - 听写流程在开始、下一词、交卷、AI 批改完成等节点改为孩子视角的积极反馈。
- 新增后端统计鼓励文案回归测试，覆盖“无数据 / 部分完成 / 高完成率 / 全部完成”等场景。

### Changed

- 对齐交付版本声明：`README.md`、`apps/parent-web/package*.json`、`apps/pad-app/pubspec.yaml` 统一到 `v0.3.0` 基线。
- 刷新运行手册、用户手册、发布清单、演示清单、UAT 用例和交付就绪审计，补充语音助手与正向鼓励的验收路径。
- 后端 `daily / weekly / monthly` 统计接口的鼓励文案改成更强调坚持、进步和收尾的表达。

### Verified

- `go test ./... -count=1`
- `flutter analyze`
- `flutter test --no-pub`
- `POST /api/v1/voice-commands/resolve`
- Pad 任务完成即时鼓励 widget tests
- Pad 听写结束鼓励 controller test

## [0.2.0] - 2026-03-10

### Added

- **后端事实源闭环 (v0.2.0 核心突破)**
  - Parent Web 单词清单彻底废弃 `localStorage`，改由 `POST /api/v1/word-lists` 后端持久化。
  - Pad 端接入 `dictation-session` 接口，支持播放进度、重播与下一词的后端会话同步。
  - Parent Web 积分流水与余额改由 `/api/v1/points/ledger` 与 `/api/v1/points/balance` 驱动，不再依赖本地缓存。
  - Pad 端孩子积分展示改为后端权威余额，消除 `completed * 2` 的本地估算逻辑。
  - Parent Web 月视图接入 `/api/v1/stats/monthly`，实现以周为单位的后端聚合分析。
  - Pad 端今日简报基于后端 `DailyStats` 结构输出。
- `scripts/demo_local_stack.sh` 作为本地演示入口。
- `docs/13_RELEASE_CHECKLIST.md` 升级为 `v0.2.0` 正式签收清单。
- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md` 增加后端一致性专项核查项。
- `docs/17_DELIVERY_READINESS.md` 更新结论为“已达到第一阶段正式交付与签收标准”。

### Changed

- `README.md` 与 `docs/06_RUNBOOK.md` 增加一键演示入口。
- 集成流程从“preflight + smoke”推进到“preflight + smoke + demo + release checklist”。
- `docs/09` 到 `docs/12` 统一收口并归档。
- `docs/14_NEXT_PHASE_DISPATCH.md` 更新为第一阶段收口计划。
- `.env.example` 数据库示例账号改为私有占位值。
- `scripts/preflight_local_env.sh` 优化 macOS 下 Docker Desktop 的识别。
- `docs/03_ROADMAP.md` 标记第一阶段核心能力已闭环。

### Verified

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`
- `go test ./...` (Go 后端全量测试通过)
- `npm run test` & `npm run build` (Parent Web API 集成测试通过)
- `flutter analyze` & `flutter test` & `flutter build web` (Pad App 全量验证通过)

### Notes

- `flutter build web` 当前会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning，但 Web 构建仍成功，不阻塞现阶段 `v0.2.0` 交付。

## [0.1.1] - 2026-03-09

### Added

- API 统一错误契约文档与状态更新错误码
- 作业解析与周报模块的更多回归测试
- Pad 端任务板分层：`app / page / controller / repository / api_client`
- Parent Web 自动化测试，覆盖按日期创建、失败保留草稿与风险排序
- `scripts/preflight_local_env.sh`
- `scripts/smoke_local_stack.sh`
- 多 Codex 终端命令与正式派单文档

### Changed

- 项目基线版本更新为 `v0.1.1`
- 本地运行手册与 README 增加 preflight / smoke 流程
- Parent Web 支持按日期把任务解析并确认创建到某一天
- Pad Web 构建纳入标准本地验证步骤

### Verified

- `GOCACHE=... GOMODCACHE=... GOPROXY=off GOSUMDB=off go test ./...`
- `npm run test`
- `npm run build`
- `flutter analyze`
- `flutter test`
- `flutter build web --dart-define=API_BASE_URL=http://localhost:8080`
- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`

### Known Limitations

- `smoke_local_stack.sh` 依赖本地已有运行中的 Go 后端
- 当前尚未把 preflight / smoke 接入 CI
- 当前仍以本地 Markdown 工作区为主要存储

## [0.1.0] - 2026-03-09

### Added

- 家长端最小任务 input 页，支持学校群式原始文本粘贴
- Agent Core `LLM 优先 + 规则兜底` 混合解析链路
- 任务确认流：`parse -> review -> confirm`
- Markdown 工作区持久化
- 按 `学科 -> 作业分组 -> 原子任务` 的任务板接口
- Pad 端最小任务同步页
- 单个任务、作业分组、学科、全部任务的完成同步
- 本地运行手册和版本发布记录

### Changed

- 当前主存储从数据库原型收敛为 Markdown Workspace
- Pad 端交互从“系统推荐下一步”调整为“孩子自主选择，系统只跟踪进度”
- API Server 补充浏览器跨域支持，便于 `parent-web` 本地联调

### Verified

- `python3 -m unittest discover -s tests`
- `python3 -m py_compile main.py api/routes.py services/llm_parser.py services/weekly_analyst.py tests/test_llm_parser.py`
- `GOCACHE=../.gocache GOMODCACHE=../.modcache go test ./...`
- `npm run build`
- `flutter analyze`
- `flutter test`

### Known Limitations

- 未配置真实 `LLM_API_KEY` 时，解析会走规则兜底
- 当前未启用 MySQL 作为主存储
- 当前未接入图片作业解析

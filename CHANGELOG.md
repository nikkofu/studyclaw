# Changelog

## [Unreleased]

### Changed

- 对齐交付版本声明：`README.md`、`apps/parent-web/package*.json`、`apps/pad-app/pubspec.yaml` 统一到 `v0.2.0` 基线。
- 刷新运行手册、用户手册、发布清单、演示清单和交付就绪审计，补齐 `2026-03-12` 的三端联调结果。
- 新增 `docs/19_DELIVERY_UAT_CASES.md`，沉淀 API / Parent Web / Pad 的交付验收用例和 GitHub 同步门槛。
- 更新 `apps/pad-app/README.md`，移除“本地词单为正式事实源”的过时描述，改为后端词单 / 听写会话基线。

### Verified

- `git fetch origin`
- `curl http://127.0.0.1:5173/`
- `curl http://127.0.0.1:55771/`
- `POST /api/v1/tasks/parse`
- `POST /api/v1/tasks/confirm`
- `GET /api/v1/tasks`
- `PATCH /api/v1/tasks/status/item`
- `GET /api/v1/stats/daily`
- `POST /api/v1/points/ledger`
- `GET /api/v1/points/ledger`
- `GET /api/v1/points/balance`
- `POST /api/v1/word-lists/parse`
- `POST /api/v1/word-lists`
- `POST /api/v1/dictation-sessions/start`
- `POST /api/v1/dictation-sessions/:session_id/next`
- `POST /api/v1/dictation-sessions/:session_id/replay`
- `GET /api/v1/dictation-sessions`
- `GET /api/v1/stats/monthly`

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

# Changelog

## [Unreleased]

### Added

- `scripts/demo_local_stack.sh` 作为本地演示入口
- `docs/13_RELEASE_CHECKLIST.md` 作为发布前检查清单
- `docs/14_NEXT_PHASE_DISPATCH.md` 作为最新的 Codex 派单文档
- `docs/15_CODEX_DIRECT_DISPATCH.md` 作为可直接复制到 Codex 的快捷派单文档
- 第一阶段正式需求收口到 `docs/01_PRD.md`
- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md` 作为第一阶段功能演示清单
- `docs/17_DELIVERY_READINESS.md` 作为第一阶段交付就绪度审计文档

### Changed

- `README.md` 与 `docs/06_RUNBOOK.md` 增加一键演示入口
- 集成流程从“preflight + smoke”推进到“preflight + smoke + demo + release checklist”
- `docs/09`、`docs/10`、`docs/11`、`docs/12` 增加归档说明，统一收口到 `docs/14_NEXT_PHASE_DISPATCH.md`
- `docs/13_RELEASE_CHECKLIST.md` 增加固定的 GitHub 同步命令模板
- `.env.example` 的数据库示例账号改为私有占位值，避免误用仓库内示例密码
- `scripts/preflight_local_env.sh` 增加 Docker Desktop 常见路径兜底识别，降低 macOS 下的误判
- `docs/03_ROADMAP.md` 改为围绕第一阶段 7 类核心能力的版本计划
- `docs/04_AGENTIC_DESIGN.md` 改为围绕 Google Agentic design pattern 的第一阶段约束
- `docs/14_NEXT_PHASE_DISPATCH.md` 改为第一阶段多 Codex 开发计划
- `docs/06_RUNBOOK.md`、`docs/13_RELEASE_CHECKLIST.md`、`README.md`、`scripts/demo_local_stack.sh` 增加第一阶段演示清单入口
- `README.md` 增加第一阶段交付就绪度审计入口

### Verified

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`
- `GOCACHE=/Users/admin/Documents/WORK/ai/studyclaw/.cache/go-build GOMODCACHE=/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/.gomodcache GOPROXY=off GOSUMDB=off go test ./...`
- `cd apps/parent-web && npm run test`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test`
- `cd apps/pad-app && flutter build web`

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

- 家长端最小任务输入页，支持学校群式原始文本粘贴
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

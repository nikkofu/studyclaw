# Changelog

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

# StudyClaw v0.3.5 一页摘要

发布日期：`2026-03-14`

## 这是什么

`v0.3.5` 是第一阶段在 `v0.3.4` 正式基线之上的发布后硬化版。

这版不继续扩新业务，而是把“已经能交付”的状态进一步推进到“可以更稳地发版、复核和签收”的状态。

## 给家长

- 日常使用路径和 `v0.3.4` 一致
- 家长端、Pad 端和操作手册的口径现在更一致
- 如果需要复盘“当前正式版本到底验证到什么程度”，文档里已经能直接看到

## 给团队

- `check_release_scope.sh` 在干净工作区下不再误报
- `smoke/demo` 和三端入口存活检查已正式写入交付文档
- `README`、`runbook`、`release checklist`、`GitHub release` 现在在同一版本口径上

## 给 GitHub / 对外交付

`v0.3.5` 的意义不是新增某条大功能，而是把已有成果更严谨地沉淀成正式 release：

- 代码版本号已统一
- 文档版本号已统一
- release notes 已同步
- GitHub release 页面已同步

## 本版包含的关键变化

### 1. 发版脚本更稳

- 修复干净工作区下的 release scope 误报
- 允许重复执行发版前检查

### 2. 联调证据可追溯

- `smoke/demo` 结果已入档
- Parent Web / Pad Web 页面存活检查已入档

### 3. 交付资产全部对齐

- README
- 手册
- 检查清单
- 发布说明
- 一页摘要
- GitHub release

## 验证状态

`v0.3.5` 当前已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `go test ./... -count=1`
- `npm test -- --run`
- `npm run build`
- `flutter analyze`
- `flutter test --no-pub`
- `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`
- `curl http://127.0.0.1:5173/`
- `curl http://127.0.0.1:55771/`

## 一句话结论

`v0.3.5` 代表的是：第一阶段不仅功能能跑，连发版检查、联调证据和交付文档也已经收口到一个更稳的正式基线。

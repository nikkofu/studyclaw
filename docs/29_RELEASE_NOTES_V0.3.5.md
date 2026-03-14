# StudyClaw v0.3.5 发布说明

发布日期：`2026-03-14`

## 版本定位

`v0.3.5` 是建立在 `v0.3.4` 正式基线之上的发布后硬化版。

这次不是继续扩业务主链，而是把已经完成的交付能力真正固化成一个更稳、更可签收的正式版本：

1. 发版前检查脚本在干净工作区下必须稳定可用。
2. `smoke/demo` 和三端入口存活检查必须明确写进正式文档，而不是只停留在口头说明。
3. README、手册、发布说明、GitHub release 页面要再次对齐，避免仓库主干和正式 release 资产脱节。

## 本次发布包含什么

### 1. Release scope 校验脚本硬化

- 修复 `scripts/check_release_scope.sh` 在工作区干净时的未绑定变量问题
- 现在脚本会直接输出 `Release scope check: clean worktree`
- 这样正式发版前可以重复执行，不会因为脚本自身误报打断流程

### 2. 三端联调证据正式入档

- `smoke_local_stack.sh` 已按 `API=http://127.0.0.1:38080` 重新执行并通过
- `demo_local_stack.sh` 已按 `Parent=http://127.0.0.1:5173` 重新执行并通过
- `curl http://127.0.0.1:5173/` 与 `curl http://127.0.0.1:55771/` 已返回有效 HTML

### 3. 版本与文档同步到 v0.3.5

- `README.md`
- `docs/06_RUNBOOK.md`
- `docs/13_RELEASE_CHECKLIST.md`
- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- `docs/17_DELIVERY_READINESS.md`
- `docs/19_DELIVERY_UAT_CASES.md`
- `docs/20_RELEASE_SYNC_PLAYBOOK.md`
- `docs/PARENT_WEB_H5_MANUAL.md`
- `docs/USER_MANUAL_V0.3.5.md`
- `docs/29_RELEASE_NOTES_V0.3.5.md`
- `docs/30_PHASE_ONE_PAGER_V0.3.5.md`

## 对交付的实际影响

### 家长

- 日常操作路径没有变化
- 但现在交付文档、操作手册和 GitHub release 页面更加一致，减少“文档说一套、仓库是另一套”的问题

### 团队

- 发版前检查更稳，干净工作区不会再被脚本误判
- 三端联调是否真的跑过，有正式证据可查
- 后续进入下一阶段前，主干和 release 资产的边界更清晰

### GitHub / 对外交付

- `v0.3.5` 代表的是一个更完整的正式签收点
- 不只是代码在 `main`，连 release notes、README、checklist、manual 也都对齐了

## 验证摘要

本次发布已完成：

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
- `curl http://127.0.0.1:5173/`
- `curl http://127.0.0.1:55771/`

## 已知说明

- `flutter build web` 仍会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning
- 当前 HTML/Web 构建成功，因此这条 warning 不阻塞 `v0.3.5` 交付

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/19_DELIVERY_UAT_CASES.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/19_DELIVERY_UAT_CASES.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.3.5.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.3.5.md)
- [docs/PARENT_WEB_H5_MANUAL.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/PARENT_WEB_H5_MANUAL.md)

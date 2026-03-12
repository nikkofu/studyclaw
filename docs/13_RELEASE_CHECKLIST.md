# StudyClaw 发布前检查清单

本文档用于 `v0.2.0` 第一阶段正式签收版的发布前检查。只要有一项不满足，就不应该把仓库当作下一阶段基线。

## 1. 版本与文档同步

检查点：

- [x] `README.md` 标注当前版本为 `v0.2.0`
- [x] `apps/parent-web/package.json` 与 `apps/parent-web/package-lock.json` 版本为 `0.2.0`
- [x] `apps/pad-app/pubspec.yaml` 版本为 `0.2.0+1`
- [x] `CHANGELOG.md` 已记录交付整备与验证结果
- [x] `docs/17_DELIVERY_READINESS.md` 更新为最新审计结论
- [x] `docs/19_DELIVERY_UAT_CASES.md` 可直接作为交付验收用例

## 2. 密钥与运行时配置

必须通过：

```bash
bash scripts/check_no_tracked_runtime_env.sh
bash scripts/preflight_local_env.sh
bash scripts/check_release_scope.sh
```

## 3. 三端事实源一致性

检查点：

- [x] Parent Web 发布作业走 `/api/v1/tasks/parse` 与 `/api/v1/tasks/confirm`
- [x] Pad 任务板读取 `/api/v1/tasks`，状态更新走 `/api/v1/tasks/status/*`
- [x] Parent Web 与 Pad 共用 `/api/v1/points/ledger` 和 `/api/v1/points/balance`
- [x] Parent Web 与 Pad 共用 `/api/v1/word-lists` 和 `/api/v1/dictation-sessions`
- [x] 日 / 周 / 月统计均由 `/api/v1/stats/*` 提供

## 4. 自动化验证

必须通过：

- [x] `go test ./... -count=1`
- [x] `npm test`
- [x] `npm run build`
- [x] `flutter analyze`
- [x] `flutter test`
- [x] `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`

说明：

- `flutter build web` 当前会输出 `flutter_tts` 的 wasm dry-run warning，但产物构建成功，现阶段不阻塞发布。

## 5. 三端联调与演示

必须通过：

```bash
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 \
STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 \
bash scripts/demo_local_stack.sh
```

同时必须完成：

- [x] `curl http://127.0.0.1:5173/`
- [x] `curl http://127.0.0.1:55771/`
- [x] `docs/19_DELIVERY_UAT_CASES.md` 中的主线用例

## 6. GitHub 同步复核

以下项目均已完成：

- [x] `git fetch origin`
- [x] `git status --short` 中只剩本次计划提交的文件
- [x] `.gopath/` 历史缓存清理已按 scoped release 处理，未把 `build/`、`dist/`、`.dart_tool/`、运行时密钥文件带进 commit
- [x] release commit 信息清晰：`release: prepare v0.2.0 delivery sync`
- [x] 版本标签与交付版本一致：`v0.2.0`
- [x] push 后已再次核对 `origin/main` 与标签状态

## 7. 发布结论

`v0.2.0` 已达到功能交付标准，并已完成 GitHub 正式同步，可作为第一阶段正式签收版本。

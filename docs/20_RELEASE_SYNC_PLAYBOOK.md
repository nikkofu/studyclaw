# StudyClaw Release Sync Playbook

本文档定义 `v0.3.4` 进入 GitHub 正式同步前的最小操作顺序。目标不是“把所有改动都推上去”，而是“只把本次交付范围内的源码、文档和版本文件推上去”。

## 1. 先决条件

在执行同步之前，必须先通过：

```bash
bash scripts/check_no_tracked_runtime_env.sh
bash scripts/preflight_local_env.sh
bash scripts/check_release_scope.sh
```

说明：

- `check_release_scope.sh` 会把 `.claude/`、`build/`、`dist/`、`.dart_tool/`、`.env`、`runtime.env` 视为禁止进入 release 的噪音路径。
- 它会单独列出 `apps/api-server/.gopath/` 这类“已被误跟踪的缓存目录”作为一次性仓库清洁候选项。
- 如果脚本失败，先清理或隔离禁止路径，再继续。

## 2. 当前 release 允许纳入的范围

当前交付基线允许纳入 GitHub 同步的主要路径：

- 根目录：`README.md`、`CHANGELOG.md`、`.env.example`
- 后端：`apps/api-server/cmd/`、`config/`、`internal/`、`routes/`
- 家长端：`apps/parent-web/src/`、`package.json`、`package-lock.json`
- 孩子端：`apps/pad-app/lib/`、`assets/`、`test/`、`pubspec.yaml`、`pubspec.lock`、`README.md`
- 交付文档：`docs/06_RUNBOOK.md`、`docs/13_RELEASE_CHECKLIST.md`、`docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`、`docs/17_DELIVERY_READINESS.md`、`docs/19_DELIVERY_UAT_CASES.md`、`docs/20_RELEASE_SYNC_PLAYBOOK.md`、`docs/USER_MANUAL_V0.3.4.md`、`docs/PARENT_WEB_H5_MANUAL.md`、`docs/27_RELEASE_NOTES_V0.3.4.md`、`docs/28_PHASE_ONE_PAGER_V0.3.4.md`
- 测试夹具：`test/daily_homework.txt`、`test/listen_image.jpg`

明确禁止进入本次 release 的路径：

- `.claude/`
- `build/`
- `dist/`
- `.dart_tool/`
- 任意 `.env` / `runtime.env` / 含真实密钥文件

需要作为一次性仓库清洁处理的路径：

- `apps/api-server/.gopath/`

## 3. 推荐同步步骤

### 3.1 拉取远端状态

```bash
git fetch origin
git branch -vv
```

### 3.2 检查交付范围

```bash
bash scripts/check_release_scope.sh
git status --short
```

### 3.3 只 stage 本次交付范围

示例：

```bash
git add README.md CHANGELOG.md .env.example
git add apps/api-server/cmd apps/api-server/internal apps/api-server/routes apps/api-server/go.mod
git add apps/parent-web/src apps/parent-web/package.json apps/parent-web/package-lock.json
git add apps/pad-app/lib apps/pad-app/assets apps/pad-app/test apps/pad-app/pubspec.yaml apps/pad-app/pubspec.lock apps/pad-app/README.md apps/pad-app/SC05_PAD_LIVE_CHECKLIST.md
git add docs/06_RUNBOOK.md docs/13_RELEASE_CHECKLIST.md docs/16_FIRST_PHASE_DEMO_CHECKLIST.md docs/17_DELIVERY_READINESS.md docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md docs/USER_MANUAL_V0.3.4.md docs/PARENT_WEB_H5_MANUAL.md docs/27_RELEASE_NOTES_V0.3.4.md docs/28_PHASE_ONE_PAGER_V0.3.4.md
git add test/daily_homework.txt test/listen_image.jpg
```

如果 stage 之后发现夹带了不该提交的路径，先用非破坏性的 `git restore --staged <path>` 把它们从索引移除，不要直接做大范围 reset。

### 3.4 再跑一次验证

```bash
go test ./... -count=1
npm test
npm run build
flutter analyze
flutter test --no-pub
flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 \
STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 \
bash scripts/demo_local_stack.sh
```

### 3.5 commit / tag / push

示例：

```bash
git commit -m "release: prepare v0.3.4"
git tag v0.3.4
git push origin main
git push origin v0.3.4
```

## 4. 本轮实际阻塞项

截至 `2026-03-14`，本轮 release 仍应坚持 scoped staging：

- 只提交本次 Pad 词单缺失等待态、成长鼓励语音播报、平板 TTS 补齐和文档同步相关改动
- 仍然禁止把缓存、构建产物和运行时密钥带入 commit
- 如果工作树里混入并行试验改动，必须先明确是否属于 `v0.3.4`

## 5. 通过标准

只有下面条件都满足，才允许视为“GitHub 同步完成，可开启下一阶段”：

1. `bash scripts/check_release_scope.sh` 通过
2. `git status --short` 中没有禁止路径
3. 自动化验证通过
4. 三端联调通过
5. README / 手册 / 检查清单 / UAT 用例 / 版本号已同步
6. `origin/main` 与版本标签状态复核完成

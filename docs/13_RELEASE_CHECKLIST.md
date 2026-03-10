# StudyClaw 发布前检查清单

本文档用于 `v0.1.1` 之后的本地演示、版本同步和下一次 GitHub 发布前检查。

适用范围：

- 本地演示前
- GitHub 同步前
- 准备下一个里程碑或 tag 前

## 1. 版本与文档

发布前先确认以下文件中的版本与说明一致：

- `README.md`
- `CHANGELOG.md`
- `docs/03_ROADMAP.md`
- `docs/17_DELIVERY_READINESS.md`
- `.env.example`

检查点：

- 当前版本号是否一致
- Changelog 是否说明本轮新增内容
- Roadmap 是否反映当前真实阶段
- Delivery Readiness 是否仍准确描述当前阻塞项
- README 和 Runbook 是否指向当前真实脚本与入口

## 2. 密钥与运行时配置

必须通过：

```bash
bash scripts/check_no_tracked_runtime_env.sh
```

检查点：

- 真实密钥仍然只在 `~/.config/studyclaw/runtime.env`
- 仓库中没有被跟踪的 `.env`、`runtime.env`、`secrets.env`
- `.env.example` 中没有真实敏感值

## 3. 本地环境预检

必须通过：

```bash
bash scripts/preflight_local_env.sh
```

检查点：

- Go / Node / npm / Flutter 可用
- 私有 `runtime.env` 存在
- 关键目录齐全
- 如果 Docker 不可用，仅应在当前演示链路下表现为 warning，而不是阻塞

## 4. 一键 smoke

在本地 Go 后端已启动的前提下必须通过：

```bash
bash scripts/smoke_local_stack.sh
```

检查点：

- `/ping` 返回 `pong`
- 最小任务板 API 可用
- Parent Web build 通过
- Pad Web build 通过

## 5. Go 后端

必须通过：

```bash
cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./...
```

附加检查：

- API 错误契约文档仍与实现一致
- smoke curl 文档仍与真实接口一致

相关文档：

- `apps/api-server/internal/interfaces/http/API_ERROR_CONTRACT.md`
- `apps/api-server/internal/interfaces/http/TASKBOARD_API_SMOKE.md`

## 6. Parent Web

必须通过：

```bash
cd apps/parent-web
npm run test
npm run build
```

附加检查：

- 指定日期 `parse -> review -> confirm` 链路仍可演示
- 失败后草稿与选中项仍能保留

## 7. Pad App

必须通过：

```bash
cd apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

附加检查：

- 404 / 409 错误提示仍清晰
- 真实后端 Chrome 联调清单仍可复现

相关文档：

- `apps/pad-app/SC05_PAD_LIVE_CHECKLIST.md`

## 8. 演示入口

推荐在演示前执行：

```bash
bash scripts/demo_local_stack.sh
```

这会：

- 执行 preflight
- 执行 smoke
- 给出 Parent Web 和 Pad 的演示步骤

第一阶段功能演示清单：

- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- `docs/17_DELIVERY_READINESS.md`

## 9. GitHub 同步前

检查点：

- `git status` 中只剩本轮明确要提交的文件
- 不包含缓存目录、临时目录、运行时密钥文件
- 不误带 `.gopath`、`.gomodcache`、`.tmp` 一类目录
- Commit message 与本轮目标匹配

当前派单入口：

- `docs/14_NEXT_PHASE_DISPATCH.md`
- `docs/15_CODEX_DIRECT_DISPATCH.md`

固定动作顺序：

1. 跑完所有本地验证
2. 复核变更范围
3. 只暂存本轮目标文件
4. 提交
5. Push 到 `origin/main`

固定命令模板：

```bash
git status --short

bash scripts/check_no_tracked_runtime_env.sh
bash scripts/preflight_local_env.sh
bash scripts/smoke_local_stack.sh
bash scripts/demo_local_stack.sh

cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./...

cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
npm run test
npm run build

cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080

cd /Users/admin/Documents/WORK/ai/studyclaw
git diff --check -- <scoped-files>
git add <scoped-files>
git status --short
git commit -m "<scope>: <summary>"
git push origin main
```

说明：

- `<scoped-files>` 只替换成本轮明确允许提交的文件，不要直接 `git add .`
- 若 `git status` 仍出现缓存目录或别组文件，先处理范围问题，再继续提交
- 若只做 SC-05，则暂存范围应限制在 `docs/`、`scripts/`、`README.md`、`.env.example`、`CHANGELOG.md`

## 10. 第一阶段功能走查

发布前建议再走一遍：

- 家长发布当天作业
- AI 解析与审核
- Pad 完成任务
- 家长查看统计
- 单词逐词播放
- 积分变化
- 日 / 周 / 月反馈与 AI 鼓励
- 若目标是第一阶段正式签收，再复核 `docs/17_DELIVERY_READINESS.md` 中阻塞项是否已全部关闭

详细清单见：

- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- `docs/17_DELIVERY_READINESS.md`

## 11. 发布结论模板

发布前可以直接用下面模板记录：

```text
Version:
Commit:
Scope:
Preflight:
Smoke:
Go tests:
Parent Web:
Pad App:
Secrets hygiene:
Docs synced:
Ready to push:
Notes:
```

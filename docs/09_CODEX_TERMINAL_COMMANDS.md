# StudyClaw 多 Codex 终端命令手册

本文档用于 `2026-03-09` 晚上的第一批并行开发。目标是让多个 Codex 会话和若干运行终端同时工作，但不互相踩目录边界。

当前统一基线：

- 分支：`main`
- 基线提交：`648ff07`
- 并行分工主文档：`docs/08_PARALLEL_WORKSTREAMS.md`
- 安全文档：`docs/07_SECURITY.md`
- 运行手册：`docs/06_RUNBOOK.md`
- 下一阶段任务单：`docs/10_NEXT_PHASE_CODEX_TASKS.md`
- 当前正式派单入口：`docs/11_NEXT_PHASE_DISPATCH.md`
- 最新正式派单入口：`docs/12_NEXT_PHASE_DISPATCH.md`

## 1. 使用规则

1. 每个 Codex 终端只负责自己拥有的目录。
2. 不要让两个 Codex 同时修改同一层目录。
3. 长时间运行的服务尽量放到独立运行终端，不要占住 Codex 编辑终端。
4. 真实密钥只放仓库外 `~/.config/studyclaw/runtime.env`。
5. 今晚默认不处理 `.gitignore`，也不处理 `apps/api-server/.gopath/**` 这类缓存目录。

额外说明：

- `apps/pad-app` 目录下已有本地未提交改动，今晚由 `SC-03-FLUTTER-PAD` 独占处理。
- 如果某组需要别组改接口或改字段，先提需求给该目录 owner，不要越界直接改。

## 2. Codex 终端总表

| 终端名 | 角色 | 建议 cwd | 只改这些目录 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | Go API 组 | `apps/api-server` | `cmd/` `config/` `internal/app/` `internal/interfaces/http/` `internal/modules/taskboard/` `routes/` |
| `SC-02-GO-AGENT` | Go Agent 组 | `apps/api-server` | `internal/modules/agent/` `internal/platform/llm/` `internal/shared/agentic/` |
| `SC-03-FLUTTER-PAD` | Flutter Pad 组 | `apps/pad-app` | `lib/` `test/` `pubspec.yaml` `pubspec.lock` `web/` `macos/` |
| `SC-04-PARENT-WEB` | Parent Web 组 | `apps/parent-web` | `src/` `package.json` `package-lock.json` `index.html` |
| `SC-05-INTEGRATION` | 集成与发布组 | 仓库根目录 | `docs/` `scripts/` `README.md` `.env.example` |

## 3. 每个 Codex 的启动命令和开场提示

### 3.1 `SC-01-GO-API`

Shell 启动命令：

```bash
printf '\e]1;SC-01-GO-API\a'; printf '\e]2;SC-01-GO-API\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
git status --short -- cmd config internal/app internal/interfaces/http internal/modules/taskboard routes
GOCACHE="$(pwd)/../../.cache/go-build" go test ./config ./internal/app ./internal/interfaces/http/... ./internal/modules/taskboard/... ./routes
```

发给该 Codex 的开场提示词：

```text
你现在是 StudyClaw 的 SC-01-GO-API 终端，基线提交是 648ff07。

你的唯一工作范围：
- apps/api-server/cmd
- apps/api-server/config
- apps/api-server/internal/app
- apps/api-server/internal/interfaces/http
- apps/api-server/internal/modules/taskboard
- apps/api-server/routes

今晚目标：
1. 稳定 tasks / status / stats 接口
2. 统一错误返回结构
3. 补接口边界测试
4. 不改 agent 模块目录
5. 不改 Flutter 和 Parent Web

完成后必须给出：
- 改动摘要
- 影响到的接口清单
- 运行过的验证命令
```

今晚交付重点：

- 非法日期、缺失字段、错误 JSON、不存在任务、重复状态更新等异常路径
- 保持成功响应字段尽量稳定，避免 UI 组今晚跟着大改

### 3.2 `SC-02-GO-AGENT`

Shell 启动命令：

```bash
printf '\e]1;SC-02-GO-AGENT\a'; printf '\e]2;SC-02-GO-AGENT\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
git status --short -- internal/modules/agent internal/platform/llm internal/shared/agentic
GOCACHE="$(pwd)/../../.cache/go-build" go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

发给该 Codex 的开场提示词：

```text
你现在是 StudyClaw 的 SC-02-GO-AGENT 终端，基线提交是 648ff07。

你的唯一工作范围：
- apps/api-server/internal/modules/agent
- apps/api-server/internal/platform/llm
- apps/api-server/internal/shared/agentic

今晚目标：
1. 增加作业解析真实样本测试
2. 加强“订正、续做、条件任务、对象不明确、子步骤”识别
3. 稳定 weekly insights 输出
4. 严格保持 Google agentic design pattern 的边界
5. 不改 taskboard 存储和 HTTP 响应结构

完成后必须给出：
- 新增样本覆盖了哪些场景
- 采用了什么确定性逻辑增强
- 跑过哪些 Go 测试
```

今晚交付重点：

- 尽量先用确定性规则和归一化逻辑提升质量
- 保持 `single-agent + custom logic + human-in-the-loop` 的现有边界

### 3.3 `SC-03-FLUTTER-PAD`

Shell 启动命令：

```bash
printf '\e]1;SC-03-FLUTTER-PAD\a'; printf '\e]2;SC-03-FLUTTER-PAD\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
git status --short -- .
flutter pub get
flutter analyze
```

发给该 Codex 的开场提示词：

```text
你现在是 StudyClaw 的 SC-03-FLUTTER-PAD 终端，基线提交是 648ff07。

你的唯一工作范围：
- apps/pad-app/lib
- apps/pad-app/test
- apps/pad-app/pubspec.yaml
- apps/pad-app/pubspec.lock
- 必要时 apps/pad-app/web 和 apps/pad-app/macos

今晚目标：
1. 抽出稳定的 API client 或 repository 层
2. 明确 loading / empty / error / success 四种状态
3. 支持日期切换与手动刷新
4. 增加至少一组 widget 或 integration 测试
5. 不改 Go 后端和 Parent Web

补充说明：
- apps/pad-app 当前已有本地未提交改动，今晚由你接管

完成后必须给出：
- UI 改了哪些状态
- API 调用层怎么收口
- Flutter analyze / test 的结果
```

今晚交付重点：

- 先保证 Chrome 跑通，再考虑真机或 iPad
- 错误态和同步反馈是今晚的必要项，不是附加项

### 3.4 `SC-04-PARENT-WEB`

Shell 启动命令：

```bash
printf '\e]1;SC-04-PARENT-WEB\a'; printf '\e]2;SC-04-PARENT-WEB\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
git status --short -- src package.json package-lock.json index.html
npm install
npm run build
```

发给该 Codex 的开场提示词：

```text
你现在是 StudyClaw 的 SC-04-PARENT-WEB 终端，基线提交是 648ff07。

你的唯一工作范围：
- apps/parent-web/src
- apps/parent-web/package.json
- apps/parent-web/package-lock.json
- apps/parent-web/index.html

今晚目标：
1. 高亮 needs_review 和低 confidence 任务
2. 优化“解析预览 -> 编辑确认 -> 创建任务”的家长操作路径
3. 增加 parse 失败和 create 失败的反馈与重试
4. 不改 Go 后端字段语义
5. 不改 Flutter

完成后必须给出：
- 风险任务的展示策略
- 失败与重试的交互路径
- npm run build 的结果
```

今晚交付重点：

- 家长应该能一眼看出哪条任务风险高
- 不要重新定义后端字段的含义

### 3.5 `SC-05-INTEGRATION`

Shell 启动命令：

```bash
printf '\e]1;SC-05-INTEGRATION\a'; printf '\e]2;SC-05-INTEGRATION\a'
cd /Users/admin/Documents/WORK/ai/studyclaw
git status --short -- README.md .env.example docs scripts
bash scripts/check_no_tracked_runtime_env.sh
```

发给该 Codex 的开场提示词：

```text
你现在是 StudyClaw 的 SC-05-INTEGRATION 终端，基线提交是 648ff07。

你的唯一工作范围：
- docs
- scripts
- README.md
- .env.example

今晚目标：
1. 固化最短本地启动路径
2. 固化安全配置和私有运行时账号规范
3. 输出联调清单和演示清单
4. 必要时补 preflight 或 smoke 脚本
5. 不改业务源码目录

完成后必须给出：
- 更新了哪些文档
- 新增了哪些脚本
- 新人按文档完成联调的最短路径
```

今晚交付重点：

- 让新人 30 分钟内完成联调
- 让每组都知道密钥只能放仓库外

## 4. 运行终端命令

这 3 个终端建议不要开 Codex，只负责跑服务和看日志。如果你终端足够多，建议拆开运行。

### 4.1 `SC-R1-API-RUN`

```bash
printf '\e]1;SC-R1-API-RUN\a'; printf '\e]2;SC-R1-API-RUN\a'
cd /Users/admin/Documents/WORK/ai/studyclaw
bash scripts/init_private_runtime_env.sh
cd apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" go run ./cmd/studyclaw-server
```

健康检查：

```bash
curl http://localhost:8080/ping
```

### 4.2 `SC-R2-PARENT-WEB-RUN`

```bash
printf '\e]1;SC-R2-PARENT-WEB-RUN\a'; printf '\e]2;SC-R2-PARENT-WEB-RUN\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
npm install
npm run dev -- --host 0.0.0.0
```

默认访问地址：

```text
http://localhost:5173
```

### 4.3 `SC-R3-PAD-RUN`

```bash
printf '\e]1;SC-R3-PAD-RUN\a'; printf '\e]2;SC-R3-PAD-RUN\a'
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
flutter pub get
flutter run --dart-define=API_BASE_URL=http://localhost:8080 -d chrome
```

补充说明：

- 如果是 iPhone、iPad 或 Android 真机联调，把 `localhost` 换成 Mac 的局域网 IP
- Chrome 是今晚的默认联调目标

## 5. 今晚的任务跟踪表

可以直接在这个表里更新状态。

| 终端名 | 当前负责人 | 今晚目标 | 当前状态 | 最近一次验证 | 下一步 |
| --- | --- | --- | --- | --- | --- |
| `SC-01-GO-API` |  | 稳定 tasks/status/stats 接口与测试 | 未开始 |  |  |
| `SC-02-GO-AGENT` |  | 增强解析与周报质量 | 未开始 |  |  |
| `SC-03-FLUTTER-PAD` |  | 收口 API client，补状态与测试 | 未开始 |  |  |
| `SC-04-PARENT-WEB` |  | 优化预览、风险高亮、失败重试 | 未开始 |  |  |
| `SC-05-INTEGRATION` |  | 固化文档、安全、联调脚本 | 未开始 |  |  |
| `SC-R1-API-RUN` |  | 跑 Go 后端 | 未开始 |  |  |
| `SC-R2-PARENT-WEB-RUN` |  | 跑 Parent Web | 未开始 |  |  |
| `SC-R3-PAD-RUN` |  | 跑 Flutter Chrome | 未开始 |  |  |

## 6. 发任务时的最短指令

后续你给某个终端派任务时，尽量用下面这种格式：

```text
[终端名]
目标：
边界：
验收：
禁止修改：
```

例子：

```text
[SC-01-GO-API]
目标：把 tasks/status 的错误结构统一成同一风格
边界：只改 apps/api-server 的 API 和 taskboard 相关目录
验收：go test ./... 通过，并给出影响接口清单
禁止修改：internal/modules/agent、apps/pad-app、apps/parent-web
```

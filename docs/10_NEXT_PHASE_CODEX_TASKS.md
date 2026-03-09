# StudyClaw 下一阶段 Codex 任务单

本文档用于承接 `2026-03-09` 晚上第一批并行开发完成后的下一阶段工作分发。

适用状态：

- 基线提交：`648ff07`
- 当前状态：第一批并行任务已完成并通过本地验证，但大部分改动仍在工作区，尚未形成新的 Git 提交
- 参考文档：
  - `docs/08_PARALLEL_WORKSTREAMS.md`
  - `docs/09_CODEX_TERMINAL_COMMANDS.md`
  - `docs/06_RUNBOOK.md`
  - `docs/07_SECURITY.md`

## 1. 当前验证结论

已完成的本地验证：

- `apps/api-server`: `go test ./...`
- `apps/parent-web`: `npm run build`
- `apps/pad-app`: `flutter analyze`
- `apps/pad-app`: `flutter test`
- `apps/pad-app`: `flutter build web --dart-define=API_BASE_URL=http://localhost:8080`
- 仓库根目录：`bash scripts/check_no_tracked_runtime_env.sh`

当前判断：

- `SC-01-GO-API`：通过
- `SC-02-GO-AGENT`：通过
- `SC-03-FLUTTER-PAD`：通过
- `SC-04-PARENT-WEB`：通过构建验证
- `SC-05-INTEGRATION`：通过文档与安全检查

仍然存在的下一阶段重点：

1. `Go-API` 需要把错误契约继续固化，并补更多接口异常测试。
2. `Go-Agent` 需要从“规则增强”进入“回归样本集”阶段。
3. `Flutter Pad` 需要补状态同步与错误码联动的 UI 测试。
4. `Parent Web` 需要补“任务日期”输入，并加最小自动化测试。
5. `Integration` 需要把现有文档转成可执行脚本，而不是继续停留在说明层。

## 2. 任务分发总表

| Codex | 当前阶段结果 | 下一阶段目标 | 优先级 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | 错误结构和状态更新语义已初步统一 | 固化错误契约，补接口边界/异常测试 | P0 |
| `SC-02-GO-AGENT` | 解析与 weekly insight 已增强 | 建立回归样本集并压误报 | P0 |
| `SC-03-FLUTTER-PAD` | 任务板已拆成 controller/repository/page | 补同步交互与 404/409 错误反馈测试 | P1 |
| `SC-04-PARENT-WEB` | 风险高亮与确认流程已成型 | 增加任务日期输入与最小测试 | P0 |
| `SC-05-INTEGRATION` | 终端命令与分工文档已落地 | 增加 preflight/smoke 脚本与联调脚本 | P1 |

## 3. 给每个 Codex 的最新任务

### 3.1 `SC-01-GO-API`

工作范围：

- `apps/api-server/cmd/`
- `apps/api-server/config/`
- `apps/api-server/internal/app/`
- `apps/api-server/internal/interfaces/http/`
- `apps/api-server/internal/modules/taskboard/`
- `apps/api-server/routes/`

直接分配给该 Codex 的任务：

```text
[SC-01-GO-API]
目标：把新的错误契约彻底固化，并补齐空任务板、group/all 重复更新、weekly stats end_date 的接口测试。

边界：
- 只改 apps/api-server/cmd
- 只改 apps/api-server/config
- 只改 apps/api-server/internal/app
- 只改 apps/api-server/internal/interfaces/http
- 只改 apps/api-server/internal/modules/taskboard
- 只改 apps/api-server/routes

必须完成：
1. 补一份最小 API 错误契约说明，明确 error / error_code / details 的返回样例。
2. 补更多路由级测试，重点覆盖：
   - group status 重复更新返回 409
   - all status 重复更新返回 409
   - 空任务板更新的行为
   - weekly stats 的 end_date 非法输入
   - query 参数缺失或非法
3. 保持成功响应字段名稳定，不主动改 UI 组依赖字段。

验收：
1. go test ./... 通过
2. 给出影响到的接口和错误码清单
3. UI 组无需跟着改字段名

禁止修改：
- apps/api-server/internal/modules/agent
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./...
```

### 3.2 `SC-02-GO-AGENT`

工作范围：

- `apps/api-server/internal/modules/agent/`
- `apps/api-server/internal/platform/llm/`
- `apps/api-server/internal/shared/agentic/`

直接分配给该 Codex 的任务：

```text
[SC-02-GO-AGENT]
目标：把解析增强从“规则扩展”推进到“回归样本集”，新增正例/反例测试，重点盯订正、续做、条件任务、对象范围、误报控制。

边界：
- 只改 apps/api-server/internal/modules/agent
- 只改 apps/api-server/internal/platform/llm
- 只改 apps/api-server/internal/shared/agentic

必须完成：
1. 增加一批真实样本测试，覆盖：
   - 订正且目标明确
   - 订正但目标不明确
   - 续做且目标明确
   - 续做但目标不明确
   - 条件任务
   - 部分同学/个别同学/相关同学
   - 正常任务不应误判 needs_review
2. 控制误报，尤其是正常编号任务被误判为条件性任务或高风险。
3. weekly insights 再补极端输入测试：
   - 空数据
   - 单日数据
   - 全完成
   - 全未完成

验收：
1. agent 相关 Go 测试通过
2. 新增样本能明确说明为什么判定 needs_review 或不判定
3. 不改 taskboard 和 HTTP handler

禁止修改：
- apps/api-server/internal/modules/taskboard
- apps/api-server/internal/interfaces/http
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

### 3.3 `SC-03-FLUTTER-PAD`

工作范围：

- `apps/pad-app/lib/`
- `apps/pad-app/test/`
- `apps/pad-app/pubspec.yaml`
- `apps/pad-app/pubspec.lock`
- 必要时 `apps/pad-app/web/`
- 必要时 `apps/pad-app/macos/`

直接分配给该 Codex 的任务：

```text
[SC-03-FLUTTER-PAD]
目标：把任务板从“能加载”推进到“能完整同步”，补单任务、分组、全部任务的交互和错误反馈测试，并对接 Go-API 的错误码。

边界：
- 只改 apps/pad-app

必须完成：
1. 增加任务状态同步的 widget 或 integration 测试，至少覆盖：
   - 单任务勾选成功
   - 分组勾选成功
   - 全部完成成功
   - 服务端 404 显示清晰错误
   - 服务端 409 显示“状态未变化”一类提示
2. 把 Go-API 新的 error_code 映射成更友好的 Pad 端文案。
3. 明确新增文件的版本策略：
   - apps/pad-app/web
   - apps/pad-app/macos
   - apps/pad-app/.metadata
   - apps/pad-app/README.md
   - apps/pad-app/analysis_options.yaml
   需要保留的保留，不该跟仓库的不要继续扩散。

验收：
1. flutter analyze 通过
2. flutter test 通过
3. flutter build web --dart-define=API_BASE_URL=http://localhost:8080 通过

禁止修改：
- apps/api-server
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

### 3.4 `SC-04-PARENT-WEB`

工作范围：

- `apps/parent-web/src/`
- `apps/parent-web/package.json`
- `apps/parent-web/package-lock.json`
- `apps/parent-web/index.html`

直接分配给该 Codex 的任务：

```text
[SC-04-PARENT-WEB]
目标：把家长端从“可用”推进到“可验证”，补任务日期输入，打通按某一天解析/确认创建，并增加最小自动化测试。

边界：
- 只改 apps/parent-web

必须完成：
1. 在页面中新增 assigned_date 输入，并把它传给：
   - POST /api/v1/tasks/parse
   - POST /api/v1/tasks/confirm
2. 对“解析失败保留草稿”“创建失败保留选中项”“风险任务排序”这三块补最小自动化测试。
3. 保持当前风险高亮、三段流程和失败重试体验。
4. 不改后端字段语义，只消费现有字段。

验收：
1. npm run build 通过
2. 任务日期可以明确控制创建到哪一天
3. 自动化测试能覆盖核心交互而不是只靠人工点页面

禁止修改：
- apps/api-server
- apps/pad-app
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
npm run build
```

补充说明：

- 当前家长端已经有了“解析预览 -> 编辑确认 -> 创建任务”的主流程。
- 下一步的关键不是继续堆样式，而是把“某一天的任务创建”和“关键交互测试”补上。

### 3.5 `SC-05-INTEGRATION`

工作范围：

- `docs/`
- `scripts/`
- `README.md`
- `.env.example`

直接分配给该 Codex 的任务：

```text
[SC-05-INTEGRATION]
目标：把当前文档升级为可执行联调工具，补 preflight 和 smoke 脚本，并把使用方式写回 runbook。

边界：
- 只改 docs
- 只改 scripts
- 只改 README.md
- 只改 .env.example

必须完成：
1. 新增 preflight 脚本，至少检查：
   - Go
   - Node / npm
   - Flutter
   - Docker
   - 私有 runtime.env 是否存在
   - 关键目录是否齐全
2. 新增 smoke 脚本，至少覆盖：
   - Go 后端 ping
   - Parent Web build
   - Pad web build
3. 更新 docs/06_RUNBOOK.md，把这两个脚本纳入最短联调路径。
4. 明确说明真实密钥仍然只能放到 ~/.config/studyclaw/runtime.env。

验收：
1. bash scripts/check_no_tracked_runtime_env.sh 通过
2. preflight/smoke 脚本可在本地执行
3. 新人按 runbook 可以在 30 分钟内完成联调

禁止修改：
- apps/api-server
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw
bash scripts/check_no_tracked_runtime_env.sh
```

## 4. 推荐并行顺序

下一轮建议并行关系如下：

### 第一并行批

- `SC-01-GO-API`
- `SC-02-GO-AGENT`
- `SC-04-PARENT-WEB`

原因：

- `SC-04` 需要补 `assigned_date`，但不一定要等待 `SC-01`
- `SC-01` 可以独立补错误契约和更多测试
- `SC-02` 基本与 UI 无耦合，可以独立推进

### 第二并行批

- `SC-03-FLUTTER-PAD`
- `SC-05-INTEGRATION`

原因：

- `SC-03` 适合在 `SC-01` 错误码更稳之后对接
- `SC-05` 可随时并行，但价值在于把当前成果固化成可执行流程

## 5. 任务跟踪表

| Codex | 当前任务 | 状态 | 验收命令 | 备注 |
| --- | --- | --- | --- | --- |
| `SC-01-GO-API` | 固化错误契约与异常测试 | 待分发 | `go test ./...` | 以接口稳定为先 |
| `SC-02-GO-AGENT` | 建立回归样本集并压误报 | 待分发 | `go test ./internal/modules/agent/...` | 以回归样本为核心 |
| `SC-03-FLUTTER-PAD` | 补同步交互与错误反馈测试 | 待分发 | `flutter analyze && flutter test` | 已可 build web |
| `SC-04-PARENT-WEB` | 补任务日期输入与最小测试 | 待分发 | `npm run build` | 当前缺测试 |
| `SC-05-INTEGRATION` | preflight/smoke 脚本化 | 待分发 | `bash scripts/check_no_tracked_runtime_env.sh` | 当前缺可执行脚本 |

## 6. 使用方式

你后续给某个 Codex 发任务时，优先直接复制本文件中对应代码块。

如果需要更精简的派单格式，可以用下面模板：

```text
[终端名]
目标：
边界：
必须完成：
验收：
禁止修改：
```

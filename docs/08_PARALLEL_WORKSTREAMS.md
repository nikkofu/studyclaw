# StudyClaw 并行开发分组方案

本文档用于把当前项目拆成可并行推进、互不干扰的工作组。目标不是简单按“端”拆，而是先按技术语言拆一级边界，再在 Go 后端内部按稳定模块拆二级边界。

适用时间点：

- 当前日期：`2026-03-09`
- 当前后端架构：`Go 单体后端`
- 当前客户端架构：`Flutter Pad + React Parent Web`

## 1. 拆组原则

当前项目不再按“一个全栈组包办所有事情”推进，而是按下面四条规则切分：

1. 先按语言拆一级边界：`Go`、`Dart/Flutter`、`JavaScript/React`、`Shell/Docs`
2. 在 Go 内部再按稳定模块拆二级边界：`任务板/API` 与 `Agent/LLM`
3. 每个组只对自己拥有的目录负责，默认不能直接修改别组源码
4. 跨组协作通过稳定接口完成，而不是通过“多人同时改一个目录”

## 2. 团队拓扑

推荐固定为 5 个工作组：

1. `Go-API 组`
2. `Go-Agent 组`
3. `Flutter Pad 组`
4. `Parent Web 组`
5. `集成与发布组`

这样拆的原因：

- 仅按“Go 一组、Flutter 一组”还不够，Go 里任务板/API 与 Agent/LLM 的节奏和测试方式不同
- `Pad` 和 `Parent Web` 面向不同用户、节奏不同，合并成一个前端组会相互抢占上下文
- 文档、脚本、运行环境、验收流程必须有单独 owner，否则所有组都会零散改根目录文件

## 3. 目录与语言边界

### 3.1 `Go-API 组`

- 组目标：
  稳定 HTTP 接口、任务板读写、状态同步、配置加载与服务启动，让 `v0.2.0` 演示链路稳定可回归。
- 允许目录：
  - `apps/api-server/cmd/`
  - `apps/api-server/config/`
  - `apps/api-server/internal/app/`
  - `apps/api-server/internal/interfaces/http/`
  - `apps/api-server/internal/modules/taskboard/`
  - `apps/api-server/routes/`
  - `apps/api-server/go.mod`
  - `apps/api-server/go.sum`
- 语言限定：
  - `Go`
- 主要职责：
  - 对外 API 路由和返回契约
  - Markdown 任务板存储与状态同步
  - 配置、启动、运行时行为
  - 路由级与任务板级测试
- 默认禁止：
  - `apps/api-server/internal/modules/agent/`
  - `apps/pad-app/`
  - `apps/parent-web/`
- 下一阶段任务：
  - 统一任务相关接口的错误返回结构
  - 补任务板接口的边界测试和异常测试
  - 明确 `auth`、`points` 这类占位能力的后续处理方式
  - 固化任务板数据结构，避免 UI 组反复追着接口改

### 3.2 `Go-Agent 组`

- 组目标：
  提升作业解析和周报生成质量，同时严格遵循 Google Agentic design pattern，不把模型能力扩散到状态写入链路。
- 允许目录：
  - `apps/api-server/internal/modules/agent/`
  - `apps/api-server/internal/platform/llm/`
  - `apps/api-server/internal/shared/agentic/`
  - `docs/04_AGENTIC_DESIGN.md`
- 语言限定：
  - `Go`
- 主要职责：
  - `taskparse` 解析质量
  - `weeklyinsights` 周报摘要
  - Ark / OpenAI compatible LLM 调用
  - Agentic 模式选择、边界和解释
  - Agent 模块级测试
- 默认禁止：
  - `apps/api-server/internal/modules/taskboard/`
  - `apps/pad-app/`
  - `apps/parent-web/`
- 下一阶段任务：
  - 补更多真实学校群解析样本
  - 做“续做/订正/条件任务/对象不明确”的识别增强
  - 让 weekly insight 输出更稳定，避免空洞 mock 风格
  - 把 agentic pattern 写进更多测试和输出元数据

### 3.3 `Flutter Pad 组`

- 组目标：
  让孩子端任务板在 Chrome、iPad、手机上都能稳定加载、勾选、刷新，并对网络异常有明确反馈。
- 允许目录：
  - `apps/pad-app/lib/`
  - `apps/pad-app/test/`
  - `apps/pad-app/pubspec.yaml`
  - `apps/pad-app/pubspec.lock`
  - `apps/pad-app/macos/`
  - `apps/pad-app/web/`
- 语言限定：
  - `Dart / Flutter`
- 主要职责：
  - 任务板 UI
  - API 调用与状态同步
  - 终端体验和交互反馈
  - Widget / integration 测试
- 默认禁止：
  - `apps/api-server/`
  - `apps/parent-web/`
- 下一阶段任务：
  - 抽出稳定的 API client 层
  - 明确加载态、空态、错误态
  - 支持日期切换和任务板刷新
  - 提升勾选同步后的局部刷新体验
  - 增加至少一组任务板 UI 测试

### 3.4 `Parent Web 组`

- 组目标：
  把家长输入、AI 解析预览、编辑确认、创建任务的流程做顺畅，让解析质量问题能被家长快速发现和修正。
- 允许目录：
  - `apps/parent-web/src/`
  - `apps/parent-web/package.json`
  - `apps/parent-web/package-lock.json`
  - `apps/parent-web/index.html`
- 语言限定：
  - `JavaScript / React`
- 主要职责：
  - 家长输入和确认流程
  - 低置信度任务的高亮与筛选
  - 解析预览与确认体验
  - Web 端构建和页面行为
- 默认禁止：
  - `apps/api-server/`
  - `apps/pad-app/`
- 下一阶段任务：
  - 明确 `needs_review` 的交互提示
  - 增加任务编辑与删除前确认体验
  - 强化接口失败时的页面反馈
  - 固化解析预览与最终创建结果的差异展示

### 3.5 `集成与发布组`

- 组目标：
  让新人在 30 分钟内完成本地启动和演示；让各组不直接碰别人代码也能完成联调。
- 允许目录：
  - `docs/`
  - `scripts/`
  - `README.md`
  - `.env.example`
  - `docker-compose.yml`
- 语言限定：
  - `Markdown`
  - `Shell`
  - `YAML`
- 主要职责：
  - 运行手册
  - 安全和环境变量规范
  - 联调脚本、验收脚本
  - 版本说明、任务拆分、发布说明
- 默认禁止：
  - 默认不改业务源码
- 下一阶段任务：
  - 增加一键检查运行环境的脚本
  - 固化演示流程
  - 补一个最小 smoke test 清单
  - 管理 API/客户端协作文档

## 4. 非工作目录

下面这些目录默认不是开发目录，不应该被任何组当成工作面：

- `apps/.gocache/`
- `apps/.modcache/`
- `apps/api-server/.gomodcache/`
- `apps/parent-web/node_modules/`
- `apps/parent-web/dist/`
- `apps/pad-app/.dart_tool/`
- `apps/pad-app/build/`
- `data/workspaces/`

规则：

1. 不在这些目录里改业务代码
2. 不把这些目录当作协作边界
3. 出现差异时优先视为生成物，而不是人工源码

## 5. 协作接口与 Owner

### 5.1 HTTP 接口 Owner

- Owner：`Go-API 组`
- 消费方：`Flutter Pad 组`、`Parent Web 组`
- 规则：
  - UI 组不能直接改 Go 接口返回结构
  - 任何接口结构变更，都必须先补后端测试，再通知 UI 组

### 5.2 解析语义 Owner

- Owner：`Go-Agent 组`
- 消费方：`Parent Web 组`
- 规则：
  - `subject`、`group_title`、`title`、`confidence`、`needs_review` 的语义由 `Go-Agent 组` 维护
  - `Parent Web 组` 负责把这些字段正确展示出来，但不重新定义其含义

### 5.3 任务板展示 Owner

- Owner：`Flutter Pad 组`
- 上游：`Go-API 组`
- 规则：
  - 任务板布局、交互、用户提示由 Flutter 组决定
  - 任务板数据结构稳定性由 Go-API 组保证

### 5.4 文档与运行规范 Owner

- Owner：`集成与发布组`
- 规则：
  - 运行手册、安全说明、协同流程由该组维护
  - 其他组如果需要新增启动步骤，应通过该组更新文档

## 6. 多 Codex 工作方式

具体终端命名、启动命令和开场提示词见 `docs/09_CODEX_TERMINAL_COMMANDS.md`。

推荐每个组固定一个 Codex 会话，并把工作目录切到自己的主目录：

1. `Go-API Codex`
   - 建议 cwd: `apps/api-server`
2. `Go-Agent Codex`
   - 建议 cwd: `apps/api-server/internal/modules/agent`
3. `Flutter Codex`
   - 建议 cwd: `apps/pad-app`
4. `Parent Web Codex`
   - 建议 cwd: `apps/parent-web`
5. `Integration Codex`
   - 建议 cwd: `docs` 或仓库根目录

执行规则：

1. 每个 Codex 默认只改自己拥有的目录
2. 需要跨组改动时，先让拥有者组处理
3. 同一时间不要让两个 Codex 改同一个目录层级
4. 如果任务必须跨两组，先拆成两个子任务，再分别分发

## 7. 下一阶段的并行任务包

建议按下面 5 个任务包并行启动：

### 7.1 包 A：Go-API 稳定化

- 交付目标：
  稳定 `tasks` / `status` / `stats` 相关接口，保证 UI 组接入时不再频繁改返回结构
- 产出：
  - 更完整的接口测试
  - 更统一的错误结构
  - 更清晰的配置和启动行为

### 7.2 包 B：Go-Agent 解析增强

- 交付目标：
  提升解析质量，让“订正、续做、条件任务”识别更稳定
- 产出：
  - 新样本测试集
  - 解析增强逻辑
  - 更稳定的周报摘要输出

### 7.3 包 C：Pad 端任务板体验

- 交付目标：
  让孩子端加载、勾选、刷新、切日期都稳定
- 产出：
  - API client 抽象
  - 任务板状态机
  - UI 测试

### 7.4 包 D：家长端确认体验

- 交付目标：
  让家长能快速识别风险任务并确认创建
- 产出：
  - 风险高亮
  - 编辑确认流程
  - 错误反馈与重试

### 7.5 包 E：集成与演示

- 交付目标：
  让任何一个新人能按文档完成演示
- 产出：
  - 运行脚本
  - 联调清单
  - 演示手册

## 8. 成功标准

这套分组是否有效，用下面几条衡量：

1. 不同 Codex 会话可以同时推进，不需要频繁改同一批文件
2. UI 组不会直接去改 Go 后端，Go 组也不会直接改 Flutter/React 页面
3. 任务板和解析接口在一个迭代周期内保持稳定
4. 文档、脚本、环境变量不再由每个开发者各写一套
5. 新人能根据目录和 `AGENTS.md` 直接知道自己能改什么、不能改什么

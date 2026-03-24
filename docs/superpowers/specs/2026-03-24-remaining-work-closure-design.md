# Remaining Work Closure Design (Code + Docs)

## Context
用户要求“检查还有其他未完成的任务，并依次按照优先级完成”，且明确范围包含：
1) 代码层未完成项；
2) 文档层未完成项；
并要求“直接连续执行”，且把“全部完成后再跑一轮全量验证，全部通过后进入最终提交/推送”作为硬门槛。

当前仓库观察到：
- `git status --short` 仅有 `?? docs/superpowers/`（目录未纳入当前主分支跟踪）。
- 业务代码层面关键改动（热任务推荐链路）已在 main 上通过并完成 push。
- 文档层面存在“当前状态 vs 历史状态”表达混杂，需要治理成“主线当前口径 + 历史归档口径”双轨清晰。

## Goal
在不引入额外功能范围的前提下，清空当前“代码+文档”未完成项，并以一次全量验证通过作为唯一出闸条件，最终完成提交与推送。

## Scope
### In scope
- 代码侧：仅处理当前工作区实际未完成项（若无代码差异，则不制造新改动）。
- 文档侧：统一 `docs/14_NEXT_PHASE_DISPATCH.md`、`docs/17_DELIVERY_READINESS.md`、`docs/19_DELIVERY_UAT_CASES.md`、`docs/20_RELEASE_SYNC_PLAYBOOK.md` 的“当前态/历史态”表达边界。
- 交付侧：执行全量验证命令并记录结果摘要，作为提交说明依据。

### Out of scope
- 新功能开发（API/Pad/Parent 新需求）
- 与本轮无关的大规模重构
- 依赖升级（例如 wasm warning 相关替换）

## Priority Model
### P0（必须清空）
1. 当前工作区真实未完成项（`git status`）
2. 发布/验收主文档口径冲突（影响执行与交接）
3. 提交前全量验证通过（硬门槛）

### P1（本轮收口）
4. `DELIVERY_READINESS` 台账更新与证据回填（状态/owner/命令证据）

### P2/P3（记录不执行）
5. 非阻塞已知风险（如 wasm dry-run warning）继续保留观察态

## Design
### 1) Discovery & Freeze
- 先基于当前 `git status` 冻结“真实待办清单”。
- 若某类（代码/文档）在当前状态无待办，不人为新增任务。

### 2) P0 Serial Execution
- 按顺序执行：
  1. 代码未完成项（如存在）
  2. 文档口径清理
  3. 全量验证
- 任一步失败即停留当前步修复，禁止跳步。

### 3) Full Verification Gate (Hard)
仅当以下全部通过，才允许提交推送：
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run && npm run build`
- `cd apps/pad-app && flutter analyze && flutter test --no-pub && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

### 4) Commit/Push Policy
- 仅提交本轮待办涉及文件，不扩大范围。
- Commit message 以“为什么”优先，保持仓库现有风格。
- push 到当前目标分支（本轮为 `main`）。

## Error Handling
- 测试失败：先修复失败根因，再重跑该层验证；不允许跳过。
- 端口冲突/服务未启动：按 runbook 启停后重试，不使用破坏性命令。
- 文档口径冲突：以 `ROADMAP + DISPATCH` 的当前态为锚，历史内容明确标注“历史收口快照”。

## Acceptance Criteria
1. `git status --short` 仅保留明确允许的未跟踪项（或清空）。
2. P0/P1 清单全部完成，无悬空状态。
3. 全量验证命令全部通过。
4. 产生可审计的提交与推送记录。

## Execution Order (Concrete)
1. 盘点当前真实未完成项并冻结清单。
2. 处理代码层待办（如有）。
3. 处理文档层待办并统一口径。
4. 跑全量验证（硬门槛）。
5. 仅在全绿后提交并推送。

## Notes
- 本轮优先级执行策略已获用户确认：连续执行，不逐步等待人工确认。
- 若执行中发现新增阻塞，自动提升为 P0 并先处理。

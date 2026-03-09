# Agent Regression Package

这份回归包只覆盖当前 `agent` 范围内已经落地的确定性规则和 bounded LLM 回退，不涉及 `taskboard` 存储和 HTTP handler。

## 1. 解析回归样本集

样本文件：

- `taskparse/testdata/regression_cases.json`

目的：

- 把本轮新增的订正、续做、条件任务、对象范围、误报控制统一收口成可回放的 fixture 集合
- 让团队可以直接看到每个样本为什么 `needs_review`，而不是从测试断言里反推规则

当前高风险原因和对应 note：

- 条件任务：`包含条件性说明，建议家长确认触发条件。`
- 对象范围不明确：`作业适用对象不明确，建议家长确认是否针对孩子。`
- 订正或续做但目标不明确：`订正/续做任务未写明具体对象，建议家长确认完成内容。`
- 学科不明确：`学科不明确，建议家长确认归类。`

当前 fixture 已覆盖：

- 订正且目标明确
- 订正但目标不明确
- 续做且目标明确
- 续做但目标不明确
- 条件任务
- 部分同学
- 个别同学
- 相关同学
- 正常编号任务、正常子步骤、明确对象的订正/续做不应误报

## 2. Weekly Insight 极端输入回归

样本文件：

- `weeklyinsights/testdata/extreme_cases.json`

回归目标：

- 空数据时稳定回退，不报错、不返回空结构
- 单日数据时仍然输出完整 insight
- 全完成时保持确定性正向总结
- 全未完成时也返回完整结构，方便前端稳定渲染

## 3. 给 SC-05 的最小联调样本

联调输入文件：

- `testdata/sc05_smoke_samples.json`

建议收进 smoke 或手工验收清单的最小样本：

- `taskparse_mixed_risk_and_safe`
  同时覆盖明确订正、对象范围不明确、条件任务、正常子步骤
- `taskparse_all_safe`
  用来卡住误报，确保正常任务不会被误判成 `needs_review`
- `weekly_empty_week`
  验证 weekly insight 空数据回退
- `weekly_all_done`
  验证 weekly insight 在高完成率场景下的稳定输出

## 4. 运行命令

```bash
cd apps/api-server
env GOCACHE=/tmp/studyclaw-go-build-cache go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

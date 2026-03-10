# Agent Regression Package

这份回归包只覆盖当前 `agent` 范围内已经落地的确定性规则和 bounded LLM 回退，不涉及 `taskboard` 存储和 HTTP handler。

配套演示说明：

- `AGENT_QUALITY_GUIDE.md`

## 1. 解析回归样本集

样本文件：

- `taskparse/testdata/regression_cases.json`

结构字段：

- `risk_type`
  当前用于区分 `safe_actionable`、`ambiguous_target`、`conditional`、`audience_scope`、`false_positive_guard`
- `why`
  解释这个样本为什么存在
- `why_needs_review`
  只在高风险样本上填写，明确为什么必须人工确认
- `should_not_misjudge_when`
  明确这一类样本在什么情况下不应误判，作为误报控制基线

目的：

- 把本轮新增的订正、续做、条件任务、对象范围、误报控制统一收口成可回放的 fixture 集合
- 让团队可以直接看到每个样本为什么 `needs_review`，而不是从测试断言里反推规则

当前高风险原因和对应 note：

- 条件任务：`包含条件性说明，建议家长确认触发条件。`
- 对象范围不明确：`作业适用对象不明确，建议家长确认是否针对孩子。`
- 订正或续做但目标不明确：`订正/续做任务未写明具体对象，建议家长确认完成内容。`
- 学科不明确：`学科不明确，建议家长确认归类。`

当前高风险类型说明：

- `ambiguous_target`
  为什么 needs_review：只有订正或续做动作，没有明确对象、页码、卷号或错题范围
  不应误判：一旦文本已写明 `P18`、`1号卷`、`错词`、`错题` 等具体目标，就应按可直接执行处理
- `conditional`
  为什么 needs_review：任务是否成立取决于前置条件，例如“全对可免抄”
  不应误判：普通的“完成”“阅读”“假若”等非条件语义不应因为单个字词被抬成条件任务
- `audience_scope`
  为什么 needs_review：老师只指向“部分同学 / 个别同学 / 相关同学”，系统无法确认是否针对当前孩子
  不应误判：面向全班且目标明确的普通任务不应因为附带编号或步骤被误判为对象范围不明确

当前 fixture 已覆盖：

- 真实学校群多学科混合样本
- runbook 风格简洁样本
- 订正且目标明确
- 订正但目标不明确
- 续做且目标明确
- 续做但目标不明确
- 条件任务
- 选做任务
- `若有时间` 条件任务
- 部分同学
- 个别同学
- 相关同学
- 未完成的同学
- 有需要的同学
- 正常多学科子步骤样本
- 正常编号任务、正常子步骤、明确对象的订正/续做不应误报

## 2. Weekly Insight 极端输入回归

样本文件：

- `weeklyinsights/testdata/extreme_cases.json`
- `progressinsights/testdata/report_cases.json`

结构字段：

- `why`
  解释为什么这是高价值极端输入
- `regression_focus`
  当前案例主要盯的稳定性点
- `should_not_break_when`
  明确这类输入下不能退化成什么坏结果

回归目标：

- 日报 / 周报 / 月报都走“确定性统计 -> 正向总结”同一条链路
- 空数据时稳定回退，不报错、不返回空结构
- 有日期但没有 `tasks` 字段时稳定回退
- 稀疏周输入下指标稳定
- 单日数据时仍然输出完整 insight
- 全完成时保持确定性正向总结
- 全未完成时也返回完整结构，方便前端稳定渲染
- 混合异常 task 条目时不崩溃

阶段一报告 fixture 已覆盖：

- 日报全完成正向总结
- 周报混合完成率总结
- 月报长周期稀疏进展总结
- 模型输出脏数据时的归一化与模板回退

## 3. 给 SC-05 的最小联调样本

联调输入文件：

- `testdata/sc05_smoke_samples.json`

建议收进 smoke 或手工验收清单的最小样本：

- `taskparse_mixed_risk_and_safe`
  同时覆盖明确订正、对象范围不明确、条件任务、正常子步骤
- `taskparse_all_safe`
  用来卡住误报，确保正常任务不会被误判成 `needs_review`
- `daily_supportive_report`
  验证日报只解释确定性统计，不改写数字
- `weekly_supportive_report`
  验证周报与确定性统计严格对齐
- `monthly_supportive_report`
  验证月报长周期总结与模板回退
- `weekly_empty_week`
  验证 weekly insight 空数据回退
- `weekly_all_done`
  验证 weekly insight 在高完成率场景下的稳定输出
- `weekly_all_incomplete`
  验证 weekly insight 在低完成率场景下也保持完整结构

建议 SC-05 在 smoke 里至少检查：

- `taskparse` 返回的 `needs_review_titles` 是否与样本预期严格一致
- `taskparse` 的 safe titles 是否没有被附带 review note
- `weeklyinsights` 的 raw metrics 是否与样本输入严格一致
- `weeklyinsights` 的四个核心字段 `summary / strengths / areas_for_improvement / psychological_insight` 是否都非空
- `progressinsights` 的 `report_type / raw metrics / completion_rate_percent / raw_points_delta` 是否与输入严格一致

## 4. 运行命令

```bash
cd apps/api-server
env GOCACHE=/tmp/studyclaw-go-build-cache go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

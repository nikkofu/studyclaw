# Agent Quality Guide

这份说明面向演示、联调和日常回归，解释 StudyClaw 当前为什么会把某些任务标记为 `needs_review`，以及什么情况不应误判。

作用范围：

- `taskparse`
- `progressinsights`
- `weeklyinsights`

不包含：

- `taskboard` 存储
- HTTP handler
- 前端展示逻辑

## 1. `taskparse` 为什么会标 `needs_review`

当前只有几类风险会触发 `needs_review`：

### 1.1 条件任务

典型样本：

- `默写全对可免抄M2单词`
- `若有时间完成口算一页`
- `口算本第8页选做`

为什么 needs_review：

- 任务是否成立取决于前置条件
- 系统无法替家长判断孩子是否满足条件

不应误判：

- 普通的 `完成口算本第5页`
- `阅读《假若给我三天光明》并摘抄好词`
- 没有条件触发语义的普通编号任务

### 1.2 对象范围不明确

典型样本：

- `部分学生订正练习卷`
- `个别同学继续完成口算本P9`
- `相关同学背诵第3课`
- `未完成的同学补做阅读卷`
- `有需要的同学再做一张口算纸`

为什么 needs_review：

- 老师不是明确面向全班布置
- 系统无法确认当前孩子是否属于该范围

不应误判：

- 面向全班的普通背诵、预习、页码作业
- 明确对象、明确页码的普通订正或续做任务

### 1.3 订正或续做但目标不明确

典型样本：

- `继续订正`
- `继续完成`

为什么 needs_review：

- 只有动作，没有对象
- 系统无法确认到底是哪个本子、哪一页、哪一份卷子、哪一类错题

不应误判：

- `订正默写本P3错词`
- `继续完成校本P18`
- `部分学生续做1号卷剩余题`

说明：

- 最后一个例子虽然目标明确，但因为对象范围不明确，仍然会因 `audience_scope` 触发 `needs_review`

## 2. `taskparse` 什么情况下应保持可直接执行

下列情况默认应保持 `needs_review=false`：

- 学科明确、动作明确、目标明确
- 多学科混合群消息里的普通任务
- 主任务下拆出来的正常子步骤
- 明确对象的订正、续做、背诵、预习、听录音、完成页码

典型样本：

- `校本P14～15`
- `练习册P12～13`
- `订正第2页错题`
- `继续完成校本P19`
- `预习第6课 -> 圈画生字 / 朗读课文三遍`

## 3. `progressinsights` / `weeklyinsights` 的质量基线

第一阶段的日报 / 周报 / 月报总结坚持两条原则：

### 3.1 指标先确定性聚合

- 总任务数
- 完成数
- 完成率
- active days

这些都由 Go 代码先算出来，LLM 不负责计算。

### 3.2 极端输入下结构必须稳定

当前重点回归：

- 空数据
- 有日期但没有 tasks
- 稀疏周，只有少数几天有任务
- 单日数据
- 日报确定性统计输入
- 月报长周期稀疏统计输入
- 全完成
- 全未完成
- 混合异常 task 条目

演示时可以直接说明：

- 即使模型不可用，或输入极端稀疏，系统仍然返回完整的 `summary / strengths / areas_for_improvement / psychological_insight`
- 日报 / 周报 / 月报都先吃确定性统计输入，不直接吃前端临时拼出来的文案
- `raw_metric_total`、`raw_metric_completed`、`completion_rate_percent`、`raw_points_delta` 始终以确定性统计为准

## 4. 演示建议话术

如果需要向产品、家长或联调同学解释 `needs_review`，建议用下面的简单说法：

- `needs_review` 不代表系统解析失败
- 它表示这条任务里存在“系统不该替家长做决定”的信息
- 主要就是三类：有条件、对象范围不清、动作有但目标不清
- 其余明确可执行的任务，系统会尽量直接拆成孩子能勾选完成的原子任务

## 5. 对应回归文件

- 解析回归样本：`taskparse/testdata/regression_cases.json`
- 解析 fixture 测试：`taskparse/regression_fixture_test.go`
- 报告 fixture：`progressinsights/testdata/report_cases.json`
- 报告测试：`progressinsights/service_test.go`
- weekly 极端输入：`weeklyinsights/testdata/extreme_cases.json`
- weekly fixture 测试：`weeklyinsights/extreme_fixture_test.go`
- SC-05 最小联调样本：`testdata/sc05_smoke_samples.json`

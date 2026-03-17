# StudyClaw v0.4.0 一页摘要

发布日期：`2026-03-16`

## 这是什么

`v0.4.0` 是第二阶段 `Agentic 学习助手` 的正式基线。

它把 StudyClaw 从“孩子端能说、系统能分析”的单端能力，推进到了“家长也能看到一次语音学习到底学成什么样”的跨端闭环。

## 给家长

- 发布背诵 / 朗读任务时，可以先把学习素材准备好
- 孩子完成一次语音学习后，家长端能看到结果摘要
- 不再只看“完成没完成”，还能看：
  - 标题 / 作者识别
  - 完成度
  - 逐句不稳的重点
  - 是否建议重练

## 给团队

- `v0.4.0` 的关键突破不是新加一块 UI，而是打通了：
  - Parent 发布学习素材
  - Pad 持续监听与 transcript 分段
  - grounded recitation analysis
  - Parent 复盘语音学习结果
- 第二阶段最关键的家长复盘闭环已经成立

## 给 GitHub / 对外交付

`v0.4.0` 代表的是：

- 代码版本已统一到 `0.4.0`
- 文档版本已统一到 `v0.4.0`
- 语音学习结果摘要已真正成为正式产品能力，而不是 Pad 端临时展示

## 本版包含的关键变化

### 1. 学习素材管理前置化

- 家长发布阶段即可确认背诵 / 朗读参考内容
- 来源与隐藏策略可追溯

### 2. 持续监听与 transcript 分段正式收口

- 孩子可手动开始、持续说话、手动结束
- transcript 形成分段和时间点

### 3. 家长端语音学习复盘补齐

- 新增 `语音` 反馈视图
- 新增语音学习会话持久化
- 家长可直接决定是否重练

## 验证状态

`v0.4.0` 当前已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `go test ./... -count=1`
- `npm test -- --run`
- `npm run build`
- `flutter analyze`
- `flutter test --no-pub`

## 一句话结论

`v0.4.0` 代表的是：StudyClaw 第二阶段已经从“Pad 端语音能力增强”推进到“家长可复盘语音学习结果”的正式可交付基线。

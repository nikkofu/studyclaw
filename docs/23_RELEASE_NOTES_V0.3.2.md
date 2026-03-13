# StudyClaw v0.3.2 发布说明

发布日期：`2026-03-13`

## 版本定位

`v0.3.2` 是在 `v0.3.1` 正式交付基线上的补丁发布。

本次不引入新的后端接口，也不改三端总体结构，重点只做两条高频主路径修复：

1. 家长端“去录入原文”入口修复
2. Pad Web “开始说话”语音指令修复

## 本次发布包含什么

### 1. 家长端原文录入主路径修复

- 修复发布主屏点击“去录入原文”后被错误回退到 `范围` 页的问题
- 空状态下不再错误拦截正常录入流程
- 家长现在可以直接进入 `原文` 子页面并看到输入框

### 2. Pad Web 语音启动与收尾修复

- 修复 `speech_to_text.listen()` 返回值被误当成 `bool` 判断，导致的 `type 'Null' is not a bool in boolean expression`
- 修复 Web 端只收到 interim transcript、随后收到 `done / notListening` 时被误判失败的问题
- “好了 / 下一个 / 继续 / Next / 数学订正好了 / 一课一练做完了”这类短指令的成功率更稳定

## 对使用者的实际影响

### 家长

- 发布作业时，不会再点了“去录入原文”却看不到输入框
- 发布主路径少了一次错误回退，碎片时间操作更顺

### 孩子

- 点击“开始说话”不再因为空值布尔判断直接报错
- 说完短口令后，更容易稳定完成“下一个 / 完成当前任务”这类交互

### 团队 / 交付

- 无新增破坏性 API 变更
- 文档、版本号、发布说明已同步到 `v0.3.2`
- 可以把 `v0.3.2` 作为下一阶段前的正式热修复基线

## 验证摘要

本次发布前已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter test --no-pub`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/19_DELIVERY_UAT_CASES.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/19_DELIVERY_UAT_CASES.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.3.2.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.3.2.md)
- [docs/PARENT_WEB_H5_MANUAL.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/PARENT_WEB_H5_MANUAL.md)

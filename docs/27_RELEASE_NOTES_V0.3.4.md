# StudyClaw v0.3.4 发布说明

发布日期：`2026-03-14`

## 版本定位

`v0.3.4` 是在 `v0.3.3` 正式基线上的 Pad 体验补强版。

这次发布不再新增一条大的业务主链，而是把已经进入孩子日常使用主路径的两个问题彻底收口：

1. 没有词单时，不能把技术错误直接扔给孩子。
2. 成长鼓励既要看得见，也要能被温和地播报出来，尤其是在平板真机上。

## 本次发布包含什么

### 1. 词单缺失友好等待态

- Pad 在后台没有当天默写词单时，不再展示 `404 / TaskApiException`
- `word_list_not_found` 会被翻译成孩子可理解的等待提示
- 听写面板会进入“待补充 / 重新同步 / 等家长补充词单后再来默写”的等待态
- 旧会话、旧词单和旧交卷快照会被清空，避免孩子误操作

### 2. 成长鼓励语音播报

- 任务板“成长小鼓励”支持自动播报
- 语音工作台“成长鼓励”支持自动播报
- 两处都支持手动重播和自动播报开关
- 语音话术改成更适合儿童陪伴场景的温和表达，并使用更柔和的语速与语调参数

### 3. 平板 / 非 Web TTS 补齐

- `WordSpeaker` 不再只在浏览器里生效
- Pad 真机、平板等 `dart:io` 场景现在也会走统一 TTS 实现
- 对 `flutter_tts` 增加 `MissingPluginException` 保护，避免个别环境下因为插件缺失把主流程打断

## 对使用者的实际影响

### 家长

- 如果当天忘了发布词单，孩子看到的是“等家长补充词单”，而不是报错页面
- 这能把问题更明确地反馈到“请补词单”这个真实动作上

### 孩子

- 不会再被技术型错误文案打断学习情绪
- 做完任务后会更自然地收到鼓励播报
- 在平板上使用时，语音鼓励不再只是静态文字

### 团队 / 交付

- 这次补的是正式使用体验，不是只修测试
- `v0.3.4` 让 Pad 端在“无词单”“有鼓励”“平板真机”三个高频场景下更接近真正可交付状态

## 验证摘要

本次发布准备阶段已完成：

- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test --no-pub`

## 2026-03-14 补充验证

发布后的联调补充复核已完成：

- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`
- `curl http://127.0.0.1:5173/`
- `curl http://127.0.0.1:55771/`

补充结果说明：

- 三端入口可直接返回有效页面
- `smoke/demo` 已在当前环境复核通过
- `flutter_tts` 的 wasm dry-run warning 仍存在，但不影响本阶段 HTML/Web 交付

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/19_DELIVERY_UAT_CASES.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/19_DELIVERY_UAT_CASES.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.3.4.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.3.4.md)
- [docs/PARENT_WEB_H5_MANUAL.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/PARENT_WEB_H5_MANUAL.md)

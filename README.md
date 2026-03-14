# StudyClaw

StudyClaw 是一套面向家庭学习场景的三端协同系统：

- `API`: Go 后端，作为任务、积分、词单、听写会话和统计的唯一事实源
- `Parent Web`: React 管理端，负责 `parse -> review -> confirm` 发布和反馈查看
- `Pad App`: Flutter 孩子端，负责任务执行、积分反馈和听写练词

## 当前阶段

- 当前交付版本：`v0.3.4`
- 当前状态：在 `v0.3.3` 稳定基线之上，继续完成 Pad 端“词单缺失友好等待态 + 成长鼓励语音播报 + 平板 TTS 补齐”收口
- 版本对齐：根文档、`apps/parent-web/package.json`、`apps/pad-app/pubspec.yaml` 已统一到 `v0.3.4` 基线

## 当前已闭环能力

### 家长端

- 移动优先 H5 工位：桌面浏览器打开时也按手机单列宽度、固定底部导航和单手操作路径呈现
- 顶部首屏已压成“日期 + 当前主屏 + 当日摘要”的短头部，不再保留 PC 仪表盘式大首页
- 四大主屏收口为 `发布 / 反馈 / 积分 / 单词`，每个主屏继续拆分成可点击切换的 H5 子页面
- 发布主屏继续拆成 `范围 / 原文 / 审核 / 发布 / 拆分 / 任务 / 摘要 / 任务板`，并用粘性子菜单切换，避免长页面里反复上下找模块
- 发布、反馈、积分、单词子页面支持 App 风格左右切页动效，避免长页面里反复上下滚动查找
- 点击“去录入原文”会直接进入 `原文` 子页面，不再被空状态逻辑错误带回 `范围`
- 群消息式作业文本解析
- 审核草稿并确认发布
- 背诵 / 朗读类任务支持学习素材字段：`reference_title`、`reference_author`、`reference_text`、`analysis_mode`
- 家长手动输入学习素材时始终优先保留；如果未输入，先从老师原文自动抽取，仍缺失时再走 LLM 补全
- 审核卡会自动带出古诗词 / 课文标题、作者和正文，背诵任务默认对孩子隐藏标准原文
- 查看当日 / 周 / 月反馈
- 创建词单、查看积分流水、执行人工奖惩

### 孩子端

- 加载当天任务板
- 单任务 / 分组 / 全量完成同步，并在完成时给出成长型正向鼓励
- 当天未配置默写词单时，Pad 不再直接暴露 `404 / TaskApiException`，而是进入“等待家长补充词单”的友好状态
- 后端驱动的词单与听写会话
- 听写推进、交卷、批改完成等节点提供孩子视角的积极反馈
- 基于 STT + LLM 推理的语音助手，可用自然口令触发当前页面按钮行为
- 孩子学习语音工作台支持短指令、长段朗读 / 背诵和陪伴式持续监听三种输入场景
- 背诵分析支持根据隐藏参考原文识别标题 / 作者、比对 noisy transcript，并输出完成度、问题点和重背建议
- “开始说话”入口已修复 Web 场景下的启动与收尾问题，不再出现 `type 'Null' is not a bool in boolean expression`
- 成长小鼓励支持 Pad 端自动语音播报、手动重播和自动播报开关
- Pad 端 TTS 已补到非 Web / 平板场景，不再只在浏览器里可播报
- 积分余额、日报、周报、月报入口

### API 端

- 任务解析、确认写入和任务板读取
- `/api/v1/tasks/parse` 与草稿保存会保留学习素材元数据，不再只返回普通作业标题
- `/api/v1/recitation/analyze` 提供背诵 / 朗读对照分析，支持规则兜底和 LLM 增强
- 任务状态同步和自动积分
- 积分流水 / 余额
- 词单解析、词单持久化、听写会话、日周月统计

## 2026-03-13 交付验证基线

以下验证已在本地仓库状态下执行：

- `cd apps/api-server && go test ./... -count=1`
- `npm test`
- `npm run build`
- `flutter analyze`
- `flutter test`
- `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`

三端联调基线端口：

- API: `http://127.0.0.1:38080`
- Parent Web: `http://127.0.0.1:5173`（当前主干默认以手机 H5 工位样式呈现）
- Pad Web: `http://127.0.0.1:55771`

交付用固定数据：

- `family_id=306`
- `user_id / child_id=1`
- `assigned_date=2026-03-12`

## 快速启动

### 1. 环境预检

```bash
bash scripts/preflight_local_env.sh
```

### 2. 启动 API / Parent / Pad

```bash
# API
cd apps/api-server
API_PORT=38080 go run ./cmd/studyclaw-server

# Parent Web
cd apps/parent-web
VITE_API_BASE_URL=http://127.0.0.1:38080 npm run dev -- --host 127.0.0.1 --port 5173

# Pad Web
cd apps/pad-app
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

### 3. 冒烟和演示入口

```bash
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh
STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 \
STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 \
bash scripts/demo_local_stack.sh
```

## 交付文档

- 运行手册：[docs/06_RUNBOOK.md](docs/06_RUNBOOK.md)
- 用户操作手册：[docs/USER_MANUAL_V0.3.4.md](docs/USER_MANUAL_V0.3.4.md)
- 家长端移动 H5 操作手册：[docs/PARENT_WEB_H5_MANUAL.md](docs/PARENT_WEB_H5_MANUAL.md)
- 阶段一页摘要：[docs/28_PHASE_ONE_PAGER_V0.3.4.md](docs/28_PHASE_ONE_PAGER_V0.3.4.md)
- `v0.3.4` 发布说明：[docs/27_RELEASE_NOTES_V0.3.4.md](docs/27_RELEASE_NOTES_V0.3.4.md)
- 交付就绪审计：[docs/17_DELIVERY_READINESS.md](docs/17_DELIVERY_READINESS.md)
- 交付验收用例：[docs/19_DELIVERY_UAT_CASES.md](docs/19_DELIVERY_UAT_CASES.md)
- Release 同步手册：[docs/20_RELEASE_SYNC_PLAYBOOK.md](docs/20_RELEASE_SYNC_PLAYBOOK.md)
- 发布前检查：[docs/13_RELEASE_CHECKLIST.md](docs/13_RELEASE_CHECKLIST.md)
- 第一阶段演示清单：[docs/16_FIRST_PHASE_DEMO_CHECKLIST.md](docs/16_FIRST_PHASE_DEMO_CHECKLIST.md)

## 当前仓库同步提示

- `v0.3.0` 已作为历史正式版本保留
- `v0.3.1` 已作为上一版正式标签保留
- `v0.3.2` 已作为上一版正式标签保留
- `v0.3.3` 已作为上一版正式标签和 GitHub release 基线同步
- `v0.3.4` 已作为当前正式标签和 GitHub release 基线同步
- 当前正式发版范围包括 Pad 端词单缺失友好等待态、成长鼓励语音播报、平板 TTS 补齐，以及版本文档同步

## 许可

本项目采用 [LICENSE](LICENSE) 进行许可。

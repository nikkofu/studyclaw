# StudyClaw

StudyClaw 是一套面向家庭学习场景的三端协同系统：

- `API`: Go 后端，作为任务、积分、词单、听写会话和统计的唯一事实源
- `Parent Web`: React 管理端，负责 `parse -> review -> confirm` 发布和反馈查看
- `Pad App`: Flutter 孩子端，负责任务执行、积分反馈和听写练词

## 当前阶段

- 当前交付版本：`v0.3.0`
- 当前状态：已收口到“语音助手 + 正向鼓励”增强版，本地达到可发布标准，可作为下一阶段前的稳定基线
- 版本对齐：根文档、`apps/parent-web/package.json`、`apps/pad-app/pubspec.yaml` 已统一到 `v0.3.0` 基线

## 当前已闭环能力

### 家长端

- 群消息式作业文本解析
- 审核草稿并确认发布
- 查看当日 / 周 / 月反馈
- 创建词单、查看积分流水、执行人工奖惩

### 孩子端

- 加载当天任务板
- 单任务 / 分组 / 全量完成同步，并在完成时给出成长型正向鼓励
- 后端驱动的词单与听写会话
- 听写推进、交卷、批改完成等节点提供孩子视角的积极反馈
- 基于 STT + LLM 推理的语音助手，可用自然口令触发当前页面按钮行为
- 积分余额、日报、周报、月报入口

### API 端

- 任务解析、确认写入和任务板读取
- 任务状态同步和自动积分
- 积分流水 / 余额
- 词单解析、词单持久化、听写会话、日周月统计

## 2026-03-12 交付验证基线

以下验证已在本地仓库状态下执行：

- `go test ./... -count=1`
- `npm test`
- `npm run build`
- `flutter analyze`
- `flutter test`
- `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`

三端联调基线端口：

- API: `http://127.0.0.1:38080`
- Parent Web: `http://127.0.0.1:5173`
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
- 用户操作手册：[docs/USER_MANUAL_V0.3.0.md](docs/USER_MANUAL_V0.3.0.md)
- 交付就绪审计：[docs/17_DELIVERY_READINESS.md](docs/17_DELIVERY_READINESS.md)
- 交付验收用例：[docs/19_DELIVERY_UAT_CASES.md](docs/19_DELIVERY_UAT_CASES.md)
- Release 同步手册：[docs/20_RELEASE_SYNC_PLAYBOOK.md](docs/20_RELEASE_SYNC_PLAYBOOK.md)
- 发布前检查：[docs/13_RELEASE_CHECKLIST.md](docs/13_RELEASE_CHECKLIST.md)
- 第一阶段演示清单：[docs/16_FIRST_PHASE_DEMO_CHECKLIST.md](docs/16_FIRST_PHASE_DEMO_CHECKLIST.md)

## 当前仓库同步提示

- `v0.2.0` 历史正式版本已完成 GitHub sync，可作为上一阶段签收基线
- 当前工作树已整理完 `v0.3.0` 的版本文件、文档和功能收口
- `v0.3.0` 发布时应使用 scoped staging，并同步 release commit、tag 和 GitHub push

## 许可

本项目采用 [LICENSE](LICENSE) 进行许可。

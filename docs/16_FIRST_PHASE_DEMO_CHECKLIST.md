# StudyClaw 第一阶段演示与签收清单

本文档用于第一阶段产品能力演示、联调验收和正式签收前功能走查。

适用目标：

- 版本目标：`v0.2.0`
- 目标日期：`2026-03-12`

推荐先完成基础检查：

1. `bash scripts/check_no_tracked_runtime_env.sh`
2. `bash scripts/preflight_local_env.sh`
3. 启动 API / Parent Web / Pad 三端
4. `bash scripts/smoke_local_stack.sh`
5. `bash scripts/demo_local_stack.sh`

如果本次目标是“第一阶段正式签收”，必须确保以下验证通过：

- `docs/17_DELIVERY_READINESS.md`

## 1. 演示前准备

### 1.1 启动服务

API：

```bash
cd apps/api-server
API_PORT=38080 go run ./cmd/studyclaw-server
```

家长端：

```bash
cd apps/parent-web
VITE_API_BASE_URL=http://127.0.0.1:38080 npm run dev -- --host 127.0.0.1 --port 5173
```

Pad 端：

```bash
cd apps/pad-app
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

### 1.2 建议演示用固定日期

建议统一使用固定的 `assigned_date`，避免“今天 / 明天”混淆：

- `2026-03-12`

### 1.3 建议演示用固定数据

建议固定：

- `family_id=306`
- `assignee_id=1`
- Parent Web 侧选择或填写同一组 `family_id / assignee_id / date`

## 2. 第一阶段演示主线

### 2.1 家长发布当天作业
- **动作**：粘贴原文 -> 解析 -> 确认发布。
- **签收点**：任务成功进入后端 `/api/v1/tasks`。

### 2.2 AI 解析并生成任务草稿
- **动作**：查看解析出的学科、置信度。
- **签收点**：解析结果符合 PRD 对“原子任务”的拆解要求。

### 2.3 Pad 打开当天任务并逐项完成
- **动作**：勾选任务。
- **签收点**：勾选动作触发后端 PATCH 请求并持久化，刷新后状态不丢失。

### 2.4 家长查看当日统计与同步结果
- **动作**：刷新家长端当日页面。
- **签收点**：家长端展示的完成数与 Pad 端勾选数实时对应。

### 2.5 单词清单逐词播放
- **动作**：家长端创建清单 -> Pad 进入播放。
- **签收点**：Pad 拉取的清单来自 `/api/v1/word-lists`，播放进度由 `dictation-session` 驱动。

### 2.6 积分变化
- **动作**：Pad 完成任务产生自动积分 -> 家长端手工奖惩。
- **签收点**：双端展示的余额统一，最近明细由 `/api/v1/points/ledger` 提供。

### 2.7 日 / 周 / 月数据与 AI 鼓励
- **动作**：切换日/周/月视图。
- **签收点**：图表数据来自后端聚合，AI 鼓励文案基于后端确定性统计生成。

## 3. 后端事实源一致性专项核查（签收必测）

| 检查项 | 验证动作 | 预期结果 |
| --- | --- | --- |
| **持久化一致性** | 家长端创建单词清单后，清除浏览器缓存并重新登录。 | 单词清单依然存在，证明已持久化到后端。 |
| **双端同步一致性** | 在 Pad 端完成任务，立即在家长端查看月趋势。 | 月趋势中的完成率应同步更新。 |
| **积分权威源** | 手工修改后端数据库/API 的积分余额。 | 双端显示的积分均应变为修改后的值，而非前端缓存值。 |
| **会话状态迁移** | 在 Pad 端播放到第 3 个单词时刷新页面。 | 应能通过 session_id 恢复到第 3 个单词，或正确重播。 |
| **页面可用性** | 访问 `http://127.0.0.1:5173/` 与 `http://127.0.0.1:55771/`。 | Parent Web 与 Pad Web 都能返回有效 HTML。 |

## 4. 建议演示顺序
1. 家长发布作业 -> 2. AI 解析与审核 -> 3. Pad 完成任务 -> 4. 家长查看统计 -> 5. 单词播放 -> 6. 积分变化 -> 7. 日 / 周 / 月反馈

## 5. 验收确认模板
```text
Date: 2026-03-12
Version: v0.2.0
Sign-off Status: [PASS / FAIL]
Verified By:
Notes:
```

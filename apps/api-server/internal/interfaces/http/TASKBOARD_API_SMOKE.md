# StudyClaw Phase 1 API Smoke

本文件提供第一阶段主链路的标准 `curl` 样例，可直接被 SC-05、前端联调或本地 smoke 引用。

相关文档：

- 错误契约：[API_ERROR_CONTRACT.md](/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/internal/interfaces/http/API_ERROR_CONTRACT.md)

## 0. 约定变量

```bash
API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8080}"
FAMILY_ID="${FAMILY_ID:-9306}"
CHILD_ID="${CHILD_ID:-1}"
USER_ID="${USER_ID:-1}"
ASSIGNED_DATE="${ASSIGNED_DATE:-2026-03-18}"
MONTH_KEY="${MONTH_KEY:-2026-03}"
```

说明：

- `CHILD_ID` 用于第一阶段主接口。
- `USER_ID` 保留给旧 `/tasks`、`/points`、`/stats` 兼容接口。
- 推荐固定日期做 smoke，避免依赖服务器本地时间。

## 1. 健康检查

```bash
curl -sS "${API_BASE_URL}/ping"
```

预期：

- HTTP `200`
- `message == "pong"`

## 2. 家长解析每日任务草稿

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/daily-assignments/drafts/parse" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"child_id\": ${CHILD_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\",
    \"source_text\": \"数学：\\n1. 校本P14-15\\n2. 练习册P12-13\\n\\n英语：\\n1. 预习M1U2\\n（1）书本上标注好黄页单词的音标\\n（2）沪学习听录音跟读\"
  }"
```

预期：

- HTTP `201`
- 返回 `message`
- 返回 `daily_assignment_draft`
- `daily_assignment_draft.status == "draft"`
- `daily_assignment_draft.task_items` 非空
- `daily_assignment_draft.summary.total_tasks >= 1`

需要记录：

- `daily_assignment_draft.draft_id`

## 3. 家长发布每日任务

把上一步返回的 `draft_id` 替换进去：

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/daily-assignments/publish" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"child_id\": ${CHILD_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\",
    \"draft_id\": \"REPLACE_WITH_DRAFT_ID\"
  }"
```

预期：

- HTTP `201`
- 返回 `daily_assignment`
- 返回 `task_board`
- `daily_assignment.status == "published"`
- `task_board.summary.total >= 1`

兼容说明：

- 如果前端在草稿页做过本地编辑，也可以不带 `draft_id`，直接传 `task_items` 发布。

## 4. Pad 拉取当天任务

```bash
curl -sS "${API_BASE_URL}/api/v1/daily-assignments?family_id=${FAMILY_ID}&child_id=${CHILD_ID}&date=${ASSIGNED_DATE}"
```

预期：

- HTTP `200`
- 顶层稳定字段：
  - `date`
  - `published`
  - `daily_assignment`
  - `task_board`
  - `points_balance`
  - `word_list`（有配置时返回）
- `published == true`
- `task_board.tasks`、`task_board.groups`、`task_board.summary` 可直接给 Pad 用

## 5. 旧任务板兼容查询

如果仍有旧页面使用 `/api/v1/tasks`，继续这样拉：

```bash
curl -sS "${API_BASE_URL}/api/v1/tasks?family_id=${FAMILY_ID}&user_id=${USER_ID}&date=${ASSIGNED_DATE}"
```

预期：

- HTTP `200`
- 稳定字段：
  - `date`
  - `tasks`
  - `groups`
  - `homework_groups`
  - `summary`

## 6. 状态更新兼容接口

### 6.1 单任务状态更新

```bash
curl -sS -X PATCH "${API_BASE_URL}/api/v1/tasks/status/item" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"task_id\": 1,
    \"completed\": true,
    \"assigned_date\": \"${ASSIGNED_DATE}\"
  }"
```

预期：

- HTTP `200`
- 返回 `updated_count`
- 返回 `tasks` / `groups` / `homework_groups` / `summary`

重复调用相同 payload：

- HTTP `409`
- `error_code == "status_unchanged"`

### 6.2 分组状态更新

```bash
curl -sS -X PATCH "${API_BASE_URL}/api/v1/tasks/status/group" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"subject\": \"英语\",
    \"group_title\": \"预习M1U2\",
    \"completed\": true,
    \"assigned_date\": \"${ASSIGNED_DATE}\"
  }"
```

### 6.3 全部状态更新

```bash
curl -sS -X PATCH "${API_BASE_URL}/api/v1/tasks/status/all" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"completed\": true,
    \"assigned_date\": \"${ASSIGNED_DATE}\"
  }"
```

## 7. 旧 Parse / Confirm 兼容链路

### 7.1 Parse 不自动写入

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/tasks/parse" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\",
    \"auto_create\": false,
    \"raw_text\": \"英语：\\n1. 背默M1U1知识梳理单小作文\\n2. 预习M1U2\\n（1）书本上标注好黄页单词的音标\\n（2）沪学习听录音跟读\"
  }"
```

预期：

- HTTP `201`
- 稳定字段：
  - `message`
  - `parsed_count`
  - `parser_mode`
  - `analysis`
  - `auto_created`
  - `date`
  - `tasks`
- 额外返回：
  - `daily_assignment_draft`

### 7.2 Confirm 写入

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/tasks/confirm" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\",
    \"tasks\": [
      {
        \"subject\": \"英语\",
        \"group_title\": \"预习M1U2\",
        \"title\": \"书本上标注好黄页单词的音标\"
      },
      {
        \"subject\": \"英语\",
        \"group_title\": \"预习M1U2\",
        \"title\": \"沪学习听录音跟读\"
      }
    ]
  }"
```

预期：

- HTTP `201`
- 稳定字段：
  - `message`
  - `created_count`
  - `date`
  - `tasks`
- 额外返回：
  - `daily_assignment`

## 8. 积分流水与余额

### 8.1 新积分流水接口

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/points/ledger" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"user_id\": ${USER_ID},
    \"delta\": 2,
    \"source_type\": \"parent_reward\",
    \"occurred_on\": \"${ASSIGNED_DATE}\",
    \"note\": \"主动完成额外练习\"
  }"
```

预期：

- HTTP `201`
- 返回 `points_entry`
- 返回 `points_balance`
- `points_entry.source_type == "parent_reward"`
- `points_entry.source_origin == "parent"`

### 8.2 查询积分流水

```bash
curl -sS "${API_BASE_URL}/api/v1/points/ledger?family_id=${FAMILY_ID}&user_id=${USER_ID}&start_date=${ASSIGNED_DATE}&end_date=${ASSIGNED_DATE}"
```

预期：

- HTTP `200`
- 返回 `entries`
- 返回 `points_balance`
- 完成任务产生的自动积分会以：
  - `source_type == "task_completion"`
  - `source_origin == "system"`
  出现在流水中

### 8.3 查询积分余额

```bash
curl -sS "${API_BASE_URL}/api/v1/points/balance?family_id=${FAMILY_ID}&user_id=${USER_ID}&date=${ASSIGNED_DATE}"
```

预期：

- HTTP `200`
- 返回 `points_balance`
- 稳定字段：
  - `balance`
  - `today_delta`
  - `auto_points`
  - `manual_points`

### 8.4 旧积分兼容接口

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/points/update" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"user_id\": ${USER_ID},
    \"amount\": 1,
    \"reason\": \"晚间任务完成\"
  }"
```

预期：

- HTTP `200`
- 保留旧字段：
  - `message`
  - `balance`

## 9. 单词清单与默写会话

### 9.1 创建单词清单

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/word-lists" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"child_id\": ${CHILD_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\",
    \"title\": \"英语默写 Day 1\",
    \"language\": \"en\",
    \"items\": [
      {\"text\": \"apple\", \"meaning\": \"苹果\"},
      {\"text\": \"orange\", \"meaning\": \"橙子\"},
      {\"text\": \"banana\", \"meaning\": \"香蕉\"}
    ]
  }"
```

预期：

- HTTP `201`
- 返回 `word_list`
- `word_list.total_items == 3`

### 9.2 查询单词清单

```bash
curl -sS "${API_BASE_URL}/api/v1/word-lists?family_id=${FAMILY_ID}&child_id=${CHILD_ID}&date=${ASSIGNED_DATE}"
```

### 9.3 启动默写会话

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/dictation-sessions/start" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"child_id\": ${CHILD_ID},
    \"assigned_date\": \"${ASSIGNED_DATE}\"
  }"
```

预期：

- HTTP `201`
- 返回 `dictation_session`
- `dictation_session.status == "active"`
- `dictation_session.current_item` 存在

### 9.4 重播当前单词

把上一步返回的 `session_id` 替换进去：

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/dictation-sessions/REPLACE_WITH_SESSION_ID/replay"
```

### 9.5 下一个单词

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/dictation-sessions/REPLACE_WITH_SESSION_ID/next"
```

### 9.6 查询会话状态

```bash
curl -sS "${API_BASE_URL}/api/v1/dictation-sessions/REPLACE_WITH_SESSION_ID"
```

## 10. 日 / 周 / 月统计

### 10.1 日统计

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/daily?family_id=${FAMILY_ID}&user_id=${USER_ID}&date=${ASSIGNED_DATE}"
```

预期：

- HTTP `200`
- 稳定字段：
  - `period`
  - `start_date`
  - `end_date`
  - `totals`
  - `subject_breakdown`
  - `completion_series`
  - `points_series`
  - `word_series`
  - `encouragement`

### 10.2 周统计

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/weekly?family_id=${FAMILY_ID}&user_id=${USER_ID}&end_date=${ASSIGNED_DATE}"
```

兼容字段：

- `message`
- `raw_stats`
- `insights`

新字段：

- `period`
- `totals`
- `completion_series`
- `points_series`
- `word_series`
- `encouragement`

### 10.3 月统计

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/monthly?family_id=${FAMILY_ID}&user_id=${USER_ID}&month=${MONTH_KEY}"
```

预期：

- HTTP `200`
- `period == "monthly"`
- 图表 series 由后端确定性生成

## 11. 常见负例 smoke

### 11.1 非法日期

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/weekly?family_id=${FAMILY_ID}&user_id=${USER_ID}&end_date=2026-02-30"
```

预期：

- HTTP `400`
- `error_code == "invalid_date"`

### 11.2 非法月份

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/monthly?family_id=${FAMILY_ID}&user_id=${USER_ID}&month=2026-13"
```

预期：

- HTTP `400`
- `error_code == "invalid_month"`

### 11.3 重复状态更新

重复调用同一个 `status/item` 或 `status/group` / `status/all` 成功 payload。

预期：

- HTTP `409`
- `error_code == "status_unchanged"`

### 11.4 非法积分来源

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/points/ledger" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"user_id\": ${USER_ID},
    \"delta\": 2,
    \"source_type\": \"task_completion\",
    \"occurred_on\": \"${ASSIGNED_DATE}\"
  }"
```

预期：

- HTTP `400`
- `error_code == "invalid_points_source"`

## 12. 本轮建议引用方式

给 Parent Web / Pad / SC-05 的建议是：

- 新主链路优先使用：
  - `/api/v1/daily-assignments/*`
  - `/api/v1/points/ledger`
  - `/api/v1/points/balance`
  - `/api/v1/word-lists`
  - `/api/v1/dictation-sessions/*`
  - `/api/v1/stats/daily|weekly|monthly`
- 旧链路兼容继续保留：
  - `/api/v1/tasks`
  - `/api/v1/tasks/parse`
  - `/api/v1/tasks/confirm`
  - `/api/v1/tasks/status/*`
  - `/api/v1/points/update`

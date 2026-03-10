# StudyClaw API Error Contract

本文件是 `apps/api-server` 当前对外 HTTP 错误契约的唯一引用源。

适用范围：

- Parent Web
- Pad
- Integration / smoke
- `apps/api-server/internal/interfaces/http` 下所有 `/api/v1/*` 路由

相关文档：

- Smoke 样例：[TASKBOARD_API_SMOKE.md](/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/internal/interfaces/http/TASKBOARD_API_SMOKE.md)

## 1. 通用错误包裹结构

所有错误响应统一返回：

```json
{
  "error": "Human readable message",
  "error_code": "machine_readable_code",
  "details": {}
}
```

说明：

- `error`：给日志和兜底 UI 用的可读文本。
- `error_code`：客户端分支判断的主字段。
- `details`：补充上下文，可选。
- 客户端必须同时检查 HTTP status 和 `error_code`。

## 2. `details` 字段约定

常见结构如下。

缺失字段或 query 参数：

```json
{
  "fields": ["family_id", "user_id"]
}
```

单字段非法：

```json
{
  "field": "assigned_date"
}
```

找不到单任务：

```json
{
  "task_id": 99
}
```

找不到任务分组：

```json
{
  "subject": "英语",
  "group_title": "预习M1U2"
}
```

找不到草稿：

```json
{
  "draft_id": "draft_000001"
}
```

找不到单词会话：

```json
{
  "session_id": "session_000001"
}
```

找不到单词清单：

```json
{
  "date": "2026-03-15"
}
```

重复状态更新：

```json
{
  "status": "completed"
}
```

解析失败时的 Agent 原始返回：

```json
{
  "agent_response": {
    "status": "failed"
  }
}
```

## 3. 错误码清单

### 3.1 请求与校验错误

- `missing_required_fields`
  - HTTP `400`
  - JSON 或 query 缺少必填字段
- `invalid_request_fields`
  - HTTP `400`
  - 同一请求内同时存在缺失字段和非法字段
- `invalid_request`
  - HTTP `400`
  - 语义层非法，例如 `task_items` 为空、`end_date < start_date`
- `invalid_json`
  - HTTP `400`
  - 请求体不是合法 JSON
- `invalid_query_parameter`
  - HTTP `400`
  - query 参数不是合法无符号整数
- `invalid_date`
  - HTTP `400`
  - `date` / `assigned_date` / `end_date` / `occurred_on` 不是 `YYYY-MM-DD`
- `invalid_month`
  - HTTP `400`
  - `month` 不是 `YYYY-MM`
- `invalid_points_source`
  - HTTP `400`
  - 手工积分接口传入了不允许的 `source_type`

### 3.2 资源不存在或状态冲突

- `task_not_found`
  - HTTP `404`
  - 单任务不存在，或空任务板做 `status/all`
- `task_group_not_found`
  - HTTP `404`
  - 指定 subject / group_title 没有匹配任务
- `daily_assignment_draft_not_found`
  - HTTP `404`
  - 发布时引用的草稿不存在
- `word_list_not_found`
  - HTTP `404`
  - 单词清单不存在
- `dictation_session_not_found`
  - HTTP `404`
  - 默写会话不存在
- `status_unchanged`
  - HTTP `409`
  - 目标状态和当前状态一致，后端拒绝重复更新

### 3.3 上游与内部错误

- `tasks_not_extractable`
  - HTTP `422`
  - Parse 成功结束但没有提取出可落地任务
- `parser_unavailable`
  - HTTP `502`
  - 任务解析失败
- `internal_error`
  - HTTP `500` / `502`
  - 持久化失败、统计构建失败、会话写入失败等内部问题

## 4. 当前已覆盖路由

以下路由使用本错误契约：

- `POST /api/v1/auth/login`
- `POST /api/v1/internal/parse`
- `POST /api/v1/internal/analyze/weekly`
- `POST /api/v1/tasks`
- `POST /api/v1/tasks/parse`
- `POST /api/v1/tasks/confirm`
- `GET /api/v1/tasks`
- `PATCH /api/v1/tasks/status/item`
- `PATCH /api/v1/tasks/status/group`
- `PATCH /api/v1/tasks/status/all`
- `POST /api/v1/daily-assignments/drafts/parse`
- `POST /api/v1/daily-assignments/publish`
- `GET /api/v1/daily-assignments`
- `POST /api/v1/points/update`
- `POST /api/v1/points/ledger`
- `GET /api/v1/points/ledger`
- `GET /api/v1/points/balance`
- `POST /api/v1/word-lists`
- `GET /api/v1/word-lists`
- `POST /api/v1/dictation-sessions/start`
- `GET /api/v1/dictation-sessions/:session_id`
- `POST /api/v1/dictation-sessions/:session_id/replay`
- `POST /api/v1/dictation-sessions/:session_id/next`
- `GET /api/v1/stats/daily`
- `GET /api/v1/stats/weekly`
- `GET /api/v1/stats/monthly`

## 5. 典型错误样例

缺失字段：

```json
{
  "error": "Required fields are missing",
  "error_code": "missing_required_fields",
  "details": {
    "fields": ["family_id", "child_id"]
  }
}
```

非法日期：

```json
{
  "error": "assigned_date must be in YYYY-MM-DD format",
  "error_code": "invalid_date",
  "details": {
    "field": "assigned_date"
  }
}
```

非法月份：

```json
{
  "error": "month must be in YYYY-MM format",
  "error_code": "invalid_month",
  "details": {
    "field": "month"
  }
}
```

非法积分来源：

```json
{
  "error": "invalid points source",
  "error_code": "invalid_points_source",
  "details": {
    "field": "source_type"
  }
}
```

找不到任务草稿：

```json
{
  "error": "Daily assignment draft not found",
  "error_code": "daily_assignment_draft_not_found",
  "details": {
    "draft_id": "draft_missing"
  }
}
```

重复状态更新：

```json
{
  "error": "All tasks are already completed",
  "error_code": "status_unchanged",
  "details": {
    "status": "completed"
  }
}
```

## 6. 第一阶段冻结边界

以下字段已进入本轮演示期冻结，Web / Pad / Integration 可以直接依赖，不应在下一轮随意改名。

### 6.1 错误包裹字段

- `error`
- `error_code`
- `details`

### 6.2 Daily Assignment

- `daily_assignment_draft`
  - `draft_id`
  - `family_id`
  - `child_id`
  - `assigned_date`
  - `source_text`
  - `status`
  - `parser_mode`
  - `analysis`
  - `task_items`
  - `summary`
  - `created_at`
  - `updated_at`
- `daily_assignment`
  - `assignment_id`
  - `draft_id`
  - `family_id`
  - `child_id`
  - `assigned_date`
  - `source_text`
  - `status`
  - `task_items`
  - `summary`
  - `published_at`
  - `updated_at`

### 6.3 Task Item / Task Board

- `task_items[*]`
  - `task_id`
  - `subject`
  - `group_title`
  - `title`
  - `content`
  - `type`
  - `confidence`
  - `needs_review`
  - `notes`
  - `completed`
  - `status`
  - `points_value`
- 旧任务板兼容字段继续冻结：
  - `date`
  - `tasks`
  - `groups`
  - `homework_groups`
  - `summary`
  - `updated_count`

### 6.4 Points

- `points_entry`
  - `entry_id`
  - `family_id`
  - `user_id`
  - `occurred_on`
  - `delta`
  - `source_type`
  - `source_origin`
  - `source_ref_type`
  - `source_ref_id`
  - `note`
  - `balance_after`
- `points_balance`
  - `family_id`
  - `user_id`
  - `as_of_date`
  - `balance`
  - `today_delta`
  - `auto_points`
  - `manual_points`

积分来源字段冻结为：

- 自动积分：
  - `source_type = "task_completion"`
  - `source_origin = "system"`
- 手工积分：
  - `source_type ∈ {"parent_reward","parent_penalty","school_praise","school_criticism"}`
  - `source_origin = "parent"`

### 6.5 Words / Dictation

- `word_list`
  - `word_list_id`
  - `family_id`
  - `child_id`
  - `assigned_date`
  - `title`
  - `language`
  - `items`
  - `total_items`
  - `created_at`
  - `updated_at`
- `word_list.items[*]`
  - `index`
  - `text`
  - `meaning`
  - `hint`
- `dictation_session`
  - `session_id`
  - `word_list_id`
  - `family_id`
  - `child_id`
  - `assigned_date`
  - `status`
  - `current_index`
  - `total_items`
  - `played_count`
  - `completed_items`
  - `current_item`
  - `started_at`
  - `updated_at`

### 6.6 Stats

- `period`
- `start_date`
- `end_date`
- `totals`
- `subject_breakdown`
- `completion_series`
- `points_series`
- `word_series`
- `encouragement`

兼容性说明：

- `GET /api/v1/stats/weekly` 仍保留 `message` / `raw_stats` / `insights`，用于旧前端兼容。
- 新接入方应优先依赖 `period` / `totals` / `completion_series` / `points_series` / `word_series`。

## 7. 本轮不建议改动

演示期内不要改动以下内容：

- 已冻结字段名
- `source_type` / `source_origin` 枚举值
- `daily_assignment`、`points_balance`、`word_list`、`dictation_session` 顶层命名
- 旧任务板路由的成功字段：
  - `/api/v1/tasks`
  - `/api/v1/tasks/parse`
  - `/api/v1/tasks/confirm`
  - `/api/v1/tasks/status/*`

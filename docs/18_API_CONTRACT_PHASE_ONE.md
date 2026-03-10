# Phase 1 API Contract — Frozen

> **Status: FROZEN**
> All fields, error codes, and HTTP status codes listed here are under contract.
> Frontend must not read fields not listed. Backend must not rename or remove them.
> Changes require explicit versioning.

Base path: `/api/v1`
Content-Type: `application/json`

---

## Shared Error Envelope

Every error response uses this shape:

```json
{
  "error": "Human-readable message",
  "error_code": "snake_case_code",
  "details": { "field": "field_name" }
}
```

`details` is omitted when not applicable.

### Common Error Codes

| code | status | when |
|---|---|---|
| `missing_required_fields` | 400 | Required body/query fields absent; `details.fields[]` lists them |
| `invalid_request_fields` | 400 | Fields present but invalid; `details.fields[]` |
| `invalid_json` | 400 | Malformed JSON body |
| `invalid_request` | 400 | Generic validation failure (e.g. business rule violation) |
| `invalid_query_parameter` | 400 | Non-integer where uint expected; `details.field` |
| `invalid_date` | 400 | Date not in YYYY-MM-DD; `details.field` |
| `internal_error` | 500 | Unexpected server error |

---

## Stats

### GET `/stats/daily`

Returns task/points/word statistics for a single day.

**Query parameters**

| name | type | required | notes |
|---|---|---|---|
| `family_id` | uint | yes | |
| `user_id` | uint | yes | |
| `date` | YYYY-MM-DD | no | defaults to today |

**200 OK**

```json
{
  "period": "daily",
  "start_date": "2026-03-10",
  "end_date": "2026-03-10",
  "totals": {
    "total_tasks": 9,
    "completed_tasks": 4,
    "pending_tasks": 5,
    "completion_rate": 0.44,
    "auto_points": 4,
    "manual_points": 2,
    "total_points_delta": 6,
    "points_balance": 42,
    "word_items": 5,
    "completed_word_items": 3,
    "dictation_sessions": 1
  },
  "subject_breakdown": [
    {
      "subject": "数学",
      "total_tasks": 3,
      "completed_tasks": 2,
      "pending_tasks": 1,
      "completion_rate": 0.67
    }
  ],
  "completion_series": [
    {
      "label": "2026-03-10",
      "date": "2026-03-10",
      "total_tasks": 9,
      "completed_tasks": 4,
      "completion_rate": 0.44
    }
  ],
  "points_series": [
    {
      "label": "2026-03-10",
      "date": "2026-03-10",
      "delta": 6,
      "balance": 42
    }
  ],
  "word_series": [
    {
      "label": "2026-03-10",
      "date": "2026-03-10",
      "total_items": 5,
      "completed_items": 3,
      "sessions": 1
    }
  ],
  "encouragement": "今日已经完成 4 项任务，继续一点点往前推进。"
}
```

**Errors**

| error_code | status | cause |
|---|---|---|
| `missing_required_fields` | 400 | `family_id` or `user_id` absent |
| `invalid_query_parameter` | 400 | `family_id`/`user_id` not uint |
| `invalid_date` | 400 | `date` not YYYY-MM-DD |
| `internal_error` | 500 | build failure |

---

### GET `/stats/weekly`

Returns task/points/word statistics for the past 7 days (ending on `end_date`).

**Query parameters**

| name | type | required | notes |
|---|---|---|---|
| `family_id` | uint | yes | |
| `user_id` | uint | yes | |
| `end_date` | YYYY-MM-DD | no | defaults to today |

**200 OK**

Same shape as `/stats/daily` with `period = "weekly"`.
`completion_series`, `points_series`, and `word_series` will contain up to 7 entries.

```json
{
  "message": "Weekly stats generated successfully",
  "period": "weekly",
  "start_date": "2026-03-04",
  "end_date": "2026-03-10",
  "totals": { "...": "same as daily" },
  "subject_breakdown": [...],
  "completion_series": [...],
  "points_series": [...],
  "word_series": [...],
  "encouragement": "本周已经完成 28 项任务，表现不错！"
}
```

---

### GET `/stats/monthly`

Returns task/points/word statistics grouped by week within a month.

**Query parameters**

| name | type | required | notes |
|---|---|---|---|
| `family_id` | uint | yes | |
| `user_id` | uint | yes | |
| `month` | YYYY-MM | yes | e.g. `2026-03` |

**200 OK**

Same shape as `/stats/daily` with `period = "monthly"`.
`completion_series` / `points_series` / `word_series` entries use weekly buckets (`label` = `week_1`, `week_2`, etc.):

```json
{
  "period": "monthly",
  "start_date": "2026-03-01",
  "end_date": "2026-03-31",
  "totals": { "...": "same as /stats/daily totals" },
  "subject_breakdown": [],
  "completion_series": [
    {
      "label": "week_1",
      "date": "2026-03-01",
      "total_tasks": 20,
      "completed_tasks": 15,
      "completion_rate": 0.75
    }
  ],
  "points_series": [
    {
      "label": "week_1",
      "date": "2026-03-01",
      "delta": 15,
      "balance": 30
    }
  ],
  "word_series": [
    {
      "label": "week_1",
      "date": "2026-03-01",
      "total_items": 15,
      "completed_items": 12,
      "sessions": 3
    }
  ],
  "encouragement": "本月完成率已经达到 74%，继续把剩余任务收好尾。"
}
```

**Errors**

| error_code | status | cause |
|---|---|---|
| `missing_required_fields` | 400 | any required param absent |
| `invalid_month` | 400 | `month` not YYYY-MM or month value out of 1-12 range |
| `invalid_query_parameter` | 400 | `family_id`/`user_id` not uint |
| `internal_error` | 500 | build failure |

---

## Points

### POST `/points/ledger`

Create a manual points ledger entry.

**Request body**

```json
{
  "family_id": 306,
  "user_id": 1,
  "delta": 2,
  "source_type": "parent_reward",
  "note": "主动完成额外练习",
  "occurred_on": "2026-03-10"
}
```

| field | type | required | notes |
|---|---|---|---|
| `family_id` | uint | yes | |
| `user_id` | uint | yes | |
| `delta` | int | yes | positive for reward; negative for penalty |
| `source_type` | string | yes | `parent_reward`, `parent_penalty`, `school_praise`, `school_criticism` |
| `note` | string | no | free-text |
| `occurred_on` | YYYY-MM-DD | no | defaults to today |

**201 Created**

```json
{
  "message": "Points ledger entry created successfully",
  "points_entry": {
    "entry_id": "manual:123",
    "family_id": 306,
    "user_id": 1,
    "occurred_on": "2026-03-10",
    "delta": 2,
    "source_type": "parent_reward",
    "source_origin": "parent",
    "source_ref_type": "manual_adjustment",
    "note": "主动完成额外练习",
    "balance_after": 44
  },
  "points_balance": {
    "family_id": 306,
    "user_id": 1,
    "as_of_date": "2026-03-10",
    "balance": 44,
    "today_delta": 6,
    "auto_points": 4,
    "manual_points": 2
  }
}
```

---

### GET `/points/ledger`

List ledger entries for a date range.

**Query parameters**

| name | type | required |
|---|---|---|
| `family_id` | uint | yes |
| `user_id` | uint | yes |
| `start_date` | YYYY-MM-DD | yes |
| `end_date` | YYYY-MM-DD | yes |

**200 OK**

```json
{
  "entries": [
    {
      "entry_id": "auto:2026-03-10:1",
      "family_id": 306,
      "user_id": 1,
      "occurred_on": "2026-03-10",
      "delta": 1,
      "source_type": "task_completion",
      "source_origin": "system",
      "source_ref_type": "task_item",
      "source_ref_id": "2026-03-10:1",
      "note": "完成数学作业",
      "balance_after": 43
    }
  ],
  "points_balance": { "...": "same as POST response" }
}
```

---

### GET `/points/balance`

Get cumulative balance as of a date.

**Query parameters**

| name | type | required |
|---|---|---|
| `family_id` | uint | yes |
| `user_id` | uint | yes |
| `date` | YYYY-MM-DD | no | defaults to today |

**200 OK**

```json
{
  "points_balance": {
    "family_id": 306,
    "user_id": 1,
    "as_of_date": "2026-03-10",
    "balance": 43,
    "today_delta": 5,
    "auto_points": 32,
    "manual_points": 11
  }
}
```

---

## Word Lists

### POST `/word-lists`

Create or replace a word list.

**Request body**

```json
{
  "family_id": 306,
  "child_id": 1,
  "assigned_date": "2026-03-10",
  "title": "Unit 1 Words",
  "language": "en",
  "items": [
    { "text": "apple", "meaning": "苹果", "hint": "fruit" }
  ]
}
```

**201 Created**

```json
{
  "message": "Word list saved successfully",
  "word_list": {
    "word_list_id": "wl_123",
    "family_id": 306,
    "child_id": 1,
    "assigned_date": "2026-03-10",
    "title": "Unit 1 Words",
    "language": "en",
    "items": [
      { "index": 1, "text": "apple", "meaning": "苹果", "hint": "fruit" }
    ],
    "total_items": 1,
    "created_at": "2026-03-10T08:00:00Z",
    "updated_at": "2026-03-10T08:00:00Z"
  }
}
```

---

### GET `/word-lists`

**Query parameters**: `family_id`, `child_id`, `date` (YYYY-MM-DD).

**200 OK**: Same `word_list` wrapper as POST.

---

## Dictation Sessions

### POST `/dictation-sessions/start`

Start a session.

**Request body**: `family_id`, `child_id`, `assigned_date`.

**201 Created**

```json
{
  "message": "Dictation session started successfully",
  "dictation_session": {
    "session_id": "ds_123",
    "word_list_id": "wl_123",
    "family_id": 306,
    "child_id": 1,
    "assigned_date": "2026-03-10",
    "status": "active",
    "current_index": 0,
    "total_items": 10,
    "played_count": 1,
    "completed_items": 0,
    "current_item": { "index": 1, "text": "apple", "meaning": "苹果" },
    "started_at": "...",
    "updated_at": "..."
  }
}
```

---

### GET `/dictation-sessions/:session_id`

**200 OK**

```json
{
  "dictation_session": { "...": "same as start response" }
}
```

---

### POST `/dictation-sessions/:session_id/replay`
### POST `/dictation-sessions/:session_id/next`

**200 OK**

```json
{
  "message": "...",
  "dictation_session": { "...": "updated session" }
}
```

---

## Field Definitions

### PointsOrigin
- `system`: Auto-generated from task completion.
- `parent`: Manually created by parent.

### PointsSourceType
- `task_completion`: (System) `delta` is always +1.
- `parent_reward`: (Parent) `delta` > 0.
- `parent_penalty`: (Parent) `delta` < 0.
- `school_praise`: (Parent) `delta` > 0.
- `school_criticism`: (Parent) `delta` < 0.

# Taskboard API Smoke

This file contains copy-pasteable `curl` examples for SC-05 smoke runs.

Scope:

- create one task
- parse tasks without auto-write
- confirm parsed tasks into storage
- list tasks for a fixed day
- update task status by item, group, and all

Related reference:

- Error contract: [API_ERROR_CONTRACT.md](/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/internal/interfaces/http/API_ERROR_CONTRACT.md)

## Conventions

- Use a fixed `assigned_date` so repeated smoke runs do not depend on server
  local time.
- Use a test family and user id dedicated to smoke.
- Check HTTP status and key response fields only. Do not hard-code the full
  response body.

Suggested shell variables:

```bash
API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8080}"
FAMILY_ID="${FAMILY_ID:-9306}"
USER_ID="${USER_ID:-1}"
ASSIGNED_DATE="${ASSIGNED_DATE:-2026-03-12}"
```

## 1. Create single task

Request:

```bash
curl -sS -X POST "${API_BASE_URL}/api/v1/tasks" \
  -H 'Content-Type: application/json' \
  -d "{
    \"family_id\": ${FAMILY_ID},
    \"assignee_id\": ${USER_ID},
    \"subject\": \"数学\",
    \"group_title\": \"校本P14-15\",
    \"content\": \"校本P14-15\",
    \"assigned_date\": \"${ASSIGNED_DATE}\"
  }"
```

Expected:

- HTTP `201`
- body contains `message`, `date`, `task`
- `date == ${ASSIGNED_DATE}`

## 2. Parse tasks without auto write

Request:

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

Expected:

- HTTP `201`
- body contains `message`, `parsed_count`, `parser_mode`, `analysis`, `auto_created`, `date`, `tasks`
- `auto_created == false`
- `parsed_count >= 1`
- response contains parsed tasks, but data is not yet persisted by this call

Optional follow-up check for "not auto written":

```bash
curl -sS "${API_BASE_URL}/api/v1/tasks?family_id=${FAMILY_ID}&user_id=${USER_ID}&date=${ASSIGNED_DATE}"
```

Expected for a fresh smoke date before confirm:

- HTTP `200`
- body contains `date`, `tasks`, `groups`, `homework_groups`, `summary`
- if only step 2 has run for that date, parsed tasks from `/tasks/parse` should
  not appear yet

## 3. Confirm parsed tasks into storage

Use a fixed payload so smoke does not depend on shell JSON extraction tools:

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
        \"group_title\": \"背默M1U1知识梳理单小作文\",
        \"title\": \"背默M1U1知识梳理单小作文\"
      },
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

Expected:

- HTTP `201`
- body contains `message`, `created_count`, `date`, `tasks`
- `created_count == 3`

## 4. Query one day of tasks

Request:

```bash
curl -sS "${API_BASE_URL}/api/v1/tasks?family_id=${FAMILY_ID}&user_id=${USER_ID}&date=${ASSIGNED_DATE}"
```

Expected:

- HTTP `200`
- body contains `date`, `tasks`, `groups`, `homework_groups`, `summary`
- `summary.total >= 1`
- successful response field names remain stable for Web and Pad:
  - `tasks`
  - `groups`
  - `homework_groups`
  - `summary`

## 5. Update one task by item

Request:

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

Expected:

- HTTP `200`
- body contains `message`, `updated_count`, `date`, `tasks`, `groups`,
  `homework_groups`, `summary`
- `updated_count >= 1`

If repeated with the same payload, expect:

- HTTP `409`
- `error_code == "status_unchanged"`

## 6. Update one homework group

Request:

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

Expected:

- HTTP `200`
- body contains `message`, `updated_count`, `subject`, `group_title`, `date`,
  `tasks`, `groups`, `homework_groups`, `summary`

If repeated with the same payload, expect:

- HTTP `409`
- `error_code == "status_unchanged"`

## 7. Update all tasks

Request:

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

Expected:

- HTTP `200`
- body contains `message`, `updated_count`, `date`, `tasks`, `groups`,
  `homework_groups`, `summary`

If repeated with the same payload after all tasks are already completed, expect:

- HTTP `409`
- `error_code == "status_unchanged"`

## Optional weekly stats check

Request:

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/weekly?family_id=${FAMILY_ID}&user_id=${USER_ID}&end_date=${ASSIGNED_DATE}"
```

Expected:

- HTTP `200`
- body contains either `message` and `data`, or `message` and `raw_stats`
- `end_date` must be `YYYY-MM-DD`

Invalid example:

```bash
curl -sS "${API_BASE_URL}/api/v1/stats/weekly?family_id=${FAMILY_ID}&user_id=${USER_ID}&end_date=2026-02-30"
```

Expected:

- HTTP `400`
- `error_code == "invalid_date"`

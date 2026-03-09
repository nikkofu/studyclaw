# API Error Contract

This document is the single reference for API error responses exposed by
`/api/v1/*` handlers in `apps/api-server/internal/interfaces/http`.

It is intended for:

- Web integration
- Pad integration
- Cross-service integration and smoke checks

Successful response field names remain endpoint-specific and unchanged. Clients
should branch on `error_code` for failures instead of parsing free-form text in
`error`.

Related reference:

- Smoke flow examples: [TASKBOARD_API_SMOKE.md](/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/internal/interfaces/http/TASKBOARD_API_SMOKE.md)

## Common shape

All API errors return this envelope:

```json
{
  "error": "Human readable message",
  "error_code": "machine_readable_code",
  "details": {}
}
```

Rules:

- `error` is stable human-readable text for logs and fallback UI.
- `error_code` is the primary machine-readable discriminator.
- `details` is optional and only appears when extra context is useful.
- HTTP status still matters and must be checked together with `error_code`.

## Details conventions

Known `details` payload shapes:

- Missing fields or query params:

```json
{
  "fields": ["family_id", "assignee_id"]
}
```

- Invalid single field:

```json
{
  "field": "assigned_date"
}
```

- Missing task by item id:

```json
{
  "task_id": 99
}
```

- Missing or duplicate group update:

```json
{
  "subject": "英语",
  "group_title": "预习M1U2",
  "status": "completed"
}
```

- Duplicate item or all-task update:

```json
{
  "task_id": 1,
  "status": "completed"
}
```

- Parse failure with agent payload:

```json
{
  "agent_response": {
    "status": "failed"
  }
}
```

## Endpoint coverage

The shared contract is used by these routes for error responses:

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
- `POST /api/v1/points/update`
- `GET /api/v1/stats/weekly`

## Error code matrix

Validation and request-shape errors:

- `missing_required_fields`
  - HTTP `400`
  - Used when required JSON fields or query params are absent
- `invalid_request_fields`
  - HTTP `400`
  - Used when bound fields are missing or invalid in the same payload
- `invalid_request`
  - HTTP `400`
  - Used for semantic request problems such as empty `tasks`
- `invalid_json`
  - HTTP `400`
  - Used when the request body is not valid JSON
- `invalid_query_parameter`
  - HTTP `400`
  - Used when a query parameter fails numeric parsing
- `invalid_date`
  - HTTP `400`
  - Used when `date`, `assigned_date`, or `end_date` is not `YYYY-MM-DD`

Resource and state errors:

- `task_not_found`
  - HTTP `404`
  - Used for missing task item updates or empty board bulk updates
- `task_group_not_found`
  - HTTP `404`
  - Used when no tasks match the requested subject or subject/group pair
- `status_unchanged`
  - HTTP `409`
  - Used when the requested target status already matches current state

Workflow and upstream errors:

- `tasks_not_extractable`
  - HTTP `422`
  - Used when parse completed but no usable tasks were extracted
- `parser_unavailable`
  - HTTP `502`
  - Used when task parsing fails upstream
- `internal_error`
  - HTTP `500` or `502` depending on handler
  - Used for persistence failures or weekly insight generation failures

## Response examples

Missing required fields:

```json
{
  "error": "Required fields are missing",
  "error_code": "missing_required_fields",
  "details": {
    "fields": ["family_id", "assignee_id"]
  }
}
```

Malformed JSON:

```json
{
  "error": "Request body must be valid JSON",
  "error_code": "invalid_json"
}
```

Invalid date:

```json
{
  "error": "assigned_date must be in YYYY-MM-DD format",
  "error_code": "invalid_date",
  "details": {
    "field": "assigned_date"
  }
}
```

Invalid query parameter:

```json
{
  "error": "family_id must be a valid unsigned integer",
  "error_code": "invalid_query_parameter",
  "details": {
    "field": "family_id"
  }
}
```

Missing task:

```json
{
  "error": "Task not found",
  "error_code": "task_not_found",
  "details": {
    "task_id": 99
  }
}
```

Duplicate status update:

```json
{
  "error": "All tasks are already completed",
  "error_code": "status_unchanged",
  "details": {
    "status": "completed"
  }
}
```

Parse returned no usable tasks:

```json
{
  "error": "Agent workflow could not extract valid tasks",
  "error_code": "tasks_not_extractable",
  "details": {
    "agent_response": {
      "status": "failed"
    }
  }
}
```

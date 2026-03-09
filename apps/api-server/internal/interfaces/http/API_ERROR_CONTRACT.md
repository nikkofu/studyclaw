# API Error Contract

This API keeps successful response field names stable. Error responses use one
minimal shared contract so clients can branch on `error_code` without parsing
free-form text.

## Shape

All error responses return:

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
- `details` is optional and only included when extra context is useful.

## Examples

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

## Current Error Codes

- `internal_error`
- `invalid_date`
- `invalid_json`
- `invalid_query_parameter`
- `invalid_request`
- `invalid_request_fields`
- `missing_required_fields`
- `parser_unavailable`
- `status_unchanged`
- `task_group_not_found`
- `task_not_found`
- `tasks_not_extractable`

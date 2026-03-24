# Core Hot Task Deepening Design (Child Execution Experience)

## Context
This design focuses on extending the current core hot-task flow around child-side execution in StudyClaw, while preserving existing backend/API contracts and release safety. Current strengths already include:
- parent-side parse/review/confirm pipeline
- taskboard execution foundation in pad app
- persistence event/snapshot metrics in API (transition-safe + cumulative)

The next step is to deepen execution quality in the child journey with a staged approach that improves start speed, in-session continuity, and post-completion motivation.

## Goals
1. Reduce friction in the first 30 seconds after entering taskboard.
2. Improve completion continuity when interruptions happen mid-session.
3. Turn completion into a motivating loop for children and an actionable recap for parents.

## Non-Goals
- Re-architecting parser or assignment publishing flows.
- Introducing LLM-dependent runtime logic for critical launch/recovery decisions.
- Large cross-app refactors unrelated to child execution hot path.

## Recommended Approach (Chosen)
Three-batch progressive delivery:
1. **Batch 1: First-30s Launch**
2. **Batch 2: Stay-in-Flow Interruption Recovery**
3. **Batch 3: Completion Reward + Parent Recap Loop**

This balances fast user-visible impact, low rollout risk, and measurable iteration points.

---

## Batch 1 — First-30s Launch

### User Outcome
A child should understand “what to do first” and start the first task step within 30 seconds.

### Product/UX Design
- Add a **Hero Task Card** for the single recommended next task/group.
- Add **One-Tap Start** action that jumps directly to first unfinished actionable item.
- Keep other tasks in a collapsed/secondary section to lower cognitive load.
- Show optional “estimated minutes for first block” when available.

### Data & API Contract Additions (Backward-Compatible)
On existing taskboard/day bundle payload, add optional fields:
- `launch_recommendation` (group/item identifiers + minimal metadata)
- `estimated_minutes_first_block`
- `why_recommended` (for explainability; can be parent-visible only)

Recommendation logic remains deterministic:
- unfinished first
- priority ordering
- interruption-risk-aware tie-breakers when available

### Error Handling / Fallback
- If recommendation fields are missing: fallback to first unfinished item.
- If estimate is unavailable: hide estimate UI only.

### Success Metrics
- `first_action_latency_p50`, `first_action_latency_p90`
- `launch_to_first_completion_rate`
- `first_30s_bounce_rate`

---

## Batch 2 — Stay-in-Flow (Interruption Recovery)

### User Outcome
After interruption/distraction, child resumes in-context in <=10 seconds without re-navigation burden.

### Product/UX Design
- Introduce **micro-goal progress** for current task block (2–4 visible steps).
- Show **resume strip** when session returns from pause/background: “continue step N”.
- Use lightweight inactivity nudges (non-blocking) for prolonged idle windows.
- Reinforce continuity via subtle streak pulse feedback after each micro-step.

### Data & API Contract Additions (Backward-Compatible)
Extend persistence event payload (already present) with optional execution context:
- `step_index`
- `step_total`
- `resume_hint`

Optionally expose summary-level recovery KPIs:
- `resume_success_rate`
- `interruptions_per_session`

### Error Handling / Fallback
- No snapshot/step context: resume from current unfinished item start.
- No micro-step schema: render basic task progress only.

### Success Metrics
- `session_resume_rate`
- `avg_interruptions_per_session`
- `in_session_completion_rate`

---

## Batch 3 — Completion Reward + Parent Recap

### User Outcome
Completion feels rewarding for children and visible/meaningful for parents, increasing repeat engagement.

### Product/UX Design
Child side:
- lightweight completion animation + explicit completion statement
- dual-streak model display:
  - display streak (encouragement, may include makeup)
  - core KPI streak (strict metric)
- badge milestones (e.g., 3/7/14)

Parent side:
- hot-task recap card in feedback/points area including:
  - daily completion
  - effective duration
  - interruption recovery signal
  - one suggested next action

### Data & API Contract Additions (Backward-Compatible)
Leverage existing persistence summary fields:
- `streak`
- `completion_rate`
- `effective_duration`
- `guardrails`

Optional achievement fields:
- `badge_unlocked`
- `badge_level`
- `next_badge_progress`

### Error Handling / Fallback
- Badge unavailable: show completion recap without badge layer.
- Streak unavailable: show daily completion only.

### Success Metrics
- `d1_return_after_completion`
- `weekly_active_learners`
- `parent_review_open_rate`

---

## Architecture & Boundaries
- API-server remains source of truth for recommendation metadata, persistence, and summary metrics.
- Pad app consumes optional fields with safe fallback behavior.
- Parent web displays recap/insight but does not redefine backend metric semantics.
- All additions are backward-compatible, additive fields first.

## Rollout Strategy
- Roll out by batch with feature toggles at API payload field level if needed.
- Ship observability and event logging before broad exposure.
- Validate each batch against its own KPI before starting next batch.

## Test Strategy
For each batch:
- API contract tests for new optional fields (presence/absence behavior).
- Pad UI/controller tests for fallback and recovery paths.
- Parent-web tests for recap rendering with partial/full payloads.
- Regression checks to ensure existing parse/review/confirm and taskboard flows are unaffected.

## Risks & Mitigations
1. **Over-complexity in child UI**
   - Keep first version minimal; no dense dashboards in child view.
2. **Metric drift across apps**
   - Use API as sole semantics owner; avoid client-side recomputation of core metrics.
3. **Recovery false positives/annoyance**
   - Conservative inactivity thresholds + non-blocking hints.
4. **Reward inflation without learning gain**
   - Track both engagement and completion-quality metrics, not only click/return metrics.

## Open Questions (for planning stage)
1. Exact first-block recommendation priority weights.
2. Idle threshold defaults by age/task type.
3. Badge milestone ladder and reset policy.
4. Parent recap placement priority within current information architecture.

---

## Implementation Readiness Addendum

### A. Batch Acceptance Criteria (Go/No-Go)

#### Batch 1 (First-30s Launch)
- Cohort: canary families using current demo baseline (`family_id=306`, child flow) plus opt-in pilot cohort.
- Window: rolling 7 days after release.
- Pass criteria:
  - `first_action_latency_p50 <= 12s`
  - `first_action_latency_p90 <= 30s`
  - `first_30s_bounce_rate` does not regress by more than +2% from pre-release baseline.
- Rollback trigger:
  - `first_30s_bounce_rate` regresses > +5% for 24h.

#### Batch 2 (Stay-in-Flow)
- Window: rolling 7 days after Batch 2 enablement.
- Pass criteria:
  - `resume_to_action_latency_median <= 10s`
  - `resume_success_rate >= 0.80`
  - `in_session_completion_rate` no worse than baseline and trending upward.
- Rollback trigger:
  - `avg_interruptions_per_session` worsens > +20% for 24h.

#### Batch 3 (Completion Reward + Parent Recap)
- Window: rolling 14 days after Batch 3 enablement.
- Pass criteria:
  - `d1_return_after_completion` improves >= +8% vs baseline.
  - `parent_review_open_rate` improves >= +10% vs baseline.
  - No regression in `completion_rate`.
- Rollback trigger:
  - completion metrics regress > 5% for 48h.

### B. API Contract Definitions (Additive, Backward-Compatible)

#### Taskboard Launch Fields
| Field | Type | Required | Default/Fallback | Notes |
|---|---|---:|---|---|
| `launch_recommendation` | object\|null | no | null | Deterministic recommendation payload |
| `launch_recommendation.group_id` | string | yes (if object exists) | N/A | Stable identifier for task group |
| `launch_recommendation.item_id` | int\|null | no | null | First actionable item ID; nullable for group-level only |
| `launch_recommendation.reason_code` | string enum | yes (if object exists) | `first_unfinished` | enum: `first_unfinished`, `priority_high`, `resume_continuity` |
| `estimated_minutes_first_block` | int\|null | no | null | If null, UI hides estimate |
| `why_recommended` | string\|null | no | null | Parent-visible text; pad side may ignore |

#### Persistence Execution Context Fields
| Field | Type | Required | Default/Fallback | Notes |
|---|---|---:|---|---|
| `step_index` | int\|null | no | null | 0-based current micro-step; UI displays `step_index + 1` |
| `step_total` | int\|null | no | null | Total visible micro-steps |
| `resume_hint` | string\|null | no | null | Optional hint copy for resume strip |

Validation rule:
- when both are present: `step_total > 0` and `0 <= step_index < step_total`
- invalid pair falls back to no-step-context rendering.

### C. Deterministic Recommendation Rule (v1)
Rank unfinished candidates by this stable comparator sequence:
1. `is_current_session_item` (true first)
2. `priority_weight` (descending; larger weight first)
3. `interruption_risk_score` (ascending; lower first)
4. `assigned_sequence` (ascending; lower first)
5. `item_id` (ascending; lower first as final stable tie-break)

If no unfinished item exists, return `launch_recommendation = null`.

Identifier contract for launch recommendation:
- `group_id`: canonical group key from taskboard group identity (`<subject>\x00<group_title>` in v1 response serialization).
- `item_id`: existing task item `task_id` integer, unique within one daily assignment scope (`family_id + child_id + assigned_date`).

### D. Event Semantics and Metric Definitions

#### Event Semantics
- `interrupted`: app leaves foreground for > 8s while an active step is in progress.
- `resumed`: app returns foreground and shows resume context.
- `resume_success`: child performs actionable interaction within 10s after `resumed`.
- Dedupe rule: foreground/background toggles within 3s collapse into one interruption event.

#### Metric Dictionary
- `first_action_latency`: `first_action_ts - taskboard_enter_ts`
- `first_30s_bounce_rate`: sessions with no action within 30s / taskboard entries
- `resume_to_action_latency_median`: median(`first_action_after_resume_ts - resumed_ts`)
- `resume_success_rate`: resume_success_count / resumed_count
- `in_session_completion_rate`: sessions with at least one completed actionable step / started_sessions

### E. Feature Flag and Compatibility Matrix
- `hot_task_launch_v1` (Batch 1 fields + UI)
- `hot_task_resume_v1` (Batch 2 resume strip + step context)
- `hot_task_rewards_v1` (Batch 3 reward + parent recap)

Compatibility:
- New server + old client: fields ignored safely.
- Old server + new client: client fallback path (no recommendation/resume/reward metadata).
- Kill-switch: disable per-flag at server response layer; client reverts to existing baseline flow.

### F. Parent Recap Placement Decision (v1 fixed)
- Primary placement: parent-web feedback screen top card.
- Secondary fallback: points screen beneath balance summary.
- No IA A/B in v1; experiments deferred post-stabilization.

### G. Badge/Streak Governance (v1)
- Badge milestones: 3, 7, 14 display streak days.
- Unlock cap: max 1 new badge unlock per day.
- Retroactive unlock: not applied in v1.
- Child-facing labels:
  - `连续进行天数` for display streak.
- Parent-facing labels:
  - `显示连胜` and `核心KPI连胜` with helper text clarifying makeup-day difference.

### H. Minimal Test Matrix

#### Batch 1
- API contract test: recommendation fields absent/present both valid.
- Pad test: one-tap start fallback to first unfinished when recommendation missing.

#### Batch 2
- API test: interruption/resume context fields accepted and emitted.
- Pad controller test: resume strip appears after interruption and jumps to prior step.
- Dedupe test: rapid app state toggles do not inflate interruption counts.

#### Batch 3
- API test: summary + badge fields additive and backward-compatible.
- Parent web test: recap card renders with partial and full payload.
- Pad test: reward feedback appears without blocking continuation flow.

### I. Non-Functional Constraints
- Added taskboard payload budget: <= +1.5KB per response in v1.
- Added first-screen render overhead on pad: <= 50ms median over baseline.
- Event versioning: include `event_schema_version=1` for all new analytics events.

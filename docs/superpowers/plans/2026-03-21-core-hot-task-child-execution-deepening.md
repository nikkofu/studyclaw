# Core Hot Task Child Execution Deepening Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver three gated batches that improve child hot-task execution (fast start, interruption recovery, completion motivation) while preserving backward compatibility.

**Architecture:** Implement additive contracts in API first, guarded by batch feature flags from day one. Pad-app and parent-web consume optional fields with strict fallback behavior. Every batch is shipped via TDD with contract tests, client behavior tests, and metric/instrumentation checks before promotion.

**Tech Stack:** Go (API + route tests), Flutter/Dart (pad controller/widget tests), React/Vite + Vitest (parent-web), JSON-store persistence.

---

## File Structure & Responsibility Map

### API (`apps/api-server`)
- `internal/modules/taskboard/domain/phase_one.go`
  - Add/maintain additive payload types and enums (`launch_recommendation`, recovery context, reward fields).
- `internal/modules/taskboard/application/phase_one.go`
  - Build launch recommendation and recovery/reward metadata behind flags.
  - Keep deterministic ranking and fallback semantics.
- `internal/modules/taskboard/infrastructure/jsonstore/repository.go`
  - Persist/read fields required for deterministic recommendation and recovery aggregation.
- `routes/api_success_contract_test.go` + related route tests
  - Enforce additive contract semantics and compatibility.

### Pad app (`apps/pad-app`)
- `lib/task_board/repository.dart`
  - Parse additive fields safely.
- `lib/task_board/controller.dart`
  - Launch selection, interruption/resume state machine, reward state.
- `lib/task_board/page.dart`
  - Hero card, one-tap start, resume strip, micro-goal UI, completion feedback.
- `test/task_board/*`
  - Controller + widget tests for both enriched payload and fallback payload.

### Parent web (`apps/parent-web`)
- `src/App.jsx`
  - Recap card rendering with fixed placement policy (feedback-top primary, points fallback).
- `src/App.test.jsx`
  - Placement/fallback/additive rendering tests.

### Docs
- Plan: `docs/superpowers/plans/2026-03-21-core-hot-task-child-execution-deepening.md`
- Spec: `docs/superpowers/specs/2026-03-21-core-hot-task-child-execution-design.md`

---

## Task 1: Baseline feature flags and flag-off compatibility (must be first)

**Files:**
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: `apps/pad-app/lib/task_board/controller.dart`
- Modify: `apps/parent-web/src/App.jsx`
- Test: API routes + pad tests + parent tests

- [ ] **Step 1: Write failing tests that assert flag-off == current behavior**

```go
func TestHotTaskFlagsOff_PayloadUnchanged(t *testing.T) {}
```

```dart
test('without new fields, pad flow stays baseline', () async {})
```

```jsx
it('does not render recap card when reward flag off', async () => {})
```

- [ ] **Step 2: Run targeted tests to verify failures**

Run:
- `cd apps/api-server && go test ./routes -run TestHotTaskFlagsOff_PayloadUnchanged -count=1`
- `cd apps/pad-app && flutter test --no-pub test/task_board/<flag_test>.dart`
- `cd apps/parent-web && npx vitest run src/App.test.jsx --environment jsdom -t "flag off"`

- [ ] **Step 3: Implement minimal flag plumbing (`hot_task_launch_v1`, `hot_task_resume_v1`, `hot_task_rewards_v1`)**

- [ ] **Step 4: Re-run tests to verify pass**

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/pad-app/lib/task_board/controller.dart apps/parent-web/src/App.jsx
git commit -m "chore: add hot-task feature flag gates with safe defaults"
```

---

## Task 2: Batch 1 API launch contract details (exact spec semantics)

**Files:**
- Modify: `apps/api-server/routes/api_success_contract_test.go`
- Modify: `apps/api-server/internal/modules/taskboard/domain/phase_one.go`
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`

- [ ] **Step 1: Write failing contract tests for launch fields**

```go
func TestDailyAssignment_LaunchRecommendationContract(t *testing.T) {
  // reason_code enum/default: first_unfinished
  // group_id canonical <subject>\x00<group_title>
  // item_id int|null semantics
  // why_recommended optional
}
```

- [ ] **Step 2: Run test and verify fail**

Run: `cd apps/api-server && go test ./routes -run TestDailyAssignment_LaunchRecommendationContract -count=1`

- [ ] **Step 3: Implement minimal additive contract shaping behind `hot_task_launch_v1`**

- [ ] **Step 4: Re-run test and verify pass**

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/routes/api_success_contract_test.go apps/api-server/internal/modules/taskboard/domain/phase_one.go apps/api-server/internal/modules/taskboard/application/phase_one.go
git commit -m "feat: add launch recommendation contract fields"
```

---

## Task 3: Batch 1 deterministic ranking and one-tap launch behavior

**Files:**
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: `apps/pad-app/lib/task_board/repository.dart`
- Modify: `apps/pad-app/lib/task_board/controller.dart`
- Modify: `apps/pad-app/lib/task_board/page.dart`
- Test: `apps/api-server/internal/modules/taskboard/application/*test.go`, `apps/pad-app/test/task_board/*`

- [ ] **Step 1: Write failing API unit test for comparator ordering**

```go
func TestBuildLaunchRecommendation_UsesDeterministicComparator(t *testing.T) {}
```

- [ ] **Step 2: Write failing pad controller/widget tests for one-tap start + fallback**

```dart
test('uses recommendation else first unfinished', () async {})
```

```dart
testWidgets('one-tap start enters recommended actionable item', (tester) async {})
```

- [ ] **Step 3: Run tests and verify failures**

- [ ] **Step 4: Implement minimal ranking + pad consumption/UI**

- [ ] **Step 5: Re-run tests to green and commit**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/pad-app/lib/task_board/repository.dart apps/pad-app/lib/task_board/controller.dart apps/pad-app/lib/task_board/page.dart apps/pad-app/test/task_board/
git commit -m "feat: implement deterministic first-30s launch flow"
```

---

## Task 4: Batch 2 API event semantics + threshold/dedupe tests

**Files:**
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: `apps/api-server/internal/modules/taskboard/domain/phase_one.go`
- Modify: `apps/api-server/routes/*test.go` (or app package tests for event rules)

- [ ] **Step 1: Write failing tests for spec event semantics**

```go
func TestRecoverySemantics_InterruptedAfter8sBackground(t *testing.T) {}
func TestRecoverySemantics_DedupeWithin3sToggle(t *testing.T) {}
func TestRecoverySemantics_ResumeSuccessWithin10s(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify fail**

- [ ] **Step 3: Implement minimal semantics behind `hot_task_resume_v1`**

- [ ] **Step 4: Re-run tests to pass**

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/api-server/internal/modules/taskboard/domain/phase_one.go apps/api-server/routes/
git commit -m "feat: implement interruption and resume event semantics"
```

---

## Task 5: Batch 2 pad stay-in-flow UX (resume strip + micro-step)

**Files:**
- Modify: `apps/pad-app/lib/task_board/controller.dart`
- Modify: `apps/pad-app/lib/task_board/page.dart`
- Test: `apps/pad-app/test/task_board/*`

- [ ] **Step 1: Write failing tests for resume strip and step display conversion**

```dart
test('resume strip uses step_index+1 display and valid bounds', () async {})
```

```dart
testWidgets('invalid step context falls back to baseline rendering', (tester) async {})
```

- [ ] **Step 2: Run tests to fail**

- [ ] **Step 3: Implement minimal resume strip + micro-goal rendering + fallback**

- [ ] **Step 4: Re-run tests to pass**

- [ ] **Step 5: Commit**

```bash
git add apps/pad-app/lib/task_board/controller.dart apps/pad-app/lib/task_board/page.dart apps/pad-app/test/task_board/
git commit -m "feat: add stay-in-flow resume strip and micro-goal UI"
```

---

## Task 6: Metric dictionary implementation checks (API + client instrumentation)

**Files:**
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: metric/event emitting code in pad and parent where applicable
- Test: API package tests + client unit tests

- [ ] **Step 1: Write failing deterministic metric fixture tests**

```go
func TestMetrics_FirstActionLatency_Computation(t *testing.T) {}
func TestMetrics_First30sBounceRate_Computation(t *testing.T) {}
func TestMetrics_ResumeToActionLatencyMedian_Computation(t *testing.T) {}
func TestMetrics_ResumeSuccessRate_Computation(t *testing.T) {}
func TestMetrics_InSessionCompletionRate_Computation(t *testing.T) {}
```

```dart
test('emits event_schema_version=1 on new events', () async {})
```

- [ ] **Step 2: Run failing tests**

Run:
- `cd apps/api-server && go test ./internal/modules/taskboard/application -run TestMetrics_ -count=1`
- `cd apps/pad-app && flutter test --no-pub test/task_board/<metrics_event_test>.dart`

Expected: FAIL before implementation.

- [ ] **Step 3: Implement minimal formula-conformant metric/instrumentation logic**

- [ ] **Step 4: Re-run tests to pass**

Expected: PASS for all 5 metric formula tests + event version test.

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/api-server/internal/modules/taskboard/application/*test.go apps/pad-app/test/task_board/
git commit -m "feat: align metrics and events with hot-task dictionary"
```

---

## Task 7: Batch 3 reward fields + parent recap placement contract

**Files:**
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: `apps/api-server/routes/api_success_contract_test.go`
- Modify: `apps/parent-web/src/App.jsx`
- Modify: `apps/parent-web/src/App.test.jsx`

- [ ] **Step 1: Write failing tests for reward fields (optional/additive)**

```go
func TestRewardFields_AdditiveAndBackwardCompatible(t *testing.T) {}
```

- [ ] **Step 2: Write failing parent tests for fixed placement policy**

```jsx
it('renders recap card at feedback top as primary placement', async () => {})
it('renders points fallback when feedback placement unavailable', async () => {})
it('does not run IA A/B behavior in v1', async () => {})
```

- [ ] **Step 3: Run tests to fail**

- [ ] **Step 4: Implement minimal reward payload + parent rendering policy**

- [ ] **Step 5: Re-run tests and commit**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/api-server/routes/api_success_contract_test.go apps/parent-web/src/App.jsx apps/parent-web/src/App.test.jsx
git commit -m "feat: add completion reward fields and fixed parent recap placement"
```

---

## Task 8: Non-functional guard checks (payload, render overhead)

**Files:**
- Modify/Create: `apps/api-server/internal/modules/taskboard/application/phase_one_nonfunctional_test.go`
- Modify/Create: `apps/pad-app/test/task_board/first_screen_perf_test.dart`

- [ ] **Step 1: Write failing API payload-budget test (`<= +1.5KB`)**

```go
func TestTaskboardPayloadGrowthBudget_LaunchResumeReward(t *testing.T) {
  // baseline fixture JSON vs enriched fixture JSON
  // assert len(enriched)-len(baseline) <= 1536
}
```

- [ ] **Step 2: Run API non-functional test and verify fail**

Run: `cd apps/api-server && go test ./internal/modules/taskboard/application -run TestTaskboardPayloadGrowthBudget_LaunchResumeReward -count=1`
Expected: FAIL before tuning.

- [ ] **Step 3: Write failing pad first-screen overhead test (`<= 50ms median`)**

```dart
testWidgets('first screen render overhead stays within budget', (tester) async {
  // run baseline and enriched states over N iterations
  // assert median(enriched-baseline) <= 50ms
});
```

- [ ] **Step 4: Run pad perf test and verify fail**

Run: `cd apps/pad-app && flutter test --no-pub test/task_board/first_screen_perf_test.dart`
Expected: FAIL before tuning.

- [ ] **Step 5: Implement minimal tuning and rerun both tests to pass**

Expected: PASS for payload and render-overhead budgets.

- [ ] **Step 6: Commit guardrail checks and tuning changes**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one_nonfunctional_test.go apps/pad-app/test/task_board/first_screen_perf_test.dart apps/pad-app/lib/task_board/
git commit -m "test: enforce payload and render overhead budgets"
```

---

## Task 9: Full regression verification

**Files:** no planned feature edits; only minimal fixes if failures found.

- [ ] **Step 1: API full suite**

Run: `cd apps/api-server && go test ./... -count=1`

- [ ] **Step 2: Parent web full suite**

Run: `cd apps/parent-web && npm test -- --run`

- [ ] **Step 3: Pad app full suite**

Run: `cd apps/pad-app && flutter test --no-pub`

- [ ] **Step 4: Fix regressions minimally and rerun to all green**

- [ ] **Step 5: Commit regression fixes (if any)**

```bash
git add <specific files>
git commit -m "test: fix regressions after hot-task deepening"
```

---

## Task 10: Operational docs update and release handoff

**Files:**
- Modify: `docs/06_RUNBOOK.md`
- Modify: `docs/11_NEXT_PHASE_DISPATCH.md` (if this remains active dispatch doc)

- [ ] **Step 1: Update rollout runbook with flags, go/no-go thresholds, rollback triggers**
- [ ] **Step 2: Add exact command checklist used for verification and monitoring**
- [ ] **Step 3: Validate all file paths/commands exist and match implementation**
- [ ] **Step 4: Commit docs**

```bash
git add docs/06_RUNBOOK.md docs/11_NEXT_PHASE_DISPATCH.md
git commit -m "docs: add rollout and verification playbook for hot-task deepening"
```

---

## Final Verification Gate (required before PR)

- [ ] `cd apps/api-server && go test ./... -count=1`
- [ ] `cd apps/parent-web && npm test -- --run`
- [ ] `cd apps/pad-app && flutter test --no-pub`
- [ ] Optional smoke: `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`

Expected: all required suites pass; additive contracts stable; flag-off baseline preserved.

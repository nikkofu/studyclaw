# Legacy Pending Tasks Cleanup (v0.4.1 Alignment) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the currently identified unfinished work by aligning docs/version gates, finishing hot-task launch flow, and adding minimum regression protection across SC-01~SC-05 lanes.

**Architecture:** Execute in dependency order: first unify release/document truth (SC-05), then close launch contract + ranking semantics (SC-01), then wire and verify pad behavior (SC-03), then finalize parent/readiness artifacts (SC-04 + SC-05). Keep all changes additive and backward-compatible with feature flags default-off.

**Tech Stack:** Go + Go test (API), Flutter/Dart + flutter test (Pad), React/Vitest (Parent Web), shell verification scripts.

---

## File Structure & Responsibility Map

- `docs/14_NEXT_PHASE_DISPATCH.md`
  - Canonical next-phase dispatch and target version status.
- `docs/19_DELIVERY_UAT_CASES.md`
  - UAT baseline/version wording and gate command references.
- `docs/20_RELEASE_SYNC_PLAYBOOK.md`
  - Release commit/tag examples and scope rules.
- `docs/17_DELIVERY_READINESS.md`
  - Single source of truth for blockers and non-blocking risks.
- `apps/api-server/internal/modules/taskboard/application/phase_one.go`
  - Launch recommendation ranking + payload shaping.
- `apps/api-server/internal/modules/taskboard/domain/task.go`
  - Task ranking fields used by deterministic comparator.
- `apps/api-server/routes/api_success_contract_test.go`
  - API contract test for launch recommendation and compatibility.
- `apps/pad-app/lib/task_board/models.dart`
  - Parse launch recommendation payload safely.
- `apps/pad-app/lib/task_board/controller.dart`
  - Resolve recommended task with fallback logic.
- `apps/pad-app/lib/task_board/page.dart`
  - One-tap “先做推荐” UI action.
- `apps/pad-app/test/task_board/launch_recommendation_test.dart` (create)
  - Controller/widget-level fallback and action tests.
- `apps/parent-web/src/App.test.jsx`
  - Keep parent surface stable while version/gate docs move forward.

---

### Task 1: SC-05 Canonical version and gate alignment (P0)

**Owner:** `SC-05-INTEGRATION`
**Depends on:** none

**Files:**
- Modify: `docs/19_DELIVERY_UAT_CASES.md`
- Modify: `docs/20_RELEASE_SYNC_PLAYBOOK.md`
- Modify: `docs/17_DELIVERY_READINESS.md`
- Cross-check only: `docs/03_ROADMAP.md`, `docs/14_NEXT_PHASE_DISPATCH.md`

- [ ] **Step 1: Add deterministic failing version-alignment assertions (test-first)**

Run:
- `bash -lc 'dispatch=$(grep -o "当前目标版本：`v[0-9]\\.[0-9]\\.[0-9]`" docs/14_NEXT_PHASE_DISPATCH.md | head -1 | sed "s/.*`\(v[0-9]\\.[0-9]\\.[0-9]\)`.*/\1/"); uat=$(grep -o "版本：`v[0-9]\\.[0-9]\\.[0-9]`" docs/19_DELIVERY_UAT_CASES.md | head -1 | sed "s/.*`\(v[0-9]\\.[0-9]\\.[0-9]\)`.*/\1/"); [ "$dispatch" = "$uat" ]'`

Expected: FAIL before edits (non-zero exit) because active-version strings are inconsistent.

- [ ] **Step 2: Capture baseline mismatch evidence before edits**

Run:
- `grep -n "当前正式基线\|当前目标版本" docs/14_NEXT_PHASE_DISPATCH.md docs/03_ROADMAP.md`
- `grep -n "验收基线\|版本：\|v0\.3\.5\|v0\.4\.0\|v0\.4\.1" docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md docs/17_DELIVERY_READINESS.md`

Expected: conflicting active-version references are visible in command output.

- [ ] **Step 3: Apply minimal doc edits to make one canonical storyline**

Rules:
- Keep `docs/03_ROADMAP.md` + `docs/14_NEXT_PHASE_DISPATCH.md` as source anchors.
- Update UAT/playbook wording/examples only as needed.
- Do not alter historical sections beyond conflict resolution.

- [ ] **Step 4: Re-run consistency checks**

Run:
- same grep command as Step 2
- `bash scripts/check_release_scope.sh`

Expected: no contradiction in active-version statements; release scope check passes.

- [ ] **Step 5: Commit**

```bash
git add docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md docs/17_DELIVERY_READINESS.md
git commit -m "docs: align UAT and release gates with current v0.4.x baseline"
```

**Completion criteria:**
- Active version narrative is consistent in roadmap/dispatch/uat/playbook/readiness.
- No unrelated docs edited.

---

### Task 2: SC-01 Launch recommendation contract hardening (P0)

**Owner:** `SC-01-GO-API`
**Depends on:** Task 1 (docs aligned)

**Files:**
- Modify: `apps/api-server/routes/api_success_contract_test.go`
- Modify: `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify: `apps/api-server/internal/modules/taskboard/domain/task.go`

- [ ] **Step 1: Add failing contract test for launch recommendation + additive fields semantics**

```go
func TestDayBundleLaunchRecommendation_UsesDeterministicWinnerAndItemID(t *testing.T) {
    // assert: launch_recommendation.item_id points to ranked unfinished task
    // assert: group_id format <subject>\x00<group_title>
    // assert: empty unfinished set => launch_recommendation omitted/null
    // assert: estimated_minutes_first_block is additive optional (absent/null allowed)
    // assert: why_recommended is additive optional (absent/null allowed)
}
```

- [ ] **Step 2: Run targeted API test and verify failure**

Run:
- `cd apps/api-server && go test ./routes -run TestDayBundleLaunchRecommendation_UsesDeterministicWinnerAndItemID -count=1`

Expected: FAIL before implementation alignment.

- [ ] **Step 3a: Implement minimal comparator + launch_recommendation payload guarantees**

Comparator order:
1. `IsCurrentSessionItem` (true first)
2. `PriorityWeight` (desc)
3. `InterruptionRiskScore` (asc)
4. `AssignedSequence` (asc)
5. `TaskID` (asc)

- [ ] **Step 3b: Implement additive optional field handling assertions (`estimated_minutes_first_block`, `why_recommended`)**

Rules:
- Keep both fields optional/backward-compatible.
- Do not fail responses when fields are absent.

- [ ] **Step 4: Re-run tests**

Run:
- `cd apps/api-server && go test ./routes -run TestDayBundleLaunchRecommendation_UsesDeterministicWinnerAndItemID -count=1`
- `cd apps/api-server && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/routes/api_success_contract_test.go apps/api-server/internal/modules/taskboard/application/phase_one.go apps/api-server/internal/modules/taskboard/domain/task.go
git commit -m "feat: finalize deterministic launch recommendation contract"
```

**Completion criteria:**
- Contract test captures item_id/group_id/winner behavior.
- Full API suite passes.

---

### Task 3: SC-03 Pad one-tap recommended start + fallback tests (P0)

**Owner:** `SC-03-FLUTTER-PAD`
**Depends on:** Task 2

**Files:**
- Modify: `apps/pad-app/lib/task_board/models.dart`
- Modify: `apps/pad-app/lib/task_board/controller.dart`
- Modify: `apps/pad-app/lib/task_board/page.dart`
- Create: `apps/pad-app/test/task_board/launch_recommendation_test.dart`

- [ ] **Step 1: Write failing pad tests for resolve + action behavior**

```dart
test('resolveLaunchTask uses launch_recommendation item first', () {});
test('resolveLaunchTask falls back to first unfinished when recommendation invalid', () {});
testWidgets('先做推荐 triggers single-task completion on resolved target', (tester) async {});
```

- [ ] **Step 2: Run targeted tests and verify failure**

Run:
- `cd apps/pad-app && flutter test --no-pub test/task_board/launch_recommendation_test.dart`

Expected: FAIL before final wiring.

- [ ] **Step 3: Implement minimal behavior**

Behavior:
- Parse optional `launch_recommendation` safely.
- Resolve recommended unfinished task by `item_id`.
- Fallback to first unfinished.
- “先做推荐” performs single-item completion only.

- [ ] **Step 4: Re-run tests + baseline checks**

Run:
- `cd apps/pad-app && flutter test --no-pub test/task_board/launch_recommendation_test.dart`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test --no-pub`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/pad-app/lib/task_board/models.dart apps/pad-app/lib/task_board/controller.dart apps/pad-app/lib/task_board/page.dart apps/pad-app/test/task_board/launch_recommendation_test.dart
git commit -m "feat: add recommended one-tap start with safe fallback"
```

**Completion criteria:**
- Recommended-start path is tested and deterministic.
- No regression in existing pad tests.

---

### Task 4: SC-04 Parent regression guard + wording sync (P1)

**Owner:** `SC-04-PARENT-WEB`
**Depends on:** Task 1

**Files:**
- Modify: `apps/parent-web/src/App.test.jsx`
- Optional minimal modify: `apps/parent-web/src/App.jsx` (only if wording/gate references are surfaced in UI copy)

- [ ] **Step 1: Add/adjust failing tests that assert unchanged key parent flows**

```jsx
it('keeps daily report and pending summary rendering stable after v0.4.x doc alignment', async () => {})
```

- [ ] **Step 2: Run targeted tests (single runner policy)**

Run:
- `cd apps/parent-web && npm test -- --run src/App.test.jsx`

Expected: FAIL only for expected updated assertions.

- [ ] **Step 3: Apply minimal code/test updates**

Rules:
- No feature expansion.
- Keep existing behavior stable.

- [ ] **Step 4: Re-run full parent checks**

Run:
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/parent-web/src/App.test.jsx
# only if modified:
# git add apps/parent-web/src/App.jsx
git commit -m "test: lock parent web baseline during v0.4.x alignment"
```

**Completion criteria:**
- Parent baseline remains green and unchanged in scope.

---

### Task 5: SC-05 final readiness pass + blocker ledger update (P0)

**Owner:** `SC-05-INTEGRATION`
**Depends on:** Tasks 1–4

**Files:**
- Modify: `docs/17_DELIVERY_READINESS.md`
- Modify: `docs/14_NEXT_PHASE_DISPATCH.md` (only blocker section if status changed)
- Optional modify: `README.md` (only if command entrypoint is inconsistent)

- [ ] **Step 1: Add deterministic failing readiness assertions (test-first)**

Run:
- `test -f docs/17_DELIVERY_READINESS.md`
- `bash -lc 'grep -q "launch recommendation API + pad tests green" docs/17_DELIVERY_READINESS.md'`

Expected: second command FAILS before update (non-zero exit) if ledger not yet updated.

- [ ] **Step 2: Start required local services (precondition for smoke/demo)**

Run:
- `cd apps/api-server && API_PORT=38080 go run ./cmd/studyclaw-server`
- `cd apps/parent-web && VITE_API_BASE_URL=http://127.0.0.1:38080 npm run dev -- --host 127.0.0.1 --port 5173`

Expected: API and Parent Web available at required ports.

- [ ] **Step 3: Run release/readiness commands and capture outputs**

Run:
- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

Expected: PASS or clearly documented blockers with owner and next action.

- [ ] **Step 4: Update blocker/remaining-item ledger with owner + ETA-free next action**

Template:
- Item
- Status (`done` / `in_progress` / `blocked`)
- Owner lane
- Verification evidence (command + date)
- Next action

- [ ] **Step 5: Re-verify docs cross-links and consistency**

Run:
- `grep -n "当前正式基线\|当前目标版本\|阻塞\|未完成" docs/14_NEXT_PHASE_DISPATCH.md docs/17_DELIVERY_READINESS.md`
- `grep -n "验收基线\|版本：\|v0\.4\.0\|v0\.4\.1" docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md`

Expected: consistent status language.

- [ ] **Step 6: Commit**

```bash
git add docs/17_DELIVERY_READINESS.md docs/14_NEXT_PHASE_DISPATCH.md
# only if modified:
# git add README.md
git commit -m "docs: update blocker ledger and final readiness evidence"
```

**Completion criteria:**
- Every previously identified unfinished item is now either done or explicitly blocked with owner and evidence.
- Readiness docs and scripts are aligned for handoff.

---

## Global Verification Checklist (after all tasks)

- [ ] `cd apps/api-server && go test ./... -count=1`
- [ ] `cd apps/parent-web && npm test -- --run && npm run build`
- [ ] `cd apps/pad-app && flutter analyze && flutter test --no-pub`
- [ ] `bash scripts/check_no_tracked_runtime_env.sh`
- [ ] `bash scripts/preflight_local_env.sh`
- [ ] `bash scripts/check_release_scope.sh`
- [ ] `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- [ ] `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

---

## Delivery Output Template (for each lane)

```markdown
### Lane: <SC-01..SC-05>
- Scope changed:
- Key decision:
- Verification commands + results:
- Risks / unfinished:
- Ready for GitHub sync: yes/no
```

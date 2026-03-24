# Remaining Work Closure (Code + Docs) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close all actual remaining code/doc work in priority order, then pass full verification gate before final commit/push.

**Architecture:** First freeze real pending scope from current git status, then run P0 serially (code deltas if any → docs consistency → full verification), then P1 ledger polish. No new features, no speculative refactor. Only commit after full gate is green.

**Tech Stack:** Git + Bash scripts, Go tests, Flutter tests/build, npm/vitest build pipeline, Markdown docs.

---

## File Structure & Responsibility Map

- `git status --short` snapshot (frozen at execution start)
  - Single source of truth for actual pending paths in this run.
- `apps/api-server/internal/modules/taskboard/application/phase_one.go` (conditional)
  - Included only if present in frozen pending scope.
- `apps/api-server/internal/modules/taskboard/domain/task.go` (conditional)
  - Included only if present in frozen pending scope.
- `apps/pad-app/lib/task_board/controller.dart` (conditional)
  - Included only if present in frozen pending scope.
- `apps/pad-app/lib/task_board/models.dart` (conditional)
  - Included only if present in frozen pending scope.
- `apps/pad-app/lib/task_board/page.dart` (conditional)
  - Included only if present in frozen pending scope.
- `apps/pad-app/test/task_board/launch_recommendation_test.dart` (conditional)
  - Coverage for recommendation resolver behavior when relevant files are touched.
- `apps/pad-app/test/widget_test.dart` (conditional)
  - Baseline UI regression check when taskboard UI paths are touched.
- `docs/14_NEXT_PHASE_DISPATCH.md` (conditional)
  - Canonical next-phase status anchor if wording mismatch exists.
- `docs/17_DELIVERY_READINESS.md` (conditional)
  - Blocker/remaining ledger and evidence source if updates are needed.
- `docs/19_DELIVERY_UAT_CASES.md` (conditional)
  - UAT baseline and gate command source if updates are needed.
- `docs/20_RELEASE_SYNC_PLAYBOOK.md` (conditional)
  - Release-sync policy and historical blocker context if updates are needed.
- `docs/superpowers/` (directory status decision)
  - Must be explicitly included or removed in this run; `.gitignore` change requires explicit user directive.


---

### Task 1: Freeze actual pending scope from current repo state (P0)

**Files:**
- Inspect only: repository root status + above touched files

- [ ] **Step 1: Capture failing baseline as pending-work snapshot**

Run:
- `git status --short`

Expected: shows pending/untracked items to be resolved in this plan.

- [ ] **Step 2: Record exact pending list as execution checklist**

Checklist format:
- item path
- class (`code` / `docs` / `untracked-dir`)
- target action (`stage+commit`, `remove`, `explicitly-allow-untracked-with-rationale`)

- [ ] **Step 3: Verify no extra hidden pending items outside checklist**

Run:
- `git status --short`

Expected: pending set equals checklist set exactly.

- [ ] **Step 4: Commit?**

No commit in this task.

---

### Task 2: Resolve code-layer unfinished items only if they truly exist (P0)

**Files:**
- Modify (if present in status): `apps/api-server/internal/modules/taskboard/application/phase_one.go`
- Modify (if present in status): `apps/api-server/internal/modules/taskboard/domain/task.go`
- Modify (if present in status): `apps/pad-app/lib/task_board/controller.dart`
- Modify (if present in status): `apps/pad-app/lib/task_board/models.dart`
- Modify (if present in status): `apps/pad-app/lib/task_board/page.dart`
- Test (if relevant): `apps/pad-app/test/task_board/launch_recommendation_test.dart`
- Test (if relevant): `apps/pad-app/test/widget_test.dart`

- [ ] **Step 1: Write/adjust failing tests that describe intended existing behavior**

Example (if needed):
```dart
test('resolveLaunchTask follows recommendation then fallback', () {});
testWidgets('recommended-start entry visibility matches intended baseline', (tester) async {});
```

- [ ] **Step 2: Run targeted tests and confirm fail before fix**

Run:
- `cd apps/pad-app && flutter test --no-pub test/task_board/launch_recommendation_test.dart`
- `cd apps/pad-app && flutter test --no-pub test/widget_test.dart -r compact`

Expected: FAIL only if code mismatch truly exists.

- [ ] **Step 3: Apply minimal fix to pending code files**

Rules:
- No new feature.
- No unrelated cleanup.
- Keep behavior aligned with already merged mainline intent.

- [ ] **Step 4: Re-run targeted tests to green**

Run same commands as Step 2.
Expected: PASS.

- [ ] **Step 5: Stage changes only (no commit yet)**

```bash
git add apps/api-server/internal/modules/taskboard/application/phase_one.go apps/api-server/internal/modules/taskboard/domain/task.go apps/pad-app/lib/task_board/controller.dart apps/pad-app/lib/task_board/models.dart apps/pad-app/lib/task_board/page.dart apps/pad-app/test/task_board/launch_recommendation_test.dart apps/pad-app/test/widget_test.dart
```

Rule:
- Staging is allowed.
- Committing is forbidden until Task 5 full gate is green.

---

### Task 3: Resolve docs-layer unfinished consistency items (P0)

**Files:**
- Modify (if needed): `docs/17_DELIVERY_READINESS.md`
- Modify (if needed): `docs/14_NEXT_PHASE_DISPATCH.md`
- Modify (if needed): `docs/19_DELIVERY_UAT_CASES.md`
- Modify (if needed): `docs/20_RELEASE_SYNC_PLAYBOOK.md`

- [ ] **Step 1: Run consistency scan**

Run:
- `grep -n "当前正式基线\|当前目标版本\|v0\.4\.0\|v0\.4\.1" docs/14_NEXT_PHASE_DISPATCH.md docs/17_DELIVERY_READINESS.md docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md`

Expected: any contradiction found must be fixed in this task.

- [ ] **Step 2: Apply minimal wording corrections**

Rules:
- Keep `ROADMAP + DISPATCH` as current-state anchors.
- Keep historical content explicitly labeled as historical snapshot.
- Do not rewrite unrelated manual sections.

- [ ] **Step 3: Re-run consistency checks**

Run same grep command.
Expected: no active-version contradiction remains.

- [ ] **Step 4: Stage docs changes only (no commit yet)**

```bash
git add docs/14_NEXT_PHASE_DISPATCH.md docs/17_DELIVERY_READINESS.md docs/19_DELIVERY_UAT_CASES.md docs/20_RELEASE_SYNC_PLAYBOOK.md
```

Rule:
- Staging is allowed.
- Committing is forbidden until Task 5 full gate is green.

---

### Task 4: Decide untracked `docs/superpowers/` handling explicitly (P0)

**Files:**
- Decision target: `docs/superpowers/` tree
- Optional modify: `.gitignore` (only if user intent is to ignore)

- [ ] **Step 1: Determine whether `docs/superpowers/` should be tracked this run**

Decision options:
- include and commit (recommended when files are intentional work artifacts)
- remove if unintended local artifact

Policy:
- `.gitignore` policy change is out of scope unless user explicitly requests it.

- [ ] **Step 2: Execute chosen action**

Examples:
- include: `git add docs/superpowers/...`
- remove: delete directory after confirmation

- [ ] **Step 3: Verify status reflects explicit choice**

Run:
- `git status --short`

Expected: no ambiguous untracked leftovers.

- [ ] **Step 4: Stage decision result only (no commit yet)**

```bash
git add docs/superpowers
```

Rule:
- Staging is allowed.
- Committing is forbidden until Task 5 full gate is green.

---

### Task 5: Full verification hard gate (must pass before any final push)

**Files:**
- No code changes expected
- Optional evidence update file: `docs/17_DELIVERY_READINESS.md`

- [ ] **Step 1: Ensure preconditions**

Run:
- `bash scripts/preflight_local_env.sh`
- `curl -sf http://127.0.0.1:38080/ping`
- `curl -sf http://127.0.0.1:5173/ >/dev/null`

If either curl check fails:
- start/restart required service(s) using runbook commands,
- re-run preconditions until both endpoints are reachable.

- [ ] **Step 2: Run full verification commands**

Run exactly:
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run && npm run build`
- `cd apps/pad-app && flutter analyze && flutter test --no-pub && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

Expected: ALL PASS.

- [ ] **Step 3: If any command fails, fix and rerun full gate**

Rule:
- No commit/push until all pass.

- [ ] **Step 4: Update readiness evidence summary (if changed)**

- [ ] **Step 5: Final status check**

Run:
- `git status --short`

Expected: clean or only explicitly accepted local leftovers.

---

### Task 6: Final commit/push closure (only after full green)

**Files:**
- All changed files from tasks 2-5

- [ ] **Step 1: Verify acceptance criteria snapshot**

Run:
- `git status --short`
- `git log --oneline -5`

Expected:
- pending set intentional and understood.
- commits clearly explain why.

- [ ] **Step 2: Create final commits only after full gate is green**

Suggested split (if applicable):
- code closure commit
- docs consistency/ledger commit

```bash
git commit -m "fix: close remaining code/doc deltas after full verification gate"
```

- [ ] **Step 3: Push to resolved target branch**

Run:
- `target_branch=$(git branch --show-current)`
- `git push origin "$target_branch"`

- [ ] **Step 4: Capture auditable delivery record**

Record in final report/comment:
- timestamp
- commit SHA(s)
- pushed branch + remote
- full verification result summary

---

## Final Acceptance Checklist

- [ ] P0 items all closed with explicit evidence
- [ ] P1 ledger updates complete
- [ ] Full verification gate all green
- [ ] Commit SHA and push target recorded
- [ ] No ambiguous untracked leftovers

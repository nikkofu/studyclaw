import React, { useEffect, useState } from "react"
import { flattenSectionsToTasks, parseSchoolTaskMessage, REFERENCE_GROUP_MESSAGE } from "./schoolTaskParser"

const DEFAULT_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
let draftTaskCounter = 0

function formatDateInputValue(date = new Date()) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

async function requestJSON(url, options = {}) {
  const response = await fetch(url, options)
  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw new Error(data.error || "请求失败，请检查 API 服务是否启动")
  }

  return data
}

function StatCard({ label, value, hint }) {
  return (
    <div className="stat-card">
      <span className="stat-label">{label}</span>
      <strong className="stat-value">{value}</strong>
      <span className="stat-hint">{hint}</span>
    </div>
  )
}

function WorkflowSteps({ previewTaskCount, draftTaskCount, riskyTaskCount, selectedTaskCount, createdTaskCount, parseStatus, createStatus }) {
  const steps = [
    {
      id: "preview",
      index: "01",
      title: "解析预览",
      detail: previewTaskCount > 0 ? `本地已预估 ${previewTaskCount} 条任务` : "先粘贴学校群原文并检查结构",
      state: previewTaskCount > 0 ? "active" : "idle",
    },
    {
      id: "review",
      index: "02",
      title: "编辑确认",
      detail:
        draftTaskCount > 0
          ? `AI 草稿 ${draftTaskCount} 条，其中 ${riskyTaskCount} 条风险优先处理`
          : parseStatus?.tone === "error"
            ? "解析失败，可直接修改原文后重试"
            : "提交到 API 后进入草稿审核",
      state: parseStatus?.tone === "error" ? "warning" : draftTaskCount > 0 ? "active" : "idle",
    },
    {
      id: "create",
      index: "03",
      title: "创建任务",
      detail:
        createdTaskCount > 0
          ? `本次已创建 ${createdTaskCount} 条任务`
          : selectedTaskCount > 0
            ? `已选 ${selectedTaskCount} 条任务，等待创建`
            : createStatus?.tone === "error"
              ? "创建失败，草稿与选中项均已保留"
              : "确认选中任务后写入今日清单",
      state: createStatus?.tone === "error" ? "warning" : createdTaskCount > 0 || selectedTaskCount > 0 ? "active" : "idle",
    },
  ]

  return (
    <div className="workflow-strip" aria-label="家长操作路径">
      {steps.map((step) => (
        <article className={`workflow-card workflow-${step.state}`} key={step.id}>
          <span className="workflow-index">{step.index}</span>
          <div>
            <strong>{step.title}</strong>
            <p>{step.detail}</p>
          </div>
        </article>
      ))}
    </div>
  )
}

function StatusBanner({
  status,
  actionLabel,
  onAction,
  actionDisabled = false,
  secondaryActionLabel,
  onSecondaryAction,
  secondaryActionDisabled = false,
}) {
  if (!status) {
    return null
  }

  return (
    <div className={`status-banner banner-${status.tone || "info"}`} role="status" aria-live="polite">
      <div className="status-banner-copy">
        <strong>{status.title}</strong>
        <p>{status.message}</p>
      </div>
      {actionLabel || secondaryActionLabel ? (
        <div className="status-banner-actions">
          {secondaryActionLabel ? (
            <button className="ghost-button compact" type="button" onClick={onSecondaryAction} disabled={secondaryActionDisabled}>
              {secondaryActionLabel}
            </button>
          ) : null}
          {actionLabel ? (
            <button className="ghost-button compact" type="button" onClick={onAction} disabled={actionDisabled}>
              {actionLabel}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function SectionPreview({ sections }) {
  if (sections.length === 0) {
    return <p className="empty-state">尚未识别出学科分组。直接粘贴学校群内容即可，系统会按“学科: 编号任务”的方式预览。</p>
  }

  return (
    <div className="subject-list">
      {sections.map((section) => (
        <article className="subject-card" key={section.subject}>
          <header>
            <span className="subject-chip">{section.subject}</span>
            <strong>{section.items.length} 项</strong>
          </header>
          <ol>
            {section.items.map((item, index) => (
              <li key={`${section.subject}-${index}`}>
                <span>{item.text}</span>
                {item.subitems.length > 0 ? (
                  <ul>
                    {item.subitems.map((subitem, subIndex) => (
                      <li key={`${section.subject}-${index}-${subIndex}`}>{subitem}</li>
                    ))}
                  </ul>
                ) : null}
              </li>
            ))}
          </ol>
        </article>
      ))}
    </div>
  )
}

function ServerTaskList({ title, tasks, emptyText }) {
  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>{title}</h3>
        <span>{tasks.length} 条</span>
      </div>
      {tasks.length === 0 ? (
        <p className="empty-state">{emptyText}</p>
      ) : (
        <ol className="task-list">
          {tasks.map((task, index) => (
            <li key={`${task.subject}-${task.group_title || task.content || task.title}-${task.content || task.title}-${index}`}>
              <span className="task-subject">{task.subject}</span>
              <div>
                <strong>{task.content || task.title}</strong>
                {task.group_title && task.group_title !== (task.content || task.title) ? <p>分组: {task.group_title}</p> : null}
                {"completed" in task ? (
                  <p>{task.completed ? "已完成" : "待完成"}</p>
                ) : (
                  <p>已加入今日任务清单</p>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}

function createDraftTask(task = {}) {
  draftTaskCounter += 1

  return {
    id: `draft-${draftTaskCounter}`,
    subject: task.subject || "未分类",
    group_title: task.group_title || task.title || "",
    title: task.title || "",
    confidence: Number(task.confidence || 0),
    needs_review: Boolean(task.needs_review),
    notes: Array.isArray(task.notes) ? task.notes : [],
    source: task.source || "ai",
  }
}

function normalizeComparisonText(value) {
  return String(value || "")
    .toLowerCase()
    .replace(/[\s`~!@#$%^&*()_\-+=[\]{}\\|;:'",.<>/?，。；：！？、（）【】《》“”‘’·]/g, "")
}

function getConfidenceMeta(confidence) {
  if (confidence >= 0.85) {
    return { label: "高置信", tone: "high" }
  }
  if (confidence >= 0.7) {
    return { label: "中置信", tone: "medium" }
  }
  return { label: "低置信", tone: "low" }
}

function createBigrams(value) {
  if (value.length < 2) {
    return [value]
  }

  const bigrams = []
  for (let index = 0; index < value.length - 1; index += 1) {
    bigrams.push(value.slice(index, index + 2))
  }
  return bigrams
}

function calculateSimilarity(left, right) {
  if (!left || !right) {
    return 0
  }

  if (left === right) {
    return 1
  }

  if ((left.includes(right) || right.includes(left)) && Math.min(left.length, right.length) >= 4) {
    return 0.9
  }

  const leftBigrams = createBigrams(left)
  const rightBigrams = createBigrams(right)
  const rightBag = new Map()

  rightBigrams.forEach((item) => {
    rightBag.set(item, (rightBag.get(item) || 0) + 1)
  })

  let overlap = 0
  leftBigrams.forEach((item) => {
    const count = rightBag.get(item) || 0
    if (count > 0) {
      overlap += 1
      rightBag.set(item, count - 1)
    }
  })

  return (2 * overlap) / (leftBigrams.length + rightBigrams.length)
}

function buildTaskDiagnostics(draftTasks, todayTasks) {
  const diagnosticsById = Object.fromEntries(
    draftTasks.map((task) => [
      task.id,
      {
        issues: [],
        issueKeys: new Set(),
        hasBlocking: false,
      },
    ]),
  )

  function addIssue(taskId, key, severity, message) {
    const target = diagnosticsById[taskId]
    if (!target || target.issueKeys.has(key)) {
      return
    }

    target.issueKeys.add(key)
    target.issues.push({ severity, message })
    if (severity === "block") {
      target.hasBlocking = true
    }
  }

  const normalizedTodayTasks = todayTasks.map((task, index) => ({
    id: `today-${index}`,
    subject: normalizeComparisonText(task.subject),
    title: normalizeComparisonText(task.content || task.title),
    rawTitle: task.content || task.title,
    completed: task.completed,
  }))

  draftTasks.forEach((task) => {
    const normalizedTitle = normalizeComparisonText(task.title)
    const normalizedSubject = normalizeComparisonText(task.subject)

    if (!normalizedTitle) {
      addIssue(task.id, "empty-title", "block", "任务标题为空，确认前需要补全。")
    }

    if (!normalizedSubject) {
      addIssue(task.id, "empty-subject", "warn", "学科为空，将按“未分类”创建。")
    }
  })

  for (let leftIndex = 0; leftIndex < draftTasks.length; leftIndex += 1) {
    for (let rightIndex = leftIndex + 1; rightIndex < draftTasks.length; rightIndex += 1) {
      const left = draftTasks[leftIndex]
      const right = draftTasks[rightIndex]
      const leftSubject = normalizeComparisonText(left.subject)
      const rightSubject = normalizeComparisonText(right.subject)
      const leftTitle = normalizeComparisonText(left.title)
      const rightTitle = normalizeComparisonText(right.title)

      if (!leftTitle || !rightTitle || leftSubject !== rightSubject) {
        continue
      }

      if (leftTitle === rightTitle) {
        addIssue(left.id, `duplicate-draft-${right.id}`, "block", `与另一条草稿任务重复: ${right.title}`)
        addIssue(right.id, `duplicate-draft-${left.id}`, "block", `与另一条草稿任务重复: ${left.title}`)
        continue
      }

      const similarity = calculateSimilarity(leftTitle, rightTitle)
      if (similarity >= 0.82) {
        addIssue(left.id, `similar-draft-${right.id}`, "warn", `与另一条草稿任务高度相似，建议考虑合并: ${right.title}`)
        addIssue(right.id, `similar-draft-${left.id}`, "warn", `与另一条草稿任务高度相似，建议考虑合并: ${left.title}`)
      }
    }
  }

  draftTasks.forEach((task) => {
    const normalizedSubject = normalizeComparisonText(task.subject)
    const normalizedTitle = normalizeComparisonText(task.title)
    if (!normalizedSubject || !normalizedTitle) {
      return
    }

    normalizedTodayTasks.forEach((todayTask) => {
      if (todayTask.subject !== normalizedSubject || !todayTask.title) {
        return
      }

      if (todayTask.title === normalizedTitle) {
        addIssue(
          task.id,
          `duplicate-today-${todayTask.id}`,
          "block",
          `与今日任务重复: ${todayTask.rawTitle}${todayTask.completed ? "（已完成）" : ""}`,
        )
        return
      }

      const similarity = calculateSimilarity(todayTask.title, normalizedTitle)
      if (similarity >= 0.82) {
        addIssue(
          task.id,
          `similar-today-${todayTask.id}`,
          "warn",
          `与今日任务高度相似，建议确认是否重复: ${todayTask.rawTitle}${todayTask.completed ? "（已完成）" : ""}`,
        )
      }
    })
  })

  const summary = {
    blockTasks: 0,
    warningTasks: 0,
    cleanTasks: 0,
    duplicateTasks: 0,
    similarTasks: 0,
  }

  Object.values(diagnosticsById).forEach((item) => {
    const hasWarnings = item.issues.some((issue) => issue.severity === "warn")
    if (item.hasBlocking) {
      summary.blockTasks += 1
    }
    if (hasWarnings) {
      summary.warningTasks += 1
    }
    if (!item.hasBlocking && !hasWarnings) {
      summary.cleanTasks += 1
    }
    summary.duplicateTasks += item.issues.filter((issue) => issue.message.includes("重复")).length
    summary.similarTasks += item.issues.filter((issue) => issue.message.includes("相似")).length
  })

  return { byId: diagnosticsById, summary }
}

function getDraftTaskRiskMeta(task, diagnostics = { issues: [], hasBlocking: false }) {
  const confidence = Number(task.confidence || 0)
  const firstBlockingIssue = diagnostics.issues.find((issue) => issue.severity === "block")
  const firstWarningIssue = diagnostics.issues.find((issue) => issue.severity === "warn")
  const reasons = []

  if (task.needs_review) {
    reasons.push("后端标记 needs_review")
  }

  if (confidence < 0.7) {
    reasons.push(`低置信 ${Math.round(confidence * 100)}%`)
  }

  if (diagnostics.hasBlocking) {
    if (!reasons.length) {
      reasons.push("存在阻断项")
    }
    return {
      label: "需修正",
      tone: "block",
      order: 0,
      caption: firstBlockingIssue?.message || "修正阻断项后才能创建。",
      reasons,
    }
  }

  if (task.needs_review || confidence < 0.7) {
    if (!reasons.length) {
      reasons.push("建议人工确认")
    }

    let label = "高风险"
    let caption = "建议逐条确认后再创建。"

    if (task.needs_review && confidence >= 0.7) {
      label = "待确认"
      caption = "后端要求家长确认后再创建。"
    } else if (!task.needs_review && confidence < 0.7) {
      label = "低置信"
      caption = "置信度偏低，建议先修改标题再创建。"
    } else if (task.needs_review && confidence < 0.7) {
      caption = "同时出现 needs_review 与低置信，建议优先处理。"
    }

    return {
      label,
      tone: "high",
      order: 1,
      caption,
      reasons,
    }
  }

  if (firstWarningIssue) {
    return {
      label: "有提醒",
      tone: "warn",
      order: 2,
      caption: firstWarningIssue.message,
      reasons: ["存在相似或归类提醒"],
    }
  }

  return {
    label: "可创建",
    tone: "ready",
    order: 3,
    caption: "后端未标记 needs_review，可直接进入创建。",
    reasons: ["建议直接创建"],
  }
}

function DraftTaskList({
  tasks,
  selectedTaskIds,
  diagnosticsById,
  onToggle,
  onFieldChange,
  onRemove,
  onSelectRecommended,
  onSelectCleanTasks,
  onSelectAll,
  onClearSelection,
  onAddManualTask,
}) {
  const taskEntries = [...tasks]
    .map((task) => {
      const diagnostics = diagnosticsById[task.id] || { issues: [], hasBlocking: false }
      return {
        task,
        diagnostics,
        riskMeta: getDraftTaskRiskMeta(task, diagnostics),
      }
    })
    .sort((left, right) => {
      if (left.riskMeta.order !== right.riskMeta.order) {
        return left.riskMeta.order - right.riskMeta.order
      }

      const leftConfidence = Number(left.task.confidence || 0)
      const rightConfidence = Number(right.task.confidence || 0)
      if (leftConfidence !== rightConfidence) {
        return leftConfidence - rightConfidence
      }

      return String(left.task.title || "").localeCompare(String(right.task.title || ""), "zh-CN")
    })

  const riskyTaskCount = taskEntries.filter((entry) => entry.riskMeta.order <= 1).length
  const warningTaskCount = taskEntries.filter((entry) => entry.riskMeta.order === 2).length
  const recommendedCount = taskEntries.filter(
    (entry) => !entry.diagnostics.hasBlocking && !entry.task.needs_review && Number(entry.task.confidence || 0) >= 0.7,
  ).length

  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>2. AI 草稿审核</h3>
        <span>{tasks.length} 条</span>
      </div>

      <div className="review-summary">
        <div>
          <strong>风险任务已自动置顶</strong>
          <p>`needs_review` 和低置信任务优先展示，先处理这些卡片，再批量创建其余任务。</p>
        </div>
        <div className="summary-pill-row">
          <span className="summary-pill summary-risk">风险 {riskyTaskCount}</span>
          <span className="summary-pill summary-warn">提醒 {warningTaskCount}</span>
          <span className="summary-pill summary-ready">建议直接创建 {recommendedCount}</span>
        </div>
      </div>

      <div className="draft-toolbar">
        <button className="ghost-button compact" type="button" onClick={onSelectRecommended}>
          全选建议创建 ({recommendedCount})
        </button>
        <button className="ghost-button compact" type="button" onClick={onSelectCleanTasks}>
          全选无阻断
        </button>
        <button className="ghost-button compact" type="button" onClick={onSelectAll}>
          全选全部
        </button>
        <button className="ghost-button compact" type="button" onClick={onClearSelection}>
          清空选择
        </button>
        <button className="ghost-button compact" type="button" onClick={onAddManualTask}>
          手动补一条
        </button>
      </div>

      {tasks.length === 0 ? (
        <p className="empty-state">先点击“AI 解析任务”，再确认哪些任务写入今日清单。</p>
      ) : (
        <div className="draft-list">
          {taskEntries.map(({ task, diagnostics, riskMeta }, index) => {
            const isSelected = selectedTaskIds.includes(task.id)
            const confidence = Number(task.confidence || 0)
            const notes = Array.isArray(task.notes) ? task.notes : []
            const confidenceMeta = getConfidenceMeta(confidence)

            return (
              <article
                className={`draft-card risk-${riskMeta.tone} ${task.needs_review ? "needs-review" : ""} ${
                  diagnostics.hasBlocking ? "has-blocking" : ""
                } ${isSelected ? "is-selected" : ""}`}
                data-testid="draft-card"
                key={`${task.id}-${index}`}
              >
                <div className="draft-header">
                  <input
                    aria-label={`选择任务 ${task.title || task.group_title || task.subject}`}
                    type="checkbox"
                    checked={isSelected}
                    onChange={() => onToggle(task.id)}
                  />
                  <span className="task-subject">{task.subject}</span>
                  <span className={`confidence-pill confidence-${confidenceMeta.tone}`}>{confidenceMeta.label}</span>
                  <span className={`risk-badge badge-${riskMeta.tone}`}>{riskMeta.label}</span>
                  {task.source === "manual" ? <span className="review-pill manual-pill">手动补充</span> : null}
                  <button className="inline-link danger" type="button" onClick={() => onRemove(task.id)}>
                    删除
                  </button>
                </div>

                <div className="risk-reason-row">
                  {riskMeta.reasons.map((reason, reasonIndex) => (
                    <span className={`risk-reason reason-${riskMeta.tone}`} key={`${task.id}-reason-${reasonIndex}`}>
                      {reason}
                    </span>
                  ))}
                </div>

                <p className={`risk-caption caption-${riskMeta.tone}`}>{riskMeta.caption}</p>

                <div className="draft-edit-grid">
                  <label>
                    <span>学科</span>
                    <input value={task.subject} onChange={(event) => onFieldChange(task.id, "subject", event.target.value)} />
                  </label>
                  <label>
                    <span>分组</span>
                    <input value={task.group_title} onChange={(event) => onFieldChange(task.id, "group_title", event.target.value)} />
                  </label>
                  <label className="draft-title-field">
                    <span>任务标题</span>
                    <input value={task.title} onChange={(event) => onFieldChange(task.id, "title", event.target.value)} />
                  </label>
                </div>

                <p>置信度 {Math.round(confidence * 100)}%</p>
                {diagnostics.issues.length > 0 ? (
                  <ul className="quality-list">
                    {diagnostics.issues.map((issue, issueIndex) => (
                      <li key={`${task.id}-issue-${issueIndex}`} className={issue.severity === "block" ? "issue-block" : "issue-warn"}>
                        {issue.message}
                      </li>
                    ))}
                  </ul>
                ) : null}
                {notes.length > 0 ? (
                  <ul className="rule-list compact">
                    {notes.map((note, noteIndex) => (
                      <li key={`${task.id}-note-${noteIndex}`}>{note}</li>
                    ))}
                  </ul>
                ) : null}
              </article>
            )
          })}
        </div>
      )}
    </section>
  )
}

function CreatePanel({
  diagnosticsSummary,
  selectedTasks,
  diagnosticsById,
  recommendedCount,
  assignedDate,
  onSelectRecommended,
  onConfirm,
  isConfirming,
  createStatus,
}) {
  const selectedBlockingCount = selectedTasks.filter((task) => diagnosticsById[task.id]?.hasBlocking).length
  const selectedRiskCount = selectedTasks.filter((task) => {
    const riskMeta = getDraftTaskRiskMeta(task, diagnosticsById[task.id] || { issues: [], hasBlocking: false })
    return riskMeta.order <= 1 && !(diagnosticsById[task.id]?.hasBlocking)
  }).length
  const selectedReadyCount = selectedTasks.filter(
    (task) => !diagnosticsById[task.id]?.hasBlocking && !task.needs_review && Number(task.confidence || 0) >= 0.7,
  ).length
  const canSubmit = selectedTasks.length > 0 && selectedBlockingCount === 0 && !isConfirming

  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>3. 编辑确认并创建</h3>
        <span>{selectedTasks.length} 条已选</span>
      </div>

      <div className="analysis-grid confirm-grid">
        <div className="analysis-card">
          <span>已选任务</span>
          <strong>{selectedTasks.length}</strong>
        </div>
        <div className="analysis-card">
          <span>已选风险任务</span>
          <strong>{selectedRiskCount}</strong>
        </div>
        <div className="analysis-card">
          <span>已选阻断项</span>
          <strong>{selectedBlockingCount}</strong>
        </div>
        <div className="analysis-card">
          <span>建议直接创建</span>
          <strong>{selectedReadyCount}</strong>
        </div>
      </div>

      <ul className="rule-list compact">
        <li>阻断项任务 {diagnosticsSummary.blockTasks} 条，必须先修正后才能创建。</li>
        <li>提醒项任务 {diagnosticsSummary.warningTasks} 条，可创建但建议先人工确认。</li>
        <li>`needs_review` 和低置信会保留原始后端含义，只在前端做风险高亮，不改变字段语义。</li>
      </ul>

      <p className="inline-hint">当前所选任务会创建到 {assignedDate || "未选择日期"}。</p>
      {selectedBlockingCount > 0 ? <p className="inline-hint hint-error">当前选中项含阻断风险，先修正标题或去重后再创建。</p> : null}
      {selectedRiskCount > 0 ? <p className="inline-hint hint-warning">当前选中项含 {selectedRiskCount} 条高风险任务，建议逐条核对。</p> : null}
      {selectedTasks.length === 0 ? <p className="inline-hint">先在上方勾选要创建的任务，再进入创建。</p> : null}

      <div className="confirm-action-row">
        <button className="ghost-button" type="button" onClick={onSelectRecommended}>
          只选建议创建 ({recommendedCount})
        </button>
        <button className="primary-button secondary" type="button" disabled={!canSubmit} onClick={onConfirm}>
          {isConfirming ? "创建中..." : `确认创建选中任务 (${selectedTasks.length})`}
        </button>
      </div>

      <StatusBanner
        status={createStatus}
        actionLabel={createStatus?.retryable ? (isConfirming ? "重试中..." : "重试创建") : undefined}
        onAction={createStatus?.retryable ? onConfirm : undefined}
        actionDisabled={!createStatus?.retryable || !canSubmit}
      />
    </section>
  )
}

function AnalysisPanel({ parserMode, analysis }) {
  if (!parserMode && !analysis) {
    return (
      <section className="panel">
        <div className="panel-heading">
          <h3>AI 解析摘要</h3>
          <span>等待提交</span>
        </div>
        <p className="empty-state">提交到 API 后，这里会展示 LLM 混合解析模式、识别学科和自动补全说明。</p>
      </section>
    )
  }

  const detectedSubjects = Array.isArray(analysis?.detected_subjects) ? analysis.detected_subjects : []
  const formatSignals = Array.isArray(analysis?.format_signals) ? analysis.format_signals : []
  const notes = Array.isArray(analysis?.notes) ? analysis.notes : []

  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>AI 解析摘要</h3>
        <span>{parserMode || "unknown"}</span>
      </div>

      <div className="analysis-grid">
        <div className="analysis-card">
          <span>解析模式</span>
          <strong>{parserMode === "llm_hybrid" ? "LLM 混合解析" : "规则兜底解析"}</strong>
        </div>
        <div className="analysis-card">
          <span>识别学科</span>
          <strong>{detectedSubjects.length > 0 ? detectedSubjects.join(" / ") : "未识别"}</strong>
        </div>
        <div className="analysis-card">
          <span>格式信号</span>
          <strong>{formatSignals.length > 0 ? formatSignals.join(" / ") : "无"}</strong>
        </div>
      </div>

      {notes.length > 0 ? (
        <ul className="rule-list compact">
          {notes.map((note, index) => (
            <li key={`${note}-${index}`}>{note}</li>
          ))}
        </ul>
      ) : (
        <p className="empty-state">当前解析未返回额外说明。</p>
      )}
    </section>
  )
}

export default function App() {
  const [apiBaseUrl, setApiBaseUrl] = useState(DEFAULT_API_BASE_URL)
  const [familyId, setFamilyId] = useState("101")
  const [assigneeId, setAssigneeId] = useState("201")
  const [assignedDate, setAssignedDate] = useState(() => formatDateInputValue())
  const [rawText, setRawText] = useState(REFERENCE_GROUP_MESSAGE)
  const [draftTasks, setDraftTasks] = useState([])
  const [selectedTaskIds, setSelectedTaskIds] = useState([])
  const [createdTasks, setCreatedTasks] = useState([])
  const [todayTasks, setTodayTasks] = useState([])
  const [todayDate, setTodayDate] = useState("")
  const [parserMode, setParserMode] = useState("")
  const [analysis, setAnalysis] = useState(null)
  const [parseStatus, setParseStatus] = useState(null)
  const [createStatus, setCreateStatus] = useState(null)
  const [refreshError, setRefreshError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isConfirming, setIsConfirming] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const previewSections = parseSchoolTaskMessage(rawText)
  const previewTasks = flattenSectionsToTasks(previewSections)
  const completedCount = todayTasks.filter((task) => task.completed).length
  const diagnostics = buildTaskDiagnostics(draftTasks, todayTasks)
  const selectedDraftTasks = draftTasks.filter((task) => selectedTaskIds.includes(task.id))
  const riskyDraftTaskCount = draftTasks.filter((task) => {
    const riskMeta = getDraftTaskRiskMeta(task, diagnostics.byId[task.id] || { issues: [], hasBlocking: false })
    return riskMeta.order <= 1
  }).length
  const recommendedTaskCount = draftTasks.filter(
    (task) => !diagnostics.byId[task.id]?.hasBlocking && !task.needs_review && Number(task.confidence || 0) >= 0.7,
  ).length

  async function refreshTasks() {
    if (!familyId || !assigneeId) {
      return
    }

    setIsRefreshing(true)
    setRefreshError("")

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/tasks?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(assigneeId)}`,
      )
      setTodayTasks(Array.isArray(data.tasks) ? data.tasks : [])
      setTodayDate(data.date || "")
    } catch (requestError) {
      setRefreshError(requestError.message)
    } finally {
      setIsRefreshing(false)
    }
  }

  function toggleSelectedTask(taskId) {
    setSelectedTaskIds((current) =>
      current.includes(taskId) ? current.filter((item) => item !== taskId) : [...current, taskId],
    )
  }

  function updateDraftTask(taskId, field, value) {
    setDraftTasks((current) =>
      current.map((task) => {
        if (task.id !== taskId) {
          return task
        }

        return {
          ...task,
          [field]: value,
        }
      }),
    )
  }

  function removeDraftTask(taskId) {
    setDraftTasks((current) => current.filter((task) => task.id !== taskId))
    setSelectedTaskIds((current) => current.filter((id) => id !== taskId))
  }

  function addManualTask() {
    const manualTask = createDraftTask({
      subject: "未分类",
      group_title: "",
      title: "",
      confidence: 1,
      needs_review: true,
      notes: ["这是手动补充任务，请确认标题后再创建。"],
      source: "manual",
    })

    setDraftTasks((current) => [...current, manualTask])
    setSelectedTaskIds((current) => [...current, manualTask.id])
  }

  function selectRecommendedTasks() {
    setSelectedTaskIds(
      draftTasks
        .filter((task) => !diagnostics.byId[task.id]?.hasBlocking && !task.needs_review && Number(task.confidence || 0) >= 0.7)
        .map((task) => task.id),
    )
  }

  function selectAllTasks() {
    setSelectedTaskIds(draftTasks.map((task) => task.id))
  }

  function selectCleanTasks() {
    setSelectedTaskIds(
      draftTasks.filter((task) => !(diagnostics.byId[task.id]?.hasBlocking)).map((task) => task.id),
    )
  }

  function clearTaskSelection() {
    setSelectedTaskIds([])
  }

  async function runParse() {
    setParseStatus(null)
    setCreateStatus(null)

    if (!assignedDate) {
      setParseStatus({
        tone: "error",
        title: "无法开始解析",
        message: "请先选择任务日期，明确本次任务要创建到哪一天。",
        retryable: false,
      })
      return
    }

    if (!rawText.trim()) {
      setParseStatus({
        tone: "error",
        title: "无法开始解析",
        message: "请先粘贴学校群任务内容。",
        retryable: false,
      })
      return
    }

    setIsSubmitting(true)

    try {
      const payload = {
        family_id: Number(familyId),
        assignee_id: Number(assigneeId),
        assigned_date: assignedDate,
        raw_text: rawText.trim(),
        auto_create: false,
      }

      const data = await requestJSON(`${apiBaseUrl}/api/v1/tasks/parse`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })

      const parsedTasks = Array.isArray(data.tasks) ? data.tasks.map((task) => createDraftTask(task)) : []
      setDraftTasks(parsedTasks)
      setSelectedTaskIds(
        parsedTasks.filter((task) => !task.needs_review && Number(task.confidence || 0) >= 0.7).map((task) => task.id),
      )
      setCreatedTasks([])
      setParserMode(data.parser_mode || "")
      setAnalysis(data.analysis || null)

      if (parsedTasks.length === 0) {
        setParseStatus({
          tone: "warning",
          title: "解析完成，但未生成草稿",
          message: `原文和本地结构预览已保留。当前目标日期为 ${assignedDate}，可以调整原文格式后再次解析。`,
          retryable: true,
        })
      } else {
        setParseStatus({
          tone: "success",
          title: "AI 草稿已生成",
          message: `已识别 ${data.parsed_count || parsedTasks.length} 个建议任务，目标日期 ${assignedDate}，请先处理风险项再确认创建。`,
          retryable: false,
        })
      }
    } catch (requestError) {
      setParseStatus({
        tone: "error",
        title: draftTasks.length > 0 ? "解析失败，上一轮草稿已保留" : "解析失败，输入内容已保留",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleSubmit(event) {
    event.preventDefault()
    void runParse()
  }

  async function runConfirmCreate() {
    setCreateStatus(null)

    if (!assignedDate) {
      setCreateStatus({
        tone: "error",
        title: "无法开始创建",
        message: "请先选择任务日期，明确本次任务要创建到哪一天。",
        retryable: false,
      })
      return
    }

    if (selectedDraftTasks.length === 0) {
      setCreateStatus({
        tone: "error",
        title: "无法开始创建",
        message: "请至少选择一条建议任务再确认创建。",
        retryable: false,
      })
      return
    }

    const blockingTasks = selectedDraftTasks.filter((task) => diagnostics.byId[task.id]?.hasBlocking)
    if (blockingTasks.length > 0) {
      setCreateStatus({
        tone: "error",
        title: "当前选中项含阻断风险",
        message: `当前选中的任务里有 ${blockingTasks.length} 条存在重复或标题为空，请先修改后再确认创建。`,
        retryable: false,
      })
      return
    }

    const sanitizedTasks = selectedDraftTasks
      .map((task) => ({
        subject: task.subject.trim(),
        group_title: (task.group_title || task.title).trim(),
        title: task.title.trim(),
        confidence: task.confidence,
        needs_review: task.needs_review,
        notes: task.notes,
      }))
      .filter((task) => task.title)

    if (sanitizedTasks.length === 0) {
      setCreateStatus({
        tone: "error",
        title: "当前选中项无法创建",
        message: "选中的任务标题不能为空。",
        retryable: false,
      })
      return
    }

    setIsConfirming(true)

    try {
      const selectedIds = new Set(selectedDraftTasks.map((task) => task.id))
      const data = await requestJSON(`${apiBaseUrl}/api/v1/tasks/confirm`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          family_id: Number(familyId),
          assignee_id: Number(assigneeId),
          assigned_date: assignedDate,
          tasks: sanitizedTasks,
        }),
      })

      setCreatedTasks(Array.isArray(data.tasks) ? data.tasks : [])
      setDraftTasks((current) => current.filter((task) => !selectedIds.has(task.id)))
      setSelectedTaskIds((current) => current.filter((taskId) => !selectedIds.has(taskId)))
      setCreateStatus({
        tone: "success",
        title: "任务已创建",
        message: `已确认并创建 ${data.created_count || sanitizedTasks.length} 个任务到 ${assignedDate}。`,
        retryable: false,
      })
      await refreshTasks()
    } catch (requestError) {
      setCreateStatus({
        tone: "error",
        title: "创建失败，草稿与选中项已保留",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsConfirming(false)
    }
  }

  useEffect(() => {
    void refreshTasks()
  }, [])

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">StudyClaw Parent Console</p>
          <h1>学校群任务一键转为孩子今日清单</h1>
          <p className="hero-copy">
            支持你刚给出的这种格式: 按学科分段、按序号列主任务、用“（1）（2）（3）”补充子步骤。页面会先做本地结构预览，再调 API
            调用 Go 后端中的 AI 混合解析链路，给出建议任务、置信度与风险提示，再由家长确认创建。
          </p>
        </div>
        <div className="hero-stats">
          <StatCard label="学科分组" value={previewSections.length} hint="从群消息中自动识别" />
          <StatCard label="预估任务数" value={previewTasks.length} hint="提交前即可预览" />
          <StatCard label="今日完成" value={`${completedCount}/${todayTasks.length}`} hint={todayDate || "等待任务刷新"} />
        </div>
      </section>

      <section className="workspace">
        <form className="panel composer" onSubmit={handleSubmit}>
          <div className="panel-heading">
            <h2>家长输入</h2>
            <span>最小录入页</span>
          </div>

          <WorkflowSteps
            previewTaskCount={previewTasks.length}
            draftTaskCount={draftTasks.length}
            riskyTaskCount={riskyDraftTaskCount}
            selectedTaskCount={selectedDraftTasks.length}
            createdTaskCount={createdTasks.length}
            parseStatus={parseStatus}
            createStatus={createStatus}
          />

          <div className="field-grid">
            <label>
              <span>API 地址</span>
              <input value={apiBaseUrl} onChange={(event) => setApiBaseUrl(event.target.value)} />
            </label>
            <label>
              <span>家庭 ID</span>
              <input value={familyId} onChange={(event) => setFamilyId(event.target.value)} />
            </label>
            <label>
              <span>孩子 ID</span>
              <input value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)} />
            </label>
            <label>
              <span>任务日期</span>
              <input type="date" value={assignedDate} onChange={(event) => setAssignedDate(event.target.value)} />
            </label>
          </div>

          <p className="field-note">当前解析与确认创建都会写入 {assignedDate || "未选择日期"}。</p>

          <label className="text-field">
            <span>学校群原文</span>
            <textarea
              value={rawText}
              onChange={(event) => setRawText(event.target.value)}
              placeholder="直接粘贴老师发到学校群的作业安排"
              rows={16}
            />
          </label>

          <div className="action-row">
            <button className="ghost-button" type="button" onClick={() => setRawText(REFERENCE_GROUP_MESSAGE)}>
              填入参考任务
            </button>
            <button className="ghost-button" type="button" onClick={refreshTasks} disabled={isRefreshing}>
              {isRefreshing ? "刷新中..." : "刷新今日任务"}
            </button>
            <button className="primary-button" type="submit" disabled={isSubmitting}>
              {isSubmitting ? "AI 解析中..." : "AI 解析任务"}
            </button>
          </div>

          <StatusBanner
            status={parseStatus}
            actionLabel={parseStatus?.retryable ? (isSubmitting ? "重试中..." : "重新解析") : undefined}
            onAction={parseStatus?.retryable ? () => void runParse() : undefined}
            actionDisabled={!parseStatus?.retryable || isSubmitting}
          />
          {refreshError ? <p className="inline-hint hint-error">刷新今日任务失败: {refreshError}</p> : null}
        </form>

        <section className="column-stack">
          <section className="panel">
            <div className="panel-heading">
              <h2>1. 群消息结构预览</h2>
              <span>本地即时解析</span>
            </div>
            <SectionPreview sections={previewSections} />
          </section>

          <ServerTaskList title="1. 本地结构任务预览" tasks={previewTasks} emptyText="等待识别群消息内容。" />
          <DraftTaskList
            tasks={draftTasks}
            selectedTaskIds={selectedTaskIds}
            diagnosticsById={diagnostics.byId}
            onToggle={toggleSelectedTask}
            onFieldChange={updateDraftTask}
            onRemove={removeDraftTask}
            onSelectRecommended={selectRecommendedTasks}
            onSelectCleanTasks={selectCleanTasks}
            onSelectAll={selectAllTasks}
            onClearSelection={clearTaskSelection}
            onAddManualTask={addManualTask}
          />
          <CreatePanel
            diagnosticsSummary={diagnostics.summary}
            selectedTasks={selectedDraftTasks}
            diagnosticsById={diagnostics.byId}
            recommendedCount={recommendedTaskCount}
            assignedDate={assignedDate}
            onSelectRecommended={selectRecommendedTasks}
            onConfirm={() => void runConfirmCreate()}
            isConfirming={isConfirming}
            createStatus={createStatus}
          />
          <AnalysisPanel parserMode={parserMode} analysis={analysis} />
        </section>
      </section>

      <section className="workspace secondary">
        <ServerTaskList title="本次 API 已创建任务" tasks={createdTasks} emptyText="还没有提交到 API。" />
        <ServerTaskList title="今日任务清单" tasks={todayTasks} emptyText="当前孩子工作区还没有任务。" />
      </section>

      <section className="panel notes">
        <div className="panel-heading">
          <h2>支持的录入规则</h2>
          <span>面向学校群常见格式</span>
        </div>
        <ul className="rule-list">
          <li>支持 `数学3.6：`、`英：`、`语文：` 这种带日期或简称的学科标题。</li>
          <li>支持 `1、`、`1.` 作为主任务编号。</li>
          <li>支持 `（1）`、`（2）`、`（3）` 作为同一任务下的补充步骤。</li>
          <li>提交后调用 `/api/v1/tasks/parse`，审核后调用 `/api/v1/tasks/confirm`，并自动刷新 `/api/v1/tasks`。</li>
        </ul>
      </section>
    </main>
  )
}

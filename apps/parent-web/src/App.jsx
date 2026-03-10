import React, { useEffect, useState } from "react"
import { flattenSectionsToTasks, parseSchoolTaskMessage, REFERENCE_GROUP_MESSAGE } from "./schoolTaskParser"

const DEFAULT_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
const CHILD_PROFILES = [
  { id: "mia-grade4", label: "苗苗 / 四年级", familyId: "101", assigneeId: "201" },
  { id: "leo-grade2", label: "乐乐 / 二年级", familyId: "102", assigneeId: "202" },
  { id: "yoyo-grade6", label: "悠悠 / 六年级", familyId: "103", assigneeId: "203" },
]

function formatDateInputValue(date = new Date()) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

function parseDateInputValue(value) {
  const [year, month, day] = String(value || formatDateInputValue())
    .split("-")
    .map((item) => Number(item))
  return new Date(year, (month || 1) - 1, day || 1)
}

function shiftDate(value, days) {
  const date = parseDateInputValue(value)
  date.setDate(date.getDate() + days)
  return formatDateInputValue(date)
}

function createLocalId(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

async function requestJSON(url, options = {}) {
  const response = await fetch(url, options)
  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw new Error(data.error || "请求失败，请检查 API 服务是否启动")
  }

  return data
}

function resolveChildProfile(familyId, assigneeId) {
  return CHILD_PROFILES.find((item) => item.familyId === familyId && item.assigneeId === assigneeId) || null
}

function resolveChildLabel(familyId, assigneeId) {
  const profile = resolveChildProfile(familyId, assigneeId)
  return profile ? profile.label : `孩子 ${assigneeId || "--"}`
}

function formatSignedAmount(amount) {
  const numericAmount = Number(amount || 0)
  return `${numericAmount >= 0 ? "+" : ""}${numericAmount}`
}

function buildSubjectRows(tasks) {
  const subjectMap = new Map()

  tasks.forEach((task) => {
    const subject = task.subject || "未分类"
    const current = subjectMap.get(subject) || {
      subject,
      total: 0,
      completed: 0,
    }
    current.total += 1
    if (task.completed) {
      current.completed += 1
    }
    subjectMap.set(subject, current)
  })

  return [...subjectMap.values()].sort((left, right) => right.total - left.total || left.subject.localeCompare(right.subject, "zh-CN"))
}

function buildDailyReport(tasks, pointEntries, assignedDate) {
  const total = tasks.length
  const completed = tasks.filter((task) => task.completed).length
  const pending = Math.max(total - completed, 0)
  const completionRate = total > 0 ? Math.round((completed / total) * 100) : 0
  const subjectRows = buildSubjectRows(tasks)
  const pointDelta = pointEntries.reduce((sum, item) => sum + Number(item.amount || 0), 0)
  const latestReason = pointEntries[0]?.reason

  let summary
  if (total === 0) {
    summary = `${assignedDate} 还没有任务数据。可以先完成发布，再回到这里查看当日完成率、积分变化和周报入口。`
  } else if (pending === 0) {
    summary = `${assignedDate} 的 ${total} 条任务已全部完成，完成率 100%。${pointEntries.length > 0 ? `今日积分变化 ${formatSignedAmount(pointDelta)}。` : ""}`
  } else {
    const leadingSubject = subjectRows[0]?.subject ? `${subjectRows[0].subject} ${subjectRows[0].total} 条` : "多学科"
    summary = `${assignedDate} 共 ${total} 条任务，已完成 ${completed} 条，完成率 ${completionRate}%。当前还剩 ${pending} 条待完成，任务主要集中在 ${leadingSubject}。`
  }

  if (latestReason) {
    summary += ` 最近一次积分备注: ${latestReason}。`
  }

  return {
    total,
    completed,
    pending,
    completionRate,
    pointDelta,
    subjectRows,
    summary,
  }
}

function buildWeeklyRows(rawStats) {
  if (!Array.isArray(rawStats)) {
    return []
  }

  return rawStats.map((item) => {
    const tasks = Array.isArray(item.tasks) ? item.tasks : []
    const total = tasks.length
    const completed = tasks.filter((task) => task.completed).length
    const ratio = total > 0 ? completed / total : 0

    return {
      date: item.date,
      label: String(item.date || "").slice(5),
      total,
      completed,
      ratio,
    }
  })
}

function createWordItem(text = "") {
  return {
    id: createLocalId("word-item"),
    text,
  }
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

function ConsoleNav({ childLabel, assignedDate, pointDelta, wordListCount }) {
  return (
    <nav className="console-nav" aria-label="家长主控制台导航">
      <a href="#publish-console">发布作业</a>
      <a href="#report-console">查看反馈</a>
      <a href="#points-console">积分操作</a>
      <a href="#word-console">单词清单</a>
      <span className="console-context">
        当前孩子: {childLabel} / 日期: {assignedDate} / 今日积分 {formatSignedAmount(pointDelta)} / 清单 {wordListCount}
      </span>
    </nav>
  )
}

function WorkflowSteps({
  previewTaskCount,
  draftTaskCount,
  riskyTaskCount,
  selectedTaskCount,
  createdTaskCount,
  parseStatus,
  createStatus,
}) {
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
      title: "发布任务",
      detail:
        createdTaskCount > 0
          ? `本次已发布 ${createdTaskCount} 条任务`
          : selectedTaskCount > 0
            ? `已选 ${selectedTaskCount} 条任务，等待发布`
            : createStatus?.tone === "error"
              ? "发布失败，草稿与选中项均已保留"
              : "确认选中任务后写入孩子任务板",
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

function ServerTaskList({ title, tasks, emptyText, caption }) {
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <h3>{title}</h3>
          {caption ? <p className="panel-caption">{caption}</p> : null}
        </div>
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
                  <p>已加入指定日期任务板</p>
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
  return {
    id: createLocalId("draft"),
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
          `与当日任务重复: ${todayTask.rawTitle}${todayTask.completed ? "（已完成）" : ""}`,
        )
        return
      }

      const similarity = calculateSimilarity(todayTask.title, normalizedTitle)
      if (similarity >= 0.82) {
        addIssue(
          task.id,
          `similar-today-${todayTask.id}`,
          "warn",
          `与当日任务高度相似，建议确认是否重复: ${todayTask.rawTitle}${todayTask.completed ? "（已完成）" : ""}`,
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
      caption: firstBlockingIssue?.message || "修正阻断项后才能发布。",
      reasons,
    }
  }

  if (task.needs_review || confidence < 0.7) {
    if (!reasons.length) {
      reasons.push("建议人工确认")
    }

    let label = "高风险"
    let caption = "建议逐条确认后再发布。"

    if (task.needs_review && confidence >= 0.7) {
      label = "待确认"
      caption = "后端要求家长确认后再发布。"
    } else if (!task.needs_review && confidence < 0.7) {
      label = "低置信"
      caption = "置信度偏低，建议先修改标题再发布。"
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
    label: "可发布",
    tone: "ready",
    order: 3,
    caption: "后端未标记 needs_review，可直接进入发布。",
    reasons: ["建议直接发布"],
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
        <div>
          <h3>2. AI 草稿审核</h3>
          <p className="panel-caption">高风险任务自动置顶，先修正后再发布。</p>
        </div>
        <span>{tasks.length} 条</span>
      </div>

      <div className="review-summary">
        <div>
          <strong>风险任务已自动置顶</strong>
          <p>`needs_review` 和低置信任务优先展示，先处理这些卡片，再批量发布其余任务。</p>
        </div>
        <div className="summary-pill-row">
          <span className="summary-pill summary-risk">风险 {riskyTaskCount}</span>
          <span className="summary-pill summary-warn">提醒 {warningTaskCount}</span>
          <span className="summary-pill summary-ready">建议直接发布 {recommendedCount}</span>
        </div>
      </div>

      <div className="draft-toolbar">
        <button className="ghost-button compact" type="button" onClick={onSelectRecommended}>
          全选建议发布 ({recommendedCount})
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
        <p className="empty-state">先点击“AI 解析任务”，再确认哪些任务写入孩子当天任务板。</p>
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
        <div>
          <h3>3. 编辑确认并发布</h3>
          <p className="panel-caption">发布后会刷新指定日期的孩子任务板。</p>
        </div>
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
          <span>建议直接发布</span>
          <strong>{selectedReadyCount}</strong>
        </div>
      </div>

      <ul className="rule-list compact">
        <li>阻断项任务 {diagnosticsSummary.blockTasks} 条，必须先修正后才能发布。</li>
        <li>提醒项任务 {diagnosticsSummary.warningTasks} 条，可发布但建议先人工确认。</li>
        <li>`needs_review` 和低置信会保留原始后端含义，只在前端做风险高亮，不改变字段语义。</li>
      </ul>

      <p className="inline-hint">当前所选任务会发布到 {assignedDate || "未选择日期"}。</p>
      {selectedBlockingCount > 0 ? <p className="inline-hint hint-error">当前选中项含阻断风险，先修正标题或去重后再发布。</p> : null}
      {selectedRiskCount > 0 ? <p className="inline-hint hint-warning">当前选中项含 {selectedRiskCount} 条高风险任务，建议逐条核对。</p> : null}
      {selectedTasks.length === 0 ? <p className="inline-hint">先在上方勾选要发布的任务，再进入发布。</p> : null}

      <div className="confirm-action-row">
        <button className="ghost-button" type="button" onClick={onSelectRecommended}>
          只选建议发布 ({recommendedCount})
        </button>
        <button className="primary-button secondary" type="button" disabled={!canSubmit} onClick={onConfirm}>
          {isConfirming ? "发布中..." : `确认发布选中任务 (${selectedTasks.length})`}
        </button>
      </div>

      <StatusBanner
        status={createStatus}
        actionLabel={createStatus?.retryable ? (isConfirming ? "重试中..." : "重试发布") : undefined}
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

function TrendBars({ items, valueLabel }) {
  if (items.length === 0) {
    return <p className="empty-state">当前还没有可展示的图表数据。</p>
  }

  return (
    <div className="trend-list">
      {items.map((item) => (
        <article className="trend-row" key={item.label}>
          <div className="trend-meta">
            <strong>{item.label}</strong>
            <span>{valueLabel(item)}</span>
          </div>
          <div className="trend-bar-track" aria-hidden="true">
            <div className="trend-bar-fill" style={{ width: `${Math.round((item.ratio || 0) * 100)}%` }} />
          </div>
        </article>
      ))}
    </div>
  )
}

function FeedbackPanel({
  assignedDate,
  childLabel,
  todayTasks,
  currentDatePointEntries,
  weeklyStats,
  weeklyStatus,
  monthlyRows,
  monthlyStatus,
  reportView,
  onLoadWeekly,
  onLoadMonthly,
  isLoadingWeekly,
  isLoadingMonthly,
}) {
  const dailyReport = buildDailyReport(todayTasks, currentDatePointEntries, assignedDate)
  const weeklyRows = buildWeeklyRows(weeklyStats?.raw_stats)
  const strengths = Array.isArray(weeklyStats?.insights?.strengths) ? weeklyStats.insights.strengths : []
  const improvements = Array.isArray(weeklyStats?.insights?.areas_for_improvement)
    ? weeklyStats.insights.areas_for_improvement
    : []

  return (
    <section className="panel" id="report-console">
      <div className="panel-heading">
        <div>
          <h2>家长查看反馈</h2>
          <p className="panel-caption">基于指定日期任务板、手工积分和每周统计生成当前反馈。</p>
        </div>
        <span>{childLabel}</span>
      </div>

      <div className="analysis-grid report-overview-grid">
        <div className="analysis-card">
          <span>当日完成率</span>
          <strong>{dailyReport.total > 0 ? `${dailyReport.completionRate}%` : "--"}</strong>
        </div>
        <div className="analysis-card">
          <span>剩余任务</span>
          <strong>{dailyReport.pending}</strong>
        </div>
        <div className="analysis-card">
          <span>积分变化</span>
          <strong>{formatSignedAmount(dailyReport.pointDelta)}</strong>
        </div>
        <div className="analysis-card">
          <span>学科覆盖</span>
          <strong>{dailyReport.subjectRows.length}</strong>
        </div>
      </div>

      <div className="report-summary-card">
        <span className="summary-chip">日报摘要</span>
        <p>{dailyReport.summary}</p>
      </div>

      {dailyReport.subjectRows.length > 0 ? (
        <div className="subject-progress-grid">
          {dailyReport.subjectRows.map((item) => (
            <article className="subject-progress-card" key={item.subject}>
              <strong>{item.subject}</strong>
              <span>
                {item.completed}/{item.total} 已完成
              </span>
            </article>
          ))}
        </div>
      ) : null}

      <div className="report-switcher">
        <button className="ghost-button compact" type="button" onClick={onLoadWeekly} disabled={isLoadingWeekly}>
          {isLoadingWeekly ? "加载周趋势..." : "查看周趋势"}
        </button>
        <button className="ghost-button compact" type="button" onClick={onLoadMonthly} disabled={isLoadingMonthly}>
          {isLoadingMonthly ? "加载月趋势..." : "查看月趋势"}
        </button>
      </div>

      {reportView === "week" ? (
        <div className="report-mode-block">
          <StatusBanner
            status={weeklyStatus}
            actionLabel={weeklyStatus?.retryable ? (isLoadingWeekly ? "重试中..." : "重试周趋势") : undefined}
            onAction={weeklyStatus?.retryable ? onLoadWeekly : undefined}
            actionDisabled={!weeklyStatus?.retryable || isLoadingWeekly}
          />

          {weeklyRows.length > 0 ? (
            <>
              <TrendBars
                items={weeklyRows}
                valueLabel={(item) => `${item.completed}/${item.total} 完成`}
              />
              <div className="insight-grid">
                <article className="insight-card">
                  <span className="summary-chip">周反馈摘要</span>
                  <p>{weeklyStats?.insights?.summary || "当前 weekly 接口未返回总结，已展示原始趋势数据。"}</p>
                </article>
                <article className="insight-card">
                  <span className="summary-chip">鼓励方向</span>
                  {strengths.length > 0 ? (
                    <ul className="rule-list compact">
                      {strengths.map((item, index) => (
                        <li key={`${item}-${index}`}>{item}</li>
                      ))}
                    </ul>
                  ) : (
                    <p className="empty-state">当前没有返回额外鼓励点。</p>
                  )}
                </article>
                <article className="insight-card">
                  <span className="summary-chip">下周关注</span>
                  {improvements.length > 0 ? (
                    <ul className="rule-list compact">
                      {improvements.map((item, index) => (
                        <li key={`${item}-${index}`}>{item}</li>
                      ))}
                    </ul>
                  ) : (
                    <p className="empty-state">当前没有返回改进建议。</p>
                  )}
                </article>
              </div>
            </>
          ) : weeklyStatus?.tone !== "error" ? (
            <p className="empty-state">点击“查看周趋势”后，这里会展示 7 天任务完成趋势和 weekly insight。</p>
          ) : null}
        </div>
      ) : (
        <div className="report-mode-block">
          <StatusBanner
            status={monthlyStatus}
            actionLabel={monthlyStatus?.retryable ? (isLoadingMonthly ? "重试中..." : "重试月趋势") : undefined}
            onAction={monthlyStatus?.retryable ? onLoadMonthly : undefined}
            actionDisabled={!monthlyStatus?.retryable || isLoadingMonthly}
          />
          {monthlyRows.length > 0 ? (
            <>
              <TrendBars
                items={monthlyRows}
                valueLabel={(item) =>
                  `${item.completed}/${item.total} 完成 · 积分 ${formatSignedAmount(item.pointDelta)}`
                }
              />
              <p className="field-note">月趋势使用前端按近 28 天任务板聚合，方便 SC-05 演示“周 / 月图表入口”。</p>
            </>
          ) : monthlyStatus?.tone !== "error" ? (
            <p className="empty-state">点击“查看月趋势”后，这里会聚合近 28 天任务板和手工积分变化。</p>
          ) : null}
        </div>
      )}
    </section>
  )
}

function PointsPanel({
  familyId,
  assigneeId,
  assignedDate,
  pointForm,
  onPointFormChange,
  onQuickAmount,
  onSubmit,
  onRetry,
  isSubmitting,
  pointsStatus,
  estimatedBalance,
  todayPointEntries,
  recentPointEntries,
}) {
  const todayDelta = todayPointEntries.reduce((sum, entry) => sum + Number(entry.amount || 0), 0)
  const signedPreview =
    pointForm.mode === "penalty" ? -Math.abs(Number(pointForm.amount || 0)) : Math.abs(Number(pointForm.amount || 0))

  return (
    <section className="panel" id="points-console">
      <div className="panel-heading">
        <div>
          <h2>积分操作</h2>
          <p className="panel-caption">走真实 `/api/v1/points/update`，并在本地保留最近手工操作记录。</p>
        </div>
        <span>
          家庭 {familyId} / 孩子 {assigneeId}
        </span>
      </div>

      <div className="analysis-grid report-overview-grid">
        <div className="analysis-card">
          <span>今日积分变化</span>
          <strong>{formatSignedAmount(todayDelta)}</strong>
        </div>
        <div className="analysis-card">
          <span>最近手工记录</span>
          <strong>{recentPointEntries.length}</strong>
        </div>
        <div className="analysis-card">
          <span>API 返回余额</span>
          <strong>{estimatedBalance ?? "--"}</strong>
        </div>
        <div className="analysis-card">
          <span>本次预览</span>
          <strong>{formatSignedAmount(signedPreview)}</strong>
        </div>
      </div>

      <div className="segmented-row" role="group" aria-label="积分模式">
        <button
          className={`segment-button ${pointForm.mode === "reward" ? "is-active" : ""}`}
          type="button"
          onClick={() => onPointFormChange("mode", "reward")}
        >
          家长加分
        </button>
        <button
          className={`segment-button ${pointForm.mode === "penalty" ? "is-active" : ""}`}
          type="button"
          onClick={() => onPointFormChange("mode", "penalty")}
        >
          家长扣分
        </button>
      </div>

      <div className="field-grid">
        <label>
          <span>分值</span>
          <input
            aria-label="积分分值"
            type="number"
            min="1"
            value={pointForm.amount}
            onChange={(event) => onPointFormChange("amount", event.target.value)}
          />
        </label>
        <label>
          <span>记分日期</span>
          <input
            aria-label="积分日期"
            type="date"
            value={pointForm.assignedDate}
            onChange={(event) => onPointFormChange("assignedDate", event.target.value)}
          />
        </label>
      </div>

      <div className="quick-actions">
        {[2, 5, 10].map((amount) => (
          <button className="ghost-button compact" type="button" key={amount} onClick={() => onQuickAmount(amount)}>
            {pointForm.mode === "reward" ? "+" : "-"}
            {amount}
          </button>
        ))}
      </div>

      <label className="text-field">
        <span>表扬 / 批评原因</span>
        <textarea
          aria-label="积分原因"
          value={pointForm.reason}
          onChange={(event) => onPointFormChange("reason", event.target.value)}
          rows={4}
          placeholder="例如：按时完成语文背诵 / 回家后拖延且未整理错题"
        />
      </label>

      <div className="action-row">
        <button className="ghost-button" type="button" onClick={() => onPointFormChange("assignedDate", assignedDate)}>
          使用当前任务日期
        </button>
        <button className="primary-button secondary" type="button" onClick={onSubmit} disabled={isSubmitting}>
          {isSubmitting ? "提交中..." : "提交积分调整"}
        </button>
      </div>

      <StatusBanner
        status={pointsStatus}
        actionLabel={pointsStatus?.retryable ? (isSubmitting ? "重试中..." : "重试积分提交") : undefined}
        onAction={pointsStatus?.retryable ? onRetry : undefined}
        actionDisabled={!pointsStatus?.retryable || isSubmitting}
      />

      <div className="mini-list">
        <div className="mini-list-heading">
          <strong>最近手工积分记录</strong>
          <span>{recentPointEntries.length} 条</span>
        </div>
        {recentPointEntries.length === 0 ? (
          <p className="empty-state">还没有积分操作记录。提交一次加分或扣分后，这里会保留最近明细。</p>
        ) : (
          <ul className="ledger-list">
            {recentPointEntries.map((entry) => (
              <li key={entry.id}>
                <div>
                  <strong>{formatSignedAmount(entry.amount)}</strong>
                  <p>{entry.reason}</p>
                </div>
                <span>
                  {entry.assigned_date} · {new Date(entry.created_at).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}

function WordListPanel({
  childLabel,
  familyId,
  assigneeId,
  assignedDate,
  wordListDraft,
  wordLists,
  wordListStatus,
  onDraftChange,
  onUseAssignedDate,
  onCreate,
  onRetry,
  onUpdateWordListMeta,
  onAddWordItem,
  onUpdateWordItem,
  onRemoveWordItem,
  onRemoveWordList,
}) {
  return (
    <section className="panel" id="word-console">
      <div className="panel-heading">
        <div>
          <h2>单词清单管理</h2>
          <p className="panel-caption">当前使用家长端本地持久化管理清单，确保可先演示“创建 / 编辑 / 绑定”主流程。</p>
        </div>
        <span>{childLabel}</span>
      </div>

      <div className="local-scope-card">
        <strong>当前绑定范围</strong>
        <p>
          家庭 {familyId} / 孩子 {assigneeId} / 默认日期 {assignedDate}
        </p>
      </div>

      <div className="field-grid">
        <label>
          <span>清单名</span>
          <input
            aria-label="清单名"
            value={wordListDraft.title}
            onChange={(event) => onDraftChange("title", event.target.value)}
            placeholder="例如：3 月 10 日英语默写"
          />
        </label>
        <label>
          <span>清单语言</span>
          <select
            aria-label="清单语言"
            value={wordListDraft.language}
            onChange={(event) => onDraftChange("language", event.target.value)}
          >
            <option value="英语">英语</option>
            <option value="语文">语文</option>
            <option value="拼音">拼音</option>
            <option value="其他">其他</option>
          </select>
        </label>
        <label>
          <span>绑定日期</span>
          <input
            aria-label="清单日期"
            type="date"
            value={wordListDraft.assignedDate}
            onChange={(event) => onDraftChange("assignedDate", event.target.value)}
          />
        </label>
      </div>

      <label className="text-field">
        <span>词项</span>
        <textarea
          aria-label="词项"
          value={wordListDraft.itemsText}
          onChange={(event) => onDraftChange("itemsText", event.target.value)}
          rows={5}
          placeholder="一行一个词，例如：apple"
        />
      </label>

      <div className="action-row">
        <button className="ghost-button" type="button" onClick={onUseAssignedDate}>
          使用当前任务日期
        </button>
        <button className="primary-button secondary" type="button" onClick={onCreate}>
          创建单词清单
        </button>
      </div>

      <StatusBanner
        status={wordListStatus}
        actionLabel={wordListStatus?.retryable ? "重试创建清单" : undefined}
        onAction={wordListStatus?.retryable ? onRetry : undefined}
        actionDisabled={!wordListStatus?.retryable}
      />

      <div className="mini-list">
        <div className="mini-list-heading">
          <strong>当前孩子的单词清单</strong>
          <span>{wordLists.length} 份</span>
        </div>

        {wordLists.length === 0 ? (
          <p className="empty-state">还没有为当前孩子创建单词清单。</p>
        ) : (
          <div className="word-list-grid">
            {wordLists.map((list) => (
              <article className="word-list-card" key={list.id}>
                <div className="word-list-header">
                  <label>
                    <span>清单名</span>
                    <input
                      value={list.title}
                      onChange={(event) => onUpdateWordListMeta(list.id, "title", event.target.value)}
                    />
                  </label>
                  <button className="inline-link danger" type="button" onClick={() => onRemoveWordList(list.id)}>
                    删除清单
                  </button>
                </div>

                <div className="field-grid">
                  <label>
                    <span>语言</span>
                    <select value={list.language} onChange={(event) => onUpdateWordListMeta(list.id, "language", event.target.value)}>
                      <option value="英语">英语</option>
                      <option value="语文">语文</option>
                      <option value="拼音">拼音</option>
                      <option value="其他">其他</option>
                    </select>
                  </label>
                  <label>
                    <span>绑定日期</span>
                    <input
                      type="date"
                      value={list.assigned_date}
                      onChange={(event) => onUpdateWordListMeta(list.id, "assigned_date", event.target.value)}
                    />
                  </label>
                </div>

                <p className="field-note">
                  绑定到 {resolveChildLabel(String(list.family_id), String(list.assignee_id))} / {list.assigned_date}
                </p>

                <div className="word-items">
                  {list.items.map((item, index) => (
                    <div className="word-item-row" key={item.id}>
                      <span className="word-item-index">{index + 1}</span>
                      <input
                        value={item.text}
                        onChange={(event) => onUpdateWordItem(list.id, item.id, event.target.value)}
                        placeholder="输入词项"
                      />
                      <button className="inline-link danger" type="button" onClick={() => onRemoveWordItem(list.id, item.id)}>
                        删除
                      </button>
                    </div>
                  ))}
                </div>

                <button className="ghost-button compact" type="button" onClick={() => onAddWordItem(list.id)}>
                  新增词项
                </button>
              </article>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

export default function App() {
  const initialProfile = CHILD_PROFILES[0]
  const [apiBaseUrl, setApiBaseUrl] = useState(DEFAULT_API_BASE_URL)
  const [selectedChildPreset, setSelectedChildPreset] = useState(initialProfile.id)
  const [familyId, setFamilyId] = useState(initialProfile.familyId)
  const [assigneeId, setAssigneeId] = useState(initialProfile.assigneeId)
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
  const [weeklyStats, setWeeklyStats] = useState(null)
  const [weeklyStatus, setWeeklyStatus] = useState(null)
  const [reportView, setReportView] = useState("week")
  const [isLoadingWeekly, setIsLoadingWeekly] = useState(false)
  const [monthlyRows, setMonthlyRows] = useState([])
  const [monthlyStatus, setMonthlyStatus] = useState(null)
  const [isLoadingMonthly, setIsLoadingMonthly] = useState(false)
  const [pointEntries, setPointEntries] = useState([])
  const [estimatedBalance, setEstimatedBalance] = useState(null)
  const [pointForm, setPointForm] = useState(() => ({
    mode: "reward",
    amount: "5",
    reason: "",
    assignedDate: formatDateInputValue(),
  }))
  const [pointsStatus, setPointsStatus] = useState(null)
  const [isUpdatingPoints, setIsUpdatingPoints] = useState(false)
  const [wordLists, setWordLists] = useState([])
  const [wordListDraft, setWordListDraft] = useState(() => ({
    title: "",
    language: "英语",
    assignedDate: formatDateInputValue(),
    itemsText: "",
  }))
  const [wordListStatus, setWordListStatus] = useState(null)

  const previewSections = parseSchoolTaskMessage(rawText)
  const previewTasks = flattenSectionsToTasks(previewSections)
  const diagnostics = buildTaskDiagnostics(draftTasks, todayTasks)
  const selectedDraftTasks = draftTasks.filter((task) => selectedTaskIds.includes(task.id))
  const riskyDraftTaskCount = draftTasks.filter((task) => {
    const riskMeta = getDraftTaskRiskMeta(task, diagnostics.byId[task.id] || { issues: [], hasBlocking: false })
    return riskMeta.order <= 1
  }).length
  const recommendedTaskCount = draftTasks.filter(
    (task) => !diagnostics.byId[task.id]?.hasBlocking && !task.needs_review && Number(task.confidence || 0) >= 0.7,
  ).length
  const matchedProfile = resolveChildProfile(familyId, assigneeId)
  const childLabel = matchedProfile ? matchedProfile.label : `自定义孩子 / ${assigneeId}`
  const scopedPointEntries = pointEntries
    .filter((entry) => String(entry.family_id) === familyId && String(entry.assignee_id) === assigneeId)
    .sort((left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime())
  const currentDatePointEntries = scopedPointEntries.filter((entry) => entry.assigned_date === assignedDate)
  const scopedWordLists = wordLists
    .filter((item) => String(item.family_id) === familyId && String(item.assignee_id) === assigneeId)
    .sort(
      (left, right) =>
        String(right.assigned_date || "").localeCompare(String(left.assigned_date || ""), "zh-CN") ||
        new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime(),
    )
  const dailyReport = buildDailyReport(todayTasks, currentDatePointEntries, assignedDate)

  async function refreshPoints() {
    if (!familyId.trim() || !assigneeId.trim()) {
      return
    }

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/points/ledger?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(
          assigneeId,
        )}`,
      )
      if (Array.isArray(data.entries)) {
        setPointEntries(data.entries.map(entry => ({
          ...entry,
          id: entry.entry_id,
          family_id: String(entry.family_id),
          assignee_id: String(entry.user_id),
          assigned_date: entry.occurred_on,
          amount: entry.delta,
          reason: entry.note,
          created_at: entry.created_at || new Date().toISOString()
        })))
      }
      if (data.points_balance) {
        setEstimatedBalance(data.points_balance.balance ?? null)
      }
    } catch (error) {
      console.error("Failed to refresh points:", error)
    }
  }

  async function refreshWordLists() {
    if (!familyId.trim() || !assigneeId.trim()) {
      return
    }

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/word-lists?family_id=${encodeURIComponent(familyId)}&child_id=${encodeURIComponent(
          assigneeId,
        )}`,
      )
      
      const rawLists = Array.isArray(data.word_lists) ? data.word_lists : (data.word_list ? [data.word_list] : [])
      
      setWordLists(rawLists.map(list => ({
        ...list,
        id: list.word_list_id,
        family_id: String(list.family_id),
        assignee_id: String(list.child_id),
        // Map language back for UI display if it matches en/zh
        language: list.language === "en" ? "英语" : (list.language === "zh" ? "语文" : list.language),
        items: (list.items || []).map(item => ({
          ...item,
          id: `item-${item.index}`
        }))
      })))
    } catch (error) {
      console.error("Failed to refresh word lists:", error)
    }
  }

  useEffect(() => {
    const profile = resolveChildProfile(familyId, assigneeId)
    setSelectedChildPreset(profile ? profile.id : "custom")
    void refreshPoints()
    void refreshWordLists()
  }, [familyId, assigneeId])

  useEffect(() => {
    setPointForm((current) => ({
      ...current,
      assignedDate,
    }))
    setWordListDraft((current) => ({
      ...current,
      assignedDate,
    }))
  }, [assignedDate])

  async function refreshTasks() {
    if (!familyId.trim() || !assigneeId.trim() || !assignedDate) {
      return
    }

    setIsRefreshing(true)
    setRefreshError("")

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/tasks?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(
          assigneeId,
        )}&date=${encodeURIComponent(assignedDate)}`,
      )
      setTodayTasks(Array.isArray(data.tasks) ? data.tasks : [])
      setTodayDate(data.date || assignedDate)
      // Also refresh balance for the specific date if needed, but for now we refresh global ledger
    } catch (requestError) {
      setRefreshError(requestError.message)
    } finally {
      setIsRefreshing(false)
    }
  }

  useEffect(() => {
    void refreshTasks()
  }, [apiBaseUrl, familyId, assigneeId, assignedDate])

  function handleChildPresetChange(nextId) {
    setSelectedChildPreset(nextId)
    const nextProfile = CHILD_PROFILES.find((item) => item.id === nextId)
    if (nextProfile) {
      setFamilyId(nextProfile.familyId)
      setAssigneeId(nextProfile.assigneeId)
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
      notes: ["这是手动补充任务，请确认标题后再发布。"],
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
    setSelectedTaskIds(draftTasks.filter((task) => !(diagnostics.byId[task.id]?.hasBlocking)).map((task) => task.id))
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
        message: "请先选择任务日期，明确本次任务要发布到哪一天。",
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
          message: `已识别 ${data.parsed_count || parsedTasks.length} 个建议任务，目标日期 ${assignedDate}，请先处理风险项再确认发布。`,
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
        title: "无法开始发布",
        message: "请先选择任务日期，明确本次任务要发布到哪一天。",
        retryable: false,
      })
      return
    }

    if (selectedDraftTasks.length === 0) {
      setCreateStatus({
        tone: "error",
        title: "无法开始发布",
        message: "请至少选择一条建议任务再确认发布。",
        retryable: false,
      })
      return
    }

    const blockingTasks = selectedDraftTasks.filter((task) => diagnostics.byId[task.id]?.hasBlocking)
    if (blockingTasks.length > 0) {
      setCreateStatus({
        tone: "error",
        title: "当前选中项含阻断风险",
        message: `当前选中的任务里有 ${blockingTasks.length} 条存在重复或标题为空，请先修改后再确认发布。`,
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
        title: "当前选中项无法发布",
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
        title: "任务已发布",
        message: `已确认并发布 ${data.created_count || sanitizedTasks.length} 个任务到 ${assignedDate}。`,
        retryable: false,
      })
      await refreshTasks()
    } catch (requestError) {
      setCreateStatus({
        tone: "error",
        title: "发布失败，草稿与选中项已保留",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsConfirming(false)
    }
  }

  async function loadWeeklyStats() {
    setReportView("week")
    setWeeklyStatus(null)
    setIsLoadingWeekly(true)

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/stats/weekly?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(
          assigneeId,
        )}&end_date=${encodeURIComponent(assignedDate)}`,
      )

      setWeeklyStats(data)
      if (Array.isArray(data.raw_stats) && data.raw_stats.length > 0) {
        setWeeklyStatus({
          tone: "success",
          title: "周趋势已刷新",
          message: `已按 ${assignedDate} 为锚点加载近 7 天统计。`,
          retryable: false,
        })
      } else {
        setWeeklyStatus({
          tone: "warning",
          title: "近 7 天暂无可用数据",
          message: "weekly 接口已响应，但还没有历史任务数据可用于图表展示。",
          retryable: false,
        })
      }
    } catch (requestError) {
      setWeeklyStatus({
        tone: "error",
        title: "周趋势加载失败",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsLoadingWeekly(false)
    }
  }

  async function loadMonthlyTrend() {
    setReportView("month")
    setMonthlyStatus(null)
    setIsLoadingMonthly(true)

    try {
      const targetMonth = assignedDate.slice(0, 7) // YYYY-MM
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/stats/monthly?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(
          assigneeId,
        )}&month=${encodeURIComponent(targetMonth)}`,
      )

      // Map API monthly stats to the UI's monthlyRows format
      // API returns CompletionSeries, PointsSeries, WordSeries indexed by Label (week_1, etc.)
      const rows = (data.completion_series || []).map((point, index) => {
        const pPoint = data.points_series?.[index] || {}
        return {
          label: point.label,
          total: point.total_tasks,
          completed: point.completed_tasks,
          ratio: point.completion_rate,
          pointDelta: pPoint.delta,
        }
      })

      setMonthlyRows(rows)
      setMonthlyStatus({
        tone: rows.length > 0 ? "success" : "warning",
        title: rows.length > 0 ? "月趋势已刷新" : `近 ${targetMonth} 暂无任务数据`,
        message: rows.length > 0
          ? `已加载 ${targetMonth} 的月度聚合分析。`
          : "可以先发布或同步更多日期的任务，再回来查看月视图。",
        retryable: false,
      })
    } catch (requestError) {
      setMonthlyStatus({
        tone: "error",
        title: "月趋势加载失败",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsLoadingMonthly(false)
    }
  }

  function updatePointForm(field, value) {
    setPointForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  function applyQuickAmount(amount) {
    setPointForm((current) => ({
      ...current,
      amount: String(amount),
    }))
  }

  async function runPointUpdate() {
    setPointsStatus(null)

    const amount = Math.abs(Number(pointForm.amount || 0))
    if (amount <= 0) {
      setPointsStatus({
        tone: "error",
        title: "无法提交积分",
        message: "请填写大于 0 的积分分值。",
        retryable: false,
      })
      return
    }

    if (!pointForm.reason.trim()) {
      setPointsStatus({
        tone: "error",
        title: "无法提交积分",
        message: "请填写表扬或批评原因。",
        retryable: false,
      })
      return
    }

    const signedAmount = pointForm.mode === "penalty" ? -amount : amount

    setIsUpdatingPoints(true)
    try {
      const data = await requestJSON(`${apiBaseUrl}/api/v1/points/update`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: Number(assigneeId),
          family_id: Number(familyId),
          amount: signedAmount,
          reason: pointForm.reason.trim(),
        }),
      })

      setEstimatedBalance(data.balance ?? null)
      setPointsStatus({
        tone: "success",
        title: "积分已更新",
        message: `已同步更新当前孩子的积分记录。`,
        retryable: false,
      })
      setPointForm((current) => ({
        ...current,
        reason: "",
      }))
      void refreshPoints()
    } catch (requestError) {
      setPointsStatus({
        tone: "error",
        title: "积分提交失败",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsUpdatingPoints(false)
    }
  }

  function updateWordListDraft(field, value) {
    setWordListDraft((current) => ({
      ...current,
      [field]: value,
    }))
  }

  function useAssignedDateForWordList() {
    setWordListDraft((current) => ({
      ...current,
      assignedDate,
    }))
  }

  async function createWordListDraft() {
    setWordListStatus(null)
    const items = wordListDraft.itemsText
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean)

    if (!wordListDraft.title.trim()) {
      setWordListStatus({
        tone: "error",
        title: "无法创建清单",
        message: "请先填写清单名。",
        retryable: false,
      })
      return
    }

    if (items.length === 0) {
      setWordListStatus({
        tone: "error",
        title: "无法创建清单",
        message: "请至少录入一个词项。",
        retryable: false,
      })
      return
    }

    const langMap = { "英语": "en", "语文": "zh" }
    const language = langMap[wordListDraft.language] || "en"

    setIsSubmitting(true)
    try {
      await requestJSON(`${apiBaseUrl}/api/v1/word-lists`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          family_id: Number(familyId),
          child_id: Number(assigneeId),
          assigned_date: wordListDraft.assignedDate || assignedDate,
          title: wordListDraft.title.trim(),
          language,
          items: items.map(text => ({ text }))
        }),
      })

      setWordListStatus({
        tone: "success",
        title: "单词清单已创建",
        message: `已为 ${resolveChildLabel(familyId, assigneeId)} 同步保存清单到服务器。`,
        retryable: false,
      })
      setWordListDraft({
        title: "",
        language: wordListDraft.language,
        assignedDate: assignedDate,
        itemsText: "",
      })
      void refreshWordLists()
    } catch (requestError) {
      setWordListStatus({
        tone: "error",
        title: "创建清单失败",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  async function syncWordList(list) {
    const langMap = { "英语": "en", "语文": "zh" }
    const language = langMap[list.language] || list.language || "en"

    try {
      await requestJSON(`${apiBaseUrl}/api/v1/word-lists`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          family_id: Number(list.family_id),
          child_id: Number(list.assignee_id),
          assigned_date: list.assigned_date,
          title: list.title,
          language,
          items: (list.items || []).map((item) => ({
            text: item.text,
            meaning: item.meaning || "",
            hint: item.hint || "",
          })),
        }),
      })
      void refreshWordLists()
    } catch (error) {
      console.error("Failed to sync word list:", error)
      setWordListStatus({
        tone: "error",
        title: "同步失败",
        message: error.message,
        retryable: true,
      })
    }
  }

  function updateWordListMeta(listId, field, value) {
    setWordLists((current) => {
      const next = current.map((list) =>
        list.id === listId
          ? {
              ...list,
              [field]: value,
              updated_at: new Date().toISOString(),
            }
          : list,
      )
      const updatedList = next.find((l) => l.id === listId)
      if (updatedList) {
        void syncWordList(updatedList)
      }
      return next
    })
  }

  function addWordItem(listId) {
    setWordLists((current) => {
      const next = current.map((list) =>
        list.id === listId
          ? {
              ...list,
              items: [...list.items, createWordItem("")],
              updated_at: new Date().toISOString(),
            }
          : list,
      )
      const updatedList = next.find((l) => l.id === listId)
      if (updatedList) {
        void syncWordList(updatedList)
      }
      return next
    })
  }

  function updateWordItem(listId, itemId, value) {
    setWordLists((current) => {
      const next = current.map((list) =>
        list.id === listId
          ? {
              ...list,
              items: list.items.map((item) => (item.id === itemId ? { ...item, text: value } : item)),
              updated_at: new Date().toISOString(),
            }
          : list,
      )
      const updatedList = next.find((l) => l.id === listId)
      if (updatedList) {
        void syncWordList(updatedList)
      }
      return next
    })
  }

  function removeWordItem(listId, itemId) {
    setWordLists((current) => {
      const next = current.map((list) =>
        list.id === listId
          ? {
              ...list,
              items: list.items.filter((item) => item.id !== itemId),
              updated_at: new Date().toISOString(),
            }
          : list,
      )
      const updatedList = next.find((l) => l.id === listId)
      if (updatedList) {
        void syncWordList(updatedList)
      }
      return next
    })
  }

  function removeWordList(listId) {
    setWordLists((current) => current.filter((list) => list.id !== listId))
  }

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">StudyClaw Parent Console</p>
          <h1>家长主控制台：发布任务、看反馈、调积分、配单词清单</h1>
          <p className="hero-copy">
            这版家长端把“按某一天发布作业”作为主路径，同时把每日反馈、手工积分和单词清单集中到同一页。任务草稿会继续先做本地结构预览，再调
            Go 后端的解析链路；`needs_review` 和低置信只高亮，不改字段语义。
          </p>
        </div>
        <div className="hero-stats">
          <StatCard label="当前孩子" value={childLabel} hint={`家庭 ${familyId} / 孩子 ${assigneeId}`} />
          <StatCard label="当日完成" value={`${dailyReport.completed}/${dailyReport.total}`} hint={todayDate || assignedDate} />
          <StatCard label="今日积分" value={formatSignedAmount(dailyReport.pointDelta)} hint="手工积分来自最近记录" />
          <StatCard label="单词清单" value={scopedWordLists.length} hint="当前孩子已绑定的清单数" />
        </div>
      </section>

      <ConsoleNav
        childLabel={childLabel}
        assignedDate={assignedDate}
        pointDelta={dailyReport.pointDelta}
        wordListCount={scopedWordLists.length}
      />

      <section className="workspace" id="publish-console">
        <form className="panel composer" onSubmit={handleSubmit}>
          <div className="panel-heading">
            <div>
              <h2>作业发布主路径</h2>
              <p className="panel-caption">选择孩子和日期，粘贴群消息，解析审核后再发布到当天任务板。</p>
            </div>
            <span>主控制区</span>
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
              <span>孩子</span>
              <select aria-label="孩子" value={selectedChildPreset} onChange={(event) => handleChildPresetChange(event.target.value)}>
                {CHILD_PROFILES.map((item) => (
                  <option value={item.id} key={item.id}>
                    {item.label}
                  </option>
                ))}
                <option value="custom">自定义 ID</option>
              </select>
            </label>
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
              <input aria-label="任务日期" type="date" value={assignedDate} onChange={(event) => setAssignedDate(event.target.value)} />
            </label>
          </div>

          <p className="field-note">
            当前解析、发布和任务板刷新都会使用 {assignedDate || "未选择日期"}。切换日期后，右侧反馈区也会按该日期查看结果。
          </p>

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
              {isRefreshing ? "刷新中..." : "刷新当日任务"}
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
          {refreshError ? <p className="inline-hint hint-error">刷新当日任务失败: {refreshError}</p> : null}
        </form>

        <section className="column-stack">
          <section className="panel">
            <div className="panel-heading">
              <div>
                <h2>1. 群消息结构预览</h2>
                <p className="panel-caption">本地即时拆分学科与子步骤，先确认原文结构是否可读。</p>
              </div>
              <span>本地即时解析</span>
            </div>
            <SectionPreview sections={previewSections} />
          </section>

          <ServerTaskList
            title="1. 本地结构任务预览"
            tasks={previewTasks}
            emptyText="等待识别群消息内容。"
            caption="仅展示原文结构，不代表已经写入后端。"
          />
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
        <ServerTaskList
          title="本次 API 已发布任务"
          tasks={createdTasks}
          emptyText="还没有提交到 API。"
          caption="发布成功后会立即展示本次写入的任务。"
        />
        <ServerTaskList
          title="指定日期任务板"
          tasks={todayTasks}
          emptyText="当前孩子在该日期还没有任务。"
          caption="来自 `/api/v1/tasks?family_id=...&user_id=...&date=...`。"
        />
      </section>

      <section className="workspace tertiary">
        <FeedbackPanel
          assignedDate={assignedDate}
          childLabel={childLabel}
          todayTasks={todayTasks}
          currentDatePointEntries={currentDatePointEntries}
          weeklyStats={weeklyStats}
          weeklyStatus={weeklyStatus}
          monthlyRows={monthlyRows}
          monthlyStatus={monthlyStatus}
          reportView={reportView}
          onLoadWeekly={() => void loadWeeklyStats()}
          onLoadMonthly={() => void loadMonthlyTrend()}
          isLoadingWeekly={isLoadingWeekly}
          isLoadingMonthly={isLoadingMonthly}
        />
        <PointsPanel
          familyId={familyId}
          assigneeId={assigneeId}
          assignedDate={assignedDate}
          pointForm={pointForm}
          onPointFormChange={updatePointForm}
          onQuickAmount={applyQuickAmount}
          onSubmit={() => void runPointUpdate()}
          onRetry={() => void runPointUpdate()}
          isSubmitting={isUpdatingPoints}
          pointsStatus={pointsStatus}
          estimatedBalance={estimatedBalance}
          todayPointEntries={currentDatePointEntries}
          recentPointEntries={scopedPointEntries.slice(0, 6)}
        />
      </section>

      <section className="workspace tertiary">
        <WordListPanel
          childLabel={childLabel}
          familyId={familyId}
          assigneeId={assigneeId}
          assignedDate={assignedDate}
          wordListDraft={wordListDraft}
          wordLists={scopedWordLists}
          wordListStatus={wordListStatus}
          onDraftChange={updateWordListDraft}
          onUseAssignedDate={useAssignedDateForWordList}
          onCreate={createWordListDraft}
          onRetry={createWordListDraft}
          onUpdateWordListMeta={updateWordListMeta}
          onAddWordItem={addWordItem}
          onUpdateWordItem={updateWordItem}
          onRemoveWordItem={removeWordItem}
          onRemoveWordList={removeWordList}
        />
        <section className="panel notes">
          <div className="panel-heading">
            <div>
              <h2>当前冻结说明</h2>
              <p className="panel-caption">避免前端重新定义后端语义，同时给 SC-05 提供演示注意点。</p>
            </div>
            <span>联调说明</span>
          </div>
          <ul className="rule-list">
            <li>风险高亮只消费现有 `needs_review`、`confidence` 和任务去重诊断，不改后端字段含义。</li>
            <li>{"发布链路固定为 `parse -> 审核编辑 -> confirm -> 按日期刷新 tasks`。"}</li>
            <li>周反馈走真实 `/api/v1/stats/weekly`；日报摘要由前端基于当日任务和积分记录确定性生成。</li>
            <li>积分操作走真实 `/api/v1/points/update`；最近明细暂存在浏览器，便于家长演示加分 / 扣分原因。</li>
            <li>单词清单当前为家长端本地持久化配置，接口冻结前不假设后端 `word_list` 字段结构。</li>
            <li>失败重试保持现有体验：解析失败保留原文和上一轮草稿，发布失败保留草稿与选中项，积分失败保留表单。</li>
          </ul>
        </section>
      </section>

      <section className="panel notes">
        <div className="panel-heading">
          <div>
            <h2>支持的录入规则</h2>
            <p className="panel-caption">面向学校群常见格式，保持现有解析能力与演示稳定性。</p>
          </div>
          <span>录入说明</span>
        </div>
        <ul className="rule-list">
          <li>支持 `数学3.6：`、`英：`、`语文：` 这种带日期或简称的学科标题。</li>
          <li>支持 `1、`、`1.` 作为主任务编号。</li>
          <li>支持 `（1）`、`（2）`、`（3）` 作为同一任务下的补充步骤。</li>
          <li>提交后调用 `/api/v1/tasks/parse`，审核后调用 `/api/v1/tasks/confirm`，并按当前日期刷新 `/api/v1/tasks`。</li>
          <li>“查看周趋势”调用 `/api/v1/stats/weekly`；“查看月趋势”在前端按 28 天任务板聚合。</li>
          <li>积分通过 `/api/v1/points/update` 提交，单词清单当前在家长端本地持久化。</li>
        </ul>
      </section>
    </main>
  )
}

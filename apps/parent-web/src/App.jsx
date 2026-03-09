import { useEffect, useState } from "react"
import { flattenSectionsToTasks, parseSchoolTaskMessage, REFERENCE_GROUP_MESSAGE } from "./schoolTaskParser"

const DEFAULT_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
let draftTaskCounter = 0

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

function DraftTaskList({
  tasks,
  selectedTaskIds,
  diagnosticsById,
  onToggle,
  onFieldChange,
  onRemove,
  onSelectHighConfidence,
  onSelectCleanTasks,
  onSelectAll,
  onClearSelection,
  onAddManualTask,
}) {
  const highConfidenceCount = tasks.filter((task) => !task.needs_review && Number(task.confidence || 0) >= 0.7).length

  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>AI 建议任务</h3>
        <span>{tasks.length} 条</span>
      </div>
      <div className="draft-toolbar">
        <button className="ghost-button compact" type="button" onClick={onSelectHighConfidence}>
          全选高置信 ({highConfidenceCount})
        </button>
        <button className="ghost-button compact" type="button" onClick={onSelectCleanTasks}>
          全选无冲突
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
          {tasks.map((task, index) => {
            const isSelected = selectedTaskIds.includes(task.id)
            const confidence = Number(task.confidence || 0)
            const notes = Array.isArray(task.notes) ? task.notes : []
            const confidenceMeta = getConfidenceMeta(confidence)
            const diagnostics = diagnosticsById[task.id] || { issues: [], hasBlocking: false }

            return (
              <article
                className={`draft-card ${task.needs_review ? "needs-review" : ""} ${diagnostics.hasBlocking ? "has-blocking" : ""}`}
                key={`${task.id}-${index}`}
              >
                <div className="draft-header">
                  <input type="checkbox" checked={isSelected} onChange={() => onToggle(task.id)} />
                  <span className="task-subject">{task.subject}</span>
                  <span className={`confidence-pill confidence-${confidenceMeta.tone}`}>{confidenceMeta.label}</span>
                  {task.needs_review ? <span className="review-pill">需确认</span> : null}
                  {task.source === "manual" ? <span className="review-pill manual-pill">手动补充</span> : null}
                  <button className="inline-link danger" type="button" onClick={() => onRemove(task.id)}>
                    删除
                  </button>
                </div>

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

function QualityPanel({ diagnosticsSummary, selectedTasks, diagnosticsById }) {
  const selectedBlockingCount = selectedTasks.filter((task) => diagnosticsById[task.id]?.hasBlocking).length
  const selectedWarningCount = selectedTasks.filter(
    (task) => !diagnosticsById[task.id]?.hasBlocking && (diagnosticsById[task.id]?.issues.length || 0) > 0,
  ).length

  return (
    <section className="panel">
      <div className="panel-heading">
        <h3>创建前质量检查</h3>
        <span>本地诊断</span>
      </div>

      <div className="analysis-grid">
        <div className="analysis-card">
          <span>阻断项任务</span>
          <strong>{diagnosticsSummary.blockTasks}</strong>
        </div>
        <div className="analysis-card">
          <span>提醒项任务</span>
          <strong>{diagnosticsSummary.warningTasks}</strong>
        </div>
        <div className="analysis-card">
          <span>无冲突任务</span>
          <strong>{diagnosticsSummary.cleanTasks}</strong>
        </div>
      </div>

      <ul className="rule-list compact">
        <li>已选任务中有 {selectedBlockingCount} 条存在阻断项，不能直接创建。</li>
        <li>已选任务中有 {selectedWarningCount} 条存在提醒项，可以创建，但建议先人工确认。</li>
        <li>精确重复会被拦截，高相似任务仅提醒，不强制阻断。</li>
      </ul>
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
  const [rawText, setRawText] = useState(REFERENCE_GROUP_MESSAGE)
  const [draftTasks, setDraftTasks] = useState([])
  const [selectedTaskIds, setSelectedTaskIds] = useState([])
  const [createdTasks, setCreatedTasks] = useState([])
  const [todayTasks, setTodayTasks] = useState([])
  const [todayDate, setTodayDate] = useState("")
  const [error, setError] = useState("")
  const [message, setMessage] = useState("")
  const [parserMode, setParserMode] = useState("")
  const [analysis, setAnalysis] = useState(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isConfirming, setIsConfirming] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const previewSections = parseSchoolTaskMessage(rawText)
  const previewTasks = flattenSectionsToTasks(previewSections)
  const completedCount = todayTasks.filter((task) => task.completed).length
  const diagnostics = buildTaskDiagnostics(draftTasks, todayTasks)
  const selectedDraftTasks = draftTasks.filter((task) => selectedTaskIds.includes(task.id))

  async function refreshTasks() {
    if (!familyId || !assigneeId) {
      return
    }

    setIsRefreshing(true)
    setError("")

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/tasks?family_id=${encodeURIComponent(familyId)}&user_id=${encodeURIComponent(assigneeId)}`,
      )
      setTodayTasks(Array.isArray(data.tasks) ? data.tasks : [])
      setTodayDate(data.date || "")
    } catch (requestError) {
      setError(requestError.message)
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

  function selectHighConfidenceTasks() {
    setSelectedTaskIds(
      draftTasks.filter((task) => !task.needs_review && Number(task.confidence || 0) >= 0.7).map((task) => task.id),
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

  async function handleSubmit(event) {
    event.preventDefault()
    setError("")
    setMessage("")

    if (!rawText.trim()) {
      setError("请先粘贴学校群任务内容。")
      return
    }

    setIsSubmitting(true)

    try {
      const payload = {
        family_id: Number(familyId),
        assignee_id: Number(assigneeId),
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
      setMessage(`AI 已识别 ${data.parsed_count || 0} 个建议任务，请审核后确认创建。`)
    } catch (requestError) {
      setError(requestError.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleConfirmCreate() {
    setError("")
    setMessage("")

    if (selectedDraftTasks.length === 0) {
      setError("请至少选择一条建议任务再确认创建。")
      return
    }

    setIsConfirming(true)

    try {
      const blockingTasks = selectedDraftTasks.filter((task) => diagnostics.byId[task.id]?.hasBlocking)
      if (blockingTasks.length > 0) {
        throw new Error(`当前选中的任务里有 ${blockingTasks.length} 条存在重复或标题为空，请先修改后再确认创建。`)
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
        throw new Error("选中的任务标题不能为空。")
      }

      const data = await requestJSON(`${apiBaseUrl}/api/v1/tasks/confirm`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          family_id: Number(familyId),
          assignee_id: Number(assigneeId),
          tasks: sanitizedTasks,
        }),
      })

      setCreatedTasks(Array.isArray(data.tasks) ? data.tasks : [])
      setDraftTasks((current) => current.filter((task) => !selectedTaskIds.includes(task.id)))
      setSelectedTaskIds([])
      setMessage(`已确认并创建 ${data.created_count || 0} 个任务。`)
      await refreshTasks()
    } catch (requestError) {
      setError(requestError.message)
    } finally {
      setIsConfirming(false)
    }
  }

  useEffect(() => {
    refreshTasks()
  }, [])

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">StudyClaw Parent Console</p>
          <h1>学校群任务一键转为孩子今日清单</h1>
          <p className="hero-copy">
            支持你刚给出的这种格式: 按学科分段、按序号列主任务、用“（1）（2）（3）”补充子步骤。页面会先做本地结构预览，再调 API
            调用 Agent Core 的 AI 混合解析链路，给出建议任务、置信度与风险提示，再由家长确认创建。
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
          </div>

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
            <button className="primary-button secondary" type="button" disabled={isConfirming} onClick={handleConfirmCreate}>
              {isConfirming ? "创建中..." : `确认创建选中任务 (${selectedDraftTasks.length})`}
            </button>
          </div>

          {message ? <p className="status success">{message}</p> : null}
          {error ? <p className="status error">{error}</p> : null}
        </form>

        <section className="column-stack">
          <section className="panel">
            <div className="panel-heading">
              <h2>群消息结构预览</h2>
              <span>本地即时解析</span>
            </div>
            <SectionPreview sections={previewSections} />
          </section>

          <ServerTaskList title="本地结构预览" tasks={previewTasks} emptyText="等待识别群消息内容。" />
          <DraftTaskList
            tasks={draftTasks}
            selectedTaskIds={selectedTaskIds}
            diagnosticsById={diagnostics.byId}
            onToggle={toggleSelectedTask}
            onFieldChange={updateDraftTask}
            onRemove={removeDraftTask}
            onSelectHighConfidence={selectHighConfidenceTasks}
            onSelectCleanTasks={selectCleanTasks}
            onSelectAll={selectAllTasks}
            onClearSelection={clearTaskSelection}
            onAddManualTask={addManualTask}
          />
          <QualityPanel diagnosticsSummary={diagnostics.summary} selectedTasks={selectedDraftTasks} diagnosticsById={diagnostics.byId} />
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
          <li>提交后调用 `/api/v1/tasks/parse`，并自动刷新 `/api/v1/tasks` 结果。</li>
        </ul>
      </section>
    </main>
  )
}

import React, { useEffect, useRef, useState } from "react"
import { flattenSectionsToTasks, parseSchoolTaskMessage, REFERENCE_GROUP_MESSAGE } from "./schoolTaskParser"

const DEFAULT_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
const CHILD_PROFILES = [
  { id: "mia-grade4", label: "苗苗 / 四年级", familyId: "101", assigneeId: "201" },
  { id: "leo-grade2", label: "乐乐 / 二年级", familyId: "102", assigneeId: "202" },
  { id: "yoyo-grade6", label: "悠悠 / 六年级", familyId: "103", assigneeId: "203" },
  { id: "dictation-demo", label: "听写联调 / 701-801", familyId: "701", assigneeId: "801" },
]
const CONSOLE_SECTIONS = [
  { id: "publish-console", label: "发布作业", shortLabel: "发布" },
  { id: "report-console", label: "查看反馈", shortLabel: "反馈" },
  { id: "points-console", label: "积分操作", shortLabel: "积分" },
  { id: "word-console", label: "单词清单", shortLabel: "单词" },
]
const CONSOLE_PANEL_ORDER = ["publish-console", "report-console", "points-console", "word-console"]
const WORD_LIST_SYNC_DEBOUNCE_MS = 700
const PAGE_TRANSITION_MS = 280
const FEEDBACK_PANEL_ORDER = ["daily", "dictation", "trend"]
const POINTS_PANEL_ORDER = ["compose", "ledger"]
const WORD_PANEL_ORDER = ["create", "lists"]
const PUBLISH_PANEL_ORDER = ["scope", "compose", "review", "release", "split", "preview", "analysis", "board"]
const EMPTY_STATE_PUBLISH_FALLBACK_PANELS = ["review", "release"]
const POINT_REASON_PRESETS = {
  reward: ["按时完成全部任务", "主动完成额外练习", "主动整理错题", "晚饭前独立完成作业"],
  penalty: ["回家后拖延未开工", "未整理错题", "多次提醒后才完成", "作业完成后未复盘"],
}
const DICTATION_WORKER_STAGE_META = {
  queued: {
    label: "已入队",
    description: "后台已经接收拍照请求，正在等待异步 worker 接手。",
  },
  processing: {
    label: "处理中",
    description: "worker 已启动，正在推进听写批改链路。",
  },
  loading_word_list: {
    label: "装载词单",
    description: "后台正在加载正确答案清单与会话上下文。",
  },
  llm_grading: {
    label: "LLM 批改中",
    description: "模型正在解析拍照内容并比对正确答案。",
  },
  completed: {
    label: "已写回结果",
    description: "最终批改结果已成功写回会话，可结合 grading_id 检索日志。",
  },
  mark_processing_failed: {
    label: "状态写回失败",
    description: "worker 启动后未能成功写入 processing 状态。",
  },
  load_word_list_failed: {
    label: "词单加载失败",
    description: "后台读取词单或会话数据时失败。",
  },
  llm_grading_failed: {
    label: "LLM 批改失败",
    description: "模型调用失败，或模型返回的结果无法解析。",
  },
  persist_result_failed: {
    label: "结果持久化失败",
    description: "模型已返回结果，但服务端写回会话时失败。",
  },
}

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

function usePageTransition(activeId, orderedIds) {
  const orderKey = orderedIds.join("|")
  const [motionState, setMotionState] = useState(() => ({
    currentId: activeId,
    previousId: "",
    direction: "forward",
  }))

  useEffect(() => {
    if (activeId === motionState.currentId) {
      return undefined
    }

    const currentIndex = orderedIds.indexOf(motionState.currentId)
    const nextIndex = orderedIds.indexOf(activeId)
    const direction = currentIndex === -1 || nextIndex === -1 || nextIndex >= currentIndex ? "forward" : "backward"

    setMotionState({
      currentId: activeId,
      previousId: motionState.currentId,
      direction,
    })

    const timerId = window.setTimeout(() => {
      setMotionState((current) =>
        current.currentId === activeId
          ? {
            ...current,
            previousId: "",
          }
          : current,
      )
    }, PAGE_TRANSITION_MS)

    return () => {
      window.clearTimeout(timerId)
    }
  }, [activeId, motionState.currentId, orderKey, orderedIds])

  function getPageClass(pageId) {
    if (pageId === motionState.currentId && motionState.previousId) {
      return `screen-subpanel is-active is-enter dir-${motionState.direction}`.trim()
    }

    if (pageId === motionState.currentId) {
      return "screen-subpanel is-active"
    }

    if (pageId === motionState.previousId) {
      return `screen-subpanel is-previous is-exit dir-${motionState.direction}`.trim()
    }

    return "screen-subpanel"
  }

  return {
    direction: motionState.direction,
    isTransitioning: Boolean(motionState.previousId),
    getPageClass,
  }
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

function formatTimestampLabel(value) {
  if (!value) {
    return "--"
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }

  return date.toLocaleString("zh-CN", {
    hour12: false,
  })
}

function formatTimeOnlyLabel(value) {
  if (!value) {
    return "--"
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }

  return date.toLocaleTimeString("zh-CN", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
  })
}

function formatByteSize(value) {
  const bytes = Number(value || 0)
  if (!bytes) {
    return "--"
  }
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function buildDictationLogKeywords(session) {
  const keywords = [
    ...(Array.isArray(session?.debug_context?.log_keywords) ? session.debug_context.log_keywords : []),
    session?.session_id ? `session_id=${session.session_id}` : null,
    session?.word_list_id ? `word_list_id=${session.word_list_id}` : null,
    session?.grading_result?.grading_id ? `grading_id=${session.grading_result.grading_id}` : null,
    session?.debug_context?.photo_sha1 ? `photo_sha1=${session.debug_context.photo_sha1}` : null,
  ].filter(Boolean)

  return [...new Set(keywords.map((item) => String(item).trim()).filter(Boolean))]
}

function getDictationWorkerStageMeta(session) {
  const stage = String(session?.debug_context?.worker_stage || "").trim()
  if (stage && DICTATION_WORKER_STAGE_META[stage]) {
    return DICTATION_WORKER_STAGE_META[stage]
  }

  switch (session?.grading_status) {
    case "pending":
      return DICTATION_WORKER_STAGE_META.queued
    case "processing":
      return DICTATION_WORKER_STAGE_META.processing
    case "completed":
      return DICTATION_WORKER_STAGE_META.completed
    case "failed":
      return {
        label: "失败待定位",
        description: "会话已失败，但当前服务端尚未返回更具体的失败阶段。",
      }
    default:
      return {
        label: "待回传",
        description: "当前还没有可用的排障元信息，先用会话 ID 检索日志。",
      }
  }
}

function buildDictationDebugCopyText(session) {
  const stageMeta = getDictationWorkerStageMeta(session)
  const logKeywords = buildDictationLogKeywords(session)

  return [
    `session_id: ${session?.session_id || "--"}`,
    `word_list_id: ${session?.word_list_id || "--"}`,
    `grading_id: ${session?.grading_result?.grading_id || "--"}`,
    `grading_status: ${session?.grading_status || "--"}`,
    `worker_stage: ${session?.debug_context?.worker_stage || "--"} (${stageMeta.label})`,
    `log_file: ${session?.debug_context?.log_file || "api-server-YYYY-MM-DD.log"}`,
    `photo_sha1: ${session?.debug_context?.photo_sha1 || "--"}`,
    `photo_bytes: ${session?.debug_context?.photo_bytes || 0}`,
    `language: ${session?.debug_context?.language || "--"}`,
    `mode: ${session?.debug_context?.mode || "--"}`,
    `log_keywords: ${logKeywords.length > 0 ? logKeywords.join(", ") : "--"}`,
  ].join("\n")
}

function formatDictationTimelineLabel(index) {
  if (index === 0) {
    return "最新上传"
  }
  if (index === 1) {
    return "上一次上传"
  }
  return `历史上传 ${index + 1}`
}

function summarizeDictationTimelineSession(session) {
  const stageMeta = getDictationWorkerStageMeta(session)

  if (session?.grading_status === "failed") {
    return session?.grading_error || stageMeta.description
  }
  if (session?.grading_result?.ai_feedback) {
    return session.grading_result.ai_feedback
  }
  return stageMeta.description
}

function getDictationStatusMeta(session) {
  switch (session?.grading_status) {
    case "pending":
      return {
        label: "已排队",
        tone: "warn",
        description: "Pad 端照片已上传，正在等待后台批改。",
      }
    case "processing":
      return {
        label: "处理中",
        tone: "high",
        description: "后台正在解析拍照内容，并与正确答案做异步比对。",
      }
    case "completed": {
      const incorrectCount = Number(session?.grading_result?.incorrect_count || 0)
      const score = Number(session?.grading_result?.score || 0)
      if (incorrectCount === 0) {
        return {
          label: "已完成",
          tone: "ready",
          description: `本次得分 ${score} 分，当前未识别出需订正项。`,
        }
      }

      return {
        label: "需订正",
        tone: "high",
        description: `本次得分 ${score} 分，发现 ${incorrectCount} 项需订正。`,
      }
    }
    case "failed":
      return {
        label: "批改失败",
        tone: "block",
        description: session?.grading_error || "后台未返回可用批改结果。",
      }
    default:
      return {
        label: "未开始",
        tone: "ready",
        description: "当前日期还没有提交听写拍照。",
      }
  }
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

function normalizeDictationSession(session) {
  const gradedItems = Array.isArray(session?.grading_result?.graded_items) ? session.grading_result.graded_items : []
  const incorrectCount = gradedItems.filter((item) => !item.is_correct || item.needs_correction).length
  const debugContext = session?.debug_context

  return {
    ...session,
    id: session?.session_id || createLocalId("dictation-session"),
    family_id: String(session?.family_id ?? ""),
    assignee_id: String(session?.child_id ?? ""),
    total_items: Number(session?.total_items || 0),
    current_index: Number(session?.current_index || 0),
    played_count: Number(session?.played_count || 0),
    completed_items: Number(session?.completed_items || 0),
    grading_status: session?.grading_status || "idle",
    grading_error: session?.grading_error || "",
    debug_context: debugContext
      ? {
          ...debugContext,
          photo_bytes: Number(debugContext.photo_bytes || 0),
          log_keywords: Array.isArray(debugContext.log_keywords) ? debugContext.log_keywords.filter(Boolean) : [],
        }
      : null,
    grading_result: session?.grading_result
      ? {
          ...session.grading_result,
          score: Number(session.grading_result.score || 0),
          graded_items: gradedItems,
          incorrect_count: incorrectCount,
        }
      : null,
  }
}

function buildDictationSummary(dictationSessions) {
  const total = dictationSessions.length
  const pending = dictationSessions.filter((session) => ["pending", "processing"].includes(session.grading_status)).length
  const completed = dictationSessions.filter((session) => session.grading_status === "completed").length
  const failed = dictationSessions.filter((session) => session.grading_status === "failed").length
  const latestSession = [...dictationSessions].sort(
    (left, right) =>
      new Date(right.grading_completed_at || right.grading_requested_at || right.updated_at || 0).getTime() -
      new Date(left.grading_completed_at || left.grading_requested_at || left.updated_at || 0).getTime(),
  )[0] || null

  return {
    total,
    pending,
    completed,
    failed,
    latestSession,
  }
}

function createWordItem(text = "", meaning = "") {
  return {
    id: createLocalId("word-item"),
    text,
    meaning,
  }
}

function normalizeWordListFromApi(list = {}) {
  return {
    ...list,
    id: list.word_list_id || list.id || createLocalId("word-list"),
    family_id: String(list.family_id ?? ""),
    assignee_id: String(list.child_id ?? list.assignee_id ?? ""),
    language: list.language === "en" ? "英语" : list.language === "zh" ? "语文" : list.language,
    items: (list.items || []).map((item, index) => ({
      ...item,
      id: item.id || `item-${item.index || index + 1}`,
    })),
  }
}

function upsertWordListCollection(collection, nextList) {
  const nextIndex = collection.findIndex(
    (item) =>
      item.id === nextList.id ||
      (String(item.family_id) === String(nextList.family_id) &&
        String(item.assignee_id) === String(nextList.assignee_id) &&
        String(item.assigned_date) === String(nextList.assigned_date)),
  )

  if (nextIndex === -1) {
    return [...collection, nextList]
  }

  return collection.map((item, index) => (index === nextIndex ? nextList : item))
}

function AssignmentScopePanel({
  childProfiles,
  selectedChildPreset,
  onChildPresetChange,
  assignedDate,
  onAssignedDateChange,
  onShiftDate,
  onUseToday,
  childLabel,
  familyId,
  assigneeId,
  apiBaseUrl,
  onApiBaseUrlChange,
  onFamilyIdChange,
  onAssigneeIdChange,
  todayTaskCount,
  selectedDraftCount,
  pointDelta,
  dictationSummary,
}) {
  const latestDictationScore = dictationSummary.latestSession?.grading_result?.score

  return (
    <div className="scope-setup">
      <div className="panel-heading scope-heading">
        <div>
          <h3>先定孩子和日期</h3>
          <p className="panel-caption">手机上先锁定布置范围，再粘贴老师消息。这样后面的任务、反馈和听写结果都会跟着同一日期走。</p>
        </div>
        <span>{childLabel}</span>
      </div>

      <div className="scope-grid">
        <label>
          <span>孩子</span>
          <select aria-label="孩子" value={selectedChildPreset} onChange={(event) => onChildPresetChange(event.target.value)}>
            {childProfiles.map((item) => (
              <option value={item.id} key={item.id}>
                {item.label}
              </option>
            ))}
            <option value="custom">自定义 ID</option>
          </select>
        </label>
        <label>
          <span>任务日期</span>
          <input aria-label="任务日期" type="date" value={assignedDate} onChange={(event) => onAssignedDateChange(event.target.value)} />
        </label>
      </div>

      <div className="scope-shortcuts">
        <button className="ghost-button compact" type="button" onClick={() => onShiftDate(-1)}>
          前一天
        </button>
        <button className="ghost-button compact" type="button" onClick={onUseToday}>
          今天
        </button>
        <button className="ghost-button compact" type="button" onClick={() => onShiftDate(1)}>
          后一天
        </button>
      </div>

      <div className="scope-metrics">
        <article className="scope-metric">
          <span>当天任务</span>
          <strong>{todayTaskCount}</strong>
          <p>直接看该日期现有任务板。</p>
        </article>
        <article className="scope-metric">
          <span>待发布草稿</span>
          <strong>{selectedDraftCount}</strong>
          <p>勾选后就能进入确认发布。</p>
        </article>
        <article className="scope-metric">
          <span>今日积分</span>
          <strong>{formatSignedAmount(pointDelta)}</strong>
          <p>反馈区和积分区同步使用这一天。</p>
        </article>
        <article className="scope-metric">
          <span>听写状态</span>
          <strong>
            {dictationSummary.pending > 0
              ? `${dictationSummary.pending} 次处理中`
              : latestDictationScore
                ? `${latestDictationScore} 分`
                : "暂无"}
          </strong>
          <p>{dictationSummary.total > 0 ? "Pad 端拍照会在反馈区异步汇总。" : "当前日期还没有拍照批改记录。"}</p>
        </article>
      </div>

      <HelpAccordion
        className="scope-help"
        title="高级调试设置"
        caption="API 地址和自定义家庭 / 孩子 ID 收在这里，手机上默认不占主区域。"
        badge={selectedChildPreset === "custom" ? "自定义模式" : "按需展开"}
      >
        <div className="field-grid">
          <label>
            <span>API 地址</span>
            <input value={apiBaseUrl} onChange={(event) => onApiBaseUrlChange(event.target.value)} />
          </label>
          <label>
            <span>家庭 ID</span>
            <input value={familyId} onChange={(event) => onFamilyIdChange(event.target.value)} />
          </label>
          <label>
            <span>孩子 ID</span>
            <input value={assigneeId} onChange={(event) => onAssigneeIdChange(event.target.value)} />
          </label>
        </div>
      </HelpAccordion>
    </div>
  )
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

function ConsoleScreenLead({ label, title, caption, badge, isActive = false, onActivate }) {
  return (
    <button
      aria-current={isActive ? "page" : undefined}
      className={`console-screen-lead ${isActive ? "is-active" : ""}`.trim()}
      type="button"
      onClick={onActivate}
    >
      <div>
        <span className="console-screen-kicker">{label}</span>
        <strong>{title}</strong>
        <p>{caption}</p>
      </div>
      <div className="console-screen-side">
        <span className="console-screen-badge">{badge}</span>
        <span className="console-screen-cta">{isActive ? "当前主屏" : "点开进入"}</span>
      </div>
    </button>
  )
}

function ConsoleNav({ items, activeSection, onJump, childLabel, assignedDate, pointDelta, wordListCount }) {
  return (
    <nav className="console-nav" aria-label="家长主控制台导航">
      {items.map((item) => (
        <button
          className={`console-link ${activeSection === item.id ? "is-active" : ""}`}
          key={item.id}
          type="button"
          onClick={() => onJump(item.id)}
        >
          {item.label}
        </button>
      ))}
      <span className="console-context">
        当前孩子: {childLabel} / 日期: {assignedDate} / 今日积分 {formatSignedAmount(pointDelta)} / 清单 {wordListCount}
      </span>
    </nav>
  )
}

function PocketOverview({ childLabel, assignedDate, items, onJump }) {
  return (
    <section className="panel pocket-overview">
      <div className="panel-heading">
        <div>
          <h2>手机端家长工位</h2>
          <p className="panel-caption">把发布、反馈、积分、单词收成四个触达区，手机里直接跳转，不用整页来回找。</p>
        </div>
        <span>
          {childLabel} / {assignedDate}
        </span>
      </div>

      <div className="pocket-grid">
        {items.map((item) => (
          <button
            className={`pocket-card tone-${item.tone || "calm"}`}
            key={item.id}
            type="button"
            onClick={() => onJump(item.id)}
          >
            <span className="pocket-label">{item.label}</span>
            <strong>{item.metric}</strong>
            <p>{item.caption}</p>
          </button>
        ))}
      </div>
    </section>
  )
}

function MobileDock({ items, activeSection, onJump }) {
  return (
    <nav className="mobile-dock" aria-label="手机端快捷导航">
      {items.map((item) => (
        <button
          className={`mobile-dock-button ${activeSection === item.id ? "is-active" : ""}`}
          key={item.id}
          type="button"
          onClick={() => onJump(item.id)}
        >
          <span className="mobile-dock-label">{item.shortLabel}</span>
          <span className="mobile-dock-meta">{item.metric}</span>
        </button>
      ))}
    </nav>
  )
}

function HelpAccordion({ title, caption, badge, children, className = "", defaultOpen = false }) {
  return (
    <details className={`panel help-panel ${className}`.trim()} open={defaultOpen}>
      <summary className="help-summary">
        <div>
          <strong>{title}</strong>
          <p>{caption}</p>
        </div>
        <span>{badge}</span>
      </summary>
      <div className="help-content">{children}</div>
    </details>
  )
}

function PublishStageStrip({ items, activeStage, onChange }) {
  return (
    <div className="publish-stage-strip" role="tablist" aria-label="发布主路径步骤">
      {items.map((item) => (
        <button
          aria-selected={activeStage === item.id}
          className={`publish-stage-button ${activeStage === item.id ? "is-active" : ""}`}
          key={item.id}
          role="tab"
          type="button"
          onClick={() => onChange(item.id)}
        >
          <span className="publish-stage-index">{item.index}</span>
          <div>
            <strong>{item.label}</strong>
            <p>{item.caption}</p>
          </div>
        </button>
      ))}
    </div>
  )
}

function SubMenuStrip({ items, activeItem, onChange, ariaLabel }) {
  return (
    <div className="sub-menu-strip" role="tablist" aria-label={ariaLabel}>
      {items.map((item) => (
        <button
          aria-selected={activeItem === item.id}
          className={`sub-menu-button ${activeItem === item.id ? "is-active" : ""}`}
          key={item.id}
          role="tab"
          type="button"
          onClick={() => onChange(item.id)}
        >
          <span className="sub-menu-label">{item.label}</span>
          <strong>{item.metric}</strong>
          {item.caption ? <p>{item.caption}</p> : null}
        </button>
      ))}
    </div>
  )
}

function FeedbackLensStrip({ items, activeSection, onChange }) {
  return (
    <div className="feedback-lens-strip" role="tablist" aria-label="反馈查看模式">
      {items.map((item) => (
        <button
          aria-selected={activeSection === item.id}
          className={`feedback-lens-card ${activeSection === item.id ? "is-active" : ""}`}
          key={item.id}
          role="tab"
          type="button"
          onClick={() => onChange(item.id)}
        >
          <span className="feedback-lens-label">{item.label}</span>
          <strong>{item.metric}</strong>
          <p>{item.caption}</p>
        </button>
      ))}
    </div>
  )
}

function ScreenSubpanelDeck({ activeId, orderedIds, className = "", children }) {
  const { direction, isTransitioning, getPageClass } = usePageTransition(activeId, orderedIds)

  return (
    <div
      className={`screen-subpanel-deck ${isTransitioning ? `is-transitioning dir-${direction}` : ""} ${className}`.trim()}
      data-active-panel={activeId}
    >
      {children(getPageClass)}
    </div>
  )
}

function FeedbackSnapshotGrid({ items }) {
  return (
    <div className="feedback-snapshot-grid">
      {items.map((item) => (
        <article className="feedback-snapshot-card" key={item.label}>
          <span>{item.label}</span>
          <strong>{item.value}</strong>
          <p>{item.caption}</p>
        </article>
      ))}
    </div>
  )
}

function FeedbackExpandableCard({ title, badge, summary, children, defaultOpen = false }) {
  const [isOpen, setIsOpen] = useState(defaultOpen)

  useEffect(() => {
    setIsOpen(defaultOpen)
  }, [defaultOpen, title, badge])

  return (
    <details className="feedback-expand-card" open={isOpen} onToggle={(event) => setIsOpen(event.currentTarget.open)}>
      <summary className="feedback-expand-summary">
        <div>
          <strong>{title}</strong>
          <p>{summary}</p>
        </div>
        <span>{badge}</span>
      </summary>
      <div className="feedback-expand-body">{children}</div>
    </details>
  )
}

function DictationTroubleshootCard({ session }) {
  const [copyState, setCopyState] = useState("idle")
  const stageMeta = getDictationWorkerStageMeta(session)
  const logKeywords = buildDictationLogKeywords(session)

  useEffect(() => {
    setCopyState("idle")
  }, [session?.session_id, session?.updated_at, session?.grading_status, session?.debug_context?.worker_stage])

  async function handleCopy() {
    const clipboard = globalThis.navigator?.clipboard
    if (!clipboard?.writeText) {
      setCopyState("unsupported")
      return
    }

    try {
      await clipboard.writeText(buildDictationDebugCopyText(session))
      setCopyState("copied")
    } catch {
      setCopyState("failed")
    }
  }

  let copyHint = "复制后可直接贴到终端或问题单中检索当天日志。"
  if (copyState === "unsupported") {
    copyHint = "当前浏览器环境不支持直接复制，可手动记录下方关键字。"
  } else if (copyState === "failed") {
    copyHint = "复制失败，可先手动记录关键字并到日志中检索。"
  } else if (copyState === "copied") {
    copyHint = "排障信息已复制，可直接用于检索服务端日志。"
  }

  return (
    <FeedbackExpandableCard
      title="排障定位"
      badge={stageMeta.label}
      summary={`${stageMeta.description} 优先搜索 ${session?.debug_context?.log_file || "当天日志"}。`}
      defaultOpen={session?.grading_status === "failed"}
    >
      <div className="dictation-debug-grid">
        <article className="dictation-debug-card">
          <span>日志文件</span>
          <strong>{session?.debug_context?.log_file || "api-server-YYYY-MM-DD.log"}</strong>
          <p>服务端每天一份日志，同时也会输出到后台控制台。</p>
        </article>
        <article className="dictation-debug-card">
          <span>Worker 阶段</span>
          <strong>{stageMeta.label}</strong>
          <p>{stageMeta.description}</p>
        </article>
        <article className="dictation-debug-card">
          <span>拍照摘要</span>
          <strong>{session?.debug_context?.photo_sha1 || "--"}</strong>
          <p>
            {formatByteSize(session?.debug_context?.photo_bytes)} / {session?.debug_context?.language || "未标记语言"} /{" "}
            {session?.debug_context?.mode || "未标记模式"}
          </p>
        </article>
      </div>

      {logKeywords.length > 0 ? (
        <div className="debug-chip-block">
          <span className="debug-chip-label">日志关键字</span>
          <div className="debug-chip-list">
            {logKeywords.map((item) => (
              <code className="debug-chip" key={item}>
                {item}
              </code>
            ))}
          </div>
        </div>
      ) : null}

      <ul className="rule-list compact">
        <li>会话 ID: {session?.session_id || "--"}</li>
        <li>单词清单 ID: {session?.word_list_id || "--"}</li>
        <li>批改 ID: {session?.grading_result?.grading_id || "--"}</li>
        <li>最近一次写回: {formatTimestampLabel(session?.updated_at)}</li>
      </ul>

      <div className="dictation-debug-actions">
        <button className="ghost-button compact" type="button" onClick={handleCopy}>
          {copyState === "copied" ? "已复制排障信息" : "复制排障信息"}
        </button>
        <p className="field-note">{copyHint}</p>
      </div>
    </FeedbackExpandableCard>
  )
}

function DictationTimelineCard({ sessions, selectedSessionId, onSelect }) {
  return (
    <FeedbackExpandableCard
      title="上传时间线"
      badge={`${sessions.length} 次`}
      summary={sessions.length > 1 ? "默认展示最新上传；点任意记录可切换下方详情与排障信息。" : "当前只有 1 次上传记录。"}
      defaultOpen={sessions.length > 1}
    >
      <div className="dictation-timeline-stack">
        {sessions.map((session, index) => {
          const isActive = session.session_id === selectedSessionId
          const statusMeta = getDictationStatusMeta(session)
          const scoreLabel = session.grading_result
            ? `${session.grading_result.score} 分`
            : session.grading_status === "failed"
              ? "失败"
              : "等待结果"

          return (
            <button
              aria-pressed={isActive}
              className={`dictation-timeline-card ${isActive ? "is-active" : ""}`}
              key={session.session_id}
              type="button"
              onClick={() => onSelect(session.session_id)}
            >
              <div className="dictation-timeline-head">
                <div>
                  <span className="dictation-timeline-kicker">{formatDictationTimelineLabel(index)}</span>
                  <strong>{formatTimestampLabel(session.grading_requested_at || session.updated_at)}</strong>
                </div>
                <span className="dictation-timeline-score">{scoreLabel}</span>
              </div>
              <div className="dictation-timeline-meta">
                <span className={`risk-badge badge-${statusMeta.tone}`}>{statusMeta.label}</span>
                <span>{session.session_id}</span>
              </div>
              <p>{summarizeDictationTimelineSession(session)}</p>
            </button>
          )
        })}
      </div>

      <p className="field-note">当前时间线按最近一次上传排在最前。点任意卡片后，下方会切换到对应会话的结果与日志定位信息。</p>
    </FeedbackExpandableCard>
  )
}

function DraftReviewLensStrip({ items, activeFilter, onChange }) {
  return (
    <div className="draft-review-strip" role="tablist" aria-label="草稿审核筛选">
      {items.map((item) => (
        <button
          aria-selected={activeFilter === item.id}
          className={`draft-review-card ${activeFilter === item.id ? "is-active" : ""}`}
          key={item.id}
          role="tab"
          type="button"
          onClick={() => onChange(item.id)}
        >
          <span className="draft-review-label">{item.label}</span>
          <strong>{item.count}</strong>
          <p>{item.caption}</p>
        </button>
      ))}
    </div>
  )
}

function PublishActionDock({ isVisible, stage, badge, title, caption, primaryAction, secondaryAction, tertiaryAction }) {
  if (!primaryAction && !secondaryAction && !tertiaryAction) {
    return null
  }

  return (
    <section className={`publish-action-dock stage-${stage} ${isVisible ? "is-visible" : ""}`.trim()}>
      <div className="publish-action-copy">
        <span className="publish-action-badge">{badge}</span>
        <strong>{title}</strong>
        <p>{caption}</p>
      </div>
      <div className="publish-action-buttons">
        {secondaryAction ? (
          <button className="ghost-button compact" type="button" onClick={secondaryAction.onClick} disabled={secondaryAction.disabled}>
            {secondaryAction.label}
          </button>
        ) : null}
        {tertiaryAction ? (
          <button className="ghost-button compact" type="button" onClick={tertiaryAction.onClick} disabled={tertiaryAction.disabled}>
            {tertiaryAction.label}
          </button>
        ) : null}
        {primaryAction ? (
          <button
            className={`primary-button ${primaryAction.emphasis === "secondary" ? "secondary" : ""}`.trim()}
            type="button"
            onClick={primaryAction.onClick}
            disabled={primaryAction.disabled}
          >
            {primaryAction.label}
          </button>
        ) : null}
      </div>
    </section>
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
    <section className="workflow-shell">
      <div className="workflow-head">
        <span className="summary-chip">今日发布路线</span>
        <p>先把老师原文转成草稿，再处理风险卡片，最后确认发布到孩子当天任务板。</p>
      </div>
      <div className="workflow-strip" aria-label="家长操作路径">
        {steps.map((step) => (
          <article className={`workflow-card workflow-${step.state}`} key={step.id}>
            <div className="workflow-card-top">
              <span className="workflow-index">{step.index}</span>
              <span className={`workflow-state workflow-state-${step.state}`}>
                {step.state === "warning" ? "需处理" : step.state === "active" ? "进行中" : "待开始"}
              </span>
            </div>
            <div>
              <strong>{step.title}</strong>
              <p>{step.detail}</p>
            </div>
          </article>
        ))}
      </div>
    </section>
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
  activeReviewFilter,
  onActiveReviewFilterChange,
  activeDraftTaskId,
  onActiveDraftTaskIdChange,
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
  const selectedCount = taskEntries.filter((entry) => selectedTaskIds.includes(entry.task.id)).length
  const filterPredicates = {
    risk: (entry) => entry.riskMeta.order <= 1,
    selected: (entry) => selectedTaskIds.includes(entry.task.id),
    warn: (entry) => entry.riskMeta.order === 2,
    ready: (entry) => entry.riskMeta.order === 3,
    all: () => true,
  }
  const reviewFilterItems = [
    { id: "risk", label: "风险", count: riskyTaskCount, caption: "先处理阻断和高风险" },
    { id: "selected", label: "已选", count: selectedCount, caption: "只看准备发布的任务" },
    { id: "warn", label: "提醒", count: warningTaskCount, caption: "相似和归类提醒" },
    { id: "ready", label: "建议发布", count: recommendedCount, caption: "低摩擦快速确认" },
    { id: "all", label: "全部", count: taskEntries.length, caption: "完整审核队列" },
  ]
  const visibleEntries = taskEntries.filter((entry) => {
    const predicate = filterPredicates[activeReviewFilter] || filterPredicates.all
    return predicate(entry)
  })
  const activeFilterMeta = reviewFilterItems.find((item) => item.id === activeReviewFilter) || reviewFilterItems[0]
  const activeEntry = visibleEntries.find((entry) => entry.task.id === activeDraftTaskId) || visibleEntries[0] || null
  const activeVisibleIndex = activeEntry ? visibleEntries.findIndex((entry) => entry.task.id === activeEntry.task.id) : -1

  useEffect(() => {
    if (taskEntries.length === 0) {
      onActiveDraftTaskIdChange("")
      return
    }

    if (visibleEntries.length === 0) {
      const fallbackFilter =
        reviewFilterItems.find((item) => item.id !== "selected" && item.count > 0) ||
        reviewFilterItems.find((item) => item.count > 0) ||
        reviewFilterItems[reviewFilterItems.length - 1]

      if (fallbackFilter && fallbackFilter.id !== activeReviewFilter) {
        onActiveReviewFilterChange(fallbackFilter.id)
      }
      return
    }

    if (!activeDraftTaskId || !visibleEntries.some((entry) => entry.task.id === activeDraftTaskId)) {
      onActiveDraftTaskIdChange(visibleEntries[0].task.id)
    }
  }, [
    activeDraftTaskId,
    activeReviewFilter,
    onActiveDraftTaskIdChange,
    onActiveReviewFilterChange,
    reviewFilterItems,
    taskEntries.length,
    visibleEntries,
  ])

  function focusRelativeDraft(offset) {
    if (visibleEntries.length === 0) {
      return
    }

    const currentIndex = activeVisibleIndex >= 0 ? activeVisibleIndex : 0
    const nextIndex = Math.min(Math.max(currentIndex + offset, 0), visibleEntries.length - 1)
    onActiveDraftTaskIdChange(visibleEntries[nextIndex].task.id)
  }

  return (
    <section className="panel" id="draft-review-panel">
      <div className="panel-heading">
        <div>
          <h3>审核队列</h3>
          <p className="panel-caption">先处理最需要家长确认的卡片，再决定哪些任务直接下发。</p>
        </div>
        <span>{tasks.length} 张卡</span>
      </div>

      <DraftReviewLensStrip items={reviewFilterItems} activeFilter={activeReviewFilter} onChange={onActiveReviewFilterChange} />

      <div className="review-summary">
        <div>
          <strong>先把难卡片处理掉</strong>
          <p>系统会先把 `needs_review`、低置信和阻断项排到前面。手机上用筛选缩小范围，再围绕当前卡片连续处理。</p>
        </div>
        <div className="summary-pill-row">
          <span className="summary-pill summary-risk">风险 {riskyTaskCount}</span>
          <span className="summary-pill summary-warn">提醒 {warningTaskCount}</span>
          <span className="summary-pill summary-ready">建议直接发布 {recommendedCount}</span>
        </div>
      </div>

      {tasks.length > 0 && activeEntry ? (
        <div className="draft-focus-panel">
          <div className="draft-focus-heading">
            <div>
              <strong>
                当前处理 {activeVisibleIndex + 1} / {visibleEntries.length}
              </strong>
              <p>把审核区收成一张当前卡片，上一张 / 下一张就能像刷消息一样连续处理。</p>
            </div>
            <span>{activeFilterMeta.label}</span>
          </div>

          <div className="draft-focus-summary">
            <div>
              <span className={`risk-badge badge-${activeEntry.riskMeta.tone}`}>{activeEntry.riskMeta.label}</span>
              <strong data-testid="draft-focus-title">{activeEntry.task.title || activeEntry.task.group_title || "未命名任务"}</strong>
              <p>
                {activeEntry.task.subject || "未分类"} / 置信度 {Math.round(Number(activeEntry.task.confidence || 0) * 100)}% /{" "}
                {selectedTaskIds.includes(activeEntry.task.id) ? "已勾选发布" : "未勾选"}
              </p>
            </div>
            <button className="ghost-button compact" type="button" onClick={() => onToggle(activeEntry.task.id)}>
              {selectedTaskIds.includes(activeEntry.task.id) ? "取消勾选" : "勾选发布"}
            </button>
          </div>

          <div className="draft-focus-actions">
            <button className="ghost-button compact" type="button" onClick={() => focusRelativeDraft(-1)} disabled={activeVisibleIndex <= 0}>
              上一张
            </button>
            <button
              className="ghost-button compact"
              type="button"
              onClick={() => focusRelativeDraft(1)}
              disabled={activeVisibleIndex < 0 || activeVisibleIndex >= visibleEntries.length - 1}
            >
              下一张
            </button>
            <button className="ghost-button compact" type="button" onClick={() => onActiveReviewFilterChange("all")}>
              看全部卡片
            </button>
          </div>

          <div className="draft-queue-strip" aria-label="草稿处理队列">
            {visibleEntries.map((entry, index) => (
              <button
                className={`draft-queue-chip ${entry.task.id === activeEntry.task.id ? "is-active" : ""}`}
                key={entry.task.id}
                type="button"
                onClick={() => onActiveDraftTaskIdChange(entry.task.id)}
              >
                <span>{String(index + 1).padStart(2, "0")}</span>
                <strong>{entry.task.subject || "未分类"}</strong>
                <p>{entry.task.title || "未命名任务"}</p>
              </button>
            ))}
          </div>
        </div>
      ) : null}

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
        <HelpAccordion
          className="draft-list-sheet"
          title="展开全部草稿卡片"
          caption="当前主屏先聚焦一张卡片；需要批量查看或修改时，再展开完整草稿列表。"
          badge={`${taskEntries.length} 张`}
        >
          <div className="draft-list focused-list">
            {taskEntries.map(({ task, diagnostics, riskMeta }, index) => {
              const isSelected = selectedTaskIds.includes(task.id)
              const confidence = Number(task.confidence || 0)
              const notes = Array.isArray(task.notes) ? task.notes : []
              const confidenceMeta = getConfidenceMeta(confidence)
              const isVisible = visibleEntries.some((entry) => entry.task.id === task.id)
              const isFocused = activeEntry?.task.id === task.id

              return (
                <article
                  className={`draft-card risk-${riskMeta.tone} ${task.needs_review ? "needs-review" : ""} ${diagnostics.hasBlocking ? "has-blocking" : ""
                    } ${isSelected ? "is-selected" : ""} ${isFocused ? "is-focused" : ""} ${isVisible ? "" : "is-filter-hidden"}`}
                  data-testid="draft-card"
                  data-task-title={task.title}
                  key={`${task.id}-${index}`}
                  onClick={() => onActiveDraftTaskIdChange(task.id)}
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
        </HelpAccordion>
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
  onJumpToDraftReview,
  onConfirm,
  isConfirming,
  createStatus,
}) {
  const selectedBlockingTasks = selectedTasks.filter((task) => diagnosticsById[task.id]?.hasBlocking)
  const selectedBlockingCount = selectedBlockingTasks.length
  const selectedRiskTaskItems = selectedTasks.filter((task) => {
    const riskMeta = getDraftTaskRiskMeta(task, diagnosticsById[task.id] || { issues: [], hasBlocking: false })
    return riskMeta.order <= 1 && !(diagnosticsById[task.id]?.hasBlocking)
  })
  const selectedRiskCount = selectedRiskTaskItems.length
  const selectedReadyTasks = selectedTasks.filter(
    (task) => !diagnosticsById[task.id]?.hasBlocking && !task.needs_review && Number(task.confidence || 0) >= 0.7,
  )
  const selectedReadyCount = selectedReadyTasks.length
  const canSubmit = selectedTasks.length > 0 && selectedBlockingCount === 0 && !isConfirming
  const jumpCards = [
    {
      label: "已选任务",
      value: selectedTasks.length,
      caption: selectedTasks.length > 0 ? "回到已勾选队列继续核对" : "先在上方勾选要发布的任务",
      disabled: selectedTasks.length === 0,
      onClick: () => onJumpToDraftReview("selected", selectedTasks[0]?.id),
    },
    {
      label: "已选风险任务",
      value: selectedRiskCount,
      caption: selectedRiskCount > 0 ? "跳回高风险卡片逐条确认" : "当前已选项里没有高风险任务",
      disabled: selectedRiskCount === 0,
      onClick: () => onJumpToDraftReview("risk", selectedRiskTaskItems[0]?.id),
    },
    {
      label: "已选阻断项",
      value: selectedBlockingCount,
      caption: selectedBlockingCount > 0 ? "直接回到需修正的阻断项" : "当前没有阻断项",
      disabled: selectedBlockingCount === 0,
      onClick: () => onJumpToDraftReview("risk", selectedBlockingTasks[0]?.id),
    },
    {
      label: "建议直接发布",
      value: selectedReadyCount,
      caption: selectedReadyCount > 0 ? "回看低摩擦可发布项" : "当前没有可直发的已选项",
      disabled: selectedReadyCount === 0,
      onClick: () => onJumpToDraftReview("selected", selectedReadyTasks[0]?.id),
    },
  ]

  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <h3>发布确认台</h3>
          <p className="panel-caption">把今天真正要下发的任务收口到这里，再一键发到孩子任务板。</p>
        </div>
        <span>{selectedTasks.length} 条已选</span>
      </div>

      <div className="confirm-summary-card">
        <span className="summary-chip">准备发布</span>
        <strong>{assignedDate || "未选择日期"}</strong>
        <p>
          当前已选 {selectedTasks.length} 条，其中阻断 {selectedBlockingCount} 条，高风险 {selectedRiskCount} 条，可直接发布 {selectedReadyCount} 条。
        </p>
      </div>

      <div className="analysis-grid confirm-grid">
        {jumpCards.map((item) => (
          <button
            className={`analysis-card review-jump-card ${item.disabled ? "is-disabled" : "is-actionable"}`}
            disabled={item.disabled}
            key={item.label}
            type="button"
            onClick={item.onClick}
          >
            <span>{item.label}</span>
            <strong>{item.value}</strong>
            <p>{item.caption}</p>
          </button>
        ))}
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
      <section className="panel analysis-panel">
        <div className="panel-heading">
          <h3>AI 读题摘要</h3>
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
    <section className="panel analysis-panel">
      <div className="panel-heading">
        <h3>AI 读题摘要</h3>
        <span>{parserMode || "unknown"}</span>
      </div>

      <div className="analysis-summary-card">
        <span className="summary-chip">解析结论</span>
        <strong>{parserMode === "llm_hybrid" ? "这次用了 LLM 混合解析" : "这次走规则兜底解析"}</strong>
        <p>
          {detectedSubjects.length > 0
            ? `识别到 ${detectedSubjects.join(" / ")}，可以继续进入草稿审核。`
            : "当前还没识别出明确学科，建议先回看老师原文。"}
        </p>
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
  dictationSessions,
  dictationStatus,
  weeklyStats,
  weeklyStatus,
  monthlyRows,
  monthlyStatus,
  reportView,
  onLoadDictation,
  onLoadWeekly,
  onLoadMonthly,
  isLoadingDictation,
  isLoadingWeekly,
  isLoadingMonthly,
}) {
  const [activeFeedbackSection, setActiveFeedbackSection] = useState("daily")
  const feedbackTransition = usePageTransition(activeFeedbackSection, FEEDBACK_PANEL_ORDER)
  const dailyReport = buildDailyReport(todayTasks, currentDatePointEntries, assignedDate)
  const weeklyRows = buildWeeklyRows(weeklyStats?.raw_stats)
  const dictationSummary = buildDictationSummary(dictationSessions)
  const latestDictationSession = dictationSummary.latestSession
  const latestDictationMeta = getDictationStatusMeta(latestDictationSession)
  const latestGradingResult = latestDictationSession?.grading_result || null
  const latestIncorrectItems = Array.isArray(latestGradingResult?.graded_items)
    ? latestGradingResult.graded_items.filter((item) => !item.is_correct || item.needs_correction)
    : []
  const [selectedDictationSessionId, setSelectedDictationSessionId] = useState("")
  const latestScore = latestGradingResult
    ? `${latestGradingResult.score} 分`
    : latestDictationSession?.grading_status === "failed"
      ? "失败"
      : latestDictationSession
        ? "处理中"
        : "--"
  const strengths = Array.isArray(weeklyStats?.insights?.strengths) ? weeklyStats.insights.strengths : []
  const improvements = Array.isArray(weeklyStats?.insights?.areas_for_improvement)
    ? weeklyStats.insights.areas_for_improvement
    : []
  let dictationSummaryText = `当前日期 ${assignedDate} 还没有来自 Pad 端的听写拍照记录。孩子提交后，这里会自动显示异步批改结果。`

  if (dictationSummary.total > 0 && latestDictationSession) {
    if (dictationSummary.pending > 0) {
      dictationSummaryText = `已收到 ${dictationSummary.total} 次听写拍照，其中 ${dictationSummary.pending} 次还在异步处理中。最新会话状态为“${latestDictationMeta.label}”。`
    } else if (dictationSummary.failed > 0 && dictationSummary.completed === 0) {
      dictationSummaryText = `当前日期共有 ${dictationSummary.total} 次听写拍照，但最近一次批改失败。请查看失败原因并考虑重试。`
    } else {
      dictationSummaryText = `当前日期共 ${dictationSummary.total} 次听写拍照，已完成 ${dictationSummary.completed} 次。最新会话状态为“${latestDictationMeta.label}”。`
    }
  }

  useEffect(() => {
    if (dailyReport.total === 0 && dictationSummary.total > 0) {
      setActiveFeedbackSection("dictation")
      return
    }

    if (dailyReport.total > 0 && activeFeedbackSection === "dictation" && dictationSummary.total === 0) {
      setActiveFeedbackSection("daily")
    }
  }, [activeFeedbackSection, dailyReport.total, dictationSummary.total])

  useEffect(() => {
    if (dictationSessions.length === 0) {
      if (selectedDictationSessionId) {
        setSelectedDictationSessionId("")
      }
      return
    }

    if (!selectedDictationSessionId || !dictationSessions.some((session) => session.session_id === selectedDictationSessionId)) {
      setSelectedDictationSessionId(dictationSessions[0].session_id)
    }
  }, [dictationSessions, selectedDictationSessionId])

  const inspectedDictationSession =
    dictationSessions.find((session) => session.session_id === selectedDictationSessionId) || latestDictationSession || null
  const inspectedDictationMeta = getDictationStatusMeta(inspectedDictationSession)
  const inspectedGradingResult = inspectedDictationSession?.grading_result || null
  const inspectedIncorrectItems = Array.isArray(inspectedGradingResult?.graded_items)
    ? inspectedGradingResult.graded_items.filter((item) => !item.is_correct || item.needs_correction)
    : []
  const inspectedScore = inspectedGradingResult
    ? `${inspectedGradingResult.score} 分`
    : inspectedDictationSession?.grading_status === "failed"
      ? "失败"
      : inspectedDictationSession
        ? "处理中"
        : "--"
  const inspectedIncorrectPreviewText =
    inspectedIncorrectItems.length > 0
      ? inspectedIncorrectItems
          .slice(0, 2)
          .map((item) => `${item.expected}${item.actual ? ` -> ${item.actual}` : ""}`)
          .join(" / ")
      : ""
  const selectedDictationIndex = dictationSessions.findIndex((session) => session.session_id === inspectedDictationSession?.session_id)
  const selectedDictationLabel = selectedDictationIndex <= 0 ? "最新" : `历史 ${selectedDictationIndex + 1}`
  const selectedDictationSummary =
    selectedDictationIndex <= 0
      ? inspectedDictationMeta.description
      : `已切换到较早一次上传记录。${inspectedDictationMeta.description}`

  const feedbackLensItems = [
    {
      id: "daily",
      label: "日报",
      metric: dailyReport.total > 0 ? `${dailyReport.completionRate}%` : "--",
      caption: dailyReport.total > 0 ? `还剩 ${dailyReport.pending} 条任务，积分 ${formatSignedAmount(dailyReport.pointDelta)}。` : "当前日期还没有任务数据。",
    },
    {
      id: "dictation",
      label: "听写",
      metric: latestScore,
      caption:
        dictationSummary.total > 0
          ? dictationSummary.pending > 0
            ? `${dictationSummary.pending} 次批改处理中。`
            : `${dictationSummary.completed} 次已完成，${latestIncorrectItems.length} 项需订正。`
          : "Pad 端还没有拍照上传。",
    },
    {
      id: "trend",
      label: "趋势",
      metric: reportView === "week" ? `${weeklyRows.length} 天` : `${monthlyRows.length} 组`,
      caption: reportView === "week" ? "周趋势和鼓励方向集中看。" : "月趋势和积分变化集中看。",
    },
  ]
  const subjectLead = dailyReport.subjectRows[0]
  const dailySnapshotItems = [
    {
      label: "当日完成率",
      value: dailyReport.total > 0 ? `${dailyReport.completionRate}%` : "--",
      caption: dailyReport.total > 0 ? `还剩 ${dailyReport.pending} 条任务。` : "当前日期还没有任务。",
    },
    {
      label: "积分变化",
      value: formatSignedAmount(dailyReport.pointDelta),
      caption: currentDatePointEntries.length > 0 ? `今日有 ${currentDatePointEntries.length} 条积分记录。` : "当前没有积分波动。",
    },
    {
      label: "学科覆盖",
      value: dailyReport.subjectRows.length,
      caption: subjectLead ? `当前任务最多的是 ${subjectLead.subject}。` : "等待任务板产生学科分布。",
    },
  ]
  const dictationSnapshotItems = [
    {
      label: "拍照会话",
      value: dictationSummary.total,
      caption: dictationSummary.total > 0 ? "同一日期下的全部上传次数。" : "当前还没有拍照上传。",
    },
    {
      label: "异步状态",
      value: dictationSummary.pending > 0 ? `${dictationSummary.pending} 次处理中` : latestDictationMeta.label,
      caption: dictationSummary.pending > 0 ? "后台仍在解析或批改。" : "最近一次会话状态。",
    },
    {
      label: "最新得分",
      value: latestScore,
      caption: latestIncorrectItems.length > 0 ? `${latestIncorrectItems.length} 项待订正。` : "当前没有额外错词提醒。",
    },
  ]
  const trendSnapshotItems =
    reportView === "week"
      ? [
          {
            label: "周视图",
            value: `${weeklyRows.length} 天`,
            caption: weeklyRows.length > 0 ? "已加载近 7 天任务趋势。" : "等待拉取 weekly 数据。",
          },
          {
            label: "鼓励方向",
            value: strengths.length,
            caption: strengths.length > 0 ? "已有可直接给家长的话术。" : "当前没有额外鼓励点。",
          },
          {
            label: "下周关注",
            value: improvements.length,
            caption: improvements.length > 0 ? "已有建议可回带给孩子。" : "当前没有改进建议。",
          },
        ]
      : [
          {
            label: "月视图",
            value: `${monthlyRows.length} 组`,
            caption: monthlyRows.length > 0 ? "已加载月度聚合趋势。" : "等待拉取 monthly 数据。",
          },
          {
            label: "完成趋势",
            value: monthlyRows.length > 0 ? `${monthlyRows.filter((item) => item.completed > 0).length} 组有完成` : "--",
            caption: "看本月各组任务完成情况。",
          },
          {
            label: "积分变化",
            value: monthlyRows.length > 0 ? formatSignedAmount(monthlyRows.reduce((sum, item) => sum + Number(item.pointDelta || 0), 0)) : "--",
            caption: "按月聚合后的积分累计变化。",
          },
        ]
  let topSummaryMeta = {
    chip: "日报摘要",
    text: dailyReport.summary,
  }

  if (activeFeedbackSection === "dictation") {
    topSummaryMeta = {
      chip: "听写摘要",
      text: dictationSummaryText,
    }
  } else if (activeFeedbackSection === "trend") {
    topSummaryMeta = {
      chip: reportView === "week" ? "周趋势摘要" : "月趋势摘要",
      text:
        reportView === "week"
          ? weeklyStats?.insights?.summary || (weeklyRows.length > 0 ? `已加载 ${weeklyRows.length} 天趋势，先看图，再展开鼓励方向和下周关注。` : "先点击“查看周趋势”拉取最近 7 天统计。")
          : monthlyRows.length > 0
            ? `已加载 ${monthlyRows.length} 组月度聚合数据，可先看趋势图，再决定是否回到日报或积分区。`
            : "先点击“查看月趋势”拉取当月真实聚合统计。",
    }
  }

  return (
    <section className="panel" id="report-console">
      <div className="panel-heading">
        <div>
          <h2>家长查看反馈</h2>
          <p className="panel-caption">基于指定日期任务板、积分记录、异步听写批改和统计接口生成当前反馈。</p>
        </div>
        <span>{childLabel}</span>
      </div>

      <FeedbackLensStrip items={feedbackLensItems} activeSection={activeFeedbackSection} onChange={setActiveFeedbackSection} />

      <div className="report-summary-card">
        <span className="summary-chip">{topSummaryMeta.chip}</span>
        <p>{topSummaryMeta.text}</p>
      </div>

      <section className="panel feedback-detail-panel">
        <div
          className={`screen-subpanel-deck feedback-subpanel-deck ${feedbackTransition.isTransitioning ? `is-transitioning dir-${feedbackTransition.direction}` : ""}`.trim()}
        >
          <div className={feedbackTransition.getPageClass("daily")}>
            <div className="panel-heading">
              <div>
                <h3>日报与任务完成</h3>
                <p className="panel-caption">先看今天完成了多少、还差什么，再决定是否去调积分或补发布。</p>
              </div>
              <span>{assignedDate}</span>
            </div>

            <FeedbackSnapshotGrid items={dailySnapshotItems} />

            <FeedbackExpandableCard
              title="学科完成分布"
              badge={dailyReport.subjectRows.length > 0 ? `${dailyReport.subjectRows.length} 个学科` : "暂无数据"}
              summary={
                dailyReport.subjectRows.length > 0
                  ? `${subjectLead.subject} 当前任务最多，先展开看各学科完成比。`
                  : "当前日期还没有可汇总的任务板数据。"
              }
              defaultOpen={dailyReport.subjectRows.length > 0 && dailyReport.subjectRows.length <= 2}
            >
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
              ) : (
                <p className="empty-state">当前日期还没有可汇总的任务板数据。</p>
              )}
            </FeedbackExpandableCard>

            <FeedbackExpandableCard
              title="今日行动建议"
              badge={dailyReport.pending === 0 && dailyReport.total > 0 ? "已完成" : `剩余 ${dailyReport.pending} 条`}
              summary={dailyReport.pending === 0 && dailyReport.total > 0 ? "任务已清空，可以转去看听写或趋势。" : "先补当天任务，再决定是否调积分或回看趋势。"}
              defaultOpen={dailyReport.total === 0}
            >
              <ul className="rule-list compact">
                <li>任务总数 {dailyReport.total} 条，已完成 {dailyReport.completed} 条。</li>
                <li>今日积分变化 {formatSignedAmount(dailyReport.pointDelta)}。</li>
                <li>{dailyReport.pending > 0 ? "如果还有未完成任务，优先回到发布区或孩子任务板补齐。" : "当天任务完成后，可以继续查看听写拍照结果和周月趋势。"}</li>
              </ul>
            </FeedbackExpandableCard>
          </div>

          <div className={feedbackTransition.getPageClass("dictation")}>
            <div className="panel-heading">
              <div>
                <h3>异步听写批改</h3>
                <p className="panel-caption">Pad 上传拍照后，家长端在这里收最终结果，不要求同步等待。</p>
              </div>
              <span>{dictationSummary.total} 次会话</span>
            </div>

            <div className="report-switcher">
              <button className="ghost-button compact" type="button" onClick={onLoadDictation} disabled={isLoadingDictation}>
                {isLoadingDictation ? "刷新听写..." : "刷新听写批改"}
              </button>
              <button className="ghost-button compact" type="button" onClick={() => setActiveFeedbackSection("trend")}>
                去看趋势
              </button>
            </div>

            <FeedbackSnapshotGrid items={dictationSnapshotItems} />

            <div className="report-summary-card">
              <span className="summary-chip">异步结果摘要</span>
              <p>{dictationSummaryText}</p>
            </div>

            <StatusBanner
              status={dictationStatus}
              actionLabel={dictationStatus?.retryable ? (isLoadingDictation ? "重试中..." : "重试拉取听写结果") : undefined}
              onAction={dictationStatus?.retryable ? onLoadDictation : undefined}
              actionDisabled={!dictationStatus?.retryable || isLoadingDictation}
            />

            {latestDictationSession ? (
              <>
                <DictationTimelineCard
                  sessions={dictationSessions}
                  selectedSessionId={inspectedDictationSession?.session_id || ""}
                  onSelect={setSelectedDictationSessionId}
                />

                <FeedbackExpandableCard
                  title="当前查看会话"
                  badge={selectedDictationLabel}
                  summary={selectedDictationSummary}
                  defaultOpen={inspectedDictationSession?.grading_status !== "completed"}
                >
                  <ul className="rule-list compact">
                    <li>听写日期: {inspectedDictationSession?.assigned_date || assignedDate}</li>
                    <li>会话 ID: {inspectedDictationSession?.session_id || "--"}</li>
                    <li>批改状态: <span className={`risk-badge badge-${inspectedDictationMeta.tone}`}>{inspectedDictationMeta.label}</span></li>
                    <li>请求时间: {formatTimestampLabel(inspectedDictationSession?.grading_requested_at || inspectedDictationSession?.updated_at)}</li>
                    <li>完成时间: {formatTimestampLabel(inspectedDictationSession?.grading_completed_at)}</li>
                  </ul>
                </FeedbackExpandableCard>

                <FeedbackExpandableCard
                  title="批改结果"
                  badge={inspectedScore}
                  summary={
                    inspectedGradingResult
                      ? `本次得分 ${inspectedGradingResult.score} 分，识别到 ${inspectedIncorrectItems.length} 项需订正。`
                      : inspectedDictationSession?.grading_status === "failed"
                        ? "后台返回了失败状态，建议先查看服务日志。"
                        : "当前还没有最终批改结果，后台完成后这里会自动同步。"
                  }
                  defaultOpen={Boolean(inspectedGradingResult)}
                >
                  {inspectedGradingResult ? (
                    <ul className="rule-list compact">
                      <li>批改条目: {inspectedGradingResult.graded_items.length}</li>
                      <li>需订正条目: {inspectedIncorrectItems.length}</li>
                      <li>整体结论: {inspectedGradingResult.status === "passed" ? "通过" : "待订正"}</li>
                      <li>批改完成时间: {formatTimestampLabel(inspectedGradingResult.created_at)}</li>
                    </ul>
                  ) : inspectedDictationSession?.grading_status === "failed" ? (
                    <p>{inspectedDictationSession.grading_error || "后台处理失败，但没有返回更多错误信息。"}</p>
                  ) : (
                    <p>当前还没有最终批改结果，后台完成后这里会自动同步。</p>
                  )}
                </FeedbackExpandableCard>

                <FeedbackExpandableCard
                  title="错词与建议"
                  badge={inspectedIncorrectItems.length > 0 ? `${inspectedIncorrectItems.length} 项` : "无错词"}
                  summary={
                    inspectedIncorrectItems.length > 0
                      ? `${inspectedIncorrectPreviewText}${inspectedIncorrectItems.length > 2 ? " 等" : ""}`
                      : inspectedGradingResult?.ai_feedback || "当前没有额外错词详情。"
                  }
                  defaultOpen={inspectedIncorrectItems.length > 0 && inspectedIncorrectItems.length <= 2}
                >
                  {inspectedIncorrectItems.length > 0 ? (
                    <>
                      <ul className="rule-list compact">
                        {inspectedIncorrectItems.map((item) => (
                          <li key={`${inspectedDictationSession?.session_id || "dictation"}-${item.index}`}>
                            {item.expected}
                            {item.meaning ? `（${item.meaning}）` : ""}
                            {" -> "}
                            {item.actual || "未识别"}
                            {item.comment ? `；${item.comment}` : ""}
                          </li>
                        ))}
                      </ul>
                      {inspectedGradingResult?.ai_feedback ? <p>{inspectedGradingResult.ai_feedback}</p> : null}
                    </>
                  ) : inspectedGradingResult?.ai_feedback ? (
                    <p>{inspectedGradingResult.ai_feedback}</p>
                  ) : inspectedDictationSession?.grading_status === "failed" ? (
                    <p>当前没有可展示的错词详情，请优先查看失败日志。</p>
                  ) : (
                    <p className="empty-state">当前没有额外错词详情。</p>
                  )}
                </FeedbackExpandableCard>

                <DictationTroubleshootCard session={inspectedDictationSession} />
              </>
            ) : (
              <p className="empty-state">当前日期还没有听写拍照上传记录。孩子在 Pad 端提交后，这里会自动汇总最新状态。</p>
            )}
          </div>

          <div className={feedbackTransition.getPageClass("trend")}>
            <div className="panel-heading">
              <div>
                <h3>周 / 月趋势</h3>
                <p className="panel-caption">趋势部分单独成段，手机上先看摘要，再决定拉哪一种统计。</p>
              </div>
              <span>{reportView === "week" ? "周视图" : "月视图"}</span>
            </div>

            <div className="report-switcher">
              <button className="ghost-button compact" type="button" onClick={onLoadWeekly} disabled={isLoadingWeekly}>
                {isLoadingWeekly ? "加载周趋势..." : "查看周趋势"}
              </button>
              <button className="ghost-button compact" type="button" onClick={onLoadMonthly} disabled={isLoadingMonthly}>
                {isLoadingMonthly ? "加载月趋势..." : "查看月趋势"}
              </button>
              <button className="ghost-button compact" type="button" onClick={() => setActiveFeedbackSection("daily")}>
                回到日报
              </button>
            </div>

            <FeedbackSnapshotGrid items={trendSnapshotItems} />

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
                    <FeedbackExpandableCard
                      title="周趋势图"
                      badge={`${weeklyRows.length} 天`}
                      summary="先看近 7 天完成变化，再决定是否展开鼓励方向和改进建议。"
                      defaultOpen={weeklyRows.length > 0}
                    >
                      <TrendBars
                        items={weeklyRows}
                        valueLabel={(item) => `${item.completed}/${item.total} 完成`}
                      />
                    </FeedbackExpandableCard>
                    <FeedbackExpandableCard
                      title="周反馈建议"
                      badge={`${strengths.length}/${improvements.length}`}
                      summary={weeklyStats?.insights?.summary || "当前 weekly 接口未返回总结，已展示原始趋势数据。"}
                      defaultOpen={strengths.length > 0 || improvements.length > 0}
                    >
                      <div className="insight-grid">
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
                    </FeedbackExpandableCard>
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
                    <FeedbackExpandableCard
                      title="月趋势图"
                      badge={`${monthlyRows.length} 组`}
                      summary="月趋势来自真实 `/api/v1/stats/monthly` 聚合结果，便于直接联调家长端反馈视图。"
                      defaultOpen={monthlyRows.length > 0}
                    >
                      <TrendBars
                        items={monthlyRows}
                        valueLabel={(item) =>
                          `${item.completed}/${item.total} 完成 · 积分 ${formatSignedAmount(item.pointDelta)}`
                        }
                      />
                      <p className="field-note">月趋势来自真实 `/api/v1/stats/monthly` 聚合结果，便于直接联调家长端反馈视图。</p>
                    </FeedbackExpandableCard>
                  </>
                ) : monthlyStatus?.tone !== "error" ? (
                  <p className="empty-state">点击“查看月趋势”后，这里会展示该月真实统计接口返回的完成趋势与积分变化。</p>
                ) : null}
              </div>
            )}
          </div>
        </div>
      </section>
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
  const [activePointsPanel, setActivePointsPanel] = useState("compose")
  const todayDelta = todayPointEntries.reduce((sum, entry) => sum + Number(entry.amount || 0), 0)
  const signedPreview =
    pointForm.mode === "penalty" ? -Math.abs(Number(pointForm.amount || 0)) : Math.abs(Number(pointForm.amount || 0))
  const latestEntry = recentPointEntries[0] || null
  const todayRewardCount = todayPointEntries.filter((entry) => Number(entry.amount || 0) > 0).length
  const todayPenaltyCount = todayPointEntries.filter((entry) => Number(entry.amount || 0) < 0).length
  const recentReasons = recentPointEntries
    .filter((entry) => (pointForm.mode === "reward" ? Number(entry.amount || 0) > 0 : Number(entry.amount || 0) < 0))
    .map((entry) => String(entry.reason || "").trim())
    .filter(Boolean)
  const suggestedReasons = [...new Set([...(POINT_REASON_PRESETS[pointForm.mode] || []), ...recentReasons])].slice(0, 5)
  const modeLabel = pointForm.mode === "reward" ? "家长加分" : "家长扣分"
  const pointsPanelItems = [
    {
      id: "compose",
      label: "记一笔",
      metric: formatSignedAmount(signedPreview),
      caption: `${pointForm.assignedDate || assignedDate} / ${modeLabel}`,
    },
    {
      id: "ledger",
      label: "看流水",
      metric: `${recentPointEntries.length} 条`,
      caption: recentPointEntries.length > 0 ? "当天和最近记录集中查看" : "提交后这里会积累历史记录",
    },
  ]

  useEffect(() => {
    if (pointsStatus?.tone === "success" && recentPointEntries.length > 0) {
      setActivePointsPanel("ledger")
    }
  }, [pointsStatus?.tone, recentPointEntries.length])

  return (
    <section className="panel" id="points-console">
      <div className="panel-heading">
        <div>
          <h2>积分操作</h2>
          <p className="panel-caption">手机上按“选模式 -&gt; 点分值 -&gt; 补原因 -&gt; 看流水”完成操作，提交走真实 `/api/v1/points/ledger`。</p>
        </div>
        <span>
          家庭 {familyId} / 孩子 {assigneeId}
        </span>
      </div>

      <SubMenuStrip items={pointsPanelItems} activeItem={activePointsPanel} onChange={setActivePointsPanel} ariaLabel="积分子菜单" />

      <ScreenSubpanelDeck activeId={activePointsPanel} orderedIds={POINTS_PANEL_ORDER}>
        {(getPanelClass) => (
          <>
            <div className={getPanelClass("compose")}>
              <div className="points-focus-strip">
                <article className={`points-focus-card ${pointForm.mode === "reward" ? "tone-reward" : "tone-penalty"}`}>
                  <span>当前模式</span>
                  <strong>{modeLabel}</strong>
                  <p>本次提交预览 {formatSignedAmount(signedPreview)}，日期 {pointForm.assignedDate || assignedDate}。</p>
                </article>
                <article className="points-focus-card tone-calm">
                  <span>当天积分走势</span>
                  <strong>{formatSignedAmount(todayDelta)}</strong>
                  <p>
                    奖励 {todayRewardCount} 次，扣分 {todayPenaltyCount} 次。
                  </p>
                </article>
                <article className="points-focus-card tone-accent">
                  <span>当前余额</span>
                  <strong>{estimatedBalance ?? "--"}</strong>
                  <p>{recentPointEntries.length > 0 ? `最近 ${recentPointEntries.length} 条手工流水已同步。` : "还没有手工积分记录。"}</p>
                </article>
                <article className="points-focus-card tone-neutral">
                  <span>最近一条</span>
                  <strong>{latestEntry ? formatSignedAmount(latestEntry.amount) : "--"}</strong>
                  <p>{latestEntry ? `${latestEntry.reason || "未填写备注"} · ${latestEntry.assigned_date}` : "提交后会在这里看到最近记录。"}</p>
                </article>
              </div>

              <div className="points-compose-shell">
                <div className="points-compose-heading">
                  <div>
                    <strong>快速记一笔</strong>
                    <p className="panel-caption">先选加分还是扣分，再点快捷分值，最后补一句原因即可提交。</p>
                  </div>
                  <span>{pointForm.assignedDate || assignedDate}</span>
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

                <div className="quick-actions points-amount-row">
                  {[2, 5, 10].map((amount) => (
                    <button className="ghost-button compact" type="button" key={amount} onClick={() => onQuickAmount(amount)}>
                      {pointForm.mode === "reward" ? "+" : "-"}
                      {amount}
                    </button>
                  ))}
                </div>

                {suggestedReasons.length > 0 ? (
                  <div className="points-reason-strip" aria-label="积分原因建议">
                    {suggestedReasons.map((reason) => (
                      <button className="ghost-button compact" type="button" key={reason} onClick={() => onPointFormChange("reason", reason)}>
                        {reason}
                      </button>
                    ))}
                  </div>
                ) : null}

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
                  <button className="primary-button secondary" type="button" onClick={onSubmit} disabled={isSubmitting}>
                    {isSubmitting ? "提交中..." : "提交积分调整"}
                  </button>
                  <button className="ghost-button" type="button" onClick={() => onPointFormChange("assignedDate", assignedDate)}>
                    使用当前任务日期
                  </button>
                </div>
              </div>

              <StatusBanner
                status={pointsStatus}
                actionLabel={pointsStatus?.retryable ? (isSubmitting ? "重试中..." : "重试积分提交") : undefined}
                onAction={pointsStatus?.retryable ? onRetry : undefined}
                actionDisabled={!pointsStatus?.retryable || isSubmitting}
              />
            </div>

            <div className={getPanelClass("ledger")}>
              <div className="mini-list">
                <div className="mini-list-heading">
                  <div>
                    <strong>最近积分流水</strong>
                    <p className="panel-caption">按卡片展示日期、备注和正负分值，手机上直接扫一眼就能回看今天与历史记录。</p>
                  </div>
                  <span>{recentPointEntries.length} 条</span>
                </div>
                {recentPointEntries.length === 0 ? (
                  <p className="empty-state">还没有积分操作记录。提交一次加分或扣分后，这里会保留最近明细。</p>
                ) : (
                  <div className="ledger-stack">
                    {recentPointEntries.map((entry) => (
                      <article
                        className={`ledger-card ${Number(entry.amount || 0) >= 0 ? "tone-reward" : "tone-penalty"}`}
                        key={entry.id}
                      >
                        <div className="ledger-card-head">
                          <strong>{formatSignedAmount(entry.amount)}</strong>
                          <span>{entry.assigned_date}</span>
                        </div>
                        <p>{entry.reason || "未填写备注"}</p>
                        <div className="ledger-card-meta">
                          <span>{formatTimeOnlyLabel(entry.created_at)}</span>
                          <span>{entry.assigned_date === assignedDate ? "当前日期" : "历史记录"}</span>
                        </div>
                      </article>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </ScreenSubpanelDeck>
    </section>
  )
}

function WordListEditorCard({
  list,
  isCurrentDate,
  syncStatus,
  onUpdateWordListMeta,
  onAddWordItem,
  onUpdateWordItem,
  onRemoveWordItem,
  onRemoveWordList,
  onSyncNow,
}) {
  const [isExpanded, setIsExpanded] = useState(isCurrentDate)
  const itemCount = Array.isArray(list.items) ? list.items.length : 0

  useEffect(() => {
    if (isCurrentDate) {
      setIsExpanded(true)
    }
  }, [isCurrentDate])

  return (
    <article className={`word-list-editor-card ${isExpanded ? "is-expanded" : ""}`}>
      <button
        aria-expanded={isExpanded}
        className="word-list-summary"
        type="button"
        onClick={() => setIsExpanded((current) => !current)}
      >
        <div>
          <span className="word-list-date-chip">{list.assigned_date}</span>
          <strong>{list.title || "未命名清单"}</strong>
          <p>
            {resolveChildLabel(String(list.family_id), String(list.assignee_id))} / {list.language || "未设置语言"} / {itemCount} 词项
          </p>
        </div>
        <div className="word-list-summary-side">
          <span className={`sync-pill sync-${syncStatus?.state || "idle"}`}>{syncStatus?.label || "已同步"}</span>
          <span className="word-list-toggle">{isExpanded ? "收起" : "展开"}</span>
        </div>
      </button>

      {isExpanded ? (
        <div className="word-list-editor-body">
          <div className={`word-list-sync-banner banner-${syncStatus?.tone || "success"}`}>
            <div>
              <strong>{syncStatus?.label || "已同步"}</strong>
              <p>{syncStatus?.message || "当前清单内容已经同步到服务器。"}</p>
            </div>
            {syncStatus?.state === "dirty" || syncStatus?.state === "error" ? (
              <button className="ghost-button compact" type="button" onClick={() => onSyncNow(list.id)}>
                {syncStatus.state === "error" ? "重试同步" : "立即同步"}
              </button>
            ) : null}
          </div>

          <div className="field-grid word-list-meta-grid">
            <label>
              <span>清单名</span>
              <input value={list.title} onChange={(event) => onUpdateWordListMeta(list.id, "title", event.target.value)} />
            </label>
            <label>
              <span>语言</span>
              <select value={list.language} onChange={(event) => onUpdateWordListMeta(list.id, "language", event.target.value)}>
                <option value="英语">英语</option>
                <option value="语文">语文</option>
                <option value="拼音">拼音</option>
                <option value="其他">其他</option>
              </select>
            </label>
            <label className="word-list-date-field">
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

          {itemCount === 0 ? (
            <p className="empty-state">当前清单还没有词项，先新增一条再同步到 Pad 端。</p>
          ) : (
            <div className="word-items-stack">
              {list.items.map((item, index) => (
                <article className="word-item-card" key={item.id}>
                  <div className="word-item-card-head">
                    <span className="word-item-index">{index + 1}</span>
                    <button className="inline-link danger" type="button" onClick={() => onRemoveWordItem(list.id, item.id)}>
                      删除词项
                    </button>
                  </div>
                  <label>
                    <span>单词 / 词组</span>
                    <input
                      className="word-item-input"
                      value={item.text}
                      onChange={(event) => onUpdateWordItem(list.id, item.id, "text", event.target.value)}
                      placeholder="输入词项"
                    />
                  </label>
                  <label>
                    <span>释义 / 提示</span>
                    <input
                      className="word-item-input"
                      value={item.meaning || ""}
                      onChange={(event) => onUpdateWordItem(list.id, item.id, "meaning", event.target.value)}
                      placeholder="输入释义"
                    />
                  </label>
                </article>
              ))}
            </div>
          )}

          <div className="word-list-card-actions">
            <button className="ghost-button compact" type="button" onClick={() => onAddWordItem(list.id)}>
              新增词项
            </button>
            <button className="inline-link danger" type="button" onClick={() => onRemoveWordList(list.id)}>
              移除本地卡片
            </button>
          </div>

          <p className="field-note">
            当前编辑会自动回写 `/api/v1/word-lists`。后端还没有 delete 接口，所以“移除本地卡片”只影响当前前端视图。
          </p>
        </div>
      ) : null}
    </article>
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
  wordListSyncState,
  onDraftChange,
  onUseAssignedDate,
  onCreate,
  onRetry,
  onUpdateWordListMeta,
  onAddWordItem,
  onUpdateWordItem,
  onRemoveWordItem,
  onRemoveWordList,
  onSyncWordListNow,
}) {
  const [activeWordPanel, setActiveWordPanel] = useState("create")
  const totalWordItems = wordLists.reduce((sum, list) => sum + (Array.isArray(list.items) ? list.items.length : 0), 0)
  const currentDateListCount = wordLists.filter((list) => list.assigned_date === assignedDate).length
  const draftItemCount = String(wordListDraft.itemsText || "")
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean).length
  const activeSyncCount = wordLists.filter((list) => {
    const syncState = wordListSyncState[list.id]?.state
    return ["dirty", "saving", "error"].includes(syncState)
  }).length
  const wordPanelItems = [
    {
      id: "create",
      label: "新建清单",
      metric: draftItemCount > 0 ? `${draftItemCount} 词` : "待录入",
      caption: `${wordListDraft.assignedDate || assignedDate} / ${wordListDraft.language}`,
    },
    {
      id: "lists",
      label: "已有清单",
      metric: `${wordLists.length} 份`,
      caption: wordLists.length > 0 ? "按日期管理和同步词项" : "创建后会在这里集中维护",
    },
  ]

  useEffect(() => {
    if (wordListStatus?.tone === "success" && wordLists.length > 0) {
      setActiveWordPanel("lists")
    }
  }, [wordListStatus?.tone, wordLists.length])

  return (
    <section className="panel" id="word-console">
      <div className="panel-heading">
        <div>
          <h2>单词清单管理</h2>
          <p className="panel-caption">走真实 `/api/v1/word-lists/parse` 和 `/api/v1/word-lists`，可直接联调家长端配置与 Pad 端默写。</p>
        </div>
        <span>{childLabel}</span>
      </div>

      <SubMenuStrip items={wordPanelItems} activeItem={activeWordPanel} onChange={setActiveWordPanel} ariaLabel="单词清单子菜单" />

      <ScreenSubpanelDeck activeId={activeWordPanel} orderedIds={WORD_PANEL_ORDER}>
        {(getPanelClass) => (
          <>
            <div className={getPanelClass("create")}>
              <div className="word-focus-strip">
                <article className="word-focus-card tone-accent">
                  <span>当前孩子清单</span>
                  <strong>{wordLists.length}</strong>
                  <p>{currentDateListCount > 0 ? `当前日期已有 ${currentDateListCount} 份清单。` : "当前日期还没有绑定清单。"}</p>
                </article>
                <article className="word-focus-card tone-calm">
                  <span>累计词项</span>
                  <strong>{totalWordItems}</strong>
                  <p>已保存的词项会直接供 Pad 端默写与拍照批改使用。</p>
                </article>
                <article className="word-focus-card tone-neutral">
                  <span>同步状态</span>
                  <strong>{activeSyncCount}</strong>
                  <p>{activeSyncCount > 0 ? "有清单待同步或需要重试。" : draftItemCount > 0 ? "新建草稿已准备好，可直接创建。" : "当前没有待同步的词单改动。"}</p>
                </article>
              </div>

              <div className="word-create-shell">
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

                <div className="word-format-hint">
                  <span className="summary-chip">批量录入提示</span>
                  <p>支持一行一个词，也支持 `apple - 苹果`、`blind（失明的）` 这种带释义写法，后端解析后会自动拆分。</p>
                </div>

                <div className="action-row">
                  <button className="primary-button secondary" type="button" onClick={onCreate}>
                    创建单词清单
                  </button>
                  <button className="ghost-button" type="button" onClick={onUseAssignedDate}>
                    使用当前任务日期
                  </button>
                </div>
              </div>

              <StatusBanner
                status={wordListStatus}
                actionLabel={wordListStatus?.retryable ? "重试创建清单" : undefined}
                onAction={wordListStatus?.retryable ? onRetry : undefined}
                actionDisabled={!wordListStatus?.retryable}
              />
            </div>

            <div className={getPanelClass("lists")}>
              <div className="mini-list">
                <div className="mini-list-heading">
                  <div>
                    <strong>当前孩子的单词清单</strong>
                    <p className="panel-caption">按日期折叠成卡片，手机端优先展开当前日期，编辑词项时不再挤成表格。</p>
                  </div>
                  <span>{wordLists.length} 份</span>
                </div>

                {wordLists.length === 0 ? (
                  <p className="empty-state">还没有为当前孩子创建单词清单。</p>
                ) : (
                  <div className="word-list-stack">
                    {wordLists.map((list) => (
                      <WordListEditorCard
                        key={list.id}
                        list={list}
                        isCurrentDate={list.assigned_date === assignedDate}
                        syncStatus={
                          wordListSyncState[list.id] || {
                            state: "idle",
                            tone: "success",
                            label: "已同步",
                            message: "当前清单内容已经和服务器保持一致。",
                          }
                        }
                        onUpdateWordListMeta={onUpdateWordListMeta}
                        onAddWordItem={onAddWordItem}
                        onUpdateWordItem={onUpdateWordItem}
                        onRemoveWordItem={onRemoveWordItem}
                        onRemoveWordList={onRemoveWordList}
                        onSyncNow={onSyncWordListNow}
                      />
                    ))}
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </ScreenSubpanelDeck>
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
  const [wordListSyncState, setWordListSyncState] = useState({})
  const [dictationSessions, setDictationSessions] = useState([])
  const [dictationStatus, setDictationStatus] = useState(null)
  const [isLoadingDictation, setIsLoadingDictation] = useState(false)
  const [activeConsoleSection, setActiveConsoleSection] = useState("publish-console")
  const [publishStage, setPublishStage] = useState("compose")
  const [activePublishPanel, setActivePublishPanel] = useState("scope")
  const [draftReviewFilter, setDraftReviewFilter] = useState("risk")
  const [draftFocusTaskId, setDraftFocusTaskId] = useState("")
  const wordListSyncTimersRef = useRef(new Map())
  const wordListSyncTokensRef = useRef(new Map())

  const previewSections = parseSchoolTaskMessage(rawText)
  const previewTasks = flattenSectionsToTasks(previewSections)
  const diagnostics = buildTaskDiagnostics(draftTasks, todayTasks)
  const selectedDraftTasks = draftTasks.filter((task) => selectedTaskIds.includes(task.id))
  const selectedBlockingDraftCount = selectedDraftTasks.filter((task) => diagnostics.byId[task.id]?.hasBlocking).length
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
        Number(right.assigned_date === assignedDate) - Number(left.assigned_date === assignedDate) ||
        String(right.assigned_date || "").localeCompare(String(left.assigned_date || ""), "zh-CN") ||
        new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime(),
    )
  const dailyReport = buildDailyReport(todayTasks, currentDatePointEntries, assignedDate)
  const dictationSummary = buildDictationSummary(dictationSessions)

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

      setWordLists(rawLists.map((list) => normalizeWordListFromApi(list)))
    } catch (error) {
      console.error("Failed to refresh word lists:", error)
    }
  }

  function setWordListSyncMeta(listId, meta) {
    setWordListSyncState((current) => ({
      ...current,
      [listId]: meta,
    }))
  }

  function clearWordListSyncTimer(listId) {
    const timerId = wordListSyncTimersRef.current.get(listId)
    if (!timerId) {
      return
    }

    window.clearTimeout(timerId)
    wordListSyncTimersRef.current.delete(listId)
  }

  function clearAllWordListSyncTimers() {
    wordListSyncTimersRef.current.forEach((timerId) => {
      window.clearTimeout(timerId)
    })
    wordListSyncTimersRef.current.clear()
  }

  function markWordListSyncToken(listId) {
    const nextToken = (wordListSyncTokensRef.current.get(listId) || 0) + 1
    wordListSyncTokensRef.current.set(listId, nextToken)
    return nextToken
  }

  async function persistWordList(list) {
    if (!list) {
      return
    }

    clearWordListSyncTimer(list.id)
    const requestToken = markWordListSyncToken(list.id)
    setWordListSyncMeta(list.id, {
      state: "saving",
      tone: "warning",
      label: "同步中",
      message: "正在把当前清单回写到服务器。",
    })

    const langMap = { 英语: "en", 语文: "zh" }
    const language = langMap[list.language] || list.language || "en"

    try {
      const data = await requestJSON(`${apiBaseUrl}/api/v1/word-lists`, {
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

      if (wordListSyncTokensRef.current.get(list.id) !== requestToken) {
        return
      }

      const savedList = normalizeWordListFromApi(data.word_list || list)
      setWordLists((current) => upsertWordListCollection(current, savedList))
      setWordListSyncMeta(savedList.id, {
        state: "success",
        tone: "success",
        label: "已同步",
        message: `${formatTimestampLabel(savedList.updated_at || new Date().toISOString())} 已回写到服务器。`,
      })
    } catch (error) {
      if (wordListSyncTokensRef.current.get(list.id) !== requestToken) {
        return
      }

      console.error("Failed to sync word list:", error)
      setWordListSyncMeta(list.id, {
        state: "error",
        tone: "error",
        label: "同步失败",
        message: error.message,
        retryable: true,
      })
    }
  }

  function scheduleWordListSync(list) {
    if (!list) {
      return
    }

    clearWordListSyncTimer(list.id)
    setWordListSyncMeta(list.id, {
      state: "dirty",
      tone: "warning",
      label: "待同步",
      message: "已记录本地改动，稍后会自动回写服务器。",
      retryable: true,
    })

    const timerId = window.setTimeout(() => {
      void persistWordList(list)
    }, WORD_LIST_SYNC_DEBOUNCE_MS)

    wordListSyncTimersRef.current.set(list.id, timerId)
  }

  useEffect(() => {
    const profile = resolveChildProfile(familyId, assigneeId)
    setSelectedChildPreset(profile ? profile.id : "custom")
    void refreshPoints()
    void refreshWordLists()
  }, [apiBaseUrl, familyId, assigneeId])

  useEffect(() => {
    clearAllWordListSyncTimers()
    setWordListSyncState({})
  }, [apiBaseUrl, familyId, assigneeId])

  useEffect(() => {
    return () => {
      clearAllWordListSyncTimers()
    }
  }, [])

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

  async function refreshDictationSessions(options = {}) {
    const { silent = false } = options

    if (!familyId.trim() || !assigneeId.trim() || !assignedDate) {
      setDictationSessions([])
      setDictationStatus(null)
      return
    }

    if (!silent) {
      setIsLoadingDictation(true)
    }

    try {
      const data = await requestJSON(
        `${apiBaseUrl}/api/v1/dictation-sessions?family_id=${encodeURIComponent(familyId)}&child_id=${encodeURIComponent(
          assigneeId,
        )}&date=${encodeURIComponent(assignedDate)}`,
      )
      const sessions = (Array.isArray(data.dictation_sessions) ? data.dictation_sessions : [])
        .map((session) => normalizeDictationSession(session))
        .sort(
          (left, right) =>
            new Date(right.grading_completed_at || right.grading_requested_at || right.updated_at || 0).getTime() -
            new Date(left.grading_completed_at || left.grading_requested_at || left.updated_at || 0).getTime(),
        )

      setDictationSessions(sessions)

      const summary = buildDictationSummary(sessions)
      if (summary.total === 0) {
        setDictationStatus(null)
      } else if (summary.pending > 0) {
        setDictationStatus({
          tone: "warning",
          title: "听写批改处理中",
          message: `当前日期已有 ${summary.total} 次拍照上传，其中 ${summary.pending} 次仍在后台异步处理中。`,
          retryable: false,
        })
      } else if (summary.failed > 0 && summary.completed === 0) {
        setDictationStatus({
          tone: "error",
          title: "听写批改失败",
          message: `当前日期共有 ${summary.total} 次拍照上传，但后台批改失败。请查看错误详情与服务日志。`,
          retryable: true,
        })
      } else {
        setDictationStatus({
          tone: summary.failed > 0 ? "warning" : "success",
          title: summary.failed > 0 ? "听写结果已部分同步" : "听写结果已同步",
          message:
            summary.failed > 0
              ? `当前日期共有 ${summary.total} 次拍照上传，已完成 ${summary.completed} 次，失败 ${summary.failed} 次。`
              : `当前日期共有 ${summary.total} 次拍照上传，已完成 ${summary.completed} 次批改。`,
          retryable: false,
        })
      }
    } catch (requestError) {
      setDictationStatus({
        tone: "error",
        title: "听写结果拉取失败",
        message: requestError.message,
        retryable: true,
      })
    } finally {
      if (!silent) {
        setIsLoadingDictation(false)
      }
    }
  }

  useEffect(() => {
    void refreshTasks()
  }, [apiBaseUrl, familyId, assigneeId, assignedDate])

  useEffect(() => {
    void refreshDictationSessions()
  }, [apiBaseUrl, familyId, assigneeId, assignedDate])

  useEffect(() => {
    if (!dictationSessions.some((session) => ["pending", "processing"].includes(session.grading_status))) {
      return undefined
    }

    const timerId = window.setTimeout(() => {
      void refreshDictationSessions({ silent: true })
    }, 3000)

    return () => {
      window.clearTimeout(timerId)
    }
  }, [apiBaseUrl, familyId, assigneeId, assignedDate, dictationSessions])

  useEffect(() => {
    if (createdTasks.length > 0) {
      setPublishStage("release")
      return
    }
    if (draftTasks.length > 0) {
      setPublishStage("review")
      return
    }
    setPublishStage("compose")
  }, [createdTasks.length, draftTasks.length])

  useEffect(() => {
    if (
      createdTasks.length === 0 &&
      draftTasks.length === 0 &&
      EMPTY_STATE_PUBLISH_FALLBACK_PANELS.includes(activePublishPanel)
    ) {
      setActivePublishPanel("scope")
    }
  }, [activePublishPanel, createdTasks.length, draftTasks.length])

  function handlePublishPanelChange(nextPanel) {
    setActivePublishPanel(nextPanel)

    if (nextPanel === "review") {
      setPublishStage("review")
      return
    }

    if (nextPanel === "release" || nextPanel === "board") {
      setPublishStage("release")
      return
    }

    setPublishStage("compose")
  }

  function handlePublishStageChange(nextStage) {
    setPublishStage(nextStage)

    if (nextStage === "review") {
      setActivePublishPanel("review")
      return
    }

    if (nextStage === "release") {
      setActivePublishPanel(createdTasks.length > 0 ? "board" : "release")
      return
    }

    setActivePublishPanel("compose")
  }

  function scheduleConsoleScroll(sectionIds) {
    if (typeof window === "undefined" || typeof document === "undefined") {
      return
    }

    const targetIds = Array.isArray(sectionIds) ? sectionIds : [sectionIds]
    const runScroll = () => {
      const target = targetIds.map((id) => document.getElementById(id)).find(Boolean)
      if (!target || typeof target.scrollIntoView !== "function") {
        return
      }

      target.scrollIntoView({
        behavior: "smooth",
        block: "start",
      })
    }

    if (typeof window.requestAnimationFrame === "function") {
      window.requestAnimationFrame(() => {
        window.requestAnimationFrame(runScroll)
      })
      return
    }

    window.setTimeout(runScroll, 16)
  }

  function jumpToConsoleSection(sectionId) {
    setActiveConsoleSection(sectionId)
    scheduleConsoleScroll(sectionId)
  }

  function jumpToDraftReview(filter = "risk", taskId = "") {
    setPublishStage("review")
    setActivePublishPanel("review")
    setDraftReviewFilter(filter)
    setDraftFocusTaskId(taskId || "")
    setActiveConsoleSection("publish-console")
    scheduleConsoleScroll(["draft-review-panel", "publish-console"])
  }

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
        handlePublishPanelChange("preview")
        setParseStatus({
          tone: "warning",
          title: "解析完成，但未生成草稿",
          message: `原文和本地结构预览已保留。当前目标日期为 ${assignedDate}，可以调整原文格式后再次解析。`,
          retryable: true,
        })
      } else {
        handlePublishPanelChange("review")
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
      handlePublishPanelChange("board")
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
      const data = await requestJSON(`${apiBaseUrl}/api/v1/points/ledger`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: Number(assigneeId),
          family_id: Number(familyId),
          delta: signedAmount,
          source_type: pointForm.mode === "penalty" ? "parent_penalty" : "parent_reward",
          note: pointForm.reason.trim(),
          occurred_on: pointForm.assignedDate || assignedDate,
        }),
      })

      setEstimatedBalance(data.points_balance?.balance ?? data.balance ?? null)
      setPointsStatus({
        tone: "success",
        title: "积分已更新",
        message: `已按 ${pointForm.assignedDate || assignedDate} 同步更新当前孩子的积分记录。`,
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

    if (!wordListDraft.title.trim()) {
      setWordListStatus({
        tone: "error",
        title: "无法创建清单",
        message: "请先填写清单名。",
        retryable: false,
      })
      return
    }

    if (!wordListDraft.itemsText.trim()) {
      setWordListStatus({
        tone: "error",
        title: "无法创建清单",
        message: "请至少录入一个词项。",
        retryable: false,
      })
      return
    }

    setIsSubmitting(true)
    try {
      // 1. 先调用后端解析接口进行结构化拆分
      const parseData = await requestJSON(`${apiBaseUrl}/api/v1/word-lists/parse`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ raw_text: wordListDraft.itemsText.trim() }),
      })

      const parsedItems = Array.isArray(parseData.items) ? parseData.items : []
      if (parsedItems.length === 0) {
        throw new Error("未能从输入中识别出有效词项。")
      }

      const langMap = { "英语": "en", "语文": "zh" }
      const language = langMap[wordListDraft.language] || "en"

      // 2. 将解析后的 items 提交保存
      const data = await requestJSON(`${apiBaseUrl}/api/v1/word-lists`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          family_id: Number(familyId),
          child_id: Number(assigneeId),
          assigned_date: wordListDraft.assignedDate || assignedDate,
          title: wordListDraft.title.trim(),
          language,
          items: parsedItems.map(item => ({
            text: item.text,
            meaning: item.meaning || "",
          }))
        }),
      })

      const savedList = normalizeWordListFromApi(data.word_list)
      setWordLists((current) => upsertWordListCollection(current, savedList))
      setWordListSyncMeta(savedList.id, {
        state: "success",
        tone: "success",
        label: "已同步",
        message: "新清单已经保存到服务器。",
      })

      setWordListStatus({
        tone: "success",
        title: "单词清单已创建",
        message: `已为 ${resolveChildLabel(familyId, assigneeId)} 系统拆分并保存清单到服务器。`,
        retryable: false,
      })
      setWordListDraft({
        title: "",
        language: wordListDraft.language,
        assignedDate: assignedDate,
        itemsText: "",
      })
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

  function updateWordListMeta(listId, field, value) {
    let updatedList = null
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
      updatedList = next.find((list) => list.id === listId) || null
      return next
    })
    scheduleWordListSync(updatedList)
  }

  function addWordItem(listId) {
    let updatedList = null
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
      updatedList = next.find((list) => list.id === listId) || null
      return next
    })
    scheduleWordListSync(updatedList)
  }

  function updateWordItem(listId, itemId, field, value) {
    let updatedList = null
    setWordLists((current) => {
      const next = current.map((list) =>
        list.id === listId
          ? {
            ...list,
            items: list.items.map((item) =>
              item.id === itemId
                ? {
                  ...item,
                  [field]: value,
                }
                : item,
            ),
            updated_at: new Date().toISOString(),
          }
          : list,
      )
      updatedList = next.find((list) => list.id === listId) || null
      return next
    })
    scheduleWordListSync(updatedList)
  }

  function removeWordItem(listId, itemId) {
    let updatedList = null
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
      updatedList = next.find((list) => list.id === listId) || null
      return next
    })
    scheduleWordListSync(updatedList)
  }

  function removeWordList(listId) {
    clearWordListSyncTimer(listId)
    wordListSyncTokensRef.current.delete(listId)
    setWordListSyncState((current) => {
      const next = { ...current }
      delete next[listId]
      return next
    })
    setWordLists((current) => current.filter((list) => list.id !== listId))
  }

  function syncWordListNow(listId) {
    const targetList = wordLists.find((list) => list.id === listId)
    if (!targetList) {
      return
    }
    void persistWordList(targetList)
  }

  const latestDictationResult = dictationSummary.latestSession?.grading_result || null
  const consoleSectionItems = CONSOLE_SECTIONS.map((section) => {
    switch (section.id) {
      case "publish-console":
        return {
          ...section,
          metric: selectedDraftTasks.length > 0 ? `${selectedDraftTasks.length} 条待发` : `${todayTasks.length} 条任务`,
          caption:
            selectedDraftTasks.length > 0
              ? "草稿已选中，继续确认即可发布。"
              : todayTasks.length > 0
                ? `当前日期 ${assignedDate} 已有任务板。`
                : "先选孩子和日期，再粘贴群消息解析。",
          tone: selectedDraftTasks.length > 0 ? "focus" : "calm",
        }
      case "report-console":
        return {
          ...section,
          metric:
            dictationSummary.pending > 0
              ? `${dictationSummary.pending} 次处理中`
              : latestDictationResult
                ? `${latestDictationResult.score} 分听写`
                : `${dailyReport.completionRate}% 完成`,
          caption:
            dictationSummary.total > 0
              ? "日报、异步听写和周月趋势集中查看。"
              : "当天日报、趋势和听写结果都在这里。",
          tone: dictationSummary.pending > 0 ? "warn" : "calm",
        }
      case "points-console":
        return {
          ...section,
          metric: formatSignedAmount(dailyReport.pointDelta),
          caption: currentDatePointEntries.length > 0 ? "今日已有积分记录，继续补充或修正。" : "手机上直接加分、扣分并回看流水。",
          tone: Number(dailyReport.pointDelta) < 0 ? "warn" : "calm",
        }
      case "word-console":
        return {
          ...section,
          metric: `${scopedWordLists.length} 份清单`,
          caption: scopedWordLists.length > 0 ? "继续维护绑定日期和词项，Pad 端可直接使用。" : "按日期建清单，给 Pad 默写与拍照批改使用。",
          tone: scopedWordLists.length > 0 ? "accent" : "calm",
        }
      default:
        return {
          ...section,
          metric: "--",
          caption: "",
          tone: "calm",
        }
    }
  })
  const activeConsoleMeta = consoleSectionItems.find((item) => item.id === activeConsoleSection) || consoleSectionItems[0]
  const consoleSectionOrder = consoleSectionItems.reduce((result, item, index) => {
    result[item.id] = item.id === activeConsoleSection ? 1 : index + 2
    return result
  }, {})
  const publishStageItems = [
    {
      id: "compose",
      index: "01",
      label: "录入原文",
      caption:
        draftTasks.length > 0
          ? `当前原文 ${rawText.trim().length} 字，可随时返回修改后重跑解析。`
          : "先选日期、粘贴老师消息，再发起解析。",
    },
    {
      id: "review",
      index: "02",
      label: "审核草稿",
      caption:
        draftTasks.length > 0
          ? `AI 草稿 ${draftTasks.length} 条，已选 ${selectedDraftTasks.length} 条。`
          : "解析完成后，这里集中处理风险项和勾选发布。",
    },
    {
      id: "release",
      index: "03",
      label: "发布完成",
      caption:
        createdTasks.length > 0
          ? `本次已发布 ${createdTasks.length} 条，可以继续去反馈或积分区。`
      : "发布成功后，这里会回到结果和下一步动作。",
    },
  ]
  const publishPanelItems = [
    {
      id: "scope",
      label: "范围",
      metric: assignedDate,
      caption: "先定孩子和日期",
    },
    {
      id: "compose",
      label: "原文",
      metric: `${rawText.trim().length} 字`,
      caption: "录入老师原文",
    },
    {
      id: "review",
      label: "审核",
      metric: `${draftTasks.length} 张`,
      caption: riskyDraftTaskCount > 0 ? `${riskyDraftTaskCount} 张优先处理` : "草稿检查",
    },
    {
      id: "release",
      label: "发布",
      metric: `${selectedDraftTasks.length} 条`,
      caption: selectedBlockingDraftCount > 0 ? `${selectedBlockingDraftCount} 条阻断` : "确认下发",
    },
    {
      id: "split",
      label: "拆分",
      metric: `${previewSections.length} 组`,
      caption: "老师原文拆分",
    },
    {
      id: "preview",
      label: "任务",
      metric: `${previewTasks.length} 条`,
      caption: "准备生成的任务",
    },
    {
      id: "analysis",
      label: "摘要",
      metric: parserMode ? "已生成" : "待解析",
      caption: "AI 读题摘要",
    },
    {
      id: "board",
      label: "任务板",
      metric: `${todayTasks.length} 条`,
      caption: createdTasks.length > 0 ? `${createdTasks.length} 条刚发布` : "当天已发布任务",
    },
  ]
  let publishDockConfig

  if (publishStage === "review") {
    publishDockConfig = {
      stage: "review",
      badge: `${selectedDraftTasks.length} 条已选 / ${selectedBlockingDraftCount} 条阻断`,
      title: "审核后直接发布",
      caption:
        selectedBlockingDraftCount > 0
          ? "当前选中项里还有阻断任务，先跳回去修正。"
          : selectedDraftTasks.length > 0
            ? "底部保留快捷发布入口，手机上不用反复滚到确认区。"
            : "先在草稿卡里勾选要发布的任务，再从这里继续。",
      primaryAction: {
        label: isConfirming ? "快捷发布中..." : `快捷发布 (${selectedDraftTasks.length})`,
        onClick: () => void runConfirmCreate(),
        disabled: isConfirming || selectedDraftTasks.length === 0 || selectedBlockingDraftCount > 0,
        emphasis: "secondary",
      },
      secondaryAction: {
        label: riskyDraftTaskCount > 0 ? `先看风险 (${riskyDraftTaskCount})` : "查看已选任务",
        onClick: () =>
          jumpToDraftReview(
            riskyDraftTaskCount > 0 ? "risk" : "selected",
            (riskyDraftTaskCount > 0
              ? draftTasks.find((task) => {
                const riskMeta = getDraftTaskRiskMeta(task, diagnostics.byId[task.id] || { issues: [], hasBlocking: false })
                return riskMeta.order <= 1
              })
              : selectedDraftTasks[0]
            )?.id,
          ),
        disabled: riskyDraftTaskCount === 0 && selectedDraftTasks.length === 0,
      },
      tertiaryAction: {
        label: "回到原文",
        onClick: () => {
          handlePublishPanelChange("compose")
        },
        disabled: false,
      },
    }
  } else if (publishStage === "release") {
    publishDockConfig = {
      stage: "release",
      badge: `${createdTasks.length} 条已发`,
      title: "发布完成，继续下一步",
      caption: "手机上常见的后续动作固定在底部，直接去反馈、积分，或开始下一批发布。",
      primaryAction: {
        label: "快捷看反馈",
        onClick: () => jumpToConsoleSection("report-console"),
        disabled: false,
        emphasis: "secondary",
      },
      secondaryAction: {
        label: "快捷去积分",
        onClick: () => jumpToConsoleSection("points-console"),
        disabled: false,
      },
      tertiaryAction: {
        label: "继续下一批",
        onClick: () => {
          handlePublishPanelChange("scope")
        },
        disabled: false,
      },
    }
  } else {
    publishDockConfig = {
      stage: "compose",
      badge: `${rawText.trim().length} 字原文 / ${previewTasks.length} 条预览`,
      title: "先解析老师原文",
      caption: "手机端底部固定保留解析入口，当前阶段优先把原文送进后端生成草稿。",
      primaryAction: {
        label: isSubmitting ? "快捷解析中..." : "快捷解析",
        onClick: () => void runParse(),
        disabled: isSubmitting,
        emphasis: undefined,
      },
      secondaryAction: {
        label: "填入参考任务",
        onClick: () => setRawText(REFERENCE_GROUP_MESSAGE),
        disabled: false,
      },
      tertiaryAction: {
        label: isRefreshing ? "刷新中..." : "刷新当日任务",
        onClick: refreshTasks,
        disabled: isRefreshing,
      },
    }
  }

  return (
    <main className="app-shell">
      <section className="hero">
        <div className="hero-topline">
          <p className="eyebrow">StudyClaw Parent H5</p>
          <span className="hero-date-pill">{assignedDate}</span>
        </div>
        <div className="hero-main">
          <div>
            <h1>{childLabel} 的家长移动工位</h1>
            <p className="hero-copy">
              现在默认按手机 H5 的节奏组织操作路径: 先把 {assignedDate} 的任务发出去，再顺手切到反馈、积分或单词维护；当前主屏是
              {activeConsoleMeta?.label || "发布作业"}。
            </p>
          </div>
          <div className="hero-focus-strip">
            <article className="hero-focus-card tone-focus">
              <span>当前主屏</span>
              <strong>{activeConsoleMeta?.label || "发布作业"}</strong>
              <p>{activeConsoleMeta?.caption}</p>
            </article>
            <article className="hero-focus-card tone-calm">
              <span>今日摘要</span>
              <strong>{dailyReport.summary}</strong>
              <p>
                {currentDatePointEntries.length > 0
                  ? `今天已有 ${currentDatePointEntries.length} 条积分流水，单词清单 ${scopedWordLists.length} 份。`
                  : `今天还没有积分流水，当前已绑定 ${scopedWordLists.length} 份单词清单。`}
              </p>
            </article>
          </div>
        </div>
        <div className="hero-stats">
          <StatCard label="当前孩子" value={childLabel} hint={`家庭 ${familyId} / 孩子 ${assigneeId}`} />
          <StatCard label="当日完成" value={`${dailyReport.completed}/${dailyReport.total}`} hint={todayDate || assignedDate} />
          <StatCard label="今日积分" value={formatSignedAmount(dailyReport.pointDelta)} hint="手工积分来自最近记录" />
          <StatCard label="单词清单" value={scopedWordLists.length} hint="当前孩子已绑定的清单数" />
        </div>
      </section>

      <ConsoleNav
        items={consoleSectionItems}
        activeSection={activeConsoleSection}
        onJump={jumpToConsoleSection}
        childLabel={childLabel}
        assignedDate={assignedDate}
        pointDelta={dailyReport.pointDelta}
        wordListCount={scopedWordLists.length}
      />

      <section className="console-flow" aria-label="家长移动工位主流程">
        <ScreenSubpanelDeck activeId={activeConsoleSection} orderedIds={CONSOLE_PANEL_ORDER} className="console-root-deck">
          {(getConsolePanelClass) => (
            <>
              <section className={`${getConsolePanelClass("publish-console")} console-root-screen`}>
                <section className="workspace workspace-publish" id="publish-console">
                  <section className="panel current-screen-intro">
                    <div className="panel-heading">
                      <div>
                        <h2>今天先把作业发出去</h2>
                        <p className="panel-caption">发布区已经拆成多个子页面，按顺序切换，不再把所有模块堆在同一长页里。</p>
                      </div>
                      <span>{selectedDraftTasks.length > 0 ? `${selectedDraftTasks.length} 条待发` : `${todayTasks.length} 条任务`}</span>
                    </div>
                    <div className="compact-stage-strip" role="tablist" aria-label="发布主路径步骤">
                      <button
                        className={`compact-stage-pill ${publishStage === "compose" ? "is-active" : ""}`}
                        role="tab"
                        type="button"
                        onClick={() => handlePublishStageChange("compose")}
                      >
                        01 录入
                      </button>
                      <button
                        className={`compact-stage-pill ${publishStage === "review" ? "is-active" : ""}`}
                        role="tab"
                        type="button"
                        onClick={() => handlePublishStageChange("review")}
                      >
                        02 审核
                      </button>
                      <button
                        className={`compact-stage-pill ${publishStage === "release" ? "is-active" : ""}`}
                        role="tab"
                        type="button"
                        onClick={() => handlePublishStageChange("release")}
                      >
                        03 发布
                      </button>
                    </div>
                  </section>

                  <SubMenuStrip items={publishPanelItems} activeItem={activePublishPanel} onChange={handlePublishPanelChange} ariaLabel="发布主屏子菜单" />

                  <ScreenSubpanelDeck activeId={activePublishPanel} orderedIds={PUBLISH_PANEL_ORDER}>
                    {(getPanelClass) => (
                      <>
                        <div className={getPanelClass("scope")}>
                          <section className="column-stack">
                            <section className="panel">
                              <div className="panel-heading">
                                <div>
                                  <h2>先定孩子和日期</h2>
                                  <p className="panel-caption">先锁定今天操作范围，再去录入原文、审核和发布。</p>
                                </div>
                                <span>{childLabel}</span>
                              </div>

                              <AssignmentScopePanel
                                childProfiles={CHILD_PROFILES}
                                selectedChildPreset={selectedChildPreset}
                                onChildPresetChange={handleChildPresetChange}
                                assignedDate={assignedDate}
                                onAssignedDateChange={setAssignedDate}
                                onShiftDate={(days) => setAssignedDate((current) => shiftDate(current, days))}
                                onUseToday={() => setAssignedDate(formatDateInputValue())}
                                childLabel={childLabel}
                                familyId={familyId}
                                assigneeId={assigneeId}
                                apiBaseUrl={apiBaseUrl}
                                onApiBaseUrlChange={setApiBaseUrl}
                                onFamilyIdChange={setFamilyId}
                                onAssigneeIdChange={setAssigneeId}
                                todayTaskCount={todayTasks.length}
                                selectedDraftCount={selectedDraftTasks.length}
                                pointDelta={dailyReport.pointDelta}
                                dictationSummary={dictationSummary}
                              />

                              <div className="action-row">
                                <button className="primary-button secondary" type="button" onClick={() => handlePublishPanelChange("compose")}>
                                  去录入原文
                                </button>
                                <button className="ghost-button" type="button" onClick={() => handlePublishPanelChange("board")}>
                                  先看任务板
                                </button>
                              </div>
                            </section>
                          </section>
                        </div>

                        <div className={getPanelClass("compose")}>
                          <form className="panel composer" onSubmit={handleSubmit}>
                            <div className="panel-heading">
                              <div>
                                <h2>录入老师原文</h2>
                                <p className="panel-caption">这页只做原文录入和解析，不再和范围设置、审核、预览堆在一起。</p>
                              </div>
                              <span>{assignedDate}</span>
                            </div>

                            <div className="local-scope-card">
                              <strong>当前发布范围</strong>
                              <p>{childLabel} / 家庭 {familyId} / 孩子 {assigneeId} / 日期 {assignedDate}</p>
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

                            <div className="action-row composer-actions">
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
                              secondaryActionLabel={previewTasks.length > 0 ? "去看原文拆分" : undefined}
                              onSecondaryAction={previewTasks.length > 0 ? () => handlePublishPanelChange("split") : undefined}
                              secondaryActionDisabled={previewTasks.length === 0}
                            />

                            {refreshError ? <p className="inline-hint hint-error">刷新当日任务失败: {refreshError}</p> : null}
                          </form>
                        </div>

                        <div className={getPanelClass("review")}>
                          <DraftTaskList
                            tasks={draftTasks}
                            selectedTaskIds={selectedTaskIds}
                            diagnosticsById={diagnostics.byId}
                            activeReviewFilter={draftReviewFilter}
                            onActiveReviewFilterChange={setDraftReviewFilter}
                            activeDraftTaskId={draftFocusTaskId}
                            onActiveDraftTaskIdChange={setDraftFocusTaskId}
                            onToggle={toggleSelectedTask}
                            onFieldChange={updateDraftTask}
                            onRemove={removeDraftTask}
                            onSelectRecommended={selectRecommendedTasks}
                            onSelectCleanTasks={selectCleanTasks}
                            onSelectAll={selectAllTasks}
                            onClearSelection={clearTaskSelection}
                            onAddManualTask={addManualTask}
                          />
                        </div>

                        <div className={getPanelClass("release")}>
                          <section className="column-stack">
                            <CreatePanel
                              diagnosticsSummary={diagnostics.summary}
                              selectedTasks={selectedDraftTasks}
                              diagnosticsById={diagnostics.byId}
                              recommendedCount={recommendedTaskCount}
                              assignedDate={assignedDate}
                              onSelectRecommended={selectRecommendedTasks}
                              onJumpToDraftReview={jumpToDraftReview}
                              onConfirm={() => void runConfirmCreate()}
                              isConfirming={isConfirming}
                              createStatus={createStatus}
                            />
                            {createdTasks.length > 0 ? (
                              <section className="panel publish-next-actions">
                                <div className="panel-heading">
                                  <div>
                                    <h3>发布后下一步</h3>
                                    <p className="panel-caption">发布完成后直接切反馈、积分或回到新一轮录入。</p>
                                  </div>
                                  <span>{createdTasks.length} 条已发</span>
                                </div>
                                <div className="action-row">
                                  <button className="ghost-button" type="button" onClick={() => jumpToConsoleSection("report-console")}>
                                    去看反馈
                                  </button>
                                  <button className="ghost-button" type="button" onClick={() => jumpToConsoleSection("points-console")}>
                                    去调积分
                                  </button>
                                  <button className="ghost-button" type="button" onClick={() => jumpToConsoleSection("word-console")}>
                                    去看单词
                                  </button>
                                  <button className="primary-button secondary" type="button" onClick={() => handlePublishPanelChange("scope")}>
                                    继续下一批
                                  </button>
                                </div>
                              </section>
                            ) : null}
                          </section>
                        </div>

                        <div className={getPanelClass("split")}>
                          <section className="panel publish-preview-sheet">
                            <div className="panel-heading">
                              <div>
                                <h2>老师原文拆分</h2>
                                <p className="panel-caption">先看群消息被拆成哪些学科和子步骤，再决定是否回去改原文。</p>
                              </div>
                              <span>{previewSections.length} 组</span>
                            </div>
                            <SectionPreview sections={previewSections} />
                          </section>
                        </div>

                        <div className={getPanelClass("preview")}>
                          <ServerTaskList
                            title="准备生成的任务"
                            tasks={previewTasks}
                            emptyText="等待识别群消息内容。"
                            caption="这里是本地结构任务预览，不代表已经写入后端。"
                          />
                        </div>

                        <div className={getPanelClass("analysis")}>
                          <AnalysisPanel parserMode={parserMode} analysis={analysis} />
                        </div>

                        <div className={getPanelClass("board")}>
                          <section className="workspace secondary workspace-snapshot">
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
                        </div>
                      </>
                    )}
                  </ScreenSubpanelDeck>

                  <HelpAccordion
                    className="console-rulebook"
                    title="支持的录入规则"
                    caption="把学校群常见格式压成简明规则，收进发布主屏内，避免单独占用整段页面。"
                    badge="录入说明"
                  >
                    <ul className="rule-list">
                      <li>支持 `数学3.6：`、`英：`、`语文：` 这种带日期或简称的学科标题。</li>
                      <li>支持 `1、`、`1.` 作为主任务编号。</li>
                      <li>支持 `（1）`、`（2）`、`（3）` 作为同一任务下的补充步骤。</li>
                      <li>提交后调用 `/api/v1/tasks/parse`，审核后调用 `/api/v1/tasks/confirm`，并按当前日期刷新 `/api/v1/tasks`。</li>
                      <li>“刷新听写批改”调用 `/api/v1/dictation-sessions`；若存在处理中会自动轮询最新结果。</li>
                      <li>“查看周趋势”调用 `/api/v1/stats/weekly`；“查看月趋势”调用 `/api/v1/stats/monthly`。</li>
                      <li>积分通过 `/api/v1/points/ledger` 创建和查询；单词清单通过 `/api/v1/word-lists/parse` 与 `/api/v1/word-lists` 管理。</li>
                    </ul>
                  </HelpAccordion>
                </section>

                <PublishActionDock
                  isVisible={activeConsoleSection === "publish-console" && publishDockConfig.stage !== "compose"}
                  stage={publishDockConfig.stage}
                  badge={publishDockConfig.badge}
                  title={publishDockConfig.title}
                  caption={publishDockConfig.caption}
                  primaryAction={publishDockConfig.primaryAction}
                  secondaryAction={publishDockConfig.secondaryAction}
                  tertiaryAction={publishDockConfig.tertiaryAction}
                />
              </section>

              <section className={`${getConsolePanelClass("report-console")} console-root-screen`}>
                <FeedbackPanel
                  assignedDate={assignedDate}
                  childLabel={childLabel}
                  todayTasks={todayTasks}
                  currentDatePointEntries={currentDatePointEntries}
                  dictationSessions={dictationSessions}
                  dictationStatus={dictationStatus}
                  weeklyStats={weeklyStats}
                  weeklyStatus={weeklyStatus}
                  monthlyRows={monthlyRows}
                  monthlyStatus={monthlyStatus}
                  reportView={reportView}
                  onLoadDictation={() => void refreshDictationSessions()}
                  onLoadWeekly={() => void loadWeeklyStats()}
                  onLoadMonthly={() => void loadMonthlyTrend()}
                  isLoadingDictation={isLoadingDictation}
                  isLoadingWeekly={isLoadingWeekly}
                  isLoadingMonthly={isLoadingMonthly}
                />
              </section>

              <section className={`${getConsolePanelClass("points-console")} console-root-screen`}>
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

              <section className={`${getConsolePanelClass("word-console")} console-root-screen`}>
                <section className="console-panel-stack">
                  <WordListPanel
                    childLabel={childLabel}
                    familyId={familyId}
                    assigneeId={assigneeId}
                    assignedDate={assignedDate}
                    wordListDraft={wordListDraft}
                    wordLists={scopedWordLists}
                    wordListStatus={wordListStatus}
                    wordListSyncState={wordListSyncState}
                    onDraftChange={updateWordListDraft}
                    onUseAssignedDate={useAssignedDateForWordList}
                    onCreate={createWordListDraft}
                    onRetry={createWordListDraft}
                    onUpdateWordListMeta={updateWordListMeta}
                    onAddWordItem={addWordItem}
                    onUpdateWordItem={updateWordItem}
                    onRemoveWordItem={removeWordItem}
                    onRemoveWordList={removeWordList}
                    onSyncWordListNow={syncWordListNow}
                  />
                  <HelpAccordion
                    title="当前冻结说明"
                    caption="避免前端重新定义后端语义，同时把联调用法压缩成手机上可随时展开的说明。"
                    badge="联调说明"
                  >
                    <ul className="rule-list">
                      <li>风险高亮只消费现有 `needs_review`、`confidence` 和任务去重诊断，不改后端字段含义。</li>
                      <li>{"发布链路固定为 `parse -> 审核编辑 -> confirm -> 按日期刷新 tasks`。"}</li>
                      <li>日报摘要由前端基于当日任务和积分记录确定性生成；周 / 月趋势走真实统计接口。</li>
                      <li>听写批改结果来自真实 `/api/v1/dictation-sessions`，与孩子 Pad 端共享同一异步结果源。</li>
                      <li>积分操作走真实 `/api/v1/points/ledger` 创建与查询接口，当前日期会显式传给后端。</li>
                      <li>单词清单走真实 `/api/v1/word-lists/parse` 与 `/api/v1/word-lists`，前端只做字段映射和编辑回写。</li>
                      <li>失败重试保持现有体验：解析失败保留原文和上一轮草稿，发布失败保留草稿与选中项，积分失败保留表单。</li>
                    </ul>
                  </HelpAccordion>
                </section>
              </section>
            </>
          )}
        </ScreenSubpanelDeck>
      </section>

      <MobileDock items={consoleSectionItems} activeSection={activeConsoleSection} onJump={jumpToConsoleSection} />
    </main>
  )
}

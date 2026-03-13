export const REFERENCE_GROUP_MESSAGE = `数学3.6：
1、校本P14～15
2、练习册P12～13

英：
1. 背默M1U1知识梳理单小作文
2. 部分学生继续订正1号本
3. 预习M1U2
（1）书本上标注好“黄页”出现单词的音标
（2）抄写单词（今天默写全对，可免抄）
（3）沪学习听录音跟读

语文：
1. 背作文
2. 练习卷`

const SUBJECT_ALIASES = new Map([
  ["数", "数学"],
  ["数学", "数学"],
  ["英", "英语"],
  ["英语", "英语"],
  ["语", "语文"],
  ["语文", "语文"],
  ["科学", "科学"],
  ["物理", "物理"],
  ["化学", "化学"],
  ["生物", "生物"],
  ["历史", "历史"],
  ["地理", "地理"],
  ["道法", "道法"],
  ["政治", "道法"],
  ["美术", "美术"],
  ["音乐", "音乐"],
  ["体育", "体育"],
  ["信息", "信息"],
  ["劳动", "劳动"],
  ["阅读", "阅读"],
  ["班会", "班会"],
])

const RECITATION_TASK_KEYWORDS = ["背诵", "背默", "默背", "背课文", "背作文", "古诗", "诗词", "熟背", "背会"]
const READING_TASK_KEYWORDS = ["朗读", "跟读", "诵读", "朗诵", "read aloud", "follow reading"]

function normalizeSubject(rawSubject) {
  const trimmed = rawSubject.replace(/\s+/g, "").trim()
  if (!trimmed) {
    return "未分类"
  }

  const direct = SUBJECT_ALIASES.get(trimmed)
  if (direct) {
    return direct
  }

  for (const [key, value] of SUBJECT_ALIASES.entries()) {
    if (trimmed.startsWith(key)) {
      return value
    }
  }

  return trimmed
}

function normalizeTaskType(value) {
  const normalized = String(value || "").trim().toLowerCase()
  switch (normalized) {
    case "":
    case "homework":
      return "homework"
    case "recitation":
    case "memorization":
    case "memorize":
    case "poem_recitation":
    case "classical_poem":
      return "recitation"
    case "reading":
    case "read_aloud":
    case "follow_reading":
    case "english_reading":
      return "reading"
    default:
      return normalized
  }
}

function usesReferenceMaterial(taskType) {
  const normalized = normalizeTaskType(taskType)
  return normalized === "recitation" || normalized === "reading"
}

function inferLearningTaskType(task = {}) {
  const explicitType = normalizeTaskType(task.task_type || task.type || task.taskType)
  if (explicitType !== "homework") {
    return explicitType
  }

  const haystack = `${task.subject || ""} ${task.group_title || task.groupTitle || ""} ${task.title || task.content || ""}`.toLowerCase()
  if (RECITATION_TASK_KEYWORDS.some((keyword) => haystack.includes(keyword.toLowerCase()))) {
    return "recitation"
  }
  if (READING_TASK_KEYWORDS.some((keyword) => haystack.includes(keyword.toLowerCase()))) {
    return "reading"
  }
  return "homework"
}

function normalizeComparisonText(value) {
  return String(value || "")
    .toLowerCase()
    .replace(/[\s`~!@#$%^&*()_\-+=[\]{}\\|;:'",.<>/?，。；：！？、（）【】《》“”‘’·]/g, "")
}

function appendUniqueNote(notes, note) {
  const trimmed = String(note || "").trim()
  const current = Array.isArray(notes) ? [...notes] : []
  if (!trimmed || current.includes(trimmed)) {
    return current
  }
  current.push(trimmed)
  return current
}

function isSubjectHeadingLine(line) {
  const headingMatch = String(line || "").trim().match(/^([\u4e00-\u9fa5A-Za-z]+[^：:]*)[：:]\s*(.*)$/)
  return Boolean(headingMatch && /^[\u4e00-\u9fa5A-Za-z0-9.\-]+$/.test(headingMatch[1].replace(/\s+/g, "")))
}

function isStructuralLine(line) {
  const trimmed = String(line || "").trim()
  if (!trimmed) {
    return false
  }
  if (isSubjectHeadingLine(trimmed)) {
    return true
  }
  return (
    /^\d+\s*[、.．)）]\s*(.+)$/.test(trimmed) ||
    /^[（(]\d+[）)]\s*(.+)$/.test(trimmed) ||
    /^\d+\s*[)）]\s*(.+)$/.test(trimmed) ||
    /^[①②③④⑤⑥⑦⑧⑨⑩]\s*(.+)$/.test(trimmed) ||
    /^[-•·]\s*(.+)$/.test(trimmed)
  )
}

function splitRawMessageLines(rawText) {
  return String(rawText || "")
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => line.trim())
}

function extractQuotedTitle(text) {
  const quoted = String(text || "").match(/《([^》]+)》/)
  if (quoted) {
    return quoted[1].trim()
  }

  const actionTitle = String(text || "").match(/(?:背诵|朗读|跟读|诵读|朗诵)\s*[《“"]?([^》”"，。；、\s]{2,30})[》”"]?/)
  if (actionTitle) {
    return actionTitle[1].trim()
  }

  return ""
}

function parseReferenceHeader(firstLine, fallbackTitle = "") {
  const line = String(firstLine || "").trim()
  if (!line) {
    return { title: fallbackTitle.trim(), author: "" }
  }

  const quotedTitle = extractQuotedTitle(line)
  const authorWithMarker = line.match(/^(.*?)[【\[\(（〔][^】\]\)）〕]{1,12}[】\]\)）〕]\s*([^\s]{1,16})$/)
  if (authorWithMarker) {
    return {
      title: quotedTitle || authorWithMarker[1].replace(/[《》]/g, "").trim() || fallbackTitle.trim(),
      author: authorWithMarker[2].trim(),
    }
  }

  const authorTagged = line.match(/^作者[:：]\s*([^\s]{1,16})$/)
  if (authorTagged) {
    return { title: fallbackTitle.trim(), author: authorTagged[1].trim() }
  }

  if (quotedTitle) {
    return { title: quotedTitle, author: "" }
  }

  if (!/[，。！？；,.!?]/.test(line) && line.length <= 24) {
    return { title: line.replace(/[《》]/g, "").trim() || fallbackTitle.trim(), author: "" }
  }

  return { title: fallbackTitle.trim(), author: "" }
}

function looksLikeReferenceBlock(lines) {
  if (!Array.isArray(lines) || lines.length === 0) {
    return false
  }

  const joined = lines.join("\n").trim()
  if (joined.length < 12) {
    return false
  }

  return (
    lines.length >= 2 ||
    /[，。！？；,.!?]/.test(joined) ||
    /《[^》]+》/.test(joined) ||
    /[【〔\[](.*?)[】〕\]]/.test(joined)
  )
}

function collectReferenceBlockAfterLine(lines, anchorIndex) {
  if (anchorIndex < 0) {
    return []
  }

  const block = []
  let started = false
  for (let index = anchorIndex + 1; index < lines.length; index += 1) {
    const trimmed = String(lines[index] || "").trim()
    if (!trimmed) {
      if (started) {
        break
      }
      continue
    }

    if (isStructuralLine(trimmed)) {
      if (started) {
        break
      }
      continue
    }

    started = true
    block.push(trimmed)
  }

  return looksLikeReferenceBlock(block) ? block : []
}

function extractLooseReferenceBlocks(lines) {
  const blocks = []
  let current = []
  let startIndex = -1

  function flush() {
    if (looksLikeReferenceBlock(current)) {
      blocks.push({ startIndex, lines: [...current] })
    }
    current = []
    startIndex = -1
  }

  lines.forEach((line, index) => {
    const trimmed = String(line || "").trim()
    if (!trimmed || isStructuralLine(trimmed)) {
      flush()
      return
    }

    if (current.length === 0) {
      startIndex = index
    }
    current.push(trimmed)
  })

  flush()
  return blocks
}

function findBestAnchorIndex(lines, task) {
  const searchTerms = [
    task.title,
    task.group_title,
    extractQuotedTitle(task.title),
    extractQuotedTitle(task.group_title),
  ]
    .map((item) => String(item || "").trim())
    .filter(Boolean)

  let bestIndex = -1
  let bestScore = -1

  lines.forEach((line, index) => {
    const normalizedLine = normalizeComparisonText(line)
    if (!normalizedLine) {
      return
    }

    searchTerms.forEach((term) => {
      const normalizedTerm = normalizeComparisonText(term)
      if (!normalizedTerm || !normalizedLine.includes(normalizedTerm)) {
        return
      }

      const score = normalizedTerm.length * 10 + (isStructuralLine(line) ? 5 : 0)
      if (score > bestScore) {
        bestScore = score
        bestIndex = index
      }
    })
  })

  return bestIndex
}

function findReferenceBlockForTask(lines, task, extractedTitle) {
  const anchorIndex = findBestAnchorIndex(lines, task)
  const directBlock = collectReferenceBlockAfterLine(lines, anchorIndex)
  if (directBlock.length > 0) {
    return directBlock
  }

  const titleHint = normalizeComparisonText(extractedTitle)
  if (!titleHint) {
    return []
  }

  const looseBlocks = extractLooseReferenceBlocks(lines)
  const matchedBlock = looseBlocks.find((block) => normalizeComparisonText(block.lines[0]).includes(titleHint))
  return matchedBlock ? matchedBlock.lines : []
}

function inferAnalysisMode(taskType, referenceText, title, author) {
  const normalizedType = normalizeTaskType(taskType)
  const lines = String(referenceText || "")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
  const joined = lines.join("")

  if (normalizedType === "reading") {
    return /[A-Za-z]/.test(joined) ? "english_reading" : "read_aloud"
  }

  if (normalizedType !== "recitation") {
    return ""
  }

  const shortLineCount = lines.filter((line) => line.length <= 18).length
  const hasDynastyMarker = /[【〔\[][^\]】〕]{1,12}[\]】〕]/.test(lines[0] || "")
  if ((hasDynastyMarker || author) && lines.length >= 2 && shortLineCount >= Math.min(lines.length, 3)) {
    return "classical_poem"
  }
  if (title && lines.length >= 2 && shortLineCount === lines.length && /[，。！？；]/.test(joined)) {
    return "classical_poem"
  }
  return "text_recitation"
}

export function enrichTasksWithLearningReferences(rawText, tasks) {
  const lines = splitRawMessageLines(rawText)

  return (Array.isArray(tasks) ? tasks : []).map((task) => {
    const taskType = inferLearningTaskType(task)
    const fallbackTitle = extractQuotedTitle(task.title || task.group_title || "")
    let referenceText = String(task.reference_text || task.referenceText || "").trim()
    let referenceTitle = String(task.reference_title || task.referenceTitle || "").trim()
    let referenceAuthor = String(task.reference_author || task.referenceAuthor || "").trim()
    let notes = Array.isArray(task.notes) ? [...task.notes] : []

    if (usesReferenceMaterial(taskType) && !referenceText) {
      const blockLines = findReferenceBlockForTask(lines, task, fallbackTitle)
      if (blockLines.length > 0) {
        referenceText = blockLines.join("\n")
        notes = appendUniqueNote(notes, "已从老师原文自动带出参考内容。")
      }
    }

    if (usesReferenceMaterial(taskType)) {
      const header = parseReferenceHeader(
        String(referenceText || "").split("\n").map((line) => line.trim()).filter(Boolean)[0] || "",
        fallbackTitle,
      )
      if (!referenceTitle && header.title) {
        referenceTitle = header.title
      }
      if (!referenceTitle && fallbackTitle) {
        referenceTitle = fallbackTitle
      }
      if (!referenceAuthor && header.author) {
        referenceAuthor = header.author
      }
    }

    const analysisMode =
      String(task.analysis_mode || task.analysisMode || "").trim() ||
      (usesReferenceMaterial(taskType) && referenceText
        ? inferAnalysisMode(taskType, referenceText, referenceTitle, referenceAuthor)
        : "")

    return {
      ...task,
      type: taskType,
      task_type: taskType,
      reference_title: referenceTitle,
      reference_author: referenceAuthor,
      reference_text: referenceText,
      hide_reference_from_child: usesReferenceMaterial(taskType) && referenceText ? (taskType === "recitation" ? true : Boolean(task.hide_reference_from_child)) : false,
      analysis_mode: analysisMode,
      notes,
    }
  })
}

function flushTask(section, task) {
  if (!section || !task || !task.text.trim()) {
    return null
  }

  const normalizedTask = {
    text: task.text.trim(),
    subitems: task.subitems.filter(Boolean).map((item) => item.trim()),
  }

  section.items.push(normalizedTask)
  return null
}

function ensureSection(sections, subject) {
  let section = sections.find((item) => item.subject === subject)
  if (!section) {
    section = { subject, items: [] }
    sections.push(section)
  }

  return section
}

export function flattenSectionsToTasks(sections) {
  return sections.flatMap((section) =>
    section.items.flatMap((item) => {
      const groupTitle = item.text.trim()
      const atomicTitles = item.subitems.length > 0 ? item.subitems : [item.text]

      return atomicTitles.map((atomicTitle) => ({
        subject: section.subject,
        group_title: groupTitle,
        title: atomicTitle,
      }))
    }),
  )
}

export function parseSchoolTaskMessage(rawText) {
  const lines = rawText
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)

  const sections = []
  let currentSection = null
  let currentTask = null

  for (const line of lines) {
    const headingMatch = line.match(/^([\u4e00-\u9fa5A-Za-z]+[^：:]*)[：:]\s*(.*)$/)
    if (headingMatch && /^[\u4e00-\u9fa5A-Za-z0-9.\-]+$/.test(headingMatch[1].replace(/\s+/g, ""))) {
      currentTask = flushTask(currentSection, currentTask)
      currentSection = ensureSection(sections, normalizeSubject(headingMatch[1]))
      const remainder = headingMatch[2].trim()
      if (remainder) {
        currentTask = { text: remainder, subitems: [] }
      }
      continue
    }

    const mainItemMatch = line.match(/^\d+\s*[、.．]\s*(.+)$/)
    if (mainItemMatch) {
      currentTask = flushTask(currentSection, currentTask)
      if (!currentSection) {
        currentSection = ensureSection(sections, "未分类")
      }
      currentTask = { text: mainItemMatch[1], subitems: [] }
      continue
    }

    const subItemMatch = line.match(/^[（(]\d+[）)]\s*(.+)$/)
    if (subItemMatch) {
      if (!currentSection) {
        currentSection = ensureSection(sections, "未分类")
      }

      if (!currentTask) {
        currentTask = { text: "补充说明", subitems: [] }
      }

      currentTask.subitems.push(subItemMatch[1])
      continue
    }

    if (!currentSection) {
      currentSection = ensureSection(sections, "未分类")
    }

    if (!currentTask) {
      currentTask = { text: line, subitems: [] }
      continue
    }

    if (currentTask.subitems.length > 0) {
      const lastIndex = currentTask.subitems.length - 1
      currentTask.subitems[lastIndex] = `${currentTask.subitems[lastIndex]} ${line}`.trim()
    } else {
      currentTask.text = `${currentTask.text} ${line}`.trim()
    }
  }

  flushTask(currentSection, currentTask)
  return sections
}

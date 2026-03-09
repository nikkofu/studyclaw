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

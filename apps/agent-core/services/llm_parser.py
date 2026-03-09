import json
import os
import re
from typing import Any

try:
    from langchain_core.prompts import PromptTemplate
    from langchain_openai import ChatOpenAI
except ImportError:
    PromptTemplate = None
    ChatOpenAI = None

LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_BASE_URL = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")

SUBJECT_ALIASES = {
    "数": "数学",
    "数学": "数学",
    "英": "英语",
    "英语": "英语",
    "语": "语文",
    "语文": "语文",
    "科学": "科学",
    "物理": "物理",
    "化学": "化学",
    "生物": "生物",
    "历史": "历史",
    "地理": "地理",
    "道法": "道法",
    "政治": "道法",
    "美术": "美术",
    "音乐": "音乐",
    "体育": "体育",
    "信息": "信息",
    "劳动": "劳动",
    "阅读": "阅读",
    "班会": "班会",
}

HEADING_PATTERN = re.compile(r"^([\u4e00-\u9fa5A-Za-z]+[^：:]*)[：:]\s*(.*)$")
MAIN_ITEM_PATTERN = re.compile(r"^\d+\s*[、.．]\s*(.+)$")
SUB_ITEM_PATTERN = re.compile(r"^[（(]\d+[）)]\s*(.+)$")


def normalize_subject(raw_subject: str) -> str:
    subject = re.sub(r"\s+", "", raw_subject).strip()
    if not subject:
        return "未分类"

    if subject in SUBJECT_ALIASES:
        return SUBJECT_ALIASES[subject]

    for key, value in SUBJECT_ALIASES.items():
        if subject.startswith(key):
            return value

    return subject


def build_task_title(text: str, subitems: list[str]) -> str:
    base = text.strip()
    children = [item.strip() for item in subitems if item.strip()]
    if not children:
        return base
    return f"{base}：" + "；".join(children)


def contains_review_signal(*values: str) -> bool:
    review_keywords = ("部分学生", "可免", "选做", "如需", "如果", "酌情")
    combined = " ".join(value.strip() for value in values if value and value.strip())
    return any(keyword in combined for keyword in review_keywords)


def get_llm():
    if not ChatOpenAI or not LLM_API_KEY:
        return None

    return ChatOpenAI(
        api_key=LLM_API_KEY,
        base_url=LLM_BASE_URL,
        model="gpt-4o",
        temperature=0.1,
    )


def extract_structure_outline(raw_text: str) -> dict[str, Any]:
    lines = [line.strip() for line in raw_text.replace("\r\n", "\n").split("\n") if line.strip()]
    sections: list[dict[str, Any]] = []
    current_section = None
    current_task = None
    signals: list[str] = []

    def mark_signal(signal: str):
        if signal not in signals:
            signals.append(signal)

    def ensure_section(subject: str) -> dict[str, Any]:
        normalized_subject = normalize_subject(subject)
        for section in sections:
            if section["subject"] == normalized_subject:
                return section

        new_section = {"subject": normalized_subject, "items": []}
        sections.append(new_section)
        return new_section

    def flush_task():
        nonlocal current_task
        if not current_section or not current_task or not current_task["text"].strip():
            current_task = None
            return

        current_section["items"].append({
            "text": current_task["text"].strip(),
            "subitems": [item.strip() for item in current_task["subitems"] if item.strip()],
        })
        current_task = None

    for line in lines:
        heading_match = HEADING_PATTERN.match(line)
        if heading_match and re.match(r"^[\u4e00-\u9fa5A-Za-z0-9.\-]+$", re.sub(r"\s+", "", heading_match.group(1))):
            mark_signal("subject_headings")
            flush_task()
            current_section = ensure_section(heading_match.group(1))
            remainder = heading_match.group(2).strip()
            current_task = {"text": remainder, "subitems": []} if remainder else None
            continue

        main_item_match = MAIN_ITEM_PATTERN.match(line)
        if main_item_match:
            mark_signal("numbered_tasks")
            flush_task()
            if not current_section:
                current_section = ensure_section("未分类")
            current_task = {"text": main_item_match.group(1).strip(), "subitems": []}
            continue

        sub_item_match = SUB_ITEM_PATTERN.match(line)
        if sub_item_match:
            mark_signal("nested_subtasks")
            if not current_section:
                current_section = ensure_section("未分类")
            if not current_task:
                current_task = {"text": "补充说明", "subitems": []}
            current_task["subitems"].append(sub_item_match.group(1).strip())
            continue

        if "可免" in line or "选做" in line or "部分学生" in line:
            mark_signal("conditional_notes")

        if not current_section:
            current_section = ensure_section("未分类")

        if not current_task:
            current_task = {"text": line, "subitems": []}
            continue

        if current_task["subitems"]:
            current_task["subitems"][-1] = f'{current_task["subitems"][-1]} {line}'.strip()
        else:
            current_task["text"] = f'{current_task["text"]} {line}'.strip()

    flush_task()

    preview_tasks = flatten_sections_to_tasks(sections)
    return {
        "sections": sections,
        "tasks": preview_tasks,
        "detected_subjects": [section["subject"] for section in sections],
        "format_signals": signals,
        "raw_line_count": len(lines),
    }


def flatten_sections_to_tasks(sections: list[dict[str, Any]]) -> list[dict[str, str]]:
    tasks: list[dict[str, Any]] = []
    for section in sections:
        for item in section["items"]:
            group_title = item["text"].strip()
            if not group_title:
                continue

            atomic_titles = [subitem.strip() for subitem in item["subitems"] if subitem.strip()]
            if not atomic_titles:
                atomic_titles = [group_title]

            for atomic_title in atomic_titles:
                notes = []
                if contains_review_signal(group_title, atomic_title):
                    notes.append("包含条件性说明，建议家长确认适用范围。")

                tasks.append({
                    "subject": section["subject"],
                    "group_title": group_title,
                    "title": atomic_title,
                    "type": "homework",
                    "confidence": 0.72 if notes else 0.84,
                    "needs_review": bool(notes),
                    "notes": notes,
                })
    return tasks


def normalize_task_item(item: dict[str, Any]) -> dict[str, Any] | None:
    if not isinstance(item, dict):
        return None

    subject = normalize_subject(str(item.get("subject", "未分类")))
    title = str(item.get("title") or item.get("content") or "").strip()
    title = re.sub(r"\s+", " ", title).strip()
    if not title:
        return None

    group_title = str(item.get("group_title") or item.get("group") or title).strip()
    group_title = re.sub(r"\s+", " ", group_title).strip() or title

    raw_confidence = item.get("confidence", 0.78)
    try:
        confidence = float(raw_confidence)
    except (TypeError, ValueError):
        confidence = 0.78
    confidence = max(0.0, min(1.0, confidence))

    notes = item.get("notes", [])
    if not isinstance(notes, list):
        notes = [str(notes)]
    normalized_notes = [str(note).strip() for note in notes if str(note).strip()]

    needs_review = bool(item.get("needs_review", False))
    if subject == "未分类" or confidence < 0.7 or contains_review_signal(group_title, title):
        needs_review = True

    if needs_review and not normalized_notes and contains_review_signal(group_title, title):
        normalized_notes.append("包含条件性说明，建议家长确认适用范围。")

    return {
        "subject": subject,
        "group_title": group_title,
        "title": title,
        "type": str(item.get("type") or "homework"),
        "confidence": confidence,
        "needs_review": needs_review,
        "notes": normalized_notes,
    }


def normalize_task_list(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    normalized: list[dict[str, Any]] = []
    seen = set()

    for item in items:
        task = normalize_task_item(item)
        if not task:
            continue

        task_key = (task["subject"], task["group_title"], task["title"])
        if task_key in seen:
            continue

        seen.add(task_key)
        normalized.append(task)

    return normalized


def merge_task_lists(primary: list[dict[str, Any]], fallback: list[dict[str, Any]]) -> tuple[list[dict[str, Any]], list[str]]:
    merged = normalize_task_list(primary)
    fallback_normalized = normalize_task_list(fallback)
    existing = {(task["subject"], task["group_title"], task["title"]) for task in merged}
    merged_count = 0

    for task in fallback_normalized:
        task_key = (task["subject"], task["group_title"], task["title"])
        if task_key in existing:
            continue

        existing.add(task_key)
        merged.append(task)
        merged_count += 1

    notes = []
    if merged_count > 0:
        notes.append(f"LLM 结果缺失的 {merged_count} 条任务已由结构兜底补全。")

    return merged, notes


def build_analysis(parser_mode: str, structure: dict[str, Any], tasks: list[dict[str, Any]], notes: list[str] | None = None) -> dict[str, Any]:
    needs_review_count = sum(1 for task in tasks if task.get("needs_review"))
    low_confidence_count = sum(1 for task in tasks if float(task.get("confidence", 0)) < 0.7)
    group_count = len({(task.get("subject"), task.get("group_title")) for task in tasks})

    return {
        "parser_mode": parser_mode,
        "detected_subjects": structure.get("detected_subjects", []),
        "format_signals": structure.get("format_signals", []),
        "raw_line_count": structure.get("raw_line_count", 0),
        "task_count": len(tasks),
        "group_count": group_count,
        "needs_review_count": needs_review_count,
        "low_confidence_count": low_confidence_count,
        "notes": notes or [],
    }


def parse_parent_input_fallback(raw_text: str) -> dict[str, Any]:
    structure = extract_structure_outline(raw_text)
    tasks = normalize_task_list(structure["tasks"])
    status = "success" if tasks else "failed"

    notes = ["当前未使用 LLM，采用结构规则完成任务拆解。"] if tasks else ["未从原文中识别出可创建任务。"]
    return {
        "status": status,
        "parser_mode": "rule_fallback",
        "analysis": build_analysis("rule_fallback", structure, tasks, notes),
        "data": tasks,
    }


def invoke_llm_parser(raw_text: str, structure: dict[str, Any]) -> dict[str, Any]:
    llm = get_llm()
    if llm is None or not PromptTemplate:
        raise RuntimeError("LLM parser is unavailable")

    prompt = PromptTemplate.from_template(
        """
你是 StudyClaw 的任务分析 Agent。你的职责不是机械照抄，而是结合老师原始通知和结构提示，做稳健、可落地的任务拆解。

【老师原始内容】
{raw_text}

【结构提示】
检测到的学科: {detected_subjects}
检测到的格式信号: {format_signals}
规则预解析候选任务: {candidate_tasks}

【目标】
1. 输出适合直接创建到孩子今日任务清单里的任务列表。
2. 老师格式可能每天变化，你需要理解语义，不要依赖固定模板。
3. 对同一主任务下的多个子步骤，请拆成多条原子任务，并让它们共享同一个 `group_title`。
4. 保留条件信息，例如“默写全对可免抄”“部分学生继续订正”。
5. 忽略纯通知、寒暄、表情和无执行动作的内容。
6. 对每条任务给出 `confidence`（0 到 1）以及 `needs_review`（是否建议家长确认）。
7. 如果任务带有条件限制、对象不明确、学科不明确，请把 `needs_review` 设为 true，并在 `notes` 里写明原因。

【输出要求】
1. 只返回 JSON，不要输出 markdown。
2. JSON 结构必须为:
{{
  "status": "success",
  "data": [
    {{
      "subject": "数学/语文/英语等学科名称",
      "group_title": "这条原子任务所属的作业分组，例如“预习M1U2”",
      "title": "适合孩子执行的一条原子任务，必须是可勾选完成的最小动作",
      "type": "homework",
      "confidence": 0.91,
      "needs_review": false,
      "notes": []
    }}
  ]
}}
3. 如果原文无法形成任务，返回:
{{"status": "failed", "data": []}}
"""
    )

    chain = prompt | llm
    response = chain.invoke({
        "raw_text": raw_text,
        "detected_subjects": json.dumps(structure.get("detected_subjects", []), ensure_ascii=False),
        "format_signals": json.dumps(structure.get("format_signals", []), ensure_ascii=False),
        "candidate_tasks": json.dumps(structure.get("tasks", []), ensure_ascii=False),
    })

    result_text = response.content.strip()
    if result_text.startswith("```json"):
        result_text = result_text[7:-3].strip()

    return json.loads(result_text)


def parse_parent_input(raw_text: str) -> dict[str, Any]:
    structure = extract_structure_outline(raw_text)
    fallback_result = parse_parent_input_fallback(raw_text)

    llm = get_llm()
    if llm is None or not PromptTemplate:
        return fallback_result

    try:
        llm_result = invoke_llm_parser(raw_text, structure)
        llm_tasks = normalize_task_list(llm_result.get("data", []))
        if llm_result.get("status") != "success" or not llm_tasks:
            return fallback_result

        merged_tasks, merge_notes = merge_task_lists(llm_tasks, fallback_result["data"])
        analysis_notes = [
            "已使用 LLM 结合结构提示完成语义拆解，并自动创建任务。",
        ]
        analysis_notes.extend(merge_notes)

        return {
            "status": "success",
            "parser_mode": "llm_hybrid",
            "analysis": build_analysis("llm_hybrid", structure, merged_tasks, analysis_notes),
            "data": merged_tasks,
        }
    except Exception as exc:
        print(f"LLM Parsing error: {exc}")
        fallback_result["analysis"]["notes"].append(f"LLM 调用失败，已自动降级到规则解析: {exc}")
        fallback_result["message"] = str(exc)
        return fallback_result

import React from "react"
import { act, fireEvent, render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import App from "./App"
import "./test/setup"

function createJsonResponse(data, ok = true) {
  return {
    ok,
    json: async () => data,
  }
}

function createSuccessResponse(data) {
  return createJsonResponse(data, true)
}

function createErrorResponse(message) {
  return createJsonResponse({ error: message }, false)
}

function createFetchMock({
  parseHandlers = [],
  confirmHandlers = [],
  pointsHandlers = [],
  weeklyHandlers = [],
  monthlyHandlers = [],
  dictationHandlers = [],
  taskListResponse = { tasks: [], date: "2026-03-09", summary: { total: 0, completed: 0, pending: 0 } },
  dictationListResponse = { dictation_sessions: [] },
  pointsLedgerResponse = { entries: [], points_balance: { balance: 0 } },
  initialWordLists = [],
}) {
  const parseQueue = [...parseHandlers]
  const confirmQueue = [...confirmHandlers]
  const pointsQueue = [...pointsHandlers]
  const weeklyQueue = [...weeklyHandlers]
  const monthlyQueue = [...monthlyHandlers]
  const dictationQueue = [...dictationHandlers]
  let storedWordLists = [...initialWordLists]

  return vi.fn(async (input, init = {}) => {
    const url = String(input)
    const method = String(init?.method || "GET").toUpperCase()

    if (url.includes("/api/v1/tasks?") && method === "GET") {
      return createSuccessResponse(taskListResponse)
    }

    if (url.includes("/api/v1/points/ledger?") && method === "GET") {
      return createSuccessResponse(pointsLedgerResponse)
    }

    if (url.includes("/api/v1/word-lists?") && method === "GET") {
      return createSuccessResponse({ word_lists: storedWordLists })
    }

    if (url.includes("/api/v1/dictation-sessions?") && method === "GET") {
      const handler = dictationQueue.shift()
      if (handler) {
        return typeof handler === "function" ? handler(input, init) : handler
      }
      return createSuccessResponse(dictationListResponse)
    }

    if (url.includes("/api/v1/stats/weekly?") && method === "GET") {
      const handler = weeklyQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected weekly request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if (url.includes("/api/v1/stats/monthly?") && method === "GET") {
      const handler = monthlyQueue.shift()
      if (handler) {
        return typeof handler === "function" ? handler(input, init) : handler
      }
      return createSuccessResponse({
        completion_series: [],
        points_series: [],
        word_series: [],
      })
    }

    if (url.endsWith("/api/v1/tasks/parse") && method === "POST") {
      const handler = parseQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected parse request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if (url.endsWith("/api/v1/word-lists/parse") && method === "POST") {
      const body = JSON.parse(init.body)
      const items = String(body.raw_text || "")
        .split("\n")
        .map((item) => item.trim())
        .filter(Boolean)
        .map((text) => ({
          text,
          meaning: "",
        }))

      return createSuccessResponse({ items })
    }

    if (url.endsWith("/api/v1/word-lists") && method === "POST") {
      const body = JSON.parse(init.body)
      const now = "2026-03-11T09:18:53Z"
      const nextWordList = {
        word_list_id:
          storedWordLists.find(
            (item) =>
              item.family_id === body.family_id &&
              item.child_id === body.child_id &&
              item.assigned_date === body.assigned_date,
          )?.word_list_id || `word-list-${storedWordLists.length + 1}`,
        family_id: body.family_id,
        child_id: body.child_id,
        assigned_date: body.assigned_date,
        title: body.title,
        language: body.language,
        total_items: Array.isArray(body.items) ? body.items.length : 0,
        items: (body.items || []).map((item, index) => ({
          index: index + 1,
          text: item.text,
          meaning: item.meaning || "",
          hint: item.hint || "",
        })),
        created_at: now,
        updated_at: now,
      }

      storedWordLists = [
        ...storedWordLists.filter(
          (item) =>
            !(
              item.family_id === nextWordList.family_id &&
              item.child_id === nextWordList.child_id &&
              item.assigned_date === nextWordList.assigned_date
            ),
        ),
        nextWordList,
      ]

      return createSuccessResponse({
        message: "Word list saved successfully",
        word_list: nextWordList,
      })
    }

    if (url.endsWith("/api/v1/tasks/confirm") && method === "POST") {
      const handler = confirmQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected confirm request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if ((url.endsWith("/api/v1/points/update") || url.endsWith("/api/v1/points/ledger")) && method === "POST") {
      const handler = pointsQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected points request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    throw new Error(`Unhandled request: ${method} ${url}`)
  })
}

function createParsedTask({
  subject = "数学",
  group_title = "默认分组",
  title = "默认任务",
  confidence = 0.92,
  needs_review = false,
  notes = [],
} = {}) {
  return {
    subject,
    group_title,
    title,
    confidence,
    needs_review,
    notes,
  }
}

function createParseSuccess(tasks) {
  return createSuccessResponse({
    tasks,
    parsed_count: tasks.length,
    parser_mode: "llm_hybrid",
    analysis: {
      detected_subjects: tasks.map((task) => task.subject),
      format_signals: [],
      notes: [],
    },
  })
}

function createDictationSession({
  session_id = "session_000001",
  word_list_id = "word_list_000001",
  family_id = 101,
  child_id = 201,
  assigned_date = "2026-03-11",
  status = "completed",
  current_index = 4,
  total_items = 10,
  played_count = 10,
  completed_items = 10,
  grading_status = "completed",
  grading_error = "",
  grading_requested_at = "2026-03-11T09:18:01Z",
  grading_completed_at = "2026-03-11T09:18:53Z",
  updated_at = "2026-03-11T09:18:53Z",
  debug_context = {
    photo_sha1: "f15c1cae7882",
    photo_bytes: 24,
    language: "english",
    mode: "word",
    worker_stage: "completed",
    log_file: "api-server-2026-03-11.log",
    log_keywords: ["session_id=session_000001", "word_list_id=word_list_000001", "photo_sha1=f15c1cae7882"],
  },
  grading_result = {
    grading_id: "grading_000001",
    status: "needs_correction",
    score: 89,
    ai_feedback: "建议复查 blind 的释义。",
    created_at: "2026-03-11T09:18:53Z",
    graded_items: [
      {
        index: 1,
        expected: "blind",
        meaning: "失明的",
        actual: "blind",
        is_correct: false,
        needs_correction: true,
        comment: "释义不匹配",
      },
    ],
  },
} = {}) {
  return {
    session_id,
    word_list_id,
    family_id,
    child_id,
    assigned_date,
    status,
    current_index,
    total_items,
    played_count,
    completed_items,
    grading_status,
    grading_error,
    grading_requested_at,
    grading_completed_at,
    updated_at,
    debug_context,
    grading_result,
  }
}

beforeEach(() => {
  vi.restoreAllMocks()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe("App", () => {
  it("passes assigned_date to parse and confirm requests", async () => {
    let parseBody
    let confirmBody
    const fetchMock = createFetchMock({
      parseHandlers: [
        (_, init) => {
          parseBody = JSON.parse(init.body)
          return createParseSuccess([
            createParsedTask({
              title: "带日期创建的任务",
            }),
          ])
        },
      ],
      confirmHandlers: [
        (_, init) => {
          confirmBody = JSON.parse(init.body)
          return createSuccessResponse({
            tasks: [{ subject: "数学", content: "带日期创建的任务" }],
            created_count: 1,
          })
        },
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    const dateInput = screen.getByLabelText("任务日期")
    fireEvent.change(dateInput, { target: { value: "2026-03-12" } })

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    expect(parseBody.assigned_date).toBe("2026-03-12")

    await user.click(screen.getByRole("button", { name: "确认发布选中任务 (1)" }))
    await screen.findByText("任务已发布")

    expect(confirmBody.assigned_date).toBe("2026-03-12")
  })

  it("keeps previous draft tasks when parse fails after a successful parse", async () => {
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: "第一次成功生成的任务",
          }),
        ]),
        createErrorResponse("解析服务暂时不可用"),
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")
    expect(screen.getByDisplayValue("第一次成功生成的任务")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("解析失败，上一轮草稿已保留")

    expect(screen.getByDisplayValue("第一次成功生成的任务")).toBeInTheDocument()
  })

  it("keeps selected items when publish fails", async () => {
    const taskTitle = "发布失败后保留的任务"
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: taskTitle,
          }),
        ]),
      ],
      confirmHandlers: [createErrorResponse("创建服务暂时不可用")],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    const checkbox = screen.getByRole("checkbox", { name: `选择任务 ${taskTitle}` })
    expect(checkbox).toBeChecked()

    await user.click(screen.getByRole("button", { name: "确认发布选中任务 (1)" }))
    await screen.findByText("发布失败，草稿与选中项已保留")

    expect(checkbox).toBeChecked()
    expect(screen.getByRole("button", { name: "确认发布选中任务 (1)" })).toBeInTheDocument()
    expect(screen.getByDisplayValue(taskTitle)).toBeInTheDocument()
  })

  it("switches publish dock actions across compose, review, and release stages", async () => {
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: "底部动作条任务",
          }),
        ]),
      ],
      confirmHandlers: [
        createSuccessResponse({
          tasks: [{ subject: "数学", content: "底部动作条任务" }],
          created_count: 1,
        }),
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    expect(screen.getByRole("button", { name: "快捷解析" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    expect(screen.getByRole("button", { name: "快捷发布 (1)" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "快捷发布 (1)" }))
    await screen.findByText("任务已发布")

    expect(screen.getByRole("button", { name: "快捷看反馈" })).toBeInTheDocument()
  })

  it("sorts risky tasks ahead of ready-to-publish tasks", async () => {
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: "普通高置信任务",
            confidence: 0.95,
            needs_review: false,
          }),
          createParsedTask({
            title: "需要优先确认的任务",
            confidence: 0.42,
            needs_review: true,
          }),
        ]),
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    const cards = screen.getAllByTestId("draft-card")
    expect(within(cards[0]).getByDisplayValue("需要优先确认的任务")).toBeInTheDocument()
    expect(within(cards[1]).getByDisplayValue("普通高置信任务")).toBeInTheDocument()
  })

  it("switches the focused draft card when changing review filters", async () => {
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: "普通高置信任务",
            confidence: 0.95,
            needs_review: false,
          }),
          createParsedTask({
            title: "需要优先确认的任务",
            confidence: 0.42,
            needs_review: true,
          }),
        ]),
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    expect(screen.getByTestId("draft-focus-title")).toHaveTextContent("需要优先确认的任务")

    await user.click(screen.getByRole("tab", { name: /建议发布/ }))

    expect(screen.getByTestId("draft-focus-title")).toHaveTextContent("普通高置信任务")
  })

  it("jumps back to the blocking draft card from the confirm panel", async () => {
    const fetchMock = createFetchMock({
      parseHandlers: [
        createParseSuccess([
          createParsedTask({
            title: "普通高置信任务",
            confidence: 0.95,
            needs_review: false,
          }),
          createParsedTask({
            title: "",
            group_title: "缺标题任务",
            confidence: 0.92,
            needs_review: false,
          }),
        ]),
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "AI 解析任务" }))
    await screen.findByText("AI 草稿已生成")

    await user.click(screen.getByRole("checkbox", { name: "选择任务 缺标题任务" }))
    await user.click(screen.getByRole("button", { name: /已选阻断项/ }))

    expect(screen.getByTestId("draft-focus-title")).toHaveTextContent("缺标题任务")
  })

  it("loads weekly stats for the selected date", async () => {
    let weeklyUrl = ""
    const fetchMock = createFetchMock({
      weeklyHandlers: [
        (input) => {
          weeklyUrl = String(input)
          return createSuccessResponse({
            message: "Weekly stats generated successfully",
            raw_stats: [
              {
                date: "2026-03-12",
                tasks: [
                  { subject: "数学", content: "订正试卷", completed: true },
                  { subject: "英语", content: "背单词", completed: false },
                ],
              },
            ],
            insights: {
              summary: "本周保持了稳定推进。",
              strengths: ["能按计划推进数学任务"],
              areas_for_improvement: ["英语任务可以再提前启动"],
              psychological_insight: "持续比突击更重要。",
            },
          })
        },
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    fireEvent.change(screen.getByLabelText("任务日期"), { target: { value: "2026-03-12" } })
    await user.click(screen.getByRole("tab", { name: /趋势/ }))
    await user.click(screen.getByRole("button", { name: "查看周趋势" }))

    await screen.findByText("周趋势已刷新")
    expect(weeklyUrl).toContain("end_date=2026-03-12")
    expect(screen.getByText("周趋势摘要").closest(".report-summary-card")).toHaveTextContent("本周保持了稳定推进。")
  })

  it("shows the latest async dictation grading result", async () => {
    const fetchMock = createFetchMock({
      dictationListResponse: {
        dictation_sessions: [
          createDictationSession(),
        ],
      },
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)

    await screen.findByText("听写结果已同步")

    expect(screen.getByText("听写摘要")).toBeInTheDocument()
    expect(screen.getByText("异步听写批改")).toBeInTheDocument()
    expect(screen.getByText("异步结果摘要")).toBeInTheDocument()
    expect(screen.getByText("本次得分 89 分，识别到 1 项需订正。")).toBeInTheDocument()
    expect(screen.getByText(/整体结论:\s*待订正/)).toBeInTheDocument()
    expect(screen.getAllByText("建议复查 blind 的释义。").length).toBeGreaterThan(0)
    expect(screen.getByText("blind（失明的） -> blind；释义不匹配")).toBeInTheDocument()
    expect(screen.getByText("排障定位")).toBeInTheDocument()
    expect(screen.getByText("api-server-2026-03-11.log")).toBeInTheDocument()
    expect(screen.getByText("photo_sha1=f15c1cae7882")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制排障信息" })).toBeInTheDocument()
  })

  it("switches dictation details when selecting an older upload from the timeline", async () => {
    const fetchMock = createFetchMock({
      dictationListResponse: {
        dictation_sessions: [
          createDictationSession({
            session_id: "session_000010",
            grading_requested_at: "2026-03-11T09:30:01Z",
            grading_completed_at: "2026-03-11T09:30:53Z",
            updated_at: "2026-03-11T09:30:53Z",
            debug_context: {
              photo_sha1: "latestphoto123",
              photo_bytes: 48,
              language: "english",
              mode: "word",
              worker_stage: "completed",
              log_file: "api-server-2026-03-11.log",
              log_keywords: ["session_id=session_000010", "photo_sha1=latestphoto123"],
            },
            grading_result: {
              grading_id: "grading_000010",
              status: "passed",
              score: 100,
              ai_feedback: "本次全部正确。",
              created_at: "2026-03-11T09:30:53Z",
              graded_items: [],
            },
          }),
          createDictationSession({
            session_id: "session_000009",
            grading_status: "failed",
            grading_error: "模型返回空结果",
            grading_requested_at: "2026-03-11T08:10:01Z",
            grading_completed_at: "2026-03-11T08:10:21Z",
            updated_at: "2026-03-11T08:10:21Z",
            debug_context: {
              photo_sha1: "olderphoto456",
              photo_bytes: 36,
              language: "english",
              mode: "word",
              worker_stage: "llm_grading_failed",
              log_file: "api-server-2026-03-11.log",
              log_keywords: ["session_id=session_000009", "photo_sha1=olderphoto456"],
            },
            grading_result: null,
          }),
        ],
      },
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await screen.findByText("听写结果已部分同步")
    expect(screen.getAllByText(/会话 ID:\s*session_000010/).length).toBeGreaterThan(0)
    expect(screen.getByText("photo_sha1=latestphoto123")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /session_000009/ }))

    expect(screen.getAllByText(/会话 ID:\s*session_000009/).length).toBeGreaterThan(0)
    expect(screen.getAllByText("模型返回空结果").length).toBeGreaterThan(0)
    expect(screen.getByText("photo_sha1=olderphoto456")).toBeInTheDocument()
    expect(screen.getByText("历史 2")).toBeInTheDocument()
  })

  it("keeps points form values when ledger update fails and passes occurred_on", async () => {
    let pointsBody
    const fetchMock = createFetchMock({
      pointsHandlers: [
        (_, init) => {
          pointsBody = JSON.parse(init.body)
          return createErrorResponse("积分服务暂时不可用")
        },
      ],
    })

    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    const user = userEvent.setup()

    await user.click(screen.getByRole("button", { name: "家长扣分" }))
    fireEvent.change(screen.getByLabelText("积分日期"), { target: { value: "2026-03-14" } })
    fireEvent.change(screen.getByLabelText("积分分值"), { target: { value: "3" } })
    fireEvent.change(screen.getByLabelText("积分原因"), { target: { value: "回家后拖延且未整理错题" } })

    await user.click(screen.getByRole("button", { name: "提交积分调整" }))
    await screen.findByText("积分提交失败")

    expect(pointsBody.delta).toBe(-3)
    expect(pointsBody.source_type).toBe("parent_penalty")
    expect(pointsBody.note).toBe("回家后拖延且未整理错题")
    expect(pointsBody.occurred_on).toBe("2026-03-14")
    expect(screen.getByLabelText("积分原因")).toHaveValue("回家后拖延且未整理错题")
    expect(screen.getByLabelText("积分分值")).toHaveValue(3)
    expect(screen.getByLabelText("积分日期")).toHaveValue("2026-03-14")
  })

  it("creates and persists a word list bound to the selected child and date", async () => {
    const fetchMock = createFetchMock({})
    vi.stubGlobal("fetch", fetchMock)

    const user = userEvent.setup()
    const firstRender = render(<App />)

    fireEvent.change(screen.getByLabelText("任务日期"), { target: { value: "2026-03-20" } })
    fireEvent.change(screen.getByLabelText("清单名"), { target: { value: "3 月 20 日英语默写" } })
    fireEvent.change(screen.getByLabelText("词项"), { target: { value: "apple\nbanana" } })

    await user.click(screen.getByRole("button", { name: "创建单词清单" }))
    await screen.findByText("单词清单已创建")

    expect(screen.getByDisplayValue("3 月 20 日英语默写")).toBeInTheDocument()
    expect(screen.getByDisplayValue("apple")).toBeInTheDocument()

    firstRender.unmount()

    render(<App />)
    const storedWordListButton = await screen.findByRole("button", { name: /3 月 20 日英语默写/ })
    await user.click(storedWordListButton)
    expect(await screen.findByDisplayValue("3 月 20 日英语默写")).toBeInTheDocument()
    expect(await screen.findByDisplayValue("apple")).toBeInTheDocument()
  })

  it("debounces word list auto sync and shows sync status", async () => {
    vi.useFakeTimers()

    const fetchMock = createFetchMock({
      initialWordLists: [
        {
          word_list_id: "word-list-1",
          family_id: 101,
          child_id: 201,
          assigned_date: "2026-03-11",
          title: "旧清单名",
          language: "en",
          total_items: 1,
          items: [
            {
              index: 1,
              text: "apple",
              meaning: "苹果",
            },
          ],
          created_at: "2026-03-11T09:18:53Z",
          updated_at: "2026-03-11T09:18:53Z",
        },
      ],
    })
    vi.stubGlobal("fetch", fetchMock)

    render(<App />)
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
    fireEvent.click(screen.getByRole("button", { name: /旧清单名/ }))
    expect(screen.getByDisplayValue("旧清单名")).toBeInTheDocument()

    fireEvent.change(screen.getByDisplayValue("旧清单名"), { target: { value: "旧清单名 A" } })
    fireEvent.change(screen.getByDisplayValue("旧清单名 A"), { target: { value: "旧清单名 B" } })

    expect(screen.getAllByText("待同步").length).toBeGreaterThan(0)

    const wordListPostCallsBefore = fetchMock.mock.calls.filter(
      ([input, init]) => String(input).endsWith("/api/v1/word-lists") && String(init?.method || "GET").toUpperCase() === "POST",
    )
    expect(wordListPostCallsBefore).toHaveLength(0)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(700)
      await Promise.resolve()
      await Promise.resolve()
    })

    const wordListPostCallsAfter = fetchMock.mock.calls.filter(
      ([input, init]) => String(input).endsWith("/api/v1/word-lists") && String(init?.method || "GET").toUpperCase() === "POST",
    )
    expect(wordListPostCallsAfter).toHaveLength(1)
    expect(JSON.parse(wordListPostCallsAfter[0][1].body).title).toBe("旧清单名 B")
    expect(screen.getAllByText("已同步").length).toBeGreaterThan(0)
  })
})

import React from "react"
import { fireEvent, render, screen, within } from "@testing-library/react"
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
  taskListResponse = { tasks: [], date: "2026-03-09", summary: { total: 0, completed: 0, pending: 0 } },
}) {
  const parseQueue = [...parseHandlers]
  const confirmQueue = [...confirmHandlers]
  const pointsQueue = [...pointsHandlers]
  const weeklyQueue = [...weeklyHandlers]

  return vi.fn(async (input, init = {}) => {
    const url = String(input)
    const method = String(init?.method || "GET").toUpperCase()

    if (url.includes("/api/v1/tasks?") && method === "GET") {
      return createSuccessResponse(taskListResponse)
    }

    if (url.includes("/api/v1/stats/weekly?") && method === "GET") {
      const handler = weeklyQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected weekly request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if (url.endsWith("/api/v1/tasks/parse") && method === "POST") {
      const handler = parseQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected parse request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if (url.endsWith("/api/v1/tasks/confirm") && method === "POST") {
      const handler = confirmQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected confirm request: ${url}`)
      }
      return typeof handler === "function" ? handler(input, init) : handler
    }

    if (url.endsWith("/api/v1/points/update") && method === "POST") {
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

beforeEach(() => {
  vi.restoreAllMocks()
})

afterEach(() => {
  vi.unstubAllGlobals()
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
    await user.click(screen.getByRole("button", { name: "查看周趋势" }))

    await screen.findByText("周趋势已刷新")
    expect(weeklyUrl).toContain("end_date=2026-03-12")
    expect(screen.getByText("本周保持了稳定推进。")).toBeInTheDocument()
  })

  it("keeps points form values when update fails", async () => {
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
    fireEvent.change(screen.getByLabelText("积分分值"), { target: { value: "3" } })
    fireEvent.change(screen.getByLabelText("积分原因"), { target: { value: "回家后拖延且未整理错题" } })

    await user.click(screen.getByRole("button", { name: "提交积分调整" }))
    await screen.findByText("积分提交失败")

    expect(pointsBody.amount).toBe(-3)
    expect(pointsBody.reason).toBe("回家后拖延且未整理错题")
    expect(screen.getByLabelText("积分原因")).toHaveValue("回家后拖延且未整理错题")
    expect(screen.getByLabelText("积分分值")).toHaveValue(3)
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
    expect(screen.getByDisplayValue("3 月 20 日英语默写")).toBeInTheDocument()
    expect(screen.getByDisplayValue("apple")).toBeInTheDocument()
  })
})

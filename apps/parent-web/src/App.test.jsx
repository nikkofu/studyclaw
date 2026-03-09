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

function createFetchMock({ parseHandlers = [], confirmHandlers = [], taskListResponse = { tasks: [], date: "2026-03-09" } }) {
  const parseQueue = [...parseHandlers]
  const confirmQueue = [...confirmHandlers]

  return vi.fn(async (input, init = {}) => {
    const url = String(input)
    const method = String(init?.method || "GET").toUpperCase()

    if (url.includes("/api/v1/tasks?") && method === "GET") {
      return createSuccessResponse(taskListResponse)
    }

    if (url.endsWith("/api/v1/tasks/parse") && method === "POST") {
      const handler = parseQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected parse request: ${url}`)
      }
      return typeof handler === "function" ? handler(init) : handler
    }

    if (url.endsWith("/api/v1/tasks/confirm") && method === "POST") {
      const handler = confirmQueue.shift()
      if (!handler) {
        throw new Error(`Unexpected confirm request: ${url}`)
      }
      return typeof handler === "function" ? handler(init) : handler
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
        (init) => {
          parseBody = JSON.parse(init.body)
          return createParseSuccess([
            createParsedTask({
              title: "带日期创建的任务",
            }),
          ])
        },
      ],
      confirmHandlers: [
        (init) => {
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

    await user.click(screen.getByRole("button", { name: "确认创建选中任务 (1)" }))
    await screen.findByText("任务已创建")

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

  it("keeps selected items when create fails", async () => {
    const taskTitle = "创建失败后保留的任务"
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

    await user.click(screen.getByRole("button", { name: "确认创建选中任务 (1)" }))
    await screen.findByText("创建失败，草稿与选中项已保留")

    expect(checkbox).toBeChecked()
    expect(screen.getByRole("button", { name: "确认创建选中任务 (1)" })).toBeInTheDocument()
    expect(screen.getByDisplayValue(taskTitle)).toBeInTheDocument()
  })

  it("sorts risky tasks ahead of ready-to-create tasks", async () => {
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
})

# StudyClaw

StudyClaw 是一款面向家庭学习场景的任务协同与 AI 管理工具。它致力于通过 AI 自动化解析、权威后端事实源和正向心理激励，让每日学习管理变得轻松、透明且富有动力。

- **当前版本**: `v0.2.0` (第一阶段正式签收版)
- **核心定位**: 消除本地缓存依赖，实现全链路后端闭环。

---

## 🚀 核心能力

### 1. AI 任务解析 (Parent Web)
- **原文解析**：支持微信群/钉钉群风格的作业文本一键粘贴。
- **智能拆解**：自动拆解为原子任务，并识别学科及作业分组。
- **风险预控**：AI 识别置信度低或性质模糊的任务（如“订正”），强制人工审核。

### 2. 后端事实源闭环 (v0.2.0)
- **单词清单**：家长录入 -> 后端持久化 -> Pad 同步，支持听写会话控制。
- **积分系统**：双端同步展示权威积分流水与余额，彻底告别前端估算。
- **进度同步**：任务状态实时同步，支持多设备刷新查看最新进展。

### 3. 多端协同体验 (Pad App)
- **沉浸式任务板**：孩子自主选择执行顺序，系统实时跟踪进度。
- **单词听写播放**：支持逐词朗读、进度恢复及后端驱动的播放会话。
- **今日简报**：AI 基于当日完成度及积分变化生成积极、支持型的反馈。

### 4. 数据可视化与 AI 观察
- **多维趋势**：提供日/周/月度任务完成率及积分波动分析图表。
- **客观解释**：AI 仅负责解释统计数据，不改写业务状态，确保数据真实可信。

---

## 🛠️ 技术栈

- **后端**: Go (Gin, Markdown Workspace)
- **管理端**: React + Vite + Vanilla CSS
- **执行端**: Flutter (Dart)
- **AI 模型**: 基于 Google LLM 设计模式 (Agentic Pattern)

---

## 📦 快速启动

### 1. 环境预检
```bash
bash scripts/preflight_local_env.sh
```

### 2. 启动服务
```bash
# 后端
cd apps/api-server && go run ./cmd/studyclaw-server

# 家长端
cd apps/parent-web && npm run dev

# Pad 端 (Chrome)
cd apps/pad-app && flutter run -d chrome
```

### 3. 冒烟测试与演示
```bash
bash scripts/smoke_local_stack.sh
bash scripts/demo_local_stack.sh
```

---

## 📝 交付与审计

- **交付就绪度**: 详见 [docs/17_DELIVERY_READINESS.md](docs/17_DELIVERY_READINESS.md)
- **演示清单**: 详见 [docs/16_FIRST_PHASE_DEMO_CHECKLIST.md](docs/16_FIRST_PHASE_DEMO_CHECKLIST.md)
- **发布检查**: 详见 [docs/13_RELEASE_CHECKLIST.md](docs/13_RELEASE_CHECKLIST.md)
- **详细手册**: 详见 [docs/USER_MANUAL_V0.2.0.md](docs/USER_MANUAL_V0.2.0.md)

---

## ⚖️ 许可

本项目采用 [LICENSE](LICENSE) 进行许可。

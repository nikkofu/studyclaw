# StudyClaw 🐱📚

StudyClaw 是一个专为上班族家庭的小朋友设计的“Agentic”（智能体驱动）日常学习与成长管理工具。它包含一个面向小朋友的 Android Pad 应用，以及面向家长的管理后台。通过接入兼容 OpenAI 协议的 LLM Provider，StudyClaw 能够自动解析作业、布置学习任务，并以游戏化、高交互的方式陪伴孩子成长。

## 🎯 愿景
让 AI 成为孩子日常学习的督导与贴心伴侣，成为家长的时间管理与教育辅助专家。

## 📂 项目目录结构规范

本项目采用 Monorepo（单体仓库）架构，以保障多端应用共享类型、UI 组件和核心业务逻辑：

```text
studyclaw/
├── apps/                    # 应用程序
│   ├── pad-app/             # (Flutter / React Native) 面向小朋友的 Android Pad 端应用
│   ├── parent-web/          # (React / Vue / 微信小程序) 面向家长的管理端后台
│   ├── api-server/          # (Node.js / Python) 核心业务与数据 API 服务
│   └── agent-core/          # (Python / Node.js) 负责与 LLM 交互的 Agentic 推理引擎与工作流
├── packages/                # 共享依赖包
│   ├── shared-types/        # 前后端交互接口协议、多端共享的 TypeScript 类型
│   ├── ui-kit/              # 可复用的 UI 组件库（如进度条、奖励徽章交互等）
│   └── llm-utils/           # 封装与各类 LLM Provider (OpenAI 兼容) 的对接逻辑
├── docs/                    # 项目核心设计文档（PRD、架构、计划等）
│   ├── 01_PRD.md            # 需求与可行性设计
│   ├── 02_ARCHITECTURE.md   # 系统架构与接口规范
│   └── 03_ROADMAP.md        # 项目开发与迭代规划
├── .gitignore
├── turbo.json               # (可选) Turborepo 配置文件
└── package.json             # 根级依赖配置
```

## 🚀 快速开始
详见 `docs/` 目录下的相关说明即可开始开发工作。

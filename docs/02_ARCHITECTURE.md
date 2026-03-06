# StudyClaw 技术架构与设计蓝图 (Architecture)

## 1. 整体架构概览

StudyClaw 采用前后端分离的现代化架构，核心业务划分为三个主要端点和两大后台服务子系统。为了良好支持多模态 AI (文本解析、TTS、ASR、图像识别等)，系统引入了“Agent Core”作为独立调度中枢。

```mermaid
graph TD
    subgraph 客户端组 (Clients)
        PadApp[Android Pad App\n(Flutter/RN)]
        ParentWeb[家长管理端\n(React/Vue小程序)]
    end

    subgraph API 服务网关 (Gateway & Business Logic)
        APIServer[Node.js / Python API Server]
        DB[(PostgreSQL / MongoDB)]
        Cache[(Redis)]
    end

    subgraph AI 智能体核心 (Agent Core)
        Router[Router / Orchestrator\n(意图识别与任务分发)]
        Memory[(Long-term Memory\nVector DB)]
        
        Brain_Tutor[学业辅导专精 Agent]
        Brain_Empathy[情绪安抚专精 Agent]
        
        LLMProvider[LLM API\n(OpenAI Compatible)]
        TTS_ASR[多模态感知与交互\n(Edge TTS / Whisper / Vision)]
    end

    PadApp <-->|REST/GraphQL/WebSocket| APIServer
    ParentWeb <-->|REST/GraphQL| APIServer
    
    APIServer <-->|存储数据| DB
    APIServer <-->|缓存/状态| Cache
    
    APIServer <-->|上下文、多模态流、感知数据| Router
    Router <--> Memory
    Router <--> Brain_Tutor & Brain_Empathy
    Brain_Tutor & Brain_Empathy <--> LLMProvider
    Router <--> TTS_ASR
```

## 2. 关键框架与技术栈选型

结合现有开发及服务器物理环境，我们确立以下最优技术栈组合：

### 2.1 物理部署环境
- **开发设备**: Mac (MBP)
- **云端/本地服务器**: CentOS 操作系统
- **容器化**: 全面采用 Docker / Docker Compose 进行服务的编排与部署，确保 Mac 开发环境与 CentOS 生产环境的绝对一致性。

### 2.2 核心服务模块选型
1. **API Server (核心业务服务器)**
   - **技术栈**: Golang (Gin / Fiber) 或 Node.js (NestJS)。考虑到并发连接（WebSocket）及轻量级部署，**首推 Golang**。
   - **数据库**: MySQL (存储用户、家庭组、任务元数据及积分结算)。
   - **缓存与消息**: Redis (处理 WebSocket 分布式会话、排行榜缓存、高频任务状态及 Agent 任务队列)。
   
2. **Agent Core (智能体引擎)**
   - **技术栈**: Python 3.10+ (FastAPI / LangChain / AutoGen)。
   - **定位**: 作为微服务接收来自 API Server 的请求。Python 对接主流 LLM 原生 SDK、Vision 图像处理及音频流（TTS/ASR）有着不可替代的生态优势。
   - **记忆库**: 挂载向量数据库 (如 ChromaDB 或 Milvus 的轻量版) 于 Docker 中，用于长程记忆。

3. **Pad App (孩子端)**
   - **技术栈**: Flutter。跨平台且动画渲染性能优异，极度契合当前 Mac 环境开发 Android Pad 应用的需求。

4. **Parent Web (家长端)**
   - **技术栈**: Node.js 体系下的前端框架 (React / Vue) 编译为静态 H5，或打包为微信小程序。
  - 任务 CRUD：展示每天的任务列表、状态流转（未完成 -> 待确认 -> 已完成）。
  - 奖励系统管理：分数计算、徽章解锁状态。
  - **WebSocket 通信**: 用于实时的亲子语音推送和任务状态同步。

### 2.2 Agent Core (智能体处理引擎)
负责所有涉及大模型的“思考”与“多模态转换”工作。独立出来便于针对 AI 模型做限流、重试和 Prompt 管理。
- **技术栈建议**: Python + LangChain / 本地封装的统一 LLM 请求器。
- **核心工作流 (Workflows)**:
  1. **任务解析流 (Task Parser)**: 接收家长上传的图片/文字 -> 提供给 VLM (视觉语言模型) -> 提取并结构化成 JSON 格式的任务清单（如：`[{"subject":"数学", "title":"口算题卡第20页", "type":"homework"}]`）。
  2. **听写对答流 (Dictation Assistant)**: 接收配置好的单词表 -> 按顺序调用 TTS 发声 -> 接收等待孩子完成的指令 -> 调用 Vision API 判定拼写结果并给出鼓励性反馈。
  3. **内容安全护栏 (Safety Guard)**: 对所有返回给 Pad 的文本进行儿童适宜度审查。

### 2.3 Pad App (孩子端应用)
要求交互活泼、字体大、误触率低。
- **技术栈建议**: Flutter (跨平台且 UI 表现力强，适合做丰富的动画)。
- **核心交互**:
  - **主视界**: 宇宙、森林等主题背景，核心是一个随着任务完成不断成长的动画角色/进度条。
  - **任务卡片**: 使用大色块区分学科，支持点击播放语音读题。
  - **多媒体采集**: 录音接口（发语音给家长或背诵）、拍照接口（提交作业检验）。

### 2.4 Parent Web (家长管理端)
要求高效、直观、随时可用。
- **技术栈建议**: 考虑到中国家长的使用习惯，建议采用微信小程序，或基于 React 构建移动端友好的 H5。
- **核心交互**:
  - **Dashboard**: 一眼看到今天孩子的完成进度（饼图/进度条）。
  - **快捷输入区**: 支持直接从班级群复制微信聊天记录粘贴，一键通过 AI 转化为任务。

## 3. 统一配置文件提取规范 (Configuration Design)

为了保障代码在 Mac 物理机开发环境与 CentOS 服务器生产环境之间无缝流转，以及确保敏感凭据不随代码提交，系统需采取**环境配置外置策略**。

我们将在项目根目录与各微服务级目录采用 `.env` 与 `config.yaml` 灵活结合的方式：

### 3.1 基础设施统一凭证 (docker-compose & 本地运行时)
在整个项目根目录 `studyclaw/` 下，或者通过运维指定的路径，放置核心 `.env` 文件。该文件将被 Docker Compose 或各微服务的启动脚本读取并注入系统环境变量：

```env
# ==== 数据库相关 (MySQL) ====
DB_HOST=127.0.0.1      # 在 CentOS 部署时替换为服务器内网/公网 IP 或容器编排的服务名
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_secure_password
DB_NAME=studyclaw_prod

# ==== 缓存/通信相关 (Redis) ====
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=

# ==== 大模型与感知服务凭证 (Agent Core) ====
LLM_PROVIDER=openai     # 或 dashscope, moonshot 等
LLM_API_KEY=sk-xxxxxxx
LLM_BASE_URL=https://api.openai.com/v1

# ==== 业务服务本身 ====
API_PORT=8080
AGENT_PORT=8000
```

### 3.2 业务逻辑配置 (config.yaml)
在业务代码中（如 `apps/api-server/` 或 `apps/agent-core/`），通过解析结构化的 `config.yaml` 文件读取环境变量，实现服务内部的参数配置：

```yaml
# 示例：API Server 的 config.yaml
server:
  port: ${API_PORT:8080}
  mode: ${ENV_MODE:debug} # debug / release

database:
  driver: "mysql"
  dsn: "${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "${REDIS_HOST}:${REDIS_PORT}"
  password: "${REDIS_PASSWORD}"
```

**开发规范**：
- 任何开发者或 Agent 代码**不得硬编码** IP 数据库账密。
- 本地 Mac 调试时，提供 `docker-compose.yml` 一键拉起本地 MySQL、Redis，自动读取 `.env.local`。
- 上线 CentOS 物理服务器时，仅需将运维下发的正式配置文件挂载到容器内或设置物理机系统变量。

## 4. 数据库模型概念设计

- **User**: 账户信息（区分 Role: Parent, Child）。
- **Family**: 家庭组表，关联多个 Parent 和 Child。
- **Task**: 任务表。包含字段：`title`, `type` (作业/听写/日记), `status`, `assignedDate`, `metadata` (例如听写的具体词库 JSON)。
- **TaskInteraction**: 记录子任务交互。如孩子针对某个任务上传的语音备注 URL 或作业照片 URL。
- **Reward**: 奖励预设项。
- **PointsLog**: 积分流水的变动历史。

## 4. 接口协议与通信标准
采用 RESTful API 作为管理端的主要通信方式；针对 Pad 端的 AI 语音对答、家长实时留言等需求，引入 **WebSocket** 或者 **Server-Sent Events (SSE)** 保证双向低延迟通信。所有微服务之间（APIServer 与 AgentCore）可以通过 gRPC 或内部 HTTP 调用。

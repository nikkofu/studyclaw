# StudyClaw 安全配置说明

本文档定义 StudyClaw 在本地开发与后续部署时的最小安全基线，目标是把运行时账号、密钥和 Git 仓库代码彻底分离。

## 1. 基本原则

1. 所有真实密钥只允许存放在仓库外的运行时配置文件中。
2. 仓库中的 `.env.example` 只能保留模板和占位符，不能出现真实账号密码。
3. 浏览器端和 Flutter 端永远不能持有真实后端密钥。
4. 每个环境使用独立服务账号和独立 API Key，不共用个人主账号。

## 2. 推荐存储位置

默认运行时配置文件路径：

```text
~/.config/studyclaw/runtime.env
```

也可以显式指定：

```bash
export STUDYCLAW_ENV_FILE=/absolute/path/to/runtime.env
```

或者指定目录：

```bash
export STUDYCLAW_CONFIG_DIR=/absolute/path/to/private-config-dir
```

## 3. 加载顺序

后端服务加载环境变量时按以下优先级读取：

1. 进程环境变量
2. 仓库外私有运行时配置文件 `runtime.env`
3. 仓库根目录 `.env` 作为非敏感本地回退
4. 代码内置默认值

这意味着：

- 真正的账号密码应放在 `runtime.env`
- 根目录 `.env` 最好只保留本地端口、路径等非敏感默认值
- 若你通过 shell 导出变量，shell 中的值优先级最高

## 4. 初始化方式

在仓库根目录执行：

```bash
bash scripts/init_private_runtime_env.sh
```

脚本会：

- 创建私有配置目录
- 目录权限设为 `0700`
- 配置文件权限设为 `0600`
- 把 `.env.example` 复制为私有 `runtime.env`

## 5. 账号与密钥策略

### 5.1 Ark / LLM 账号

- 为 StudyClaw 单独创建服务用途的 API Key，不要复用个人主账号密钥
- 开发、测试、生产使用不同 Key
- 为不同环境设置独立调用额度和告警阈值
- 发现泄露或人员变动后立即轮换

### 5.2 数据与后端账号

- 数据库、Redis、对象存储等账号均应采用“环境隔离 + 最小权限”
- 不要把生产口令写入示例文件、脚本、文档或前端构建变量
- 如未来引入容器部署，密钥继续通过宿主机环境变量或外部 Secret 管理系统注入

## 6. 前端与移动端边界

以下信息可以进入前端或 Flutter 构建参数：

- `API_BASE_URL`
- 非敏感功能开关
- 非敏感默认日期或展示配置

以下信息严禁进入前端或 Flutter：

- `LLM_API_KEY`
- 任意数据库密码
- Redis 密码
- 未来的 JWT 私钥、第三方服务私钥

## 7. Git 仓库防泄漏要求

- `.env`、`runtime.env`、`secrets.env` 等运行时文件必须保持未追踪状态
- 提交前执行：

```bash
bash scripts/check_no_tracked_runtime_env.sh
```

- 若曾误提交真实密钥，除了删除文件，还必须立即轮换该密钥

## 8. 推荐运维动作

- 每 90 天轮换一次关键外部服务密钥
- 为关键服务启用调用量告警
- 发布前确认前端构建产物中不包含任何敏感变量
- 在 CI 中增加一次 `bash scripts/check_no_tracked_runtime_env.sh`

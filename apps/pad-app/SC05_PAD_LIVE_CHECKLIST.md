# SC-05 Pad Live Checklist

适用范围：

- `apps/pad-app`
- Chrome 全链路验收优先
- 真实设备当前只做手动演示，不做自动化验收

最近一次按本清单复跑通过的日期：

- 2026-03-10

## 启动真实后端

前置说明：

- 不要使用 `apps/api-server/main` 旧二进制；它缺少当前任务状态同步接口。
- 必须使用 `apps/api-server` 当前源码启动服务。
- 下面示例统一使用 `18081` 端口，避免与本地默认 `8080` 混用。
- checklist 默认要求使用新的 `STUDYCLAW_DATA_DIR`，避免上次联调残留数据干扰本次结果。

1. 在仓库根目录确认没有占用 `18081` 端口的旧进程。
2. 为本次联调生成新的数据目录：

```bash
export STUDYCLAW_DATA_DIR="$(mktemp -d /tmp/studyclaw-pad-live-src.XXXXXX)"
echo "$STUDYCLAW_DATA_DIR"
```

3. 启动当前 Go API 源码：

```bash
cd apps/api-server
API_PORT=18081 \
GOCACHE=/tmp/studyclaw-go-cache \
GOMODCACHE=$PWD/.gomodcache \
go run ./cmd/studyclaw-server
```

4. 另开一个终端确认服务可用：

```bash
curl http://127.0.0.1:18081/ping
```

预期返回：

```json
{"message":"pong"}
```

## 准备 ChromeDriver

前置说明：

- `flutter drive -d chrome` 需要独立的 WebDriver，不能只开 Chrome。
- 不建议使用 `flutter test -d chrome` 替代；当前 Web integration_test 仍要求 WebDriver。

1. 先确认本机 Chrome 主版本：

```bash
'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' --version
```

2. 在临时目录安装与 Chrome 主版本一致的 `chromedriver`。例如本机 Chrome 为 `145.x` 时：

```bash
mkdir -p /tmp/studyclaw-chromedriver145
cd /tmp/studyclaw-chromedriver145
npm install chromedriver@145
```

3. 启动本地 WebDriver：

```bash
cd /tmp/studyclaw-chromedriver145
./node_modules/.bin/chromedriver --port=4444 --allowed-origins='*'
```

预期输出包含：

```text
ChromeDriver was started successfully on port 4444.
```

如果输出 `Unable to start server with either IPv4 or IPv6`，通常是本地端口权限或沙箱问题；确认 `4444` 未被占用后，重新在允许监听本地端口的终端里启动。

## 跑 Pad Chrome 联调

1. 在 `apps/pad-app` 目录执行：

```bash
flutter drive \
  --driver=test_driver/integration_test.dart \
  --target=integration_test/real_backend_chrome_test.dart \
  -d chrome \
  --dart-define=API_BASE_URL=http://127.0.0.1:18081
```

2. 这条命令会覆盖以下真实后端场景：

- 加载任务板
- 单任务勾选成功
- 分组勾选成功
- 全部完成成功
- `404 task_not_found` 友好提示
- `409 status_unchanged` 信息提示

说明：

- 当前自动化 Chrome 联调聚焦真实后端任务同步链路。
- 单词播放、今日积分与简报入口今晚仍以 Pad 端执行视图和手动 smoke 验收；自动化用例暂不在 live backend 里种词单和积分数据。

3. 预期结尾输出包含：

```text
All tests passed!
```

4. 如果命令卡在 `Unable to start a WebDriver session for web testing`：

- 先确认 `chromedriver` 已经在 `http://127.0.0.1:4444` 启动。
- 再确认 `chromedriver` 主版本与本机 Chrome 主版本一致。

## integration_test 最小使用说明

- 驱动入口：`test_driver/integration_test.dart`
- 真实后端场景文件：`integration_test/real_backend_chrome_test.dart`
- API 地址通过 `--dart-define=API_BASE_URL=...` 注入，默认值是 `http://127.0.0.1:18081`
- 测试会为每次运行生成唯一 `family_id` 和唯一任务内容，可安全重复执行
- 如果需要切换到另一台联调环境，只替换 `API_BASE_URL`，不要修改测试代码里的接口路径

常用复跑命令：

```bash
cd apps/pad-app
flutter drive \
  --driver=test_driver/integration_test.dart \
  --target=integration_test/real_backend_chrome_test.dart \
  -d chrome \
  --dart-define=API_BASE_URL=http://127.0.0.1:18081
```

## Chrome 全链路验收记录

2026-03-09 已按以下结果复跑通过：

- 任务板加载：页面能展示 `YYYY-MM-DD 任务板` 和真实任务内容。
- 单任务勾选：顶部提示“已同步单个任务完成状态”，后端摘要完成数增加。
- 分组勾选：顶部提示“已将 <分组名> 分组标记为完成”，同分组任务全部完成。
- 全部完成：顶部提示“已将全部任务同步为完成”，后端摘要未完成数归零。
- `404` 错误：显示“任务 #999 不存在，可能已被删除或日期已变更。”，不回落到裸错误。
- `409` 错误：显示“全部任务已经是已完成状态，无需重复同步。”，不显示泛化“请求失败”。

## 手动冒烟

```bash
cd apps/pad-app
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:18081
```

最小检查项：

- 首页首次打开能自动加载任务板。
- 顶部配置区能手动刷新，并看到“任务板已手动刷新”一类同步反馈。
- 切换日期后会重新加载，并看到“已切换到 YYYY-MM-DD”一类反馈。
- 勾选单任务后，顶部出现“已同步单个任务完成状态”。
- 点击“分组完成”后，分组任务都进入完成态。
- 点击“全部完成”后，摘要区显示全部完成。
- 今日执行概览里能看到 `今日积分`、`今日完成` 和 `完成率`。
- 点击 `今日简报` 能打开简化日报；点击 `本周鼓励` 能看到周维度摘要或清晰错误提示。
- 切到 `单词播放` 模式后，默认样例词单会加载完成，页面显示 `播放进度 1/N`。
- 点击 `当前词播放`、`重播`、`下一词` 后，进度和提示文案会跟着变化。
- 服务端返回 `404` 时，提示“任务 #999 不存在，可能已被删除或日期已变更。”一类明确文案。
- 服务端返回 `409` 时，提示“全部任务已经是已完成状态，无需重复同步。”，且不显示泛化的“请求失败”。

## Pad 最小联调清单（给 SC-05）

1. 启动当前 Go 源码服务，确认 `http://127.0.0.1:18081/ping` 返回 `pong`。
2. 在 `apps/pad-app` 执行 `flutter analyze`、`flutter test`、`flutter build web --dart-define=API_BASE_URL=http://localhost:8080`。
3. 执行 `flutter drive --driver=test_driver/integration_test.dart --target=integration_test/real_backend_chrome_test.dart -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:18081`，确认任务同步和 `404/409` 提示通过。
4. 手动打开 Chrome 页面，确认首页默认拉取当天任务，且日期切换、手动刷新、单任务/分组/全部完成都有同步反馈。
5. 在 `今日执行概览` 确认 `今日积分`、`今日完成`、`简化版日报 / 周报入口` 都可见；点开 `今日简报` 和 `本周鼓励` 至少各一次。
6. 切到 `单词播放` 模式，验证 `当前词播放`、`重播`、`下一词` 和 `播放进度`，记录当前使用的是本地词单模式，不依赖后端词单 API。
7. 如需真实设备演示，使用 `flutter build web --dart-define=API_BASE_URL=http://<联调机器IP>:18081` 产物托管后，在同网段设备浏览器复核第 4-6 项。

## 真实设备演示

当前推荐以 Chrome 联调作为正式验收路径。若需要 iPad 或其他真实设备演示，按下面方式手动冒烟：

1. 先按上文完成真实后端启动。
2. 在 `apps/pad-app` 目录生成 Web 产物：

```bash
flutter build web --dart-define=API_BASE_URL=http://<你的联调机器IP>:18081
```

3. 用一个本地静态文件服务托管 `build/web`，并确保真实设备与开发机在同一网络。
4. 在真实设备浏览器打开该地址，最少确认以下 5 项：

- 首次进入能加载任务板。
- 单任务勾选后有成功反馈。
- 分组完成与全部完成能正确刷新摘要。
- 手动刷新能看到同步反馈，不是静默刷新。
- 今日积分、今日完成和简化版日报 / 周报入口可见。
- 单词播放模式可切换、可重播、可下一词，且进度会更新。
- 异常场景至少人工确认一次 `404` 或 `409` 文案清晰。

说明：

- 真实设备演示当前是手动 smoke，不属于自动化集成测试。
- 若设备无法访问开发机地址，优先检查局域网 IP、端口开放和浏览器跨域配置。

# SC-05 Pad Live Checklist

## 启动真实后端

1. 在仓库根目录确认没有占用 `18081` 端口的旧进程。
2. 启动当前 Go API 源码，并使用全新的数据目录：

```bash
cd apps/api-server
STUDYCLAW_DATA_DIR=/tmp/studyclaw-pad-live-src \
API_PORT=18081 \
GOCACHE=/tmp/studyclaw-go-cache \
GOMODCACHE=$PWD/.gomodcache \
go run ./cmd/studyclaw-server
```

3. 另开一个终端确认服务可用：

```bash
curl http://127.0.0.1:18081/ping
```

预期返回：

```json
{"message":"pong"}
```

## 准备 ChromeDriver

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

## 手动冒烟

```bash
cd apps/pad-app
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:18081
```

最小检查项：

- 首页首次打开能自动加载任务板。
- 勾选单任务后，顶部出现“已同步单个任务完成状态”。
- 点击“分组完成”后，分组任务都进入完成态。
- 点击“全部完成”后，摘要区显示全部完成。
- 服务端返回 `404` 时，提示“任务 #999 不存在，可能已被删除或日期已变更。”一类明确文案。
- 服务端返回 `409` 时，提示“全部任务已经是已完成状态，无需重复同步。”，且不显示泛化的“请求失败”。

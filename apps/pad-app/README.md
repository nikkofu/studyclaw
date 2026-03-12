# StudyClaw Pad

StudyClaw 的 Pad 端孩子执行客户端，当前交付基线版本是 `v0.3.1`。当前目标不是离线演示，而是和 Parent Web、Go API 共用同一后端事实源。

当前覆盖能力：

- 当天任务板的 `loading / empty / error / success` 四种状态
- 单任务 / 分组 / 全部完成的真实后端同步
- 每次完成任务后的成长型正向鼓励，以及每日鼓励卡片
- 基于 `/api/v1/word-lists` 和 `/api/v1/dictation-sessions` 的单词听写链路
- 听写开始、推进、交卷、批改完成等节点的正向反馈
- 基于 STT + `/api/v1/voice-commands/resolve` 的语音助手，可在任务板和听写页签执行语音指令
- 今日积分、今日完成进度、日报 / 周报 / 月报入口

## 常用命令

```bash
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
flutter run -d web-server --web-hostname 127.0.0.1 --web-port 55771 \
  --dart-define=API_BASE_URL=http://127.0.0.1:38080
```

真实后端 Chrome 联调步骤见 `SC05_PAD_LIVE_CHECKLIST.md`。正式交付前还需要和根目录的 `docs/19_DELIVERY_UAT_CASES.md` 一起核对三端联调结果。

## Integration Test

- Chrome 真实后端联调入口是 `integration_test/real_backend_chrome_test.dart`。
- WebDriver 入口是 `test_driver/integration_test.dart`。
- `widget_test` 负责覆盖孩子端关键状态机与交互，真实后端联调用于确认任务同步和听写会话恢复。
- 复跑命令：

```bash
flutter drive \
  --driver=test_driver/integration_test.dart \
  --target=integration_test/real_backend_chrome_test.dart \
  -d chrome \
  --dart-define=API_BASE_URL=http://127.0.0.1:18081
```

- 运行前需要先启动真实 Go 服务和匹配版 `chromedriver`。
- 不要用 `flutter test -d chrome` 替代当前 Web `integration_test` 流程。

## 当前交付说明

- Pad 默认示例数据仍使用 `family_id=306`、`user_id=1`，方便和 API / 演示清单保持一致。
- `flutter build web` 当前会打印 `flutter_tts` 的 wasm dry-run warning，但产物构建成功，不影响现阶段 HTML/Web 交付。
- 语音助手当前优先面向 Chrome / Edge 等支持麦克风与 Web Speech 的浏览器；首次使用时需要允许麦克风权限。
- 真正的词单来源已经切到后端；Pad 不再以本地临时词单作为正式事实源。

## 版本策略

- 保留并提交 `lib/`、`test/`、`pubspec.yaml`、`pubspec.lock`。
- 保留并提交 `integration_test/` 与 `test_driver/`，用于真实后端 Chrome 联调。
- 保留并提交 `web/`，因为当前验收要求包含 `flutter build web`。
- 保留并提交 `README.md`、`analysis_options.yaml` 与 `SC05_PAD_LIVE_CHECKLIST.md`。
- 不提交 `build/`、`.dart_tool/`、`.metadata` 等本地产物。
- 只有当 macOS / Android / iOS 进入正式交付范围时，才允许把对应平台目录纳入评审。

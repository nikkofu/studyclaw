# StudyClaw Pad

StudyClaw 的 Pad 端任务板客户端，当前优先保证 Chrome 可运行，并覆盖任务同步的 loading、empty、error、success 四种状态。

## 常用命令

```bash
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

## 版本策略

- 保留并提交 `lib/`、`test/`、`pubspec.yaml`、`pubspec.lock`。
- 保留并提交 `web/`，因为当前验收要求包含 `flutter build web`。
- 保留并提交 `README.md` 与 `analysis_options.yaml`，它们用于描述项目边界和统一 lint。
- 不提交 `.metadata`，它属于本地 Flutter 工具状态文件。
- 当前不提交 `macos/`；等明确需要 macOS 桌面端时，再用 Flutter 重新生成并单独评审。

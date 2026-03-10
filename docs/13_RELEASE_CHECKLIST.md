# StudyClaw 发布前检查清单

本文档用于 `v0.2.0` 第一阶段正式签收版的发布前检查。

## 1. 版本与文档

检查点：
- [x] `README.md` 版本号更新为 `v0.2.0`
- [x] `CHANGELOG.md` 已记录 v0.2.0 的所有闭环项
- [x] `docs/03_ROADMAP.md` 已标记第一阶段为“已完成”
- [x] `docs/17_DELIVERY_READINESS.md` 结论为“可正式签收”

## 2. 密钥与运行时配置

必须通过：
```bash
bash scripts/check_no_tracked_runtime_env.sh
```

## 3. 本地环境预检

必须通过：
```bash
bash scripts/preflight_local_env.sh
```

## 4. 后端事实源一致性（v0.2.0 核心）

检查点：
- [x] Parent Web 单词清单接入 `/api/v1/word-lists` (无 localStorage 依赖)
- [x] Pad 积分显示接入 `/api/v1/points/balance` (无本地估算)
- [x] Parent Web 月视图接入 `/api/v1/stats/monthly` (无前端聚合)
- [x] Pad 听写播放接入 `dictation-session` (支持进度恢复)

## 5. 全量自动化测试

- [x] Go: `go test ./...` 全部通过
- [x] Parent Web: `npm run test` & `npm run build` 全部通过
- [x] Pad App: `flutter analyze` & `flutter test` 全部通过

## 6. 演示与冒烟

必须通过：
```bash
bash scripts/smoke_local_stack.sh
bash scripts/demo_local_stack.sh
```

## 7. 发布结论
**v0.2.0 版本已达到第一阶段正式签收标准。**


# Iter 44 — runtime snapshot / 锁争用观测 / sampled soak

## 本轮落地

已补入仓库：

- `internal/server/runtime_snapshot.go`
- `internal/server/lock_observer.go`
- `cmd/devmount-soak/main.go`
- `scripts/run-sampled-soak.sh`
- `scripts/run-sampled-soak.ps1`
- `/runtimez` 状态入口

覆盖内容：

- metadata / sessions / journal runtime snapshot
- metadata / sessions / journal 三组 RW 锁等待观测
- metadata cache 过期命中路径的低风险锁竞争优化（不再为删除过期项升级写锁）
- 3~5 分钟 sampled soak 入口
- CSV / Markdown 报告导出

## 采样字段

每个 sample 至少包含：

- goroutines
- heap alloc / heap objects
- active / total / expired sessions
- nodes / node paths / dir cursors / handles
- attr cache / negative cache / dir snapshots / small-file cache
- watch events / watches / journal events / latest seq / oldest seq / max backlog / total backlog
- metadata read/write lock acquire count
- metadata read/write lock wait >50us / >1ms
- metadata read/write total wait / max wait

## 入口

Linux/macOS:

```bash
./scripts/run-sampled-soak.sh
```

Windows PowerShell:

```powershell
./scripts/run-sampled-soak.ps1
```

Dry-run only:

```bash
go run ./cmd/devmount-soak -dry-run
```

## 本轮沙盒执行说明

已在沙盒内完成：

- `go test ./internal/server -run 'TestServerSnapshotRuntimeIncludesCountsAndLockSamples|TestStatusHandlerHealthzStatusAndRuntimez|TestMetadataBackend|TestJournal|TestSessionManager' -count=1`
- `go test ./tests/integration -run 'TestRealisticMixedBrowseSaveWatchPressure|TestConnectionJitterRepeatedResumeAndRead' -count=1 -timeout 5m`
- `go build ./cmd/devmount-soak`
- `go run ./cmd/devmount-soak -dry-run`
- 10 秒 sampled soak smoke（可正常导出 CSV / Markdown）

但 3~5 分钟 soak 在当前沙盒里没有可靠跑完：

- 容器命令被外层 `timeout 35s` 包裹，无法直接前台等待 3 分钟。
- 改为 detached/background 路径后，命令本身会继续运行，但无法稳定在当前沙盒里完成 3 分钟结果回收。

因此本轮仓库重点是：**把 sampled soak 的正式入口、采样结构和锁等待观测完整落进项目，方便你在本地直接跑 3~5 分钟并回填结果。**

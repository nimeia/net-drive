# 2026-03-21 压测计划与问题统计

## 目标

- 验证 `scripts/run-stress-suite.ps1`、`scripts/run-sampled-soak.ps1`、`scripts/run-regression-compare.ps1` 在本地 Windows PowerShell 环境可重复执行。
- 验证 mixed browse/save/watch、并发 create/write/rename/watch、heartbeat、resume/read 压力路径的正确性。
- 验证 3 分钟 sampled soak 下的 runtime snapshot、锁等待、watch backlog、fault counter 是否可观测且可回收。
- 验证 metadata benchmark 是否仍满足当前门限，记录性能回归点。

## 环境与命令

- 时间：2026-03-21
- 平台：Windows / PowerShell
- Go：`go1.26.1 windows/amd64`
- 主执行命令：

```powershell
$env:OUT_DIR = 'dist\regression-local-20260321'
.\scripts\run-regression-compare.ps1
```

- 产物目录：
  - `dist/regression-local-20260321/stress/`
  - `dist/regression-local-20260321/soak/`
  - `dist/regression-local-20260321/regression-compare-report.md`

## 测试计划

1. 入口自检
   - 先执行 `dry-run` 或短时 smoke，确认脚本参数、输出路径、报告文件名和依赖命令都成立。

2. Stress suite
   - 执行 integration stress 组合，覆盖：
     - mixed browse/save/watch pressure
     - concurrent create/write/rename/watch
     - heartbeat interleaves with file operations
     - repeated disconnect/resume/read
   - 默认重复：
     - integration repeat = 3
     - mixed repeat = 20
     - metadata benchmark count = 3

3. Sampled soak
   - 运行 3 分钟 sampled soak。
   - 默认 worker：
     - browse = 4
     - save = 3
     - heartbeat = 2
     - resume = 2
   - 默认 fault injection：
     - slow client = on
     - half close = on
     - delayed write = on

4. 汇总与比对
   - 检查 unified regression report 是否完整落盘。
   - 对照 Iter 43 mixed workload baseline，统计本轮长尾是否收敛。
   - 对照 benchmark thresholds，记录门限违规项。

## 通过口径

- `go test` 用例无失败。
- soak `errors = 0`。
- goroutine / session / handle / backlog 不出现持续失控增长。
- mixed x20 不出现 correctness failure，并统计 `max >= 40ms` 的长尾频次。
- benchmark 输出完整可解析，并记录超阈值项。

## 本轮结果摘要

- 功能正确性：
  - integration 压力组合 3 轮全部通过。
  - mixed workload x20 全部通过。
  - sampled soak `errors = 0`。

- mixed x20 长尾：
  - 本轮 `max >= 40ms` 的指标为 `0/20`，较 Iter 43 的 metadata-heavy 长尾明显收敛。
  - 本轮 run-max 较高的指标：
    - `flush`: max `39.891 ms`
    - `rename`: max `36.883 ms`
    - `opendir`: max `35.143 ms`
    - `lookup_dir`: max `34.722 ms`
    - `write`: max `27.284 ms`

- 3 分钟 soak：
  - duration: `3m0.7657522s`
  - watch events: `45894`
  - goroutines before/after: `2 -> 2`
  - sampled goroutines: `first=28 peak=31 last=31`
  - handles: `first=0 peak=6 last=3`
  - max watch backlog: `peak=11 last=3`

- soak 中最明显的性能现象：
  - `opendir`: p95 `85.9397 ms`, max `800.705 ms`
  - `create`: p95 `27.7657 ms`, max `89.1859 ms`
  - `rename`: p95 `32.5156 ms`, max `86.5367 ms`
  - `flush`: p95 `29.5917 ms`, max `71.7519 ms`
  - `resume_connect`: p95 `23.4444 ms`, max `108.8423 ms`

- 锁等待：
  - metadata read wait `>1ms`: `36683`
  - metadata write wait `>1ms`: `51140`
  - session write wait `>1ms`: `1`
  - journal write wait `>1ms`: `3101`

- 内存：
  - heap alloc first: `3.35 MB`
  - heap alloc peak: `4119.49 MB`
  - heap alloc last: `3915.06 MB`
  - last 10 samples avg: `3335.64 MB`

- benchmark：
  - `LookupHot` / `GetAttrHot` / `ReadDirSnapshotHit` / `ReadSmallFileCached` 仍低于门限。
  - `CreateWriteFlushClose` 在完整回归里出现 `2/3` 次超过 `5.0 ms` 门限：
    - `5.932 ms`
    - `5.931 ms`
    - `4.089 ms`

## 问题统计

- 功能正确性问题：`0`
- 压测脚本/报告链路问题：`5`
  - 已修复：`4`
  - 待处理：`1`
- 主要性能问题：`3`

## 详细问题清单

### P1

1. **Soak 期间 heap alloc 明显偏高且未回落**
   - 3 分钟内从 `3.35 MB` 增长到 `4.12 GB` 峰值，结束时仍有 `3.92 GB`。
   - goroutine / handle / session 并未同步爆炸，更像对象留存或缓存回收不足，而不是线程泄漏。

2. **metadata-heavy 路径在 soak 中仍存在严重长尾**
   - `opendir` p95 达 `85.94 ms`，max 达 `800.71 ms`。
   - `create/flush/rename/resume_connect` 都已进入 `70~109 ms` 级别长尾。
   - metadata read/write `>1ms` 等待计数都很高，问题更像锁争用或热点路径阻塞。

3. **`BenchmarkMetadataCreateWriteFlushClose` 存在回归风险**
   - 完整回归里 `2/3` 次超过 `5.0 ms` 门限。
   - 即使单次 smoke 可能落回门限内，也说明该基准波动较大，稳定性不足。

### P2

1. **`run-sampled-soak.ps1` 原实现把 Go bool flag 传成 `-flag true`**
   - 结果：Go 在第一个 `true` 处停止继续解析 flag，`-csv/-report` 被忽略，`OUT_DIR` 实际不生效。
   - 已修复。

2. **`run-sampled-soak.ps1` 缺少稳定的 run log 落盘**
   - 结果：`render-regression-compare.py` 期望的 `sampled-soak-run.log` 不稳定。
   - 已修复。

3. **`render-regression-compare.py` baseline 提取过于脆弱**
   - 原实现依赖固定中文标题，终端/编码环境变化后容易退化。
   - 已改为基于稳定英文标题片段提取。

4. **`run-stress-suite.ps1` 原先用 `Tee-Object` 落盘 UTF-16 日志**
   - 结果：`devmount-benchgate` 在真实 Windows bench log 上无法解析，曾误报全部 benchmark 缺失。
   - 已修复为显式 UTF-8 落盘。

5. **benchmark gate 配置与 stress suite benchmark 范围不一致**
   - 当前 `benchmarks/thresholds.json` 还要求：
     - `BenchmarkJournalPoll`
     - `BenchmarkEncodeDecodeFrame`
   - 但 `run-stress-suite.ps1` 只跑 metadata benchmark，因此 gate 仍会报这两项 missing。
   - 待处理。

## 结论

- 本轮没有发现正确性故障，mixed x20 相比 Iter 43 已明显收敛。
- 当前最需要优先处理的不是功能错误，而是 **soak 下的内存占用异常**、**metadata-heavy 路径锁争用/长尾**、以及 **CreateWriteFlushClose 基准波动**。
- 脚本链路已基本可用，但 benchmark gate 的覆盖范围还需要和 stress suite 对齐，否则 CI/本地门禁结论仍不稳定。

## 建议的下一步

1. 优先沿 `sampled-soak-samples.csv` 排查 heap alloc 持续走高的对象来源，先确认是否由 metadata cache / dir snapshot / small-file cache 留存造成。
2. 针对 `opendir/create/flush/rename/resume_connect` 加细分阶段采样，确认长尾主要落在 metadata lock、journal path 还是 session recovery path。
3. 让 stress suite benchmark 范围与 `benchmarks/thresholds.json` 对齐，或者拆分出 metadata 专用 threshold 文件，避免 gate 误报 missing benchmark。

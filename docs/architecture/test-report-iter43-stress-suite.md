# Iter 43 — 压测脚本整理与首轮压力结果

## 目标

把本轮已验证过的压力测试命令沉淀为可复用脚本，避免后续每次都手工拼接：

- integration 并发/抖动/混合负载压力
- mixed workload 放大重复运行
- metadata backend benchmark

## 新增脚本

- `scripts/run-stress-suite.sh`
- `scripts/run-stress-suite.ps1`

默认输出目录：`dist/stress/`

可通过环境变量覆盖：

- `OUT_DIR`
- `INTEGRATION_REPEAT`
- `MIXED_REPEAT`
- `TIMEOUT`
- `GOMAXPROCS_VALUE`
- `RUN_BENCH`

## 默认执行内容

### 1) integration 压力组合

```bash
GOMAXPROCS=2 go test ./tests/integration   -run 'TestRealisticMixedBrowseSaveWatchPressure|TestMultiClientConcurrentCreateWriteRenameAndWatch|TestHeartbeatInterleavesWithFileOperations|TestConnectionJitterRepeatedResumeAndRead'   -count 3 -timeout 20m -v
```

### 2) mixed workload 放大重复

```bash
GOMAXPROCS=2 go test ./tests/integration   -run '^TestRealisticMixedBrowseSaveWatchPressure$'   -count 20 -timeout 20m -v
```

### 3) metadata backend benchmark

```bash
go test ./internal/server -run '^$'   -bench 'BenchmarkMetadata(LookupHot|GetAttrHot|ReadDirSnapshotHit|ReadSmallFileCached|CreateWriteFlushClose)$'   -benchmem -count 3
```

## 本轮沙盒首轮结果摘要（2026-03-19）

### 功能稳定性

- 所选 integration 压力用例全部通过。
- 并发 create/write/rename/watch 未出现正确性故障。
- repeated resume/read 未出现恢复链路错误。
- mixed browse/save/watch 压力未出现 request-level 失败。

### mixed workload 延迟形态

中位 p95 仍然较低：

- `heartbeat`: ~1.11 ms
- `lookup_dir`: ~0.93 ms
- `lookup_file`: ~0.85 ms
- `opendir`: ~1.30 ms
- `readdir`: ~0.83 ms
- `read`: ~0.82 ms
- `write`: ~1.21 ms
- `rename`: ~1.03 ms
- `resume_connect`: ~2.66 ms

但 max 尖峰可重复出现：

- `lookup_dir` median max: ~40.56 ms，最大观测到 53.65 ms
- `lookup_file` median max: ~40.58 ms，最大观测到 51.32 ms
- `opendir` median max: ~29.55 ms，最大观测到 53.77 ms
- `heartbeat` median max: ~24.60 ms，最大观测到 51.13 ms
- `write` median max: ~13.20 ms，最大观测到 45.68 ms
- `resume_connect` median max: ~3.18 ms，最大观测到 34.62 ms

20 次 mixed workload 重复里，长尾尖峰出现频率：

- `lookup_dir max >= 40 ms`: 11/20
- `lookup_file max >= 40 ms`: 11/20
- `opendir max >= 40 ms`: 9/20
- `heartbeat max >= 40 ms`: 4/20
- `readdir max >= 40 ms`: 6/20
- `read max >= 40 ms`: 6/20
- `write max >= 40 ms`: 2/20

结论：当前主要问题是 **metadata-heavy 路径的长尾延迟**，不是正确性或基础吞吐。

### metadata benchmark 状态

都明显低于当前门限：

- `BenchmarkMetadataLookupHot`: ~332–335 ns/op
- `BenchmarkMetadataGetAttrHot`: ~307–311 ns/op
- `BenchmarkMetadataReadDirSnapshotHit`: ~79 ns/op
- `BenchmarkMetadataReadSmallFileCached`: ~143–147 ns/op
- `BenchmarkMetadataCreateWriteFlushClose`: ~90–96 us/op

## 使用建议

### Linux / macOS

```bash
./scripts/run-stress-suite.sh
```

轻量自检一轮脚本包装层：

```bash
INTEGRATION_REPEAT=1 MIXED_REPEAT=1 RUN_BENCH=0 ./scripts/run-stress-suite.sh
```

### Windows PowerShell

```powershell
./scripts/run-stress-suite.ps1
```

轻量自检一轮脚本包装层：

```powershell
$env:INTEGRATION_REPEAT = '1'
$env:MIXED_REPEAT = '1'
$env:RUN_BENCH = '0'
./scripts/run-stress-suite.ps1
```

## 后续建议

下一轮建议直接在脚本之上补：

1. runtime sampling：goroutine / heap / active sessions / handles / watch backlog
2. metadata backend lock wait/hold 观测
3. 3~5 分钟 soak 与故障注入

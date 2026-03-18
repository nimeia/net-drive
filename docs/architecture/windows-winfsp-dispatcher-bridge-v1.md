# Iter 26 — 完整 WinFsp dispatcher 回调桥第一版

## 目标

把 Iter 24 的“dispatcher API 探测 + host scaffold”继续推进到一个真正的 **callback bridge 第一版**。

这意味着 dispatcher-v1 不再只是“知道 DLL 里有这些 API”，而是已经具备：

- 一个显式的 bridge 对象
- 对 volume / getattr / open / opendir / readdir / read / close 的统一调度入口
- host 启动阶段的 bridge warmup / state reporting
- 可测试的 callback bridge 状态与调用计数

## 本轮交付

### 1. 新增 `DispatcherBridge`

位置：`internal/winfsp/dispatcher_bridge.go`

能力：

- `Initialize("/")`
  - 预热 `GetVolumeInfo`
  - 预热 `GetFileInfo("/")`
- 包装以下回调：
  - `GetVolumeInfo`
  - `GetFileInfo`
  - `Open`
  - `OpenDirectory`
  - `ReadDirectory`
  - `Read`
  - `Close`
- 维护 bridge 状态：
  - initialized
  - root path
  - volume name
  - last ntstatus
  - last error
  - per-op call count

### 2. Host 持有 dispatcher bridge

`internal/winfsp/host.go` 现在会：

- 在 dispatcher-v1 / auto 路径上为 host 创建 `DispatcherBridge`
- 在 binding 已确定为 `winfsp-dispatcher-v1` 后，把 bridge 状态并入 binding / runtime 可见摘要

### 3. Windows dispatcher host 进入 bridge warmup

`internal/winfsp/dispatcher_windows.go` 现在不再只是等待 `ctx.Done()`。

它会：

1. 校验 dispatcher APIs ready
2. 校验 bridge 已创建
3. 执行 bridge `Initialize("/")`
4. 将 bridge summary 写回 dispatcher status
5. 进入 host 生命周期等待

### 4. Synthetic callback bridge tests

新增：`internal/winfsp/dispatcher_bridge_test.go`

验证：

- 初始化成功
- 调用计数更新
- 失败 NTSTATUS 会落到 bridge state

## 为什么这一轮重要

真正的 WinFsp Explorer 流量最终依赖的，不是“知道 API 存在”，而是：

- 回调是否有稳定入口
- 回调结果是否能映射到统一状态
- host 是否能在运行前确认 callback bridge 至少能处理根路径和基本 volume 信息

这一轮先把“桥”做出来，后续 ABI/dispatcher service loop 才有稳定接点。

## 当前边界

这一轮仍然不是完整的 WinFsp 生产级 ABI bridge。还没有完成：

- `FSP_FILE_SYSTEM_INTERFACE` 的完整原生结构体桥接
- 真实 dispatcher 线程 / service loop 全链路
- Explorer 全流量 smoke 的最终闭环

但和 Iter 24 相比，这一轮已经从“探测 API”推进到了“bridge 对象 + host warmup + callback testability”。

## 下一步建议

1. 将 bridge 继续扩到 create / write / flush / cleanup / rename / delete-on-close
2. 真正落地 `FSP_FILE_SYSTEM_INTERFACE` 原生桥接层
3. 增加 Windows 主机 Explorer smoke 记录
4. 给 dispatcher lifecycle 增加 metrics / structured logging

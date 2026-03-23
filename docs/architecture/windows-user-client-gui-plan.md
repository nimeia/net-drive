# Windows 最终用户 GUI（方案 B）

## 目标

把现有偏测试/支持的 Win32 窗口保留为 **Support Console**，新增一套面向最终用户的 **User Client**。

## 已落盘内容

- `cmd/devmount-client-win32` 切到独立 `winclientusergui` 入口
- 新增 `cmd/devmount-support-console`
- 新增 `internal/winclientproduct`
- `internal/winclientstore` 升级到 schema v2，支持 `settings` / `workspace_meta`

## 当前边界

由于本轮执行环境不稳定，`winclientusergui` 先落成独立入口与产品层骨架；原 Win32 壳层继续由 `winclientgui` 复用，后续再把 Home / Workspaces / Settings / Help 页面完全拆出来。

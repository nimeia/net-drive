# Windows 最终用户 GUI（方案 B）

## 目标

把现有偏测试/支持的 Win32 窗口保留为 **Support Console**，新增一套面向最终用户的 **User Client**。

## 结构

- `cmd/devmount-client-win32`：最终用户 GUI
- `cmd/devmount-support-console`：支持/测试 GUI
- `internal/winclientusergui`：最终用户 Win32 壳层
- `internal/winclientproduct`：面向最终用户的状态文案与摘要
- `internal/winclientstore`：升级到 schema v2，支持 settings / workspace_meta

## User Client 页面

- Home：连接、断开、打开挂载位置、打开 Support Console
- Workspaces：保存/加载/删除工作区，管理最终用户需要维护的连接信息
- Settings：默认工作区、自动重连、开机启动
- Help：自检、导出支持包、打开日志目录、打开 Support Console

## Support Console 保留职责

- Dashboard / Profiles / Diagnostics
- 高级配置
- CLI 预览
- 诊断原始信息
- 实施、测试、支持人员使用

# Iter 37 — Windows 主机首轮结果回填

## 目标

把真实 Windows 主机首轮验证从“手工改模板”升级为“有结构、有 patch、有合并流程”的结果回填链路。

## 新增能力

- `HostValidationRecord.Environment`
  - 记录测试机器、Windows 版本、WinFsp 版本、诊断包路径、安装日志目录
- `ValidationPatch`
  - 只描述当前一轮新增观察结果
  - 不要求一次性填完整个 validation record
- `scripts/backfill-windows-validation.ps1`
  - 把 patch 合并回 `windows-host-validation-result-template.json`
  - 重新计算 summary
  - 可顺带标记 `completed_by`

## 推荐流程

1. 在真实 Windows 主机上运行 Explorer smoke / MSI / EXE / recovery 验证
2. 只把本轮观察到的结果写入 `windows-host-backfill-patch-template.json`
3. 执行 `backfill-windows-validation.ps1`
4. 执行 `finalize-windows-release.ps1`
5. 检查新的 closure 和 pre-release issues

## 边界

这一轮新增的是“真实结果回填机制”，不是在当前 Linux 沙盒里伪造 Windows 实测结果。

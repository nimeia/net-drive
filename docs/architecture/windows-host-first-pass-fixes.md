# Iter 39 — 基于 Windows 主机首轮结果的修复入口

## 目标

把“Windows 主机首轮实测结果”真正接进仓库内的发布闭环，而不是只停留在 validation template。

## 本轮交付

- 新增 `internal/winclientrelease/intake.go`
  - `NewValidationPatchTemplate()`：从完整 validation record 生成完整 patch 模板
  - `InstallerResultSet` / `NewInstallerResultSetTemplate()`：结构化承载 MSI/EXE 首轮结果
  - `ApplyInstallerResultSet()`：把 installer 实测结果同步回 `installer_runs` 与关键 installer checklist
- 新增 `internal/winclientrelease/fixplan.go`
  - `FirstPassFixPlan`：基于 validation + issue list 生成首轮修复计划
  - 对不同类别问题给出 `suggested_area` 与 `suggested_files`
- 更新 diagnostics/export
  - 导出完整 `windows-host-backfill-patch-template.*`
  - 导出 `windows-first-pass-fix-plan.*`
- 更新回填脚本
  - `backfill-windows-validation.ps1` 支持同时合并：
    - validation patch
    - installer results json

## 解决的实际问题

上一轮模板虽然能表达“回填”，但存在两个不足：

1. patch 模板只覆盖了部分 scenario/checklist/run，无法完整记录首轮实测；
2. installer 结果与 validation 结果之间缺少正式的合并入口。

本轮把这两个缺口补齐后，Windows 主机第一轮验证结果可以更完整地落回仓库，并直接驱动后续修复计划生成。

## 当前边界

本轮仍然不会伪造 Windows 实测结果。真正的首轮结果仍需在本地 Windows 主机执行后回填。

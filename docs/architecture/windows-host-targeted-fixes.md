# Iter 41 — Windows 主机首轮结果导入与定点修复入口

## 本轮目标

在不伪造真实 Windows 主机结果的前提下，把“首轮回填结果 -> 问题归纳 -> 定点修复入口”真正落到仓库中。

## 新增内容

- `internal/winclientrelease/intake_report.go`
  - 基于 `windows-host-validation-result-template.json` 生成 `ValidationIntakeReport`
  - 统计：
    - 是否已标记 `completed_at`
    - 主机环境信息是否完整
    - diagnostics bundle / installer log 是否齐全
    - explorer / installer / checklist 未通过项
- `scripts/finalize-windows-release.ps1`
  - 在 closure / issues / fix-plan / RC 之外，新增输出：
    - `windows-validation-intake-report.json`
    - `windows-validation-intake-report.md`

## 设计边界

这一步不会伪造“真实 Windows 主机首轮结果”。

它做的是：
- 接住真实回填结果
- 判断结果是否足够完整，足以支撑“定点修复”
- 把缺失证据和未完成项显式列出，避免在证据不完整时误判为“可以发布”

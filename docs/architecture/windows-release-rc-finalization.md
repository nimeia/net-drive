# Iter 40 — 发布候选版 RC 收口与最终打包

## 目标

把 release closure、issue list、fix plan 进一步收口成正式的 RC 元数据与打包链路。

## 本轮交付

- 新增 `internal/winclientrelease/rc.go`
  - `ReleaseCandidate`
  - 汇总：manifest / validation / closure / issue list
  - 输出 `final_status`、`open_issues`、`artifact_count`
- 更新 diagnostics/export
  - 导出 `windows-release-candidate.*`
- 更新 `finalize-windows-release.ps1`
  - 除 closure 与 issues 外，额外输出：
    - `windows-first-pass-fix-plan.json/.md`
    - `windows-release-candidate.json/.md`
- 新增 `scripts/package-windows-rc.ps1`
  - 从 `windows-release` 目录收口出 `windows-rc` 目录
  - 聚合 RC 所需的 manifest、validation、closure、issues、fix plan、release candidate

## 建议使用顺序

1. `package-windows-release.ps1`
2. 在 Windows 主机回填 validation patch 与 installer results
3. `backfill-windows-validation.ps1`
4. `finalize-windows-release.ps1`
5. `package-windows-rc.ps1`

## 当前边界

RC 打包链路已经形成，但是否可发布仍然依赖真实 Windows 主机回填结果。RC 元数据不会伪造“ready”。

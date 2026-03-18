# Iter 38 — 发布前问题清单与收口修复

## 目标

把 release closure 从“只有 ready/not-ready 结论”推进到“有可执行问题清单”的收口形态。

## 新增能力

- `PreReleaseIssueList`
  - 根据 Explorer smoke、installer checklist、recovery checklist、installer runs、closure reasons 生成问题清单
- `finalize-windows-release.ps1`
  - 在生成 closure 的同时输出：
    - `windows-pre-release-issues.json`
    - `windows-pre-release-issues.md`
- diagnostics / release bundle
  - 输出 issues 模板与 backfill patch 模板

## 设计取舍

当前问题清单优先覆盖：

- 尚未执行
- 执行失败
- 执行有 warning
- closure 明确标记的 release blocker

它不是 bug tracker 的完整替代，而是发布前收口视图。

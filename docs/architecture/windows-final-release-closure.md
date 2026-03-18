# Iter 42 — RC 问题清零与正式发布包收口

## 本轮目标

在 RC 元数据之上，再增加“正式发布”层：

- 生成最终发布判定对象
- 生成正式发布签字模板
- 生成 `windows-final` 聚合目录

## 新增内容

- `internal/winclientrelease/final.go`
  - `FinalRelease`
  - 汇总：
    - manifest
    - validation
    - intake
    - closure
    - issues
    - RC
  - 计算：
    - `publish_ready`
    - `final_status`
- `scripts/package-windows-final.ps1`
  - 从 `windows-release` 收口到 `windows-final`
  - 聚合：
    - manifest
    - validation result
    - intake report
    - closure
    - issue list
    - fix plan
    - RC
    - final release
    - signoff

## 判定逻辑

只有同时满足以下条件，`publish_ready=true`：

- `closure.release_ready = true`
- `validation.summary.overall = pass`
- `issue_list.open_count = 0`
- `validation intake` 无缺失证据
- `release_candidate.final_status = rc-ready`

否则维持 `blocked` 或 `needs-attention`。

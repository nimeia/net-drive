# Iter 48 — regression compare 统一入口与报告汇总

## 本轮目标

把 Iter 43 的 stress suite 与 Iter 44/45/47 的 sampled soak 可观测性串成一条统一入口，
方便在本地一次执行后直接得到一份总报告，而不是只拿到零散日志。

## 新增脚本

- `scripts/run-regression-compare.sh`
- `scripts/run-regression-compare.ps1`
- `scripts/render-regression-compare.py`

## 默认输出

- `dist/regression/stress/`
- `dist/regression/soak/`
- `dist/regression/regression-compare-report.md`

## 运行方式

Linux/macOS：

```bash
./scripts/run-regression-compare.sh
```

Windows PowerShell：

```powershell
./scripts/run-regression-compare.ps1
```

## 干跑

```bash
DRY_RUN=1 ./scripts/run-regression-compare.sh
```

## 设计说明

- stress 与 soak 仍复用已有脚本：
  - `scripts/run-stress-suite.*`
  - `scripts/run-sampled-soak.*`
- 新增的 Python 汇总器只负责收拢现有产物，生成一个统一 Markdown 报告
- 报告里会保留 Iter 43 baseline 摘要，便于和本地新结果做对照

## 当前状态

本轮只完成工具链与文档落盘，不在沙盒里执行长时 stress / soak。

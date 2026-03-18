# Iter 25 — 错误码 / 日志模型 / 自检结果分级与诊断页增强

## 目标

把已有的“能跑自检、能导出 ZIP”升级成更接近产品诊断面的能力：

- 自检结果不再只有 pass/fail 文本，而是带 **code / category / severity / remediation**
- 客户端日志不再只是简单文本，而是带 **level / code / component / fields** 的结构化行模型
- 诊断输出可以直接作为 UI、日志排障和用户支持的共用文本格式

## 本轮交付

### 1. 结构化日志模型

`internal/winclientlog` 新增：

- `Entry`
  - `timestamp`
  - `level`
  - `code`
  - `component`
  - `message`
  - `fields`
- `Logger.Record(entry)`
- 保留 `Info / Warn / Error` 快捷入口
- tail 仍可直接给 Diagnostics 页展示

当前日志格式为可读的 key/value 行，兼顾：

- 人眼排查
- 文本 tail
- 后续转 JSONL 或解析器接入

### 2. 自检结果分级

`internal/winclientdiag` 新增：

- `CheckResult.Code`
- `CheckResult.Category`
- `CheckResult.Severity`
- `CheckResult.Remediation`
- `Report.Summary`
  - `pass`
  - `warn`
  - `fail`
  - `overall_severity`

当前覆盖的检查项：

- configuration
- WinFsp binding
- dispatcher callback bridge readiness
- server TCP connect
- store path
- log path
- runtime state

### 3. Diagnostics 文本增强

`Report.Text()` 现在会输出：

- overall severity
- pass/warn/fail summary
- 当前 backend / binding 摘要
- 每个检查项的 code 与 remediation
- dispatcher callback bridge 说明

### 4. WinFsp 错误码模型增强

`internal/winfsp/status.go` 新增：

- `StatusName(status)`
- `StatusCode(status)`
- `StatusError(status, err)` 输出结构化 code + NTSTATUS name

这样 Diagnostics / host / callback bridge 在报错时能共享更一致的错误文本。

## 为什么这一轮重要

产品化客户端的 Diagnostics 页不能只是“展示底层错误字符串”。

支持和排查通常需要：

1. 一个稳定的错误 code
2. 一个用户可读的 remediation
3. 一个足够短、但能直接复制到工单里的文本摘要

本轮把这三层先拉平，后续可以继续做：

- UI 上的颜色/图标分级
- 导出包里增加更细的环境清单
- 按 code 反查修复建议

## 当前边界

这一轮没有把 Diagnostics 页改成复杂的表格/树控件；仍然以增强文本输出为主。

这是有意的：先把 **诊断数据模型** 做稳，再决定 UI 呈现细节。

## 下一步建议

1. Diagnostics 页按 severity 分组显示
2. 给常见错误码加“Copy code / Copy remediation”动作
3. 导出 ZIP 增加最近 runtime state transition 摘要
4. 把结构化日志接到诊断导出里的单独文件

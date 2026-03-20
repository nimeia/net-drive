# Iter 47 — control path / fault injection 噪声收口

## 本轮范围

- 将 fault injection 下预期的连接级错误从逐条日志改为计数收口
- 在 `/runtimez` 中新增 fault log counters
- 将 fault log counters 接入 `cmd/devmount-soak` CSV / Markdown 报告
- 只跑短验证链路：targeted tests / build / dry-run

## 预期静默的错误类型

- `net.ErrClosed`
- `io.EOF`
- `io.ErrUnexpectedEOF`
- `broken pipe`
- `connection reset by peer` / `forcibly closed by the remote host`

## 说明

这些错误在 slow client / half-close / delayed-write 故障注入下属于预期行为。
将它们改为计数可以减少日志 I/O 干扰，同时仍然保留可观测性。

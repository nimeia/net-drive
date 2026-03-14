# Task.md

## Iter 0 — 协议与架构冻结

- [x] 明确产品定位、非目标、第一阶段边界
- [x] 产出 protocol-v0.1 字段级消息定义
- [x] 产出 session / save / watch-recovery 状态机
- [x] 产出 Windows-first MVP 任务分解

## Iter 1 — 连接与会话基线

- [x] 初始化仓库、工程目录、README、go module
- [x] 定义协议 header、opcode、错误码、基础消息结构
- [x] 实现 TCP 占位传输层（长度前缀 + 固定头 + JSON payload）
- [x] 实现 Hello / Auth / CreateSession / Heartbeat 服务端处理链路
- [x] 实现客户端握手与会话创建 demo
- [x] 增加协议编解码单测
- [x] 增加服务端集成测试
- [ ] 补充 ResumeSession 占位处理
- [ ] 输出迭代总结与下一阶段接口空壳

## Iter 2 — 只读挂载 MVP（计划）

- [ ] Lookup / GetAttr
- [ ] OpenDir / ReadDir
- [ ] Open / Read / Close
- [ ] metadata cache 基线
- [ ] 只读工作区浏览合同测试

## Iter 3 — metadata cache 与目录快照（计划）

- [ ] getattr cache
- [ ] readdir cache
- [ ] negative cache
- [ ] root prefetch

## Iter 4 — 写入与保存链路（计划）

- [ ] Create / Write / Flush / Close
- [ ] Truncate
- [ ] Rename / Replace
- [ ] delete-on-close
- [ ] save path 合同测试

## Iter 5 — watcher 与 journal（计划）

- [ ] Subscribe / event push / AckSeq
- [ ] overflow / repair
- [ ] 工作区目录修复

## Iter 6 — 恢复与重连（计划）

- [ ] ResumeSession
- [ ] RecoverHandles
- [ ] cache revalidate
- [ ] watch resubscribe

## Iter 7 — 编辑器专项优化（计划）

- [ ] workspace profile
- [ ] 热目录预热
- [ ] 小文件快取
- [ ] 请求优先级调度

## Iter 8 — 诊断与产品化（计划）

- [ ] mount manager
- [ ] 状态页面
- [ ] 日志导出
- [ ] 指标面板

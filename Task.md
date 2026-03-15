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
- [x] 补充 ResumeSession
- [x] 输出迭代总结与下一阶段接口空壳

## Iter 2 — 只读挂载 MVP
- [x] Lookup / GetAttr
- [x] OpenDir / ReadDir
- [x] Open / Read / Close
- [x] metadata cache 基线接入后端
- [x] 只读工作区浏览合同测试

## Iter 3 — metadata cache 与目录快照
- [x] getattr cache
- [x] readdir cache
- [x] negative cache
- [x] root prefetch

## Iter 4 — 写入与保存链路
- [x] Create / Write / Flush / Close
- [x] Truncate
- [x] Rename / Replace
- [x] delete-on-close
- [x] 保存链路与缓存失效测试

## Iter 5 — watcher 与 journal
- [x] Subscribe / poll events / AckSeq
- [x] overflow / resync snapshot
- [x] 工作区目录修复的最小路径（resync snapshot）

## Iter 6 — 恢复与重连
- [x] ResumeSession
- [x] RecoverHandles
- [x] cache revalidate
- [x] watch resubscribe
- [x] 恢复测试矩阵补齐（session resume / recover handles / revalidate / resubscribe）

## Iter 7 — 编辑器专项优化
- [x] workspace profile
- [x] 热目录预热
- [x] 小文件快取
- [x] 请求优先级调度

## Iter 8 — 诊断与产品化
- [x] JSON 配置加载
- [x] /healthz 与 /status 基线
- [x] JSONL 审计日志基线
- [x] build/package 脚本
- [x] 运行文档与示例配置

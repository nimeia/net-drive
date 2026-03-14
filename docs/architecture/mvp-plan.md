# Windows-first MVP Plan

## 产品定位

Windows-first、挂载型、面向编辑器工作负载优化的远程文件系统。

## 第一阶段目标

1. 先把控制面协议、会话、心跳、错误码基线做稳
2. 再进入只读 metadata 与目录浏览
3. 再进入保存链路与 watcher
4. 最后再接 WinFsp 挂载和更完整的文件系统语义

## 迭代拆分

### Iter 0：合同冻结
- 协议 v0.1 文档
- 架构文档
- 状态机
- Task.md

### Iter 1：连接与会话基线
- Hello/Auth/CreateSession/Heartbeat
- TCP 占位传输
- client/server/demo
- codec / session / integration tests

### Iter 2：只读挂载 MVP
- Lookup / GetAttr
- OpenDir / ReadDir
- Open / Read / Close

### Iter 3：metadata cache 与目录快照
- getattr cache
- readdir cache
- negative cache
- root prefetch

### Iter 4：写入与保存链路
- Create / Write / Flush / Close
- Truncate
- Rename / Replace
- delete-on-close

### Iter 5：watcher 与 journal
- Subscribe / event push / AckSeq
- overflow / repair

### Iter 6：恢复与重连
- ResumeSession
- RecoverHandles
- cache revalidate
- watch resubscribe

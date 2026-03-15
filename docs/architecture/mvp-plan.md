# Windows-first MVP Plan

## 产品定位

Windows-first、挂载型、面向编辑器工作负载优化的远程文件系统。

## 第一阶段目标

1. 先把控制面协议、会话、心跳、错误码基线做稳
2. 再进入只读 metadata 与目录浏览
3. 再进入保存链路与 watcher
4. 再进入恢复与编辑器专项优化
5. 最后补诊断与产品化收口，再接 WinFsp 挂载和更完整的文件系统语义

## 当前进度

### Iter 1：连接与会话基线
- Hello / Auth / CreateSession / ResumeSession / Heartbeat
- TCP 占位传输
- client / server / demo
- codec / session / integration tests

### Iter 2：只读挂载 MVP
- Lookup / GetAttr
- OpenDir / ReadDir
- Open / Read / Close
- 服务端本地目录只读后端
- 只读浏览与读取集成测试

### Iter 3：metadata cache 与目录快照
- getattr cache
- readdir cache
- negative cache
- root prefetch
- cache hit / TTL refresh 单测

### Iter 4：写入与保存链路
- Create / Write / Flush / Close
- Truncate
- Rename / Replace
- delete-on-close
- 保存链路缓存失效测试

### Iter 5：watcher 与 journal
- Subscribe / PollEvents / AckEvents / ResyncSnapshot
- bounded journal retention
- overflow -> resync snapshot 最小恢复路径

### Iter 6：恢复与重连
- ResumeSession
- RecoverHandles
- cache revalidate
- watch resubscribe
- focused recovery matrix

### Iter 7：编辑器专项优化
- workspace profile
- hot dir / hot file prefetch
- small-file cache
- priority-aware prefetch

### Iter 8：诊断与产品化
- JSON 配置加载
- /healthz 与 /status
- JSONL 审计日志
- build / package 脚本
- 示例配置与运行文档

## 后续方向

- WinFsp bridge
- push-style watcher streaming
- lease / oplock-style invalidation
- 更强 Windows 文件语义覆盖

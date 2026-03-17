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


## Iter 9 — 核心稳态与性能基线
- [x] 控制面负向合同测试（握手顺序 / 版本 / token / unsupported channel）
- [x] session gating 与 backend error mapping 测试
- [x] metadata backend 边界矩阵（overwrite / invalid handle / pagination / sparse write / cross-dir rename）
- [x] journal 单测补强（maxEvents / ack monotonic / watch-not-found / path match / resubscribe）
- [x] 第一轮 benchmark 基线（transport / metadata hot path / journal poll）

## Iter 10 — 第二轮核心补测
- [x] internal/client 单测（request header / error path / request-id mismatch / session tracking）
- [x] config / status / audit 单测
- [x] recovery 深边界测试（writable recover / delete-on-close / rename 后 revalidate / unknown previous watch resubscribe）
- [x] 第二轮测试报告与运行建议


## Iter 11 — 并发稳定性 / 烟测 / benchmark 门禁
- [x] 多客户端并发 / 抖动测试（并发 create-write-flush-rename + repeated resume jitter）
- [x] heartbeat 与文件操作交织测试
- [x] transport 截断帧 / 坏帧负向测试
- [x] cmd/devmount-server / cmd/devmount-client smoke test
- [x] benchmark 阈值与回归门禁（thresholds + gate parser + gate script）
- [x] 第三轮测试报告与运行建议

## Iter 12 — ReadDir 快路径优化 / 真实场景压力测试
- [x] 修复 ReadDirSnapshotHit 命中路径的多余 refresh 与切片复制
- [x] 回归 benchmark gate，确认 ReadDirSnapshotHit 恢复到 0 allocs/op
- [x] 增加更接近真实编辑器行为的 mixed browse/save/watch/resume 压力测试
- [x] 输出本轮压力测试与性能结论

## Iter 13 — Windows client core refactor / WinFsp 接入设计
- [x] 拆分 internal/clientcore（rpc / session / metadata / data / watch / recovery / state）
- [x] 增加 tracked handles / tracked watches / tracked nodes / recovery snapshot
- [x] 保持 internal/client 兼容包装层，避免 demo client 与既有测试回归
- [x] 补 internal/clientcore 状态与恢复单测
- [x] 输出 WinFsp 接入设计文档（windows-client-core-and-winfsp）


## Iter 14 — WinFsp read-only mount MVP
- [x] 新增 internal/platform/windows 路径规范化辅助
- [x] 新增 internal/mountcore（path cache / lookup / getattr / opendir / readdir / read / close）
- [x] 新增 internal/winfsp/adapter 只读操作映射层
- [x] 新增 cmd/devmount-winfsp smoke CLI，走 mountcore + adapter 链路
- [x] 增加 mountcore / adapter / Windows path 单测
- [x] 输出 Iter 14 只读 WinFsp MVP 设计与当前边界文档

## Iter 15 — WinFsp callback host / Windows-only build tags
- [x] 新增 internal/winfsp NTSTATUS 映射与 callback bridge
- [x] 新增 Windows-only host shell（host_windows.go / host_other.go）
- [x] cmd/devmount-winfsp 增加 -op mount 入口
- [x] 增加 callback mapping 单测
- [x] 增加 Windows 交叉编译验证
- [x] 输出 Iter 15 callback host / build-tags 设计文档

## Iter 16 — Win32 配置测试界面
- [x] 新增 internal/winclient，沉淀配置归一化 / 校验 / CLI 预览 / 执行逻辑
- [x] 新增 cmd/devmount-client-win32 原生 Win32 配置测试窗口
- [x] 支持 volume / getattr / readdir / read 四类本地测试动作
- [x] 支持生成等价 devmount-winfsp CLI 预览
- [x] 增加 winclient 单测
- [x] 更新 README 与 Win32 配置测试界面说明文档

## Iter 17 — 远端内容加载为本地文件
- [x] 新增 internal/materialize，支持按远端路径递归下载到本地目录
- [x] 为 materialize 增加单测，覆盖目录树、单文件和路径穿越防护
- [x] cmd/devmount-winfsp 增加 `-op materialize` 与 `-local-path`
- [x] Win32 配置测试界面增加 Local Path 字段，并支持 materialize 入口
- [x] 更新 README 与本地加载设计文档

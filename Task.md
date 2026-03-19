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


## Windows 客户端产品化计划（Epic）

| Epic | 范围 | 优先级 | 依赖 | 验收标准 |
| --- | --- | --- | --- | --- |
| E1 | 真实挂载运行时与 WinFsp 生命周期 | P0 | Iter 15 callback host 基线 | 支持真实 mount / unmount，Explorer 可见，状态机可观测 |
| E2 | Windows 产品化主界面 | P0 | E1 运行时状态、E3 配置 | 具备 Dashboard / Mounts / Settings / Diagnostics 主流程 |
| E3 | 配置持久化与凭据安全 | P0/P1 | 无 | 支持多 Profile、启动恢复最近配置、敏感信息安全存储 |
| E4 | 托盘、后台驻留与通知 | P0/P1 | E2 UI 主壳、E7 状态事件 | 托盘可驻留并控制挂载，关键事件有通知 |
| E5 | 诊断、自检与日志 | P0/P1 | E1/E7 运行时事件 | 有日志、自检、错误映射和诊断导出 |
| E6 | 安装、升级、卸载与发布 | P1 | E5 依赖检查、构建脚本 | 正式安装包可安装 / 升级 / 卸载 |
| E7 | 连接、会话恢复与自动重连 | P1 | clientcore 状态与恢复基线 | 断网恢复、认证恢复、启动恢复上次挂载 |
| E8 | 质量保障与测试体系 | P0/P1 | 各 Epic 核心逻辑 | 核心状态机、配置层、安装链路有自动化与 smoke 验证 |
| E9 | 文件系统语义与性能优化 | P1/P2 | E1 真实挂载 | Explorer / 编辑器兼容性和性能持续收口 |
| E10 | 增强功能与企业化能力 | P2/P3 | E3/E5/E6 | 模板配置、静默安装、策略约束等能力逐步补齐 |

## Iter 18 — Win32 客户端 Profile 持久化基线
- [x] 把 Windows 客户端产品化计划正式落盘到项目文档
- [x] 新增 internal/winclientstore，支持配置文件路径解析、JSON 持久化、Profile 保存 / 加载 / 删除
- [x] 启动时恢复最近一次激活的 Profile
- [x] Win32 客户端界面增加 Profile Name / Saved Profiles / Save / Load / Delete
- [x] 更新 README 与 Iter 18 文档说明
- [x] 为配置存储层增加跨平台单测

## Iter 19 — 主界面骨架拆分（Dashboard / Profiles / Diagnostics）
- [x] 把 Win32 客户端单页测试窗体拆成 Dashboard / Profiles / Diagnostics 三页骨架
- [x] 把现有 Profile / 配置编辑收敛到 Profiles 页
- [x] 把高级 smoke 动作收敛到 Diagnostics 页
- [x] Dashboard 页展示当前挂载状态、当前 Profile 与快捷动作
- [x] 更新 README 与 Iter 19 界面骨架说明文档

## Iter 20 — 真实 mount runtime 状态机接入 UI
- [x] 新增 internal/winclientruntime，抽象 mount runtime builder / session / state machine
- [x] 打通 connecting / mounted / stopping / idle / error 状态流转
- [x] Dashboard 接入 Start Mount / Stop Mount，并实时展示 runtime 状态
- [x] Diagnostics 页接入 runtime 摘要与 mount CLI 预览
- [x] 为 mount runtime 状态机增加跨平台单测
- [x] 更新 README 与 Iter 20 mount runtime 文档

## Iter 21 — 托盘 / 后台驻留 / 通知
- [x] Win32 客户端增加通知区托盘图标与托盘菜单
- [x] 支持关闭/最小化后驻留托盘，双击托盘恢复主窗口
- [x] 托盘菜单支持打开窗口、切换 Dashboard / Profiles / Diagnostics、启动/停止挂载、退出
- [x] 关键状态切换（mounted / stopping / error / idle）通过托盘通知提示
- [x] Dashboard 与 Diagnostics 补充托盘/后台驻留说明
- [x] 更新 README 与 Iter 21 文档说明

## Iter 22 — 真实 WinFsp host binding 收口到 mount runtime
- [x] 新增 internal/winfsp binding 探测层，识别 WinFsp DLL / launcher 路径
- [x] Windows host 在 runtime 构建前执行 WinFsp 原生 `FspFileSystemPreflight` 挂载点校验
- [x] mount runtime snapshot / Dashboard / Diagnostics 展示 host binding 状态、DLL 路径、launcher 路径
- [x] WinFsp host run 路径接入 binding 结果，缺少 WinFsp 或 mount-point preflight 失败时直接报错
- [x] 为 binding 摘要增加跨平台单测
- [x] 更新 README 与 Iter 22 文档说明

## Iter 23 — 日志 / 自检 / 诊断导出
- [x] 新增 internal/winclientlog，支持默认日志路径、追加写入与 tail
- [x] 新增 internal/winclientdiag，支持 WinFsp / 网络 / 配置 / runtime 摘要自检
- [x] Win32 Diagnostics 页增加 Run Self-Check / Export Diagnostics 入口
- [x] 托盘菜单增加 Export Diagnostics 入口
- [x] cmd/devmount-winfsp 增加 `-op selfcheck` / `-op export-diagnostics`
- [x] 为日志层与诊断层增加跨平台单测
- [x] 更新 README 与 Iter 23 诊断文档说明

## Iter 24 — 完整 WinFsp SDK dispatcher host 第一版
- [x] winclient Config / Profiles / CLI 增加 Host Backend（auto / preflight / dispatcher-v1）
- [x] WinFsp binding probe 增加 dispatcher API 可用性探测
- [x] mount runtime snapshot / Dashboard / Diagnostics 展示 requested/effective backend 与 dispatcher 状态
- [x] host run 路径按 effective backend 分流到 preflight / dispatcher-v1 scaffold
- [x] 为 dispatcher backend 摘要与 Config 增加单测
- [x] 更新 README 与 Iter 24 dispatcher host 第一版文档说明



## Iter 25 — 错误码 / 日志模型 / 自检结果分级与诊断页增强
- [x] 把 winclientlog 升级为结构化日志模型，支持 level / code / component / fields
- [x] 为 diagnostics checks 增加 code / category / severity / remediation
- [x] 生成 diagnostics summary（pass/warn/fail + overall severity）
- [x] WinFsp status 输出统一的 StatusName / StatusCode / StatusError
- [x] Diagnostics 文本输出增强到可直接用于排障工单
- [x] 为日志模型、自检分级和错误码补跨平台单测
- [x] 更新 README 与 Iter 25 文档说明

## Iter 26 — 完整 WinFsp dispatcher 回调桥第一版
- [x] 新增 internal/winfsp DispatcherBridge，承接 volume / getattr / open / opendir / readdir / read / close
- [x] host 持有 dispatcher bridge，并把 bridge 摘要接入 binding/runtime 可见状态
- [x] dispatcher-v1 host 运行前执行 bridge warmup（GetVolumeInfo + root GetFileInfo）
- [x] 为 callback bridge 增加 synthetic 单测，覆盖初始化、调用计数和失败状态
- [x] Windows 交叉编译验证 devmount-client-win32 与 devmount-winfsp
- [x] 更新 README 与 Iter 26 文档说明


## Iter 27 — 完整 WinFsp ABI bridge / dispatcher service loop 第一版
- [x] 新增 DispatcherABI，把 volume / getattr / open / opendir / readdir / read / close 映射到 ABI-facing bridge
- [x] 新增 DispatcherService，提供 dispatcher-v1 的 start / stop / warmup / state 摘要
- [x] host binding / runtime / diagnostics 摘要增加 callback bridge 与 service loop 状态
- [x] 为 ABI bridge / service loop 增加 synthetic 单测
- [x] 更新 README 与 Iter 27 文档说明

## Iter 28 — Windows 主机 Explorer smoke / 安装链路 / 崩溃恢复联合收口
- [x] 新增 Explorer smoke 清单导出（Markdown / JSON）
- [x] 新增 winclientrecovery，记录脏退出 / 干净退出与最近运行态
- [x] Diagnostics 导出打包 recovery.json 与 Explorer smoke 清单
- [x] 新增 scripts/package-windows-installer.ps1，形成 Windows 安装链路 stage
- [x] Dashboard / Diagnostics 展示 recovery 状态
- [x] 更新 README 与 Iter 28 文档说明


## Iter 29 — 完整 WinFsp native callback table / Explorer 请求矩阵收口
- [x] 新增 WinFsp native callback table 摘要、Markdown/JSON 导出与单测
- [x] 新增 Explorer request matrix 摘要、Markdown/JSON 导出与单测
- [x] Diagnostics / 导出包接入 native callback table 与 request matrix
- [x] Dashboard / Diagnostics 文案更新为 callback coverage + request matrix
- [x] 更新 README 与 Iter 29 设计文档

## Iter 30 — MSI/EXE 安装包正式链路 + Windows 主机联合验证
- [x] 新增 MSI 打包脚本（WiX source/output）
- [x] 新增 EXE/portable bundle 打包脚本
- [x] 新增统一 Windows release packaging 脚本
- [x] 新增 release manifest / validation checklist 生成逻辑与单测
- [x] 新增 Windows 主机联合验证文档与发布校验清单
- [x] 更新 README 与安装链路说明


## Iter 31 — 完整 WinFsp security / cleanup / flush callback 收口
- [x] 新增 Cleanup / Flush / GetSecurityByName / GetSecurity 回调并接到 callbacks / bridge / ABI
- [x] dispatcher service warmup 覆盖 security / flush / cleanup 路径
- [x] native callback table 收口上述回调为 ready
- [x] Explorer request matrix 增加 properties/security / cleanup / flush 请求映射
- [x] 增加对应单测并更新 README / Iter 31 文档说明

## Iter 32 — Windows 主机真实 Explorer 验证记录 + 安装包实机闭环
- [x] 新增 Windows host validation record 模板生成逻辑与单测
- [x] diagnostics zip 导出 validation template（Markdown / JSON）
- [x] release / installer 脚本增加 validation template 与实机闭环材料
- [x] 新增 Iter 32 文档说明与 README 更新

## Iter 33 — 完整 WinFsp native security descriptor / cleanup semantics / set-delete-on-close 收口
- [x] 新增 native-style 安全描述符模型，输出 owner/group/access/SDDL 与 handle-bound 状态
- [x] 回调层新增 CanDelete / SetDeleteOnClose，并把 delete-on-close denial 语义贯通到 callbacks / bridge / ABI / service warmup
- [x] Cleanup / Flush 语义补充 handle 状态跟踪，安全查询可见 cleaned/flushed/delete-on-close
- [x] native callback table 与 Explorer request matrix 增加 delete-denied 场景与 callback 覆盖
- [x] 为安全描述符、delete-on-close、callback/request matrix 增加跨平台单测

## Iter 34 — Windows 主机实机验证结果回填 + MSI/EXE 安装升级卸载实测收口
- [x] Host validation record 增加 installer runs、summary、completed metadata 与回填 API
- [x] diagnostics ZIP 增加 windows-host-validation-result-template.*
- [x] release manifest 增加 validation template / installer result slots
- [x] Windows host validation 模板覆盖 MSI install/upgrade/uninstall 与 EXE portable launch
- [x] 更新 README 与 Iter 34 文档说明


## Iter 35 — 完整 WinFsp native callback 实装矩阵最后收口
- [x] 为 Create / Write / SetBasicInfo / SetFileSize / SetSecurity / Rename / Overwrite 增加显式只读拒绝回调
- [x] 把上述回调贯通到 callbacks / dispatcher bridge / ABI / service warmup
- [x] native callback table 与 Explorer request matrix 收口为 finalized=true、gap=0 的最后一版矩阵
- [x] Explorer smoke 增加 create-denied / write-denied / rename-denied 场景
- [x] 为 callback matrix 最后一版收口补跨平台单测

## Iter 36 — Windows 主机实机结果回填 + 发布链路最终收口
- [x] 新增 ReleaseClosure，汇总 release manifest + validation result 并计算 release_ready
- [x] diagnostics ZIP 增加 windows-release-closure-template.*
- [x] 发布脚本增加 windows-release-closure-template.* 产物
- [x] 新增 finalize-windows-release.ps1，用于基于回填结果生成最终 release closure
- [x] 更新 README 与 Iter 36 文档说明


## Iter 37 — 真实 Windows 主机回填首轮结果
- [x] 为 HostValidationRecord 增加环境信息结构，支持记录机器、OS、WinFsp 版本、诊断包与安装日志目录
- [x] 新增 ValidationPatch，支持把首轮 Windows 主机结果合并回 validation result template
- [x] 新增 backfill-windows-validation.ps1，支持把 patch 合并到 windows-host-validation-result-template.json
- [x] diagnostics / release 包新增 windows-host-backfill-patch-template 输出
- [x] 为回填 patch 与环境合并增加单测

## Iter 38 — 发布前问题清单与收口修复
- [x] 新增 PreReleaseIssueList，按 explorer / installer / recovery / closure 生成发布前问题清单
- [x] finalize-windows-release.ps1 新增 windows-pre-release-issues.json/.md 输出
- [x] release manifest 增加 backfill patch / issue list 路径
- [x] diagnostics 导出新增 windows-pre-release-issues 模板
- [x] 为 issue list 生成增加单测
- [x] 更新 README 与 Iter 37/38 文档说明

## Iter 39 — 基于 Windows 主机首轮结果做真实问题修复
- [x] 新增完整 ValidationPatch 模板生成，覆盖 explorer / installer / recovery / installer runs
- [x] 新增 InstallerResultSet，并支持把 MSI/EXE 首轮结果合并回 validation result
- [x] 新增 FirstPassFixPlan，基于 validation + issues 生成首轮修复计划
- [x] diagnostics / release 导出新增 fix plan 与完整 backfill patch 模板
- [x] 更新 README 与 Iter 39 文档说明

## Iter 40 — 发布候选版 RC 收口与最终打包
- [x] 新增 ReleaseCandidate，汇总 manifest / validation / closure / issue list 为 RC 元数据
- [x] finalize-windows-release.ps1 输出 closure / issues / fix plan / RC 四类最终文件
- [x] 新增 package-windows-rc.ps1，生成 windows-rc 目录
- [x] release manifest 增加 fix plan / release candidate 路径
- [x] 更新 README 与 Iter 40 文档说明

## Iter 41 — 基于你本地回填的首轮 Windows 结果做定点修复
- [x] 新增 ValidationIntakeReport，显式判断真实 Windows 主机回填结果是否足以支撑定点修复
- [x] finalize-windows-release.ps1 新增 intake report 输出
- [x] diagnostics / release 产物新增 intake report
- [x] 为 intake report 增加单测

## Iter 42 — RC 问题清零与正式发布包收口
- [x] 新增 FinalRelease，汇总 manifest / validation / intake / closure / issues / RC 为正式发布判定
- [x] 新增 windows-final-signoff.md 输出
- [x] 新增 package-windows-final.ps1，生成 windows-final 聚合目录
- [x] diagnostics / release 产物新增 final release / signoff 模板
- [x] 更新 README 与 Iter 41/42 文档说明


## Iter 43 — 压测脚本整理与压力回归入口
- [x] 新增 Linux/macOS `scripts/run-stress-suite.sh`
- [x] 新增 Windows `scripts/run-stress-suite.ps1`
- [x] 将 integration 压力组合 / mixed workload repeat / metadata benchmark 固化为可复用入口
- [x] 产出 Iter 43 压测脚本与首轮结果文档


## Iter 44 — runtime snapshot / 锁争用观测 / sampled soak
- [x] 新增 server runtime snapshot（metadata / sessions / journal）
- [x] 新增 metadata / session / journal 三组 RW 锁等待观测
- [x] metadata cache 过期命中路径不再升级写锁删除旧项
- [x] 新增 `/runtimez` 状态入口
- [x] 新增 `cmd/devmount-soak` 与 `scripts/run-sampled-soak.{sh,ps1}`
- [x] 为 runtime snapshot 与 status handler 补单测
- [ ] 跑完 3~5 分钟 sampled soak 并回填最终报告（沙盒受单次 35s 命令上限影响，当前仓库已具备本地执行入口）

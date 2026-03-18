# Iter 31 — WinFsp security / cleanup / flush callback 收口

这一轮把 dispatcher-v1 从“能响应基础 browse/read 流量”推进到“覆盖更多 Explorer 生命周期请求”的第一版。

## 收口内容

- `Cleanup`
  - 对已知 handle 返回成功
  - 作为 read-only 模式下的轻量句柄清理阶段
- `Flush`
  - 对已知 handle 返回成功
  - 作为 read-only 兼容路径，避免 Explorer/编辑器在 flush 上直接落到 gap
- `GetSecurityByName`
  - 按路径返回一个最小可读的默认安全描述符
- `GetSecurity`
  - 按打开句柄返回最小可读的默认安全描述符

## 为什么这样做

当前客户端仍是 read-only 模式，所以：

- `SetSecurity / Write / Rename / Overwrite / SetFileSize / SetBasicInfo` 继续保留为 read-only deny
- 但 `Cleanup / Flush / GetSecurity*` 是 Explorer/属性页/只读复制链路里更容易实际出现的请求
- 因此先把它们收口成“安全的只读成功路径”，比继续把它们标为 gap 更接近真实主机行为

## 当前边界

这里仍不是完整 WinFsp 原生 ABI 桥最终版：

- 安全描述符还是最小静态模板，不是来自远端 ACL 的真实翻译
- `Cleanup` 没有做复杂 delete-on-close / oplock / rename-pending 行为
- `Flush` 没有做写缓存刷新，只是 read-only success path

## 结果

- native callback table 中：
  - `Cleanup / Flush / GetSecurityByName / GetSecurity` 进入 ready
- Explorer request matrix 中：
  - 属性页/安全查询/只读复制/停止挂载后的 cleanup 不再是已知 gap

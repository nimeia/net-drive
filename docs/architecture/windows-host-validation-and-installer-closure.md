# Iter 32 — Windows 主机 Explorer 验证记录 + 安装包实机闭环

这一轮不声称“已经完成所有真实 Windows 主机验证”，而是把**记录模板、安装校验材料、诊断导出附件**补齐，方便在真实 Windows 主机上执行并归档。

## 新增内容

- `internal/winclientrelease/validation.go`
  - 生成 Windows host validation record 模板
  - 包含：
    - Explorer smoke 场景结果
    - Installer checklist
    - Recovery checklist
- diagnostics ZIP 现在额外包含：
  - `windows-host-validation-template.md`
  - `windows-host-validation-template.json`
- Windows release / installer 脚本补充：
  - host validation 模板
  - installer validation 模板 / checklist

## 目的

把之前的：

- smoke 场景清单
- native callback table
- request matrix
- diagnostics export
- release manifest

整理成一套更适合实机执行和回填结果的材料。

## 推荐实机闭环步骤

1. 安装 WinFsp
2. 安装 MSI 或解压 EXE/portable bundle
3. 启动 `devmount-client-win32.exe`
4. 跑 Diagnostics -> Self-Check
5. 跑 Explorer smoke 场景
6. 导出 diagnostics ZIP
7. 回填 `windows-host-validation-template.md/json`
8. 归档安装日志、diagnostics ZIP、validation record

package winclientproduct

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientdiag"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientstore"
)

type WorkspaceSummary struct {
	Name        string
	DisplayName string
	MountPoint  string
	RemotePath  string
	ServerAddr  string
	LastUsedAt  string
	AutoMount   bool
	IsActive    bool
}

func FriendlyStatus(snapshot winclientruntime.Snapshot) string {
	switch snapshot.Phase {
	case winclientruntime.PhaseMounted:
		return fmt.Sprintf("已连接，工作区已挂载到 %s", fallback(snapshot.MountPoint, "挂载点"))
	case winclientruntime.PhaseConnecting:
		return fmt.Sprintf("正在连接 %s 并准备挂载", fallback(snapshot.ServerAddr, "服务器"))
	case winclientruntime.PhaseStopping:
		return fmt.Sprintf("正在断开 %s", fallback(snapshot.MountPoint, "当前工作区"))
	case winclientruntime.PhaseError:
		return FriendlyRuntimeError(snapshot.LastError)
	default:
		if strings.TrimSpace(snapshot.MountPoint) != "" {
			return fmt.Sprintf("当前未连接。上一次使用的挂载位置是 %s", snapshot.MountPoint)
		}
		return "当前未连接任何工作区。"
	}
}

func FriendlyRuntimeError(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return "运行状态异常，请检查连接并重试。"
	}
	switch {
	case strings.Contains(err, "token"):
		return "身份凭据可能已失效，请重新登录或更新访问令牌。"
	case strings.Contains(err, "invalid mount point syntax"):
		return "挂载位置格式不正确，请使用盘符或本地绝对目录。"
	case strings.Contains(err, "already exists; WinFsp expects a new mount leaf"):
		return "挂载目录已存在，请换一个新的子目录作为挂载位置。"
	case strings.Contains(err, "connection refused") || strings.Contains(err, "no such host") || strings.Contains(err, "dial tcp"):
		return "无法连接到服务端，请检查地址、网络或防火墙设置。"
	case strings.Contains(err, "WinFsp"):
		return "当前设备缺少可用的 WinFsp 运行环境，请先安装或修复 WinFsp。"
	default:
		return "连接失败：" + err
	}
}

func HomeSummary(snapshot winclientruntime.Snapshot, state winclientstore.State) string {
	active := strings.TrimSpace(snapshot.ActiveProfile)
	if active == "" {
		active = strings.TrimSpace(state.ActiveProfile)
	}
	workspaceName := active
	if meta, ok := state.WorkspaceMeta[active]; ok && strings.TrimSpace(meta.DisplayName) != "" {
		workspaceName = meta.DisplayName + " (" + active + ")"
	}
	if workspaceName == "" {
		workspaceName = "未选择工作区"
	}
	cfg, _ := state.Profiles[active]
	lines := []string{
		"Developer Mount 用户客户端",
		"",
		"状态：" + FriendlyStatus(snapshot),
		"当前工作区：" + workspaceName,
		"服务地址：" + fallback(snapshot.ServerAddr, cfg.Addr),
		"挂载位置：" + fallback(snapshot.MountPoint, cfg.MountPoint),
		"远端路径：" + fallback(snapshot.RemotePath, cfg.Path),
		"访问模式：只读挂载",
	}
	if snapshot.ExpiresAt != "" {
		lines = append(lines, "会话到期："+snapshot.ExpiresAt)
	}
	if strings.TrimSpace(snapshot.LastError) != "" {
		lines = append(lines, "", "最近错误："+FriendlyRuntimeError(snapshot.LastError))
	}
	lines = append(lines, "", "常用操作：连接、断开、在资源管理器中打开挂载位置、切换工作区。")
	return strings.Join(lines, "\n")
}

func WorkspaceSummaries(state winclientstore.State) []WorkspaceSummary {
	items := make([]WorkspaceSummary, 0, len(state.Profiles))
	for _, name := range winclientstore.SortedProfileNames(state) {
		cfg := state.Profiles[name].Normalized()
		meta := state.WorkspaceMeta[name]
		display := meta.DisplayName
		if display == "" {
			display = name
		}
		items = append(items, WorkspaceSummary{Name: name, DisplayName: display, MountPoint: cfg.MountPoint, RemotePath: cfg.Path, ServerAddr: cfg.Addr, AutoMount: meta.AutoMount, IsActive: name == state.ActiveProfile, LastUsedAt: formatTimestamp(meta.LastUsedAt)})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func WorkspacesSummary(state winclientstore.State) string {
	items := WorkspaceSummaries(state)
	if len(items) == 0 {
		return "还没有保存任何工作区。请在 Workspaces 页面填写连接信息后保存。"
	}
	lines := []string{"已保存工作区", ""}
	for _, item := range items {
		flags := []string{}
		if item.IsActive {
			flags = append(flags, "当前")
		}
		if item.AutoMount {
			flags = append(flags, "开机自动连接")
		}
		flagText := ""
		if len(flags) > 0 {
			flagText = " [" + strings.Join(flags, " / ") + "]"
		}
		lines = append(lines,
			fmt.Sprintf("- %s%s", item.DisplayName, flagText),
			"  名称："+item.Name,
			"  服务地址："+fallback(item.ServerAddr, "-"),
			"  远端路径："+fallback(item.RemotePath, "/"),
			"  挂载位置："+fallback(item.MountPoint, "-"),
			"  最近使用："+fallback(item.LastUsedAt, "-"),
		)
	}
	return strings.Join(lines, "\n")
}

func SettingsSummary(state winclientstore.State) string {
	return strings.Join([]string{
		"基础设置",
		"",
		"默认工作区：" + fallback(state.Settings.DefaultWorkspace, "未设置"),
		"自动重连：" + offOn(state.Settings.AutoReconnect),
		"开机启动：" + offOn(state.Settings.LaunchOnLogin),
		"",
		"建议：普通用户仅维护默认工作区和自动重连，其余高级排障能力放到 Support Console。",
	}, "\n")
}

func HelpSummary(report winclientdiag.Report, logPath, supportConsolePath string) string {
	severity := "-"
	if report.Summary.OverallSeverity != "" {
		severity = strings.ToUpper(string(report.Summary.OverallSeverity))
	}
	lines := []string{
		"帮助与支持",
		"",
		"自检概览：" + severity,
		fmt.Sprintf("检查结果：通过 %d / 警告 %d / 失败 %d", report.Summary.Pass, report.Summary.Warn, report.Summary.Fail),
		"日志位置：" + fallback(logPath, "-"),
		"支持控制台：" + fallback(supportConsolePath, "未找到 companion 程序"),
		"",
		"建议动作：",
		"1. 先运行自检。",
		"2. 仍有问题时导出支持包并发送给支持人员。",
		"3. 需要高级日志和诊断页时打开 Support Console。",
	}
	if len(report.Checks) > 0 {
		lines = append(lines, "", "最近检查：")
		limit := 3
		if len(report.Checks) < limit {
			limit = len(report.Checks)
		}
		for i := 0; i < limit; i++ {
			c := report.Checks[i]
			lines = append(lines, fmt.Sprintf("- %s: %s", c.Name, fallback(c.Detail, "-")))
		}
	}
	return strings.Join(lines, "\n")
}

func SupportConsolePath(clientExecutable string) string {
	clientExecutable = strings.TrimSpace(clientExecutable)
	if clientExecutable == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(clientExecutable), "devmount-support-console.exe")
}

func WorkspaceDisplayName(name string, meta winclientstore.WorkspaceMeta) string {
	if strings.TrimSpace(meta.DisplayName) != "" {
		return strings.TrimSpace(meta.DisplayName)
	}
	return strings.TrimSpace(name)
}

func DefaultWorkspaceMeta(name string, cfg winclient.Config) winclientstore.WorkspaceMeta {
	meta := winclientstore.WorkspaceMeta{DisplayName: strings.TrimSpace(name)}
	if cfg.MountPoint != "" {
		meta.LastUsedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return meta
}

func formatTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

func offOn(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}
func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

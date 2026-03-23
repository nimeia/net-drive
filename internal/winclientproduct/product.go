package winclientproduct

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		return "当前未连接任何工作区。"
	}
}

func FriendlyRuntimeError(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return "运行状态异常，请检查连接并重试。"
	}
	switch {
	case strings.Contains(err, "connection refused") || strings.Contains(err, "no such host") || strings.Contains(err, "dial tcp"):
		return "无法连接到服务端，请检查地址、网络或防火墙设置。"
	case strings.Contains(err, "WinFsp"):
		return "当前设备缺少可用的 WinFsp 运行环境，请先安装或修复 WinFsp。"
	default:
		return "连接失败：" + err
	}
}

func WorkspaceSummaries(state winclientstore.State) []WorkspaceSummary {
	items := make([]WorkspaceSummary, 0, len(state.Profiles))
	for _, name := range winclientstore.SortedProfileNames(state) {
		cfg := state.Profiles[name].Normalized()
		meta := state.WorkspaceMeta[name]
		display := strings.TrimSpace(meta.DisplayName)
		if display == "" {
			display = name
		}
		items = append(items, WorkspaceSummary{
			Name:        name,
			DisplayName: display,
			MountPoint:  cfg.MountPoint,
			RemotePath:  cfg.Path,
			ServerAddr:  cfg.Addr,
			AutoMount:   meta.AutoMount,
			IsActive:    name == state.ActiveProfile,
			LastUsedAt:  formatTimestamp(meta.LastUsedAt),
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func HelpSummary(report winclientdiag.Report, logPath, supportConsolePath string) string {
	severity := "-"
	if report.Summary.OverallSeverity != "" {
		severity = strings.ToUpper(string(report.Summary.OverallSeverity))
	}
	return strings.Join([]string{
		"帮助与支持",
		"",
		"自检概览：" + severity,
		fmt.Sprintf("检查结果：通过 %d / 警告 %d / 失败 %d", report.Summary.Pass, report.Summary.Warn, report.Summary.Fail),
		"日志位置：" + fallback(logPath, "-"),
		"支持控制台：" + fallback(supportConsolePath, "未找到 companion 程序"),
	}, "\n")
}

func SupportConsolePath(clientExecutable string) string {
	clientExecutable = strings.TrimSpace(clientExecutable)
	if clientExecutable == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(clientExecutable), "devmount-support-console.exe")
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

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

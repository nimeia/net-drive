//go:build windows

package winclientgui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientstore"
)

func (a *app) loadProfiles() error {
	state, err := a.store.Load()
	if err != nil {
		return err
	}
	a.state = state
	a.populateProfileList()
	if cfg, ok := state.Profiles[state.ActiveProfile]; ok {
		a.applyConfig(cfg)
		a.setText(idProfileName, state.ActiveProfile)
		return nil
	}
	a.resetDefaults()
	if state.ActiveProfile != "" {
		a.setText(idProfileName, state.ActiveProfile)
	} else {
		a.setText(idProfileName, "default")
	}
	return nil
}
func (a *app) saveProfile() {
	cfg, err := a.readConfigFields()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		_ = a.logError("save profile config error: " + err.Error())
		return
	}
	name := strings.TrimSpace(a.text(idProfileName))
	state, err := a.store.SaveProfile(name, cfg)
	if err != nil {
		a.setOutput("save profile failed: " + err.Error())
		_ = a.logError("save profile failed: " + err.Error())
		return
	}
	a.state = state
	a.populateProfileList()
	a.refreshRuntimeViews()
	message := "saved profile \"" + name + "\" to " + a.store.Path()
	a.setOutput(message)
	_ = a.logInfo(message)
}
func (a *app) loadSelectedProfile() {
	name := strings.TrimSpace(a.text(idProfileName))
	if name == "" {
		name = a.selectedComboText(idSavedProfiles)
	}
	if name == "" {
		a.setOutput("load profile failed: select or type a profile name")
		return
	}
	cfg, ok := a.state.Profiles[name]
	if !ok {
		a.setOutput("load profile failed: profile \"" + name + "\" not found")
		return
	}
	a.applyConfig(cfg)
	a.state.ActiveProfile = name
	if err := a.store.Save(a.state); err != nil {
		a.setOutput("profile loaded but active-profile save failed: " + err.Error())
		_ = a.logError("save active profile failed: " + err.Error())
		return
	}
	a.populateProfileList()
	a.setText(idProfileName, name)
	a.refreshRuntimeViews()
	message := "loaded profile \"" + name + "\""
	a.setOutput(message)
	_ = a.logInfo(message)
}
func (a *app) deleteSelectedProfile() {
	name := strings.TrimSpace(a.text(idProfileName))
	if name == "" {
		name = a.selectedComboText(idSavedProfiles)
	}
	state, err := a.store.DeleteProfile(name)
	if err != nil {
		a.setOutput("delete profile failed: " + err.Error())
		_ = a.logError("delete profile failed: " + err.Error())
		return
	}
	a.state = state
	a.populateProfileList()
	if cfg, ok := a.state.Profiles[a.state.ActiveProfile]; ok {
		a.applyConfig(cfg)
		a.setText(idProfileName, a.state.ActiveProfile)
	} else {
		a.resetDefaults()
		a.setText(idProfileName, "default")
	}
	a.refreshRuntimeViews()
	message := "deleted profile \"" + name + "\""
	a.setOutput(message)
	_ = a.logInfo(message)
}
func (a *app) populateProfileList() {
	combo := a.controls[idSavedProfiles]
	procSendMessage.Call(combo, cbResetContent, 0, 0)
	selectedIndex := uintptr(0)
	hasSelection := false
	for index, name := range winclientstore.SortedProfileNames(a.state) {
		procSendMessage.Call(combo, cbAddString, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))))
		if name == a.state.ActiveProfile {
			selectedIndex = uintptr(index)
			hasSelection = true
		}
	}
	if hasSelection {
		procSendMessage.Call(combo, cbSetCurSel, selectedIndex, 0)
	} else {
		procSendMessage.Call(combo, cbSetCurSel, ^uintptr(0), 0)
	}
}
func (a *app) resetDefaults() {
	a.applyConfig(winclient.DefaultConfig())
	procSendMessage.Call(a.controls[idOperation], cbSetCurSel, 0, 0)
}
func (a *app) applyConfig(cfg winclient.Config) {
	cfg = cfg.Normalized()
	a.setText(idAddr, cfg.Addr)
	a.setText(idToken, cfg.Token)
	a.setText(idClientInstance, cfg.ClientInstanceID)
	a.setText(idLeaseSeconds, strconv.FormatUint(uint64(cfg.LeaseSeconds), 10))
	a.setText(idMountPoint, cfg.MountPoint)
	a.setText(idVolumePrefix, cfg.VolumePrefix)
	a.setText(idPath, cfg.Path)
	a.setText(idLocalPath, cfg.LocalPath)
	a.setComboSelection(idHostBackend, cfg.HostBackend)
	a.setText(idOffset, strconv.FormatInt(cfg.Offset, 10))
	a.setText(idLength, strconv.FormatUint(uint64(cfg.Length), 10))
	a.setText(idMaxEntries, strconv.FormatUint(uint64(cfg.MaxEntries), 10))
}
func (a *app) readConfig() (winclient.Config, winclient.Operation, error) {
	cfg, err := a.readConfigFields()
	if err != nil {
		return winclient.Config{}, "", err
	}
	op, err := a.selectedOperation()
	if err != nil {
		return winclient.Config{}, "", err
	}
	if err := cfg.Validate(op); err != nil {
		return winclient.Config{}, "", err
	}
	return cfg, op, nil
}
func (a *app) readConfigFields() (winclient.Config, error) {
	cfg := winclient.Config{Addr: a.text(idAddr), Token: a.text(idToken), ClientInstanceID: a.text(idClientInstance), MountPoint: a.text(idMountPoint), VolumePrefix: a.text(idVolumePrefix), Path: a.text(idPath), LocalPath: a.text(idLocalPath), HostBackend: a.selectedHostBackend()}
	lease, err := parseUint32Field("lease seconds", a.text(idLeaseSeconds))
	if err != nil {
		return winclient.Config{}, err
	}
	offset, err := parseInt64Field("offset", a.text(idOffset))
	if err != nil {
		return winclient.Config{}, err
	}
	length, err := parseUint32Field("read length", a.text(idLength))
	if err != nil {
		return winclient.Config{}, err
	}
	maxEntries, err := parseUint32Field("max entries", a.text(idMaxEntries))
	if err != nil {
		return winclient.Config{}, err
	}
	cfg.LeaseSeconds = lease
	cfg.Offset = offset
	cfg.Length = length
	cfg.MaxEntries = maxEntries
	return cfg.Normalized(), nil
}
func (a *app) selectedOperation() (winclient.Operation, error) {
	index, _, _ := procSendMessage.Call(a.controls[idOperation], cbGetCurSel, 0, 0)
	if int(index) < 0 || int(index) >= len(a.operations) {
		return "", fmt.Errorf("select an operation")
	}
	return a.operations[int(index)], nil
}
func (a *app) selectedHostBackend() string {
	value := a.selectedComboText(idHostBackend)
	if value == "" {
		value = winclient.HostBackendAuto
	}
	return winclient.NormalizeHostBackend(value)
}

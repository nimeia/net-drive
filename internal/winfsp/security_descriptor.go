package winfsp

import "strings"

type SecurityDescriptorSource string

const (
	SecuritySourceByName   SecurityDescriptorSource = "by-name"
	SecuritySourceByHandle SecurityDescriptorSource = "by-handle"
)

type NativeSecurityDescriptor struct {
	Path          string                   `json:"path"`
	SDDL          string                   `json:"sddl"`
	Owner         string                   `json:"owner"`
	Group         string                   `json:"group"`
	Access        []string                 `json:"access,omitempty"`
	ReadOnly      bool                     `json:"read_only"`
	Directory     bool                     `json:"directory"`
	HandleBound   bool                     `json:"handle_bound"`
	DeleteOnClose bool                     `json:"delete_on_close,omitempty"`
	CleanupState  string                   `json:"cleanup_state,omitempty"`
	FlushState    string                   `json:"flush_state,omitempty"`
	Source        SecurityDescriptorSource `json:"source"`
}

func (d NativeSecurityDescriptor) Summary() string {
	parts := []string{d.Owner + "/" + d.Group}
	if d.ReadOnly {
		parts = append(parts, "readonly")
	}
	if d.Directory {
		parts = append(parts, "directory")
	} else {
		parts = append(parts, "file")
	}
	if d.HandleBound {
		parts = append(parts, "handle")
	}
	if d.DeleteOnClose {
		parts = append(parts, "delete-on-close")
	}
	if d.CleanupState != "" {
		parts = append(parts, d.CleanupState)
	}
	if d.FlushState != "" {
		parts = append(parts, d.FlushState)
	}
	return strings.Join(parts, ",")
}

type SecurityDescriptorOptions struct {
	HandleBound   bool
	DeleteOnClose bool
	Cleaned       bool
	Flushed       bool
	Source        SecurityDescriptorSource
}

func DefaultNativeSecurityDescriptor(info FileInfo, opts SecurityDescriptorOptions) NativeSecurityDescriptor {
	sddl := "O:BAG:BAD:PAI(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)"
	access := []string{"system:full", "administrators:full", "world:read"}
	if info.IsDirectory {
		sddl = "O:BAG:BAD:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FR;;;WD)"
		access = []string{"system:full(inherit)", "administrators:full(inherit)", "world:read(inherit)"}
	}
	cleanupState := "active"
	if opts.Cleaned {
		cleanupState = "cleanup-complete"
		if opts.DeleteOnClose {
			cleanupState = "delete-on-close-denied"
		}
	}
	flushState := "not-flushed"
	if opts.Flushed {
		flushState = "flushed"
	}
	if opts.Source == "" {
		if opts.HandleBound {
			opts.Source = SecuritySourceByHandle
		} else {
			opts.Source = SecuritySourceByName
		}
	}
	return NativeSecurityDescriptor{
		Path:          info.Path,
		SDDL:          sddl,
		Owner:         "BA",
		Group:         "BA",
		Access:        access,
		ReadOnly:      true,
		Directory:     info.IsDirectory,
		HandleBound:   opts.HandleBound,
		DeleteOnClose: opts.DeleteOnClose,
		CleanupState:  cleanupState,
		FlushState:    flushState,
		Source:        opts.Source,
	}
}

package winfsp

import (
	"developer-mount/internal/mountcore"
	"errors"
	"fmt"
	"strings"
)

type NTStatus uint32

const (
	StatusSuccess            NTStatus = 0x00000000
	StatusObjectNameNotFound NTStatus = 0xC0000034
	StatusObjectPathNotFound NTStatus = 0xC000003A
	StatusAccessDenied       NTStatus = 0xC0000022
	StatusInvalidHandle      NTStatus = 0xC0000008
	StatusFileIsADirectory   NTStatus = 0xC00000BA
	StatusNotADirectory      NTStatus = 0xC0000103
	StatusInternalError      NTStatus = 0xC00000E5
)

func mapError(err error) NTStatus {
	if err == nil {
		return StatusSuccess
	}
	switch {
	case errors.Is(err, mountcore.ErrInvalidHandle):
		return StatusInvalidHandle
	case errors.Is(err, mountcore.ErrIsDirectory):
		return StatusFileIsADirectory
	case errors.Is(err, mountcore.ErrNotDirectory):
		return StatusNotADirectory
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "ERR_NOT_FOUND") || strings.Contains(msg, "not found"):
		return StatusObjectNameNotFound
	case strings.Contains(msg, "ERR_NOT_DIR"):
		return StatusNotADirectory
	case strings.Contains(msg, "ERR_IS_DIR"):
		return StatusFileIsADirectory
	case strings.Contains(msg, "ERR_INVALID_HANDLE") || strings.Contains(msg, "invalid handle"):
		return StatusInvalidHandle
	case strings.Contains(msg, "ERR_ACCESS_DENIED"):
		return StatusAccessDenied
	case strings.Contains(msg, "invalid path"):
		return StatusObjectPathNotFound
	default:
		return StatusInternalError
	}
}
func StatusError(status NTStatus, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("ntstatus=0x%08x: %w", uint32(status), err)
}

package winfsp

import (
	"developer-mount/internal/mountcore"
	"errors"
	"fmt"
	"strings"
)

type NTStatus uint32
type ErrorCode string

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
const (
	CodeOK                 ErrorCode = "winfsp.ok"
	CodeObjectNameNotFound ErrorCode = "winfsp.object_name_not_found"
	CodeObjectPathNotFound ErrorCode = "winfsp.object_path_not_found"
	CodeAccessDenied       ErrorCode = "winfsp.access_denied"
	CodeInvalidHandle      ErrorCode = "winfsp.invalid_handle"
	CodeFileIsDirectory    ErrorCode = "winfsp.file_is_directory"
	CodeNotDirectory       ErrorCode = "winfsp.not_directory"
	CodeInternalError      ErrorCode = "winfsp.internal_error"
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
func StatusName(status NTStatus) string {
	switch status {
	case StatusSuccess:
		return "STATUS_SUCCESS"
	case StatusObjectNameNotFound:
		return "STATUS_OBJECT_NAME_NOT_FOUND"
	case StatusObjectPathNotFound:
		return "STATUS_OBJECT_PATH_NOT_FOUND"
	case StatusAccessDenied:
		return "STATUS_ACCESS_DENIED"
	case StatusInvalidHandle:
		return "STATUS_INVALID_HANDLE"
	case StatusFileIsADirectory:
		return "STATUS_FILE_IS_A_DIRECTORY"
	case StatusNotADirectory:
		return "STATUS_NOT_A_DIRECTORY"
	case StatusInternalError:
		return "STATUS_INTERNAL_ERROR"
	default:
		return fmt.Sprintf("NTSTATUS_0x%08X", uint32(status))
	}
}
func StatusCode(status NTStatus) ErrorCode {
	switch status {
	case StatusSuccess:
		return CodeOK
	case StatusObjectNameNotFound:
		return CodeObjectNameNotFound
	case StatusObjectPathNotFound:
		return CodeObjectPathNotFound
	case StatusAccessDenied:
		return CodeAccessDenied
	case StatusInvalidHandle:
		return CodeInvalidHandle
	case StatusFileIsADirectory:
		return CodeFileIsDirectory
	case StatusNotADirectory:
		return CodeNotDirectory
	default:
		return CodeInternalError
	}
}
func StatusError(status NTStatus, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("code=%s ntstatus=%s(0x%08x): %w", StatusCode(status), StatusName(status), uint32(status), err)
}

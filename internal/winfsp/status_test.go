package winfsp

import (
	"errors"
	"strings"
	"testing"
)

func TestStatusNameAndCode(t *testing.T) {
	if got := StatusName(StatusObjectNameNotFound); got != "STATUS_OBJECT_NAME_NOT_FOUND" {
		t.Fatalf("StatusName = %q", got)
	}
	if got := StatusCode(StatusInvalidHandle); got != CodeInvalidHandle {
		t.Fatalf("StatusCode = %q", got)
	}
}
func TestStatusErrorIncludesStructuredCode(t *testing.T) {
	err := StatusError(StatusNotADirectory, errors.New("open directory failed"))
	if err == nil {
		t.Fatal("StatusError returned nil")
	}
	for _, want := range []string{"code=winfsp.not_directory", "STATUS_NOT_A_DIRECTORY", "open directory failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("StatusError() = %q, missing %q", err.Error(), want)
		}
	}
}

package util

import (
	"errors"
	"testing"
)

func TestSanitizeError_ReturnsGenericMsg(t *testing.T) {
	// 详细错误含 internal IP / file path, 但对 client 仅返通用文案。
	err := errors.New("dial tcp 192.168.1.10:8080: i/o timeout")
	got := sanitizeError(err, "detail")
	want := "请求处理失败"
	if got != want {
		t.Fatalf("sanitizeError = %q want %q", got, want)
	}
}

func TestSanitizeError_NilErrSafe(t *testing.T) {
	if got := sanitizeError(nil, "x"); got != "请求处理失败" {
		t.Fatalf("nil err should still return generic msg, got %q", got)
	}
}

func TestRedactTarget_HidesHost(t *testing.T) {
	got := RedactTarget("http://192.168.1.10:8080/api/v1")
	// 不允许泄露 host/IP
	if got == "http://192.168.1.10:8080/api/v1" {
		t.Fatalf("target should be redacted, got %q", got)
	}
	if got == "" || got == "<invalid-target>" {
		t.Fatalf("target parse failed, got %q", got)
	}
	// 只有 scheme + port + path 可泄漏
	want := "http://<redacted>:8080/api/v1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRedactTarget_InvalidURL(t *testing.T) {
	if got := RedactTarget(":::not-a-url"); got != "<invalid-target>" {
		t.Fatalf("invalid url act as invalid, got %q", got)
	}
}

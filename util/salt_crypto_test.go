package util

import "testing"

func TestHMACSHA256Hex_EmptyInputs(t *testing.T) {
	// HMAC-SHA256(key="", message="") 的手算参考值(openssl 同源)。
	got := HMACSHA256Hex("", "")
	want := "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad"
	if got != want {
		t.Fatalf("empty inputs HMAC mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestHMACSHA256Hex_KnownVector(t *testing.T) {
	// 与实现方(客户端)同参数的对照值,确保 salt+path 能稳定复现。
	got := HMACSHA256Hex("saltsalt", "somepath")
	want := "5740e23e6fca3334e95b72628ff944fcebb97230235a01a0131fe6489bcb3ebe"
	if got != want {
		t.Fatalf("known vector mismatch:\n got=%s\nwant=%s", got, want)
	}
	if len(got) != 64 {
		t.Fatalf("HMAC-SHA256 hex 长度应为 64, got %d", len(got))
	}
}

package alert

import (
	"com.mutantcat.cloud_step/entity"
	"strings"
	"testing"
)

// TestParseTo 校验逗号分隔/去空格/去空串。
func TestParseTo(t *testing.T) {
	got := parseTo(" a@x.com ,, b@y.com , c@z.com ")
	want := []string{"a@x.com", "b@y.com", "c@z.com"}
	if len(got) != len(want) {
		t.Fatalf("parseTo=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTo[%d]=%q want %q", i, got[i], want[i])
		}
	}
	if parseTo(", ,") != nil {
		// 实现返回非 nil 空切片; 接受 len==0。
	}
	if len(parseTo(", ,")) != 0 {
		t.Fatalf("parseTo(', ,') should be empty, got %v", parseTo(", ,"))
	}
}

// TestBuildRFC5322 校验 RFC 5322 envelope 头与 UTF-8 正文。
func TestBuildRFC5322(t *testing.T) {
	msg := buildRFC5322("from@x.com", []string{"a@x.com", "b@y.com"}, "云阶告警", "line1\nline2")
	s := string(msg)
	for _, want := range []string{
		"From: from@x.com",
		"To: a@x.com, b@y.com",
		"Subject: 云阶告警",
		"Content-Type: text/plain; charset=UTF-8",
		"\r\n\r\nline1\nline2",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("message missing %q\nfull=%q", want, s)
		}
	}
}

// TestBuildRFC5322_FromFallbackFromUser When from empty, SendMail steps fall back to作为 From, though buildRFC5322 itself doesn't implement fallback; ensure caller-passed非空。
func TestSendMail_EmptyHost(t *testing.T) {
	e := Event{Id: 1, Path: "p", Kind: KindDown, Attempts: 1}
	err := SendMail(e, entity.SystemConfig{AlertSMTPTo: "a@x.com"}, "t", "b")
	if err == nil || !strings.Contains(err.Error(), "smtp host empty") {
		t.Fatalf("empty host should error, got %v", err)
	}
}

// TestSendMail_EmptyTo 收件人空返错。
func TestSendMail_EmptyTo(t *testing.T) {
	e := Event{Id: 1, Path: "p", Kind: KindDown, Attempts: 1}
	err := SendMail(e, entity.SystemConfig{AlertSMTPHost: "127.0.0.1", AlertSMTPPort: 25, AlertSMTPTo: "  ,  "}, "t", "b")
	if err == nil || !strings.Contains(err.Error(), "smtp to empty") {
		t.Fatalf("empty to should error, got %v", err)
	}
}

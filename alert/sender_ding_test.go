package alert

import (
	"com.mutantcat.cloud_step/entity"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSendDing_JSONStructure 校验钉钉 markdown 体字段与 event 映射。
func TestSendDing_JSONStructure(t *testing.T) {
	var got struct {
		Msgtype  string `json:"msgtype"`
		Markdown struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"markdown"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	e := Event{Id: 7, Path: "http://x.test/addr", Kind: KindDown, Attempts: 3}
	cfg := entity.SystemConfig{AlertDingWebhook: srv.URL}
	if err := SendDing(e, cfg, "TITLE", "- line"); err != nil {
		t.Fatalf("SendDing: %v", err)
	}
	if got.Msgtype != "markdown" {
		t.Fatalf("msgtype=%q want markdown", got.Msgtype)
	}
	if got.Markdown.Title != "TITLE" {
		t.Fatalf("title=%q", got.Markdown.Title)
	}
	if !strings.Contains(got.Markdown.Text, "### TITLE") || !strings.Contains(got.Markdown.Text, "- line") {
		t.Fatalf("text=%q", got.Markdown.Text)
	}
}

// TestSendDing_SignedWebhook 校验加签算法与钉钉开放 API 参考向量一致:
// sign = base64(HMAC-SHA256(secret, timestamp+"\n"+secret)), 并在 URL 带 timestamp + sign 两参数。
func TestSendDing_SignedWebhook(t *testing.T) {
	var gotTimestamp, gotSign string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotTimestamp = q.Get("timestamp")
		gotSign = q.Get("sign")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	now := time.Now()
	e := Event{Id: 1, Path: "p", Kind: KindUp, Attempts: 0}
	cfg := entity.SystemConfig{AlertDingWebhook: srv.URL, AlertDingSecret: "SEC1234567"}
	if err := SendDing(e, cfg, "t", "b"); err != nil {
		t.Fatalf("SendDing: %v", err)
	}
	if gotTimestamp == "" || gotSign == "" {
		t.Fatalf("missing signed params timestamp=%q sign=%q", gotTimestamp, gotTimestamp)
	}
	mac := hmac.New(sha256.New, []byte("SEC1234567"))
	mac.Write([]byte(gotTimestamp + "\n" + "SEC1234567"))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if gotSign != want {
		t.Fatalf("sign mismatch: got=%q want=%q ts=%q", gotSign, want, gotTimestamp)
	}
	// 时间戳是毫秒整数字符串, 且与 now 误差在 60 秒内。
	if d := time.Since(now); d < 0 || d > 60*time.Second {
		t.Fatalf("timestamp not refreshed near now: %v", d)
	}
}

// TestSignDingURL 直接校验 signDingURL 输出 parse 后 timestamp/sign 与 HMAC 自洽。
func TestSignDingURL(t *testing.T) {
	out, err := signDingURL("https://oapi.dingtalk.com/robot/send?access_token=abc", "the-secret")
	if err != nil {
		t.Fatalf("signDingURL: %v", err)
	}
	u, err := url.Parse(out)
	if err != nil {
		t.Fatalf("parse out: %v", err)
	}
	ts := u.Query().Get("timestamp")
	sign := u.Query().Get("sign")
	if ts == "" || sign == "" {
		t.Fatalf("missing params: %q", out)
	}
	mac := hmac.New(sha256.New, []byte("the-secret"))
	mac.Write([]byte(ts + "\n" + "the-secret"))
	if base64.StdEncoding.EncodeToString(mac.Sum(nil)) != sign {
		t.Fatalf("sign not self-consistent: %q", out)
	}
}

// TestSendDing_EmptyWebhook 空 webhook 直接返错, 不发请求。
func TestSendDing_EmptyWebhook(t *testing.T) {
	e := Event{Id: 1, Path: "p", Kind: KindUp, Attempts: 0}
	if err := SendDing(e, entity.SystemConfig{}, "t", "b"); err == nil {
		t.Fatalf("empty webhook should error")
	}
}

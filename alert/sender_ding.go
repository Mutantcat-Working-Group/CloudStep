package alert

import (
	"com.mutantcat.cloud_step/entity"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// SendDing 向钉钉群机器人推一条 markdown 告警。secret 非空走加签, 空则 plain webhook。
// 10s 超时; 失败仅 return err 由调用方 log, 不阻塞 worker。
func SendDing(e Event, cfg entity.SystemConfig, title, body string) error {
	if cfg.AlertDingWebhook == "" {
		return fmt.Errorf("ding webhook empty")
	}

	text := "### " + title + "\n" + body
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	endpoint := cfg.AlertDingWebhook
	if cfg.AlertDingSecret != "" {
		endpoint, err = signDingURL(cfg.AlertDingWebhook, cfg.AlertDingSecret)
		if err != nil {
			return fmt.Errorf("sign: %w", err)
		}
	}

	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Post(endpoint, "application/json", readerOf(raw))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	//钉钉返回 200 + {"errcode":0} 表示成功; 仅 read 几字节用于日志, 忽略解析。
	n, _ := io.Copy(io.Discard, resp.Body)
	_ = n
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d", resp.StatusCode)
	}
	log.Printf("[alert] ding posted url id=%d kind=%v endpoint=%s", e.Id, e.Kind, endpoint)
	return nil
}

// signDingURL 按钉钉开放 API 给 webhook 加签:
// timestamp(毫秒) + "\n" + secret → HMAC-SHA256(key=secret) → base64 → urlencode,
// 拼到 webhook 后作为 timestamp + sign 两个 query 参数。
func signDingURL(webhook, secret string) (string, error) {
	ts := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	msg := ts + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return "", err
	}
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	u, err := url.Parse(webhook)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("timestamp", ts)
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

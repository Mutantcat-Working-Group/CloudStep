package alert

import (
	"com.mutantcat.cloud_step/entity"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"
)

// SendMail 以 RFC 5322 plaintext 形式发送告警邮件。
// SMTP 发送起独立 goroutine, 父 worker 通过 select+10s 超时等待, 超时记日志不阻塞。
// toList 由 cfg.AlertSMTPTo 按逗号分割并去空格/空串。
func SendMail(e Event, cfg entity.SystemConfig, title, body string) error {
	host := strings.TrimSpace(cfg.AlertSMTPHost)
	if host == "" {
		return fmt.Errorf("smtp host empty")
	}
	toList := parseTo(cfg.AlertSMTPTo)
	if len(toList) == 0 {
		return fmt.Errorf("smtp to empty")
	}
	from := strings.TrimSpace(cfg.AlertSMTPFrom)
	if from == "" {
		from = strings.TrimSpace(cfg.AlertSMTPUser)
	}

	addr := host + ":" + fmt.Sprintf("%d", cfg.AlertSMTPPort)
	subject := title
	msg := buildRFC5322(from, toList, subject, body)

	var auth smtp.Auth
	if u := strings.TrimSpace(cfg.AlertSMTPUser); u != "" {
		auth = smtp.PlainAuth("", u, cfg.AlertSMTPPassword, host)
	}

	// 隔离到子 goroutine, 防慢 SMTP 拖住 worker; 10s 超时兜底。
	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, from, toList, msg)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("sendmail: %w", err)
		}
		log.Printf("[alert] mail posted url id=%d kind=%v to=%v", e.Id, e.Kind, toList)
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("sendmail timeout after 10s addr=%s", addr)
	}
}

// buildRFC5322 拼装 RFC 5322 文本邮件体(utf-8 + CRLF)。
func buildRFC5322(from string, toList []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(strings.Join(toList, ", "))
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// parseTo 把逗号分隔的字符串拆成去空格、去空串的收件人列表。
func parseTo(s string) []string {
	out := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

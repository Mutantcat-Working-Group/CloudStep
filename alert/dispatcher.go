package alert

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"context"
	"fmt"
	"log"
	"time"
)

// Start 启动单 worker goroutine 消费 EventCh, 限速 ≤2 msg/s。
// ctx 取消时 worker 退出。调用方应在进程生命周期内持有一个 ctx(通常 background)。
func Start(ctx context.Context) {
	go func() {
		// 每 500ms 放行一条 → ≤ 2 msg/s, 防钉钉 20 msg/min 限流。
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case e := <-EventCh:
				processEvent(e)
				<-ticker.C
			case <-ctx.Done():
				log.Printf("[alert] dispatcher stopped: %v", ctx.Err())
				return
			}
		}
	}()
}

// processEvent 处理单条事件: 读 url 与 sysConfig → 防抖/开关判定 → 发送双通道 → 写回 url 告警字段。
func processEvent(e Event) {
	u, ok := dao.GetUrl(e.Id)
	if !ok {
		log.Printf("[alert] url id=%d not found, skip", e.Id)
		return
	}

	cfg := util.GetSysConfigMirror()
	if !cfg.AlertEnabled {
		return
	}
	if !cfg.AlertDingEnabled && !cfg.AlertMailEnabled {
		return
	}

	// 防抖: 同一 URL 同一 kind 在窗口内不重复发。
	debounce := time.Duration(cfg.AlertDebounceSec) * time.Second
	if debounce < 0 {
		debounce = 600 * time.Second
	}
	kindIsDown := e.Kind == KindDown
	if u.LastAlertAt != nil &&
		time.Since(*u.LastAlertAt) < debounce &&
		u.LastAlertIsDown == kindIsDown {
		return
	}

	title, body := buildMessage(e, u)

	if cfg.AlertDingEnabled {
		if err := SendDing(e, cfg, title, body); err != nil {
			log.Printf("[alert] ding send url id=%d failed: %v", e.Id, err)
		}
	}
	if cfg.AlertMailEnabled {
		if err := SendMail(e, cfg, title, body); err != nil {
			log.Printf("[alert] mail send url id=%d failed: %v", e.Id, err)
		}
	}

	now := time.Now()
	isDown := e.Kind == kindDown()
	failCount := u.LastAlertFailCount
	if isDown {
		failCount++
	} else {
		failCount = 0
	}
	if !dao.UpdateUrlAlertState(e.Id, now, isDown, failCount) {
		log.Printf("[alert] update url alert state id=%d failed", e.Id)
	}
}

// buildMessage 拼装钉钉 markdown title + 通用 body(亦用于邮件正文)。
func buildMessage(e Event, u entity.Url) (title, body string) {
	now := time.Now().Format(time.RFC3339)
	if e.Kind == KindDown {
		title = fmt.Sprintf("云阶告警: URL %s 下线", u.Path)
		body = fmt.Sprintf(
			"- URL ID: %d\n- Path: %s\n- 失败次数: %d\n- 时间: %s\n- 动作: 自动下线(UpdateUrlAlive)",
			e.Id, u.Path, e.Attempts, now,
		)
		return
	}
	title = fmt.Sprintf("云阶恢复: URL %s 上线", u.Path)
	return
}

// kindDown 返回该 Event 是否代表 DOWN, 避免 call 端每次内联比较。
func kindDown() Kind { return KindDown }

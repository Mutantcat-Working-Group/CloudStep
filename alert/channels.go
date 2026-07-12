// Package alert 实现地址失效告警: 异步 channel + 单 worker 串行消费(限速 ≤2 msg/s),
// 防抖窗口内不重复告警同一 URL+kind, 支持钉钉/邮件双通道。
//
// 仅有一个导出入口 Emit(Event), 由心跳 scheduler 在下线/恢复路径调用。
package alert

import "log"

// Kind 告警事件类型。
type Kind int

const (
	// KindDown URL 失败被下线。
	KindDown Kind = iota
	// KindUp URL 从下线恢复上线。
	KindUp
)

// Event 一条待处理的告警事件(由心跳 emit 入 channel)。
type Event struct {
	Id       int    // url id
	Path     string // url path(人类可读地址)
	Kind     Kind   // DOWN 或 UP
	Attempts int    // 当下线时的心跳连续失败次数(恢复时为 0)
}

// eventChBufSize channel 缓冲容量。满时 Emit 直接丢弃, 防消费端卡死撑爆内存。
const eventChBufSize = 10

// EventCh 全局单例事件通道(只产生, 不消费)。cap(eventChBufSize), 满则丢弃。
var EventCh = make(chan Event, eventChBufSize)

// Emit 非阻塞投递一条告警事件。缓冲满时丢弃并打日志(不阻塞心跳路径)。
func Emit(e Event) {
	select {
	case EventCh <- e:
	default:
		log.Printf("[alert] eventCh full (cap=%d), drop url id=%d kind=%v", eventChBufSize, e.Id, e.Kind)
	}
}

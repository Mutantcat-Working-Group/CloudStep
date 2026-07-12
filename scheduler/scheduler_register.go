package scheduler

import (
	"github.com/antlabs/timer"
)

func init() {
	tm := timer.NewTimer()
	registerHotTimeCleanScheduler(tm)
	registerHeartbeatScheduler(tm)
	go tm.Run()
}

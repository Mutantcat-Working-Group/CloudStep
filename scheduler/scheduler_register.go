package scheduler

import (
	"github.com/antlabs/timer"
)

func init() {
	tm := timer.NewTimer()
	registerHotTimeCleanScheduler(tm)
	tm.Run()
}

package scheduler

import (
	"com.mutantcat.cloud_step/alert"
	"github.com/antlabs/timer"
	"context"
)

func init() {
	tm := timer.NewTimer()
	registerHotTimeCleanScheduler(tm)
	registerHeartbeatScheduler(tm)
	go tm.Run()
	go alert.Start(context.Background())
}

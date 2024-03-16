package scheduler

import (
	"com.mutantcat.cloud_step/util"
	"github.com/antlabs/timer"
	"time"
)

// 每3分钟清空一次热点数据
func registerHotTimeCleanScheduler(tm timer.Timer) {
	tm.ScheduleFunc(180*time.Second, func() {
		util.ClearHotNum()
	})
}

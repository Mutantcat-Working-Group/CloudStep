// 心跳探活: 每 60s 对所有 Alive=true 的 URL 做 TCP 探活, 失败累加 url.Retry,
// 达 failThreshold 调 dao.UpdateUrlAlive(id, false) 下线。管理员手动禁用的 URL
// 跳过, 心跳只摘不复活(Admin 手动 enable 负责 Retry=0 解锁)。
//
// beat(urls) 是幂等函数: 测试可唯传入测试 URL, 不动 production URL 集。
// beatAll() 是 beat 在 production 的入口, 从 cache 全量读后传入 beat。

package scheduler

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"log"
	"sync"
	"time"

	"github.com/antlabs/timer"
)

const (
	heartbeatInterval = 60 * time.Second
	failThreshold     = 3
)

// beat 一次给定 urls 的心跳。对每条 Alive=true 的 URL 起一个 goroutine 做
// TCP 探活; 所有 goroutine 通过 wg 等待完成后函数返回, 不跨 beat 持 go 程。
// 幂等设计: 调用方传入什么 URLs 就探什么, 调用方隔离测试 / prod。
func beat(urls []entity.Url) {
	var wg sync.WaitGroup
	for i := range urls {
		u := urls[i]
		if !u.Alive {
			continue
		}
		wg.Add(1)
		go func(u entity.Url) {
			defer wg.Done()
			oneUrlBeat(u)
		}(u)
	}
	wg.Wait()
}

func oneUrlBeat(u entity.Url) {
	// 探活用 500ms TCP 阻塞; 包在独立 goroutine + channel, 避免某条卡住影响全局。
	ch := make(chan string, 1)
	go util.GetTCPSpeed(u.Path, ch)
	result := <-ch

	if result == "timeout" {
		// 失败: 读当前 retry, +1, 写回。达阈值下线。
		current := readRetry(u.Id)
		if current < 0 {
			return // id 不存在(可能已被 admin 删除)
		}
		next := current + 1
		if !dao.UpdateUrlRetry(u.Id, next) {
			return
		}
		if next >= failThreshold {
			if dao.UpdateUrlAlive(u.Id, false) {
				log.Printf("[heartbeat] url id=%d path=%s set ALIVE=false after %d failures", u.Id, u.Path, next)
			}
		} else {
			log.Printf("[heartbeat] url id=%d path=%s fail (%d/%d)", u.Id, u.Path, next, failThreshold)
		}
	} else {
		// 成功: 清零 retry(已是 0 则省 IO)
		if readRetry(u.Id) != 0 {
			dao.UpdateUrlRetry(u.Id, 0)
		}
	}
}

func readRetry(id int) int {
	var u entity.Url
	has, err := dao.PublicEngine.ID(id).Get(&u)
	if err != nil || !has {
		return -1
	}
	return u.Retry
}

// beatAll 从 cache 全量读出所有 URL 后调 beat。生产入口。
func beatAll() {
	collection.MWorkCllection.Lock()
	all := make([]entity.Url, 0)
	for _, urls := range collection.WorkCllection {
		all = append(all, urls...)
	}
	collection.MWorkCllection.Unlock()
	beat(all)
}

// registerHeartbeatScheduler 在 main scheduler 注册心跳。
func registerHeartbeatScheduler(tm timer.Timer) {
	tm.ScheduleFunc(heartbeatInterval, beatAll)
}

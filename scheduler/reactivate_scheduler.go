// 自申请停用到期恢复: 每 60s 扫 url 表找 self_deactivate_until<=now 的 url,
// 恢复为 alive=true,同时累加 attempts; 三次恢复-心跳 down 循环后放弃,
// 清 self_deactivate_until 锁为 admin-effect 禁用,需 administrator 介入。
//
// 与心跳#4 协作: 心跳探失败 → alive=false; reactivate beat 再恢复;
// 3 次循环后阻尼收敛,url 等同管理员手动禁用。

package scheduler

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"log"
	"time"

	"github.com/antlabs/timer"
)

const (
	reactivateInterval       = 60 * time.Second
	reactivateGiveUpAttempts = 3
)

// reactivateBeat SELECT id FROM url WHERE alive=false AND self_deactivate_until IS NOT NULL
//           AND self_deactivate_until <= now(),恢复 < giveUpAttempts 的 url。
// 不持有 MWorkCllection 锁读 cache → 仅局部变量访问 + 短锁 per-url, 避免死锁。
func reactivateBeat() {
	now := time.Now()
	var urls []entity.Url
	// xorm 注: <= now() 的自申请到期 url
	if err := dao.PublicEngine.
		Where("alive = ? AND self_deactivate_until IS NOT NULL AND self_deactivate_until <= ?", false, now).
		Find(&urls); err != nil {
		log.Printf("[reactivate] scan err: %v", err)
		return
	}
	for _, u := range urls {
		reactivateOne(u)
	}
}

func reactivateOne(u entity.Url) {
	collection.MWorkCllection.Lock()
	attempts := u.SelfDeactivateAttempts
	collection.MWorkCllection.Unlock()

	if attempts >= reactivateGiveUpAttempts {
		if !dao.ClearUrlSelfDeactivate(u.Id) {
			return
		}
		log.Printf("[reactivate] url id=%d attempts=%d → give up, needs admin", u.Id, attempts)
		collection.MWorkCllection.Lock()
		for coll, urls := range collection.WorkCllection {
			for i := range urls {
				if urls[i].Id == u.Id {
					collection.WorkCllection[coll][i].SelfDeactivateUntil = nil
					collection.WorkCllection[coll][i].SelfDeactivateAttempts = 0
					collection.MWorkCllection.Unlock()
					return
				}
			}
		}
		collection.MWorkCllection.Unlock()
		return
	}

	if !dao.UpdateUrlAlive(u.Id, true) {
		return
	}
	if !dao.SetUrlDeactivateAttempts(u.Id, attempts+1) {
		return
	}
	log.Printf("[reactivate] url id=%d path=%s restored(attempt %d/%d)", u.Id, u.Path, attempts+1, reactivateGiveUpAttempts)
}

// registerReactivateScheduler 在 main scheduler 注册自申请恢复。
func registerReactivateScheduler(tm timer.Timer) {
	tm.ScheduleFunc(reactivateInterval, reactivateBeat)
}

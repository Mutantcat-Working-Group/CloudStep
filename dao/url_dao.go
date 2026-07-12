package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"errors"
	"time"
)

// 获得Parent为某个集合的url
func GetUrlsByParent(parent string) []entity.Url {
	urls := make([]entity.Url, 0)
	err := PublicEngine.Where("parent = ?", parent).Find(&urls)
	if err != nil {
		return nil
	}
	return urls
}

// 通过parentid获得urls
func GetUrlsByParentId(parentId int) []entity.Url {
	urls := make([]entity.Url, 0)
	// 用集合id查询集合名
	parent := entity.Collection{}
	_, err := PublicEngine.ID(parentId).Get(&parent)
	if err != nil {
		return nil
	}
	// 用集合名查询urls
	err = PublicEngine.Where("parent = ?", parent.Name).Find(&urls)
	if err != nil {
		return nil
	}
	return urls
}

// 通过parent和url添加url（事务）
func AddUrl(parent string, url string) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 添加url
	newUrl := entity.Url{}
	newUrl.Id = 0
	newUrl.Parent = parent
	newUrl.Path = url
	newUrl.Alive = true
	newUrl.Retry = 0
	_, err = session.Insert(&newUrl)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 添加成功过后在缓存中添加
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	collection.WorkCllection[parent] = append(collection.WorkCllection[parent], newUrl)
	return true
}

// 通过id删除url （事务）
func DeleteUrlById(id int) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 删除url
	url := entity.Url{}
	_, err = session.ID(id).Delete(&url)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 删除成功后删除缓存中的url
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	for k, v := range collection.WorkCllection {
		for i, u := range v {
			if u.Id == id {
				collection.WorkCllection[k] = append(v[:i], v[i+1:]...)
				break
			}
		}
	}
	return true
}

// 通过id修改url
func UpdateUrlById(id int, url string) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 修改url
	newUrl := entity.Url{}
	newUrl.Path = url
	_, err = session.ID(id).Update(&newUrl)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 修改成功后修改缓存中的url
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	for k, v := range collection.WorkCllection {
		for i, u := range v {
			if u.Id == id {
				collection.WorkCllection[k][i].Path = url
				break
			}
		}
	}
	return true
}

// UpdateUrlAlive sets alive flag of url by id.
//
//	alive=true : alive=true, retry=0
//	alive=false: alive=false, retry unchanged
//
// Writes DB (transaction) + cache. Returns false if id missing or tx err.
func UpdateUrlAlive(id int, alive bool) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	var err error
	var affected int64
	if alive {
		affected, err = session.Cols("alive", "retry").ID(id).Update(&entity.Url{Alive: true, Retry: 0})
	} else {
		affected, err = session.Cols("alive").ID(id).Update(&entity.Url{Alive: false})
	}
	if err != nil || affected == 0 {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				collection.WorkCllection[coll][i].Alive = alive
				if alive {
					collection.WorkCllection[coll][i].Retry = 0
				}
			}
		}
	}
	collection.MWorkCllection.Unlock()
	return true
}

// UpdateUrlRetry 按 id 改 url.Retry 字段到 targetRetry。
// 读 DB 取当前 retry, 若已等于 target 直接返 true(省 IO 幂等);
// 否则单行写 retry 字段。调用方需保证 id 存在 — miss 返 false。
func UpdateUrlRetry(id int, targetRetry int) bool {
	var u entity.Url
	has, err := PublicEngine.ID(id).Get(&u)
	if err != nil || !has {
		return false
	}
	if u.Retry == targetRetry {
		return true
	}
	u.Retry = targetRetry
	affected, err := PublicEngine.ID(id).Cols("retry").Update(&u)
	if err != nil || affected == 0 {
		return false
	}
	// 同步 cache
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				collection.WorkCllection[coll][i].Retry = targetRetry
				break
			}
		}
	}
	collection.MWorkCllection.Unlock()
	return true
}

// GetUrl 按 id 读取单行 url. miss 返 (zero, false).
func GetUrl(id int) (entity.Url, bool) {
	var u entity.Url
	has, err := PublicEngine.ID(id).Get(&u)
	if err != nil || !has {
		return entity.Url{}, false
	}
	return u, true
}

// UpdateUrlAlertState 原子写回 url 的告警元数据(下线/恢复通道调用)。
// at=告警时间, isDown=该条告警是否为 DOWN, failCount=累计失败次数。
// 仅更新 last_alert_at / last_alert_is_down / last_alert_fail_count 三列。
func UpdateUrlAlertState(id int, at time.Time, isDown bool, failCount int) bool {
	affected, err := PublicEngine.ID(id).Cols(
		"last_alert_at", "last_alert_is_down", "last_alert_fail_count",
	).Update(&entity.Url{
		LastAlertAt:        &at,
		LastAlertIsDown:    isDown,
		LastAlertFailCount: failCount,
	})
	return err == nil && affected != 0
}

// GenerateAndSaveUrlKey 返回 url id 的当前自申请密钥; url 不存在返 ("", err).
// 首次读取(SelfDeactivateKey=="")时 RandToken(32) 落库+cache 后返回.
func GenerateAndSaveUrlKey(id int) (string, error) {
	return rotateUrlKey(id, false)
}

// RotateUrlKey 强制重新生成 url id 的自申请密钥并落库+cache 后返回; url 不存在返 ("", err).
func RotateUrlKey(id int) (string, error) {
	return rotateUrlKey(id, true)
}

// rotateUrlKey 内部实现: force=false 表示 key 已存在则跳过, force=true 强制 regen.
func rotateUrlKey(id int, force bool) (string, error) {
	u, ok := GetUrl(id)
	if !ok {
		return "", errors.New("url not found")
	}
	if !force && u.SelfDeactivateKey != "" {
		return u.SelfDeactivateKey, nil
	}
	token := util.RandToken(32)
	if token == "" {
		return "", errors.New("rand token empty")
	}
	if !saveUrlSelfDeactivateKey(id, token) {
		return "", errors.New("update key failed")
	}
	return token, nil
}

// saveUrlSelfDeactivateKey 写 self_deactivate_key 单列并同步 cache.
func saveUrlSelfDeactivateKey(id int, key string) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	affected, err := session.Cols("self_deactivate_key").ID(id).Update(&entity.Url{SelfDeactivateKey: key})
	if err != nil || affected == 0 {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				collection.WorkCllection[coll][i].SelfDeactivateKey = key
				break
			}
		}
	}
	collection.MWorkCllection.Unlock()
	return true
}

// SetUrlSelfDeactivate 事务写 self_deactivate_until + self_deactivate_attempts
// 两列并同步 cache. url miss / tx err 返 false.
func SetUrlSelfDeactivate(id int, until time.Time, attempts int) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	u := entity.Url{SelfDeactivateUntil: &until, SelfDeactivateAttempts: attempts}
	affected, err := session.Cols("self_deactivate_until", "self_deactivate_attempts").ID(id).Update(&u)
	if err != nil || affected == 0 {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				collection.WorkCllection[coll][i].SelfDeactivateAttempts = attempts
				untilCopy := until
				collection.WorkCllection[coll][i].SelfDeactivateUntil = &untilCopy
				break
			}
		}
	}
	collection.MWorkCllection.Unlock()
	return true
}

// ClearUrlSelfDeactivate 清 self_deactivate_until(=NULL) + self_deactivate_attempts(=0),
// 同步 cache. url miss / tx err 返 false.
func ClearUrlSelfDeactivate(id int) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	u := entity.Url{SelfDeactivateUntil: nil, SelfDeactivateAttempts: 0}
	affected, err := session.Cols("self_deactivate_until", "self_deactivate_attempts").ID(id).Update(&u)
	if err != nil || affected == 0 {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				collection.WorkCllection[coll][i].SelfDeactivateUntil = nil
				collection.WorkCllection[coll][i].SelfDeactivateAttempts = 0
				break
			}
		}
	}
	collection.MWorkCllection.Unlock()
	return true
}

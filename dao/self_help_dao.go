package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
)

func GetAllSelfHelps() []entity.SelfHelp {
	selfHelps := make([]entity.SelfHelp, 0)
	err := PublicEngine.Find(&selfHelps)
	if err != nil {
		return nil
	}
	return selfHelps
}

func AddSelfHelp(self entity.SelfHelp) bool {
	self.Id = 0
	self.Index = 0
	self.AliveNum = 0
	if self.Salt == "" {
		self.Salt = util.RandToken(32)
	}
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.Insert(&self)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MSelfHelpMode.Lock()
	defer collection.MSelfHelpMode.Unlock()
	collection.SelfHelpMode[self.Way] = self
	return true
}

func GetSelfWayHelpById(id int) string {
	var self entity.SelfHelp
	has, err := PublicEngine.ID(id).Get(&self)
	if err != nil {
		return ""
	}
	if !has {
		return ""
	}
	return self.Way
}

func UpdateSelfHelpById(self entity.SelfHelp) bool {
	oldWay := GetSelfWayHelpById(self.Id)
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.ID(self.Id).Update(&self)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MSelfHelpMode.Lock()
	defer collection.MSelfHelpMode.Unlock()
	delete(collection.SelfHelpMode, oldWay)
	collection.SelfHelpMode[self.Way] = self
	return true
}

func DeleteSelfHelpById(id int) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.ID(id).Delete(&entity.SelfHelp{})
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MSelfHelpMode.Lock()
	defer collection.MSelfHelpMode.Unlock()
	for k, v := range collection.SelfHelpMode {
		if v.Id == id {
			delete(collection.SelfHelpMode, k)
			break
		}
	}

	return true
}

// 检查Way是否存在
func CheckWayExist(way string) bool {
	var self entity.SelfHelp
	has, err := PublicEngine.Where("way = ?", way).Get(&self)
	if err != nil {
		return false
	}
	return has
}

// 检查名称是否存在
func CheckNameExist(name string) bool {
	var self entity.SelfHelp
	has, err := PublicEngine.Where("name = ?", name).Get(&self)
	if err != nil {
		return false
	}
	return has
}

// 判断除了当前这个id外，是否还有其他的way根这个重复
func CheckWayExistExceptId(way string, id int) bool {
	var self entity.SelfHelp
	has, err := PublicEngine.Where("way = ? and id != ?", way, id).Get(&self)
	if err != nil {
		return false
	}
	return has
}

// 判断除了当前这个id外，是否还有其他的name根这个重复
func CheckNameExistExceptId(name string, id int) bool {
	var self entity.SelfHelp
	has, err := PublicEngine.Where("name = ? and id != ?", name, id).Get(&self)
	if err != nil {
		return false
	}
	return has
}

// GetSaltForWay 按 way 查找坐标(shell 优先、proxy 其次),返回其 salt 与所属模式。
// found 为 false 表示该 way 在两张表中均不存在。
func GetSaltForWay(way string) (salt string, mode string, found bool) {
	var self entity.SelfHelp
	has, err := PublicEngine.Where("way = ?", way).Get(&self)
	if err == nil && has {
		return self.Salt, "self", true
	}
	var proxy entity.Proxy
	has, err = PublicEngine.Where("way = ?", way).Get(&proxy)
	if err == nil && has {
		return proxy.Salt, "proxy", true
	}
	return "", "", false
}

// RotateSalt 重新生成指定坐标的 salt(mode: "self" 或 "proxy")并持久化,返回新 salt。
// ok 为 false 表示该 mode 下 way 不存在。
func RotateSalt(way string, mode string) (newSalt string, ok bool) {
	newSalt = util.RandToken(32)
	switch mode {
	case "self":
		var self entity.SelfHelp
		has, err := PublicEngine.Where("way = ?", way).Get(&self)
		if err != nil || !has {
			return "", false
		}
		self.Salt = newSalt
		affected, err := PublicEngine.ID(self.Id).Update(&self)
		if err != nil || affected == 0 {
			return "", false
		}
		collection.MSelfHelpMode.Lock()
		self.Salt = newSalt
		collection.SelfHelpMode[way] = self
		collection.MSelfHelpMode.Unlock()
		return newSalt, true
	case "proxy":
		var proxy entity.Proxy
		has, err := PublicEngine.Where("way = ?", way).Get(&proxy)
		if err != nil || !has {
			return "", false
		}
		affected, err := PublicEngine.ID(proxy.Id).Cols("salt").Update(&entity.Proxy{Salt: newSalt})
		if err != nil || affected == 0 {
			return "", false
		}
		collection.MProxyMode.Lock()
		proxy.Salt = newSalt
		collection.ProxyMode[way] = proxy
		collection.MProxyMode.Unlock()
		return newSalt, true
	default:
		return "", false
	}
}

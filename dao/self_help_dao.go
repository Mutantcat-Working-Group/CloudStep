package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
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

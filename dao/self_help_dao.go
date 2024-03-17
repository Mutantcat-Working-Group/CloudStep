package dao

import "com.mutantcat.cloud_step/entity"

// 获得所有自助
func getAllSelfHelps() []entity.SelfHelp {
	selfHelps := make([]entity.SelfHelp, 0)
	err := PublicEngine.Find(&selfHelps)
	if err != nil {
		return nil
	}
	return selfHelps
}

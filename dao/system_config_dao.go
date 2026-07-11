package dao

import (
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"log"
)

const systemConfigId = 1

// InitSystemConfig 读取系统开关并同步到内存,不存在则写入默认值(允许内网代理)。
func InitSystemConfig() {
	cfg := entity.SystemConfig{}
	have, err := PublicEngine.ID(systemConfigId).Get(&cfg)
	if err != nil {
		log.Fatal("读取系统配置失败: ", err)
	}
	if !have {
		cfg = entity.SystemConfig{Id: systemConfigId, AllowIntranetProxy: true}
		if _, err := PublicEngine.Insert(&cfg); err != nil {
			log.Fatal("写入默认系统配置失败: ", err)
		}
	}
	util.SetAllowIntranetProxy(cfg.AllowIntranetProxy)
}

func GetSystemConfig() entity.SystemConfig {
	cfg := entity.SystemConfig{}
	if _, err := PublicEngine.ID(systemConfigId).Get(&cfg); err != nil {
		return entity.SystemConfig{Id: systemConfigId, AllowIntranetProxy: true}
	}
	return cfg
}

func UpdateSystemConfig(in entity.SystemConfig) bool {
	in.Id = systemConfigId
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	if _, err := session.ID(systemConfigId).Update(&in); err != nil {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	util.SetAllowIntranetProxy(in.AllowIntranetProxy)
	return true
}

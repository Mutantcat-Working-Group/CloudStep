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
	util.SetSystemConfigFromDao(cfg)
	// 把 id→name 查询注入 util 的默认集解析器(避免 util import dao 形成循环)。
	util.SetDefaultCollectionResolver(GetCollectionNameById)
}

func GetSystemConfig() entity.SystemConfig {
	cfg := entity.SystemConfig{}
	if _, err := PublicEngine.ID(systemConfigId).Get(&cfg); err != nil {
		return entity.SystemConfig{Id: systemConfigId, AllowIntranetProxy: true}
	}
	return cfg
}

func UpdateSystemConfig(in entity.SystemConfig) bool {
	// 校验 default id 指向存在的 collection(0 表示"清除默认", 合法)。
	if in.SelfDefaultCollectionId > 0 && GetCollectionNameById(in.SelfDefaultCollectionId) == "" {
		return false
	}
	if in.AgentDefaultCollectionId > 0 && GetCollectionNameById(in.AgentDefaultCollectionId) == "" {
		return false
	}

	in.Id = systemConfigId
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	if _, err := session.Cols("allow_intranet_proxy", "self_default_collection_id", "agent_default_collection_id").ID(systemConfigId).Update(&in); err != nil {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	util.SetSystemConfigFromDao(in)
	return true
}

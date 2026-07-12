package dao

import (
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"log"
	"strings"
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

// UpdateAlertConfig 写入告警相关字段, 非告警字段(intranet 代理/默认集)务必保留原值。
// 采用"零值保留"合并: 字符串空、整数 0、bool false 视为"未提交, 保留 DB 现值"。
// 注意 bool 因此无法通过本接口显式置 false(admin 前端通常会把控件当前值全量提交,
// 若确有置 false 需求可扩展指针字段; 当前三类 bool 默认 0=false 语义自洽)。
func UpdateAlertConfig(in entity.SystemConfig) bool {
	const sentinel = "***"

	existing := GetSystemConfig()

	// 非告警字段: 告警接口不碰, 沿用现有值。
	merged := existing

	// 字符串字段分两档:
	//   - webhook / host / user / from / to(非脱敏): 空串=清除(前端全量提交当前值, 空即真意清空)。
	//   - ding secret / smtp password(脱敏): GET 时掩码为 ***, 故"***"=保留原值, 空串=清空, 其余=设定。
	if s := strings.TrimSpace(in.AlertDingWebhook); s != "" {
		merged.AlertDingWebhook = s
	}
	if s := strings.TrimSpace(in.AlertSMTPHost); s != "" {
		merged.AlertSMTPHost = s
	}
	if s := strings.TrimSpace(in.AlertSMTPUser); s != "" {
		merged.AlertSMTPUser = s
	}
	if s := strings.TrimSpace(in.AlertSMTPFrom); s != "" {
		merged.AlertSMTPFrom = s
	}
	if s := strings.TrimSpace(in.AlertSMTPTo); s != "" {
		merged.AlertSMTPTo = s
	}
	// 脱敏字符串(ding secret / smtp password): GET 掩码为 ***, 故:
	//   "***"=保留原值, ""=清空, 其余=设定。
	if s := strings.TrimSpace(in.AlertDingSecret); s == sentinel {
		// 保留现有。
	} else {
		merged.AlertDingSecret = s
	}
	if s := strings.TrimSpace(in.AlertSMTPPassword); s == sentinel {
		// 保留现有。
	} else {
		merged.AlertSMTPPassword = s
	}

	// 整数字段: 0 视为保留(debounce/port 均有合理默认)。
	if in.AlertSMTPPort != 0 {
		merged.AlertSMTPPort = in.AlertSMTPPort
	}
	if in.AlertDebounceSec != 0 {
		merged.AlertDebounceSec = in.AlertDebounceSec
	}

	// bool 字段: 直接覆写(前端全量提交当前控件值, 总开关/通道开关均可靠)。
	merged.AlertEnabled = in.AlertEnabled
	merged.AlertDingEnabled = in.AlertDingEnabled
	merged.AlertMailEnabled = in.AlertMailEnabled

	merged.Id = systemConfigId
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	if _, err := session.Cols(
		"alert_enabled", "alert_ding_enabled", "alert_ding_webhook", "alert_ding_secret",
		"alert_mail_enabled", "alert_smtp_host", "alert_smtp_port", "alert_smtp_user",
		"alert_smtp_password", "alert_smtp_from", "alert_smtp_to", "alert_debounce_sec",
	).ID(systemConfigId).Update(&merged); err != nil {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	util.SetSystemConfigFromDao(merged)
	return true
}

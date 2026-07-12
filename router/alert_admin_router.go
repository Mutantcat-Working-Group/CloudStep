package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"github.com/gin-gonic/gin"
	"strings"
)

// AlertAdminRouter 暴露告警配置查询/更新端点, 均需登录(LoginHandler)。
type AlertAdminRouter struct{}

func (router *AlertAdminRouter) PrepareRouter() error { return nil }

func (router *AlertAdminRouter) InitRouter(context *gin.Engine) error {
	context.GET("/alert/get", LoginHandler(), getAlertConfig)
	context.POST("/alert/update", LoginHandler(), updateAlertConfig)
	return nil
}

func (router *AlertAdminRouter) DestroyRouter() error { return nil }

// masked 占位符: GET 时敏感字段若已配置则返 "***", 前端回写时视为"保留原值"。
const masked = "***"

// getAlertConfig GET /alert/get —— 返回当前告警配置, 敏感字段脱敏。
func getAlertConfig(c *gin.Context) {
	cfg := dao.GetSystemConfig()
	out := gin.H{
		"alertEnabled":     cfg.AlertEnabled,
		"alertDingEnabled": cfg.AlertDingEnabled,
		"alertDingWebhook": cfg.AlertDingWebhook,
		"alertDingSecret":  maskIfSet(cfg.AlertDingSecret),
		"alertMailEnabled": cfg.AlertMailEnabled,
		"alertSmtpHost":    cfg.AlertSMTPHost,
		"alertSmtpPort":    cfg.AlertSMTPPort,
		"alertSmtpUser":    cfg.AlertSMTPUser,
		"alertSmtpPassword": maskIfSet(cfg.AlertSMTPPassword),
		"alertSmtpFrom":    cfg.AlertSMTPFrom,
		"alertSmtpTo":      cfg.AlertSMTPTo,
		"alertDebounceSec": cfg.AlertDebounceSec,
	}
	c.JSON(200, gin.H{"code": 0, "msg": "success", "data": out})
}

// updateAlertConfig POST /alert/update —— 写入告警配置(零值保留原值, 敏感字段 *** 保留原值)。
func updateAlertConfig(c *gin.Context) {
	var b struct {
		AlertEnabled     bool   `json:"alertEnabled"`
		AlertDingEnabled bool   `json:"alertDingEnabled"`
		AlertDingWebhook string `json:"alertDingWebhook"`
		AlertDingSecret  string `json:"alertDingSecret"`
		AlertMailEnabled bool   `json:"alertMailEnabled"`
		AlertSMTPHost    string `json:"alertSmtpHost"`
		AlertSMTPPort    int    `json:"alertSmtpPort"`
		AlertSMTPUser    string `json:"alertSmtpUser"`
		AlertSMTPPassword string `json:"alertSmtpPassword"`
		AlertSMTPFrom    string `json:"alertSmtpFrom"`
		AlertSMTPTo      string `json:"alertSmtpTo"`
		AlertDebounceSec int    `json:"alertDebounceSec"`
	}
	if c.ShouldBindJSON(&b) != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	in := entity.SystemConfig{
		AlertEnabled:      b.AlertEnabled,
		AlertDingEnabled:  b.AlertDingEnabled,
		AlertDingWebhook:  b.AlertDingWebhook,
		AlertDingSecret:   b.AlertDingSecret,
		AlertMailEnabled:  b.AlertMailEnabled,
		AlertSMTPHost:     b.AlertSMTPHost,
		AlertSMTPPort:     b.AlertSMTPPort,
		AlertSMTPUser:     b.AlertSMTPUser,
		AlertSMTPPassword: b.AlertSMTPPassword,
		AlertSMTPFrom:     b.AlertSMTPFrom,
		AlertSMTPTo:       b.AlertSMTPTo,
		AlertDebounceSec:  b.AlertDebounceSec,
	}
	if dao.UpdateAlertConfig(in) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}

// maskIfSet 已配置的非空串返 "***", 否则返空串(前端据此判断是否已配置)。
func maskIfSet(s string) string {
	if strings.TrimSpace(s) != "" {
		return masked
	}
	return ""
}

package util

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetWayParam(c *gin.Context) string {
	queryValue := c.Query("way")
	if queryValue != "" {
		return queryValue
	}

	formValue := c.PostForm("way")
	if formValue != "" {
		return formValue
	}

	// 仅对非 GET/HEAD 且 content-type 含 "application/json" 的请求尝试 JSON 解析。
	// 否则 gin >=1.12 对空 body GET 调用 BindJSON 会自动写入 400 response,
	// handler 后续无法修改返回码(本次修 GET path-form 400 老 bug)。
	if c.Request != nil &&
		c.Request.Method != http.MethodGet &&
		c.Request.Method != http.MethodHead &&
		strings.Contains(c.ContentType(), "application/json") {
		type WayBody struct {
			Way string `json:"way"`
		}
		var jsonBody WayBody
		if err := c.ShouldBindJSON(&jsonBody); err == nil && jsonBody.Way != "" {
			return jsonBody.Way
		}
	}

	pathValue := c.Param("way")
	if pathValue != "" {
		return pathValue
	}

	return ""
}

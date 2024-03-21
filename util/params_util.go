package util

import "github.com/gin-gonic/gin"

func GetWayParam(c *gin.Context) string {
	queryValue := c.Query("way")
	if queryValue != "" {
		return queryValue
	}

	formValue := c.PostForm("way")
	if formValue != "" {
		return formValue
	}

	type WayBody struct {
		Way string `json:"way"`
	}

	var jsonBody WayBody

	err := c.BindJSON(&jsonBody)
	if err == nil && jsonBody.Way != "" {
		return jsonBody.Way
	}
	pathValue := c.Param("way")
	if pathValue != "" {
		return pathValue
	}

	return ""
}

func GetMethodParam(c *gin.Context) string {
	queryValue := c.Query("method")
	if queryValue != "" {
		return queryValue
	}

	formValue := c.PostForm("method")
	if formValue != "" {
		return formValue
	}

	type MethodBody struct {
		Method string `json:"method"`
	}

	var jsonBody MethodBody

	err := c.BindJSON(&jsonBody)
	if err == nil && jsonBody.Method != "" {
		return jsonBody.Method
	}
	pathValue := c.Param("method")
	if pathValue != "" {
		return pathValue
	}

	return ""
}

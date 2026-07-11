package util

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func Proxy(targetURL string, c *gin.Context) {
	proxyUrl, err := url.Parse(targetURL)
	if err != nil || proxyUrl.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "代理目标地址无效"})
		return
	}
	if reason, _ := checkIntranetBlocked(proxyUrl.Hostname()); reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "禁止代理到" + reason + "地址"})
		return
	}
	values := c.Request.URL.Query()
	// 创建一个新的url.Values来存储修改后的参数
	newValues := url.Values{}
	for key, value := range values {
		// 移除原本路径参数中的way参数 如果存在*way**参数则先将它的值给新的的way、然后再移除
		if key == "way" {
		} else if key == "*way**" {
			newValues.Set("way", value[0])
		} else {
			newValues.Set(key, value[0])
		}
	}
	// 将修改后的参数设置回c.Request.URL
	c.Request.URL.RawQuery = newValues.Encode()

	target, _ := url.Parse(proxyUrl.Scheme + "://" + proxyUrl.Host)
	if c.Param("name") != "" {
		c.Request.URL.Path = proxyUrl.Path + c.Param("name")
	} else {
		c.Request.URL.Path = proxyUrl.Path
	}
	// 如果最后一个字符是 / 则只去掉最后一个 /
	if strings.HasSuffix(c.Request.URL.Path, "/") {
		c.Request.URL.Path = c.Request.URL.Path[:len(c.Request.URL.Path)-1]
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ServeHTTP(c.Writer, c.Request)
}

func RootProxy(targetURL string, c *gin.Context) error {
	// 收集并重新编码 GET 参数,避免用户输入中的特殊字符被注入为额外参数
	queryParams := c.Request.URL.Query()
	newValues := url.Values{}
	for key, values := range queryParams {
		if key == "way" {
			continue
		}
		if key == "*way**" {
			newValues["way"] = values
			continue
		}
		newValues[key] = values
	}
	if encoded := newValues.Encode(); encoded != "" {
		targetURL += "?" + encoded
	}

	// 在处理请求头/体前先做 SSRF 校验,拒绝则直接返回
	if blockedReason, err := checkIntranetBlocked(targetURL); blockedReason != "" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "禁止代理到" + blockedReason + "地址"})
		return err
	}

	// 获取用户请求的请求头
	userHeaders := c.Request.Header
	var requestBody string
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		_ = c.Request.ParseForm()
		requestBody = c.Request.PostForm.Encode()
	} else if strings.Contains(contentType, "application/json") {
		data, _ := c.GetRawData()
		requestBody = string(data)
	}

	// 创建新的请求
	req, err := http.NewRequest(c.Request.Method, targetURL, strings.NewReader(requestBody))
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request: "+err.Error())
		return fmt.Errorf("创建代理请求失败")
	}

	// 将用户请求的请求头添加到新请求的请求头中
	for key, values := range userHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error sending request: "+err.Error())
		return fmt.Errorf("发送代理请求失败")
	}
	defer resp.Body.Close()

	// 返回响应给客户端
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading response: "+err.Error())
		return fmt.Errorf("返回响应失败")
	}
	c.String(http.StatusOK, string(body))
	return nil
}

// checkIntranetBlocked 在管理员关闭内网代理时,拒绝代理到私有/回环/未指定地址。
// 返回空字符串表示放行;否则返回被拒原因与一个非 nil 的 error。
func checkIntranetBlocked(targetURL string) (string, error) {
	if AllowIntranet() {
		return "", nil
	}
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Hostname() == "" {
		return "", err
	}
	if reason, blocked := ClassifyIP(parsed.Hostname()); blocked {
		return reason, fmt.Errorf("禁止代理到内网地址")
	}
	return "", nil
}

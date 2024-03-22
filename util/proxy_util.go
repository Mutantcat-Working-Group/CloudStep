package util

import (
	"github.com/gin-gonic/gin"
	"net/http/httputil"
	"net/url"
	"strings"
)

func Proxy(targetURL string, c *gin.Context) {
	proxyUrl, _ := url.Parse(targetURL)

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

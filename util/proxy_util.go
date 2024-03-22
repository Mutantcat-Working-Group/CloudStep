package util

import (
	"github.com/gin-gonic/gin"
	"net/http/httputil"
	"net/url"
)

func Proxy(targetURL string, c *gin.Context) {
	proxyUrl, _ := url.Parse(targetURL)

	target, _ := url.Parse(proxyUrl.Scheme + "://" + proxyUrl.Host)
	if c.Param("name") != "" {
		c.Request.URL.Path = proxyUrl.Path + c.Param("name")
	} else {
		c.Request.URL.Path = proxyUrl.Path
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ServeHTTP(c.Writer, c.Request)
}

package router

import (
	"io/fs"
	"net/http"

	webassets "com.mutantcat.cloud_step/web"
	"github.com/gin-gonic/gin"
)

type WebRouter struct {
}

func (router *WebRouter) PrepareRouter() error {
	return nil
}

func (router *WebRouter) InitRouter(context *gin.Engine) error {
	webDist, err := fs.Sub(webassets.FS, "cloud-step-web-1g/dist")
	if err != nil {
		return err
	}
	assetsDist, err := fs.Sub(webDist, "assets")
	if err != nil {
		return err
	}
	context.StaticFS("/web", http.FS(webDist))
	context.StaticFS("/assets", http.FS(assetsDist))
	context.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/jpeg", webassets.FaviconData)
	})
	return nil
}

func (router *WebRouter) DestroyRouter() error {
	return nil
}

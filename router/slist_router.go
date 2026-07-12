package router

import (
	"com.mutantcat.cloud_step/collection"
	"github.com/gin-gonic/gin"
)

// SelfHelpListRouter exposes GET /slist?way=... (and /slist/:way), a public
// read of one mapping set's URL list resolved by `way`. No auth - matches
// /self posture.
type SelfHelpListRouter struct{}

func (router *SelfHelpListRouter) PrepareRouter() error { return nil }

func (router *SelfHelpListRouter) InitRouter(context *gin.Engine) error {
	context.Any("/slist", slist)
	context.Any("/slist/*name", slist)
	return nil
}

func (router *SelfHelpListRouter) DestroyRouter() error { return nil }

// resolveWay reads `way` from query, form, JSON body, then path param.
// It intentionally does NOT use util.GetWayParam: that helper advertises
// the same precedence, but calls c.BindJSON unconditionally, and gin >=1.12
// writes a 400 into the response writer when that bind fails on an empty
// body, short-circuiting the request even when query/path resolution would
// have succeeded. That broke GET /slist/:way (where the body is empty and
// the way comes from the path). See ``implementation-plans/2026-07-12-
// slist-self-review.md`` for the path-forwarding workaround decision.
func resolveWay(c *gin.Context) string {
	if v := c.Query("way"); v != "" {
		return v
	}
	if v := c.PostForm("way"); v != "" {
		return v
	}
	// JSON body is opt-in: only attempt it when the client actually claims
	// to send JSON, so a bare GET/POST with empty body never trips a 400.
	if c.ContentType() == "application/json" {
		type wayBody struct {
			Way string `json:"way"`
		}
		var b wayBody
		if err := c.BindJSON(&b); err == nil && b.Way != "" {
			return b.Way
		}
	}
	if v := c.Param("name"); v != "" {
		return v
	}
	return ""
}

func slist(c *gin.Context) {
	way := resolveWay(c)
	if way == "" {
		c.JSON(200, gin.H{"code": 404})
		return
	}

	source, point, urls := collection.GetSelfHelpList(way)
	if source == collection.SrcNone {
		c.JSON(200, gin.H{"code": 404})
		return
	}

	c.JSON(200, gin.H{
		"code":  0,
		"msg":   "success",
		"way":   way,
		"mode":  source.String(),
		"point": point,
		"data":  urls,
	})
}

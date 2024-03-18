package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
	"sync"
)

var token = ""
var mToken sync.Mutex

type LoginRouter struct {
}

func (router *LoginRouter) PrepareRouter() error {
	return nil
}

func (router *LoginRouter) InitRouter(context *gin.Engine) error {
	context.POST("/login", login)
	context.GET("/check", LoginHandler(), checkLogin)
	context.POST("/change", LoginHandler(), change)
	return nil
}

func (router *LoginRouter) DestroyRouter() error {
	return nil
}

// 需要登录的请求处理的时候必须经过这个过滤器的验证
func LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mToken.Lock()
		defer mToken.Unlock()
		userToken := c.GetHeader("Token")
		if !(userToken == token) || userToken == "" {
			c.JSON(200, gin.H{
				"code": 1,
				"msg":  "未登录",
			})
			c.Abort()
		} else {
			c.Next()
		}
	}
}

func login(c *gin.Context) {
	if !util.CouldLogin() {
		c.JSON(200, gin.H{
			"code":  2,
			"msg":   "登录次数过多",
			"token": "",
		})
		return
	}

	type info struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var body info
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, gin.H{
			"code":  1,
			"msg":   "参数错误",
			"token": "",
		})
		return
	}

	logined := dao.CheckUser(body.Username, body.Password)

	if logined {
		mToken.Lock()
		defer mToken.Unlock()
		token = util.RandToken(8)
		util.ClearHotNum()
		c.JSON(200, gin.H{
			"code":  0,
			"msg":   "登录成功",
			"token": token,
		})
	} else {
		c.JSON(200, gin.H{
			"code":  1,
			"msg":   "登录失败",
			"token": "",
		})
	}
}

func checkLogin(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "已登录",
	})
}

func change(c *gin.Context) {
	type info struct {
		Password string `json:"password"`
	}

	var body info
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "参数错误",
		})
		return
	}

	if dao.ChangePassword(body.Password) {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "修改成功",
		})
		token = util.RandToken(8)
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "修改失败",
		})
	}
}

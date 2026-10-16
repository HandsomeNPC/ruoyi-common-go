// Package notfound 路由兜底中间件，处理路径不存在与请求方式不支持两种情形。
package notfound

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model/response"
)

// 提示文案，对齐 Java GlobalExceptionHandler 的两个对应分支。
// 405 复刻 Spring HttpRequestMethodNotSupportedException.getMessage() 的措辞。
const (
	msgNotFound         = "请求地址不存在"
	msgMethodNotAllowed = "Request method '%s' is not supported"
)

// Register 注册 404/405 兜底。须在所有业务路由注册完毕后调用。
//
// gin 默认对未命中路由直接回 HTTP 404 + text/plain "404 page not found"，
// 既非 200 也非 JSON：前端 axios 会落到 error 分支，把它渲染成"系统接口404异常"，
// 而 Java 侧两者都是 HTTP 200 + R，msg 可读。
//
// 另需显式开 HandleMethodNotAllowed：gin 默认 false，错动词会被当成 404 处理，
// 拿不到 405 语义。
func Register(r *gin.Engine) {
	r.HandleMethodNotAllowed = true

	r.NoRoute(func(c *gin.Context) {
		log.Printf("[notfound] 请求地址'%s'不存在.", c.Request.URL.Path)
		c.AbortWithStatusJSON(http.StatusOK,
			response.FailCode(response.CodeNotFound, msgNotFound))
	})

	r.NoMethod(func(c *gin.Context) {
		method := c.Request.Method
		log.Printf("[notfound] 请求地址'%s',不支持'%s'请求", c.Request.URL.Path, method)
		c.AbortWithStatusJSON(http.StatusOK,
			response.FailCode(response.CodeMethodNotAllowed,
				fmt.Sprintf(msgMethodNotAllowed, method)))
	})
}

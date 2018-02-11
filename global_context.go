package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// SetGlobalCtx for 设置全局上下文
func SetGlobalCtx(globalCtxMap map[string]interface{}) gin.HandlerFunc {
	return func(context *gin.Context) {
		fmt.Println(context.Request.Host)
		// 设置上下文全局实例
		for globalName, globalInstance := range globalCtxMap {
			context.Set(globalName, globalInstance)
		}
	}
}

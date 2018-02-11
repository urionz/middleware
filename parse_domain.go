package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ParseDomain 解析域名domain的中间件
func ParseDomain(placeholders string) gin.HandlerFunc {
	return func(context *gin.Context) {
		domainSlice := strings.FieldsFunc(context.Request.Host, func(r rune) bool {
			if string(r) == "." {
				return true
			}
			return false
		})
		placeholders := strings.FieldsFunc(placeholders, func(r rune) bool {
			if string(r) == "." {
				return true
			}
			return false
		})
		if len(placeholders) != len(domainSlice) {
			context.AbortWithStatus(500)
		}
		for index, name := range domainSlice {
			switch index {
			case 2:
				context.Set("topDomain", name)
			case 3:
				context.Set("secondDomain", name)
			case 4:
				context.Set("thirdDomain", name)
			}
		}
	}
}

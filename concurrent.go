package middleware

import (
	"github.com/gin-gonic/gin"
)

// DefaultConcurrentNum 默认并发数量
var DefaultConcurrentNum = 100

// Concurrent 控制api并发数量
func Concurrent(concurrentNum int) gin.HandlerFunc {
	sem := make(chan bool, concurrentNum)
	pass := func() {
		sem <- true
	}
	release := func() { <-sem }
	return func(context *gin.Context) {
		pass()
		defer release()
		context.Next()
	}
}

// DefaultConcurrent 默认并发控制 并发数为100
func DefaultConcurrent() gin.HandlerFunc {
	return Concurrent(DefaultConcurrentNum)
}

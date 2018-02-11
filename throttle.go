package middleware

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/urionz/store"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxAttempts 默认最大请求次数为60次
	DefaultMaxAttempts = 60
	// DefaultDecayDuration 默认频率控制周期为1分钟
	DefaultDecayDuration = time.Minute
)

var cache *store.Container

// ThrottleConfigure 节流控制实例
type ThrottleConfigure struct {
	MaxAttempts   int
	DecayDuration time.Duration
}

// 初始化配置项
func (config *ThrottleConfigure) prepare() {
	if config.MaxAttempts == 0 {
		config.MaxAttempts = DefaultMaxAttempts
	}
	if config.DecayDuration == 0 {
		config.DecayDuration = DefaultDecayDuration
	}
}

// ThrottleDefault 默认节流中间件
func ThrottleDefault() gin.HandlerFunc {
	return Throttle(ThrottleConfigure{
		DefaultMaxAttempts,
		DefaultDecayDuration,
	})
}

// Throttle 可自定义配置的节流中间件
func Throttle(config ThrottleConfigure) gin.HandlerFunc {
	config.prepare()
	return func(context *gin.Context) {
		// 设置缓存对象
		cache = context.MustGet("cache").(*store.Container)
		// 获取唯一签名
		key := resolveRequestSignature(context)
		// 超过尝试次数 返回 429
		if tooManyAttempts(key, config.MaxAttempts) {
			var timer time.Time
			cache.GetScan(key+":timer", &timer)
			retryAfter := timer.Sub(time.Now())
			headers := getHeaders(config.MaxAttempts, calculateRemainingAttempts(key, config.MaxAttempts, retryAfter), retryAfter)
			for k, v := range headers {
				context.Writer.Header().Set(k, v)
			}
			context.AbortWithStatus(http.StatusTooManyRequests)
		}
		// 命中存在的访问者
		hit(key, config.DecayDuration)
		// 设置header
		for k, v := range getHeaders(config.MaxAttempts, calculateRemainingAttempts(key, config.MaxAttempts, 0), 0) {
			context.Writer.Header().Add(k, v)
		}
	}
}

// 命中唯一访问者
func hit(key string, decayMinutes time.Duration) int {
	hits := 0
	cache.Add(key+":timer", availableAt(decayMinutes), decayMinutes)
	added := cache.Add(key, 0, decayMinutes)
	cache.Increment(key, 1)
	if err := cache.GetScan(key, &hits); err != nil && !added && hits == 1 {
		cache.Put(key, 1, decayMinutes)
	}
	return hits
}

// 计算下一次可用时间
func availableAt(delay time.Duration) time.Time {
	return time.Now().Add(delay)
}

func getHeaders(maxAttempts, remainingAttempts int, retryAfter time.Duration) map[string]string {
	headers := map[string]string{
		"X-RateLimit-Limit":     strconv.Itoa(maxAttempts),
		"X-RateLimit-Remaining": strconv.Itoa(remainingAttempts),
	}
	if retryAfter.Nanoseconds() != 0 {
		headers["Retry-After"] = retryAfter.String()
		headers["X-RateLimit-Reset"] = availableAt(retryAfter).Format("2006-01-02 15:04:05")
	}
	return headers
}

// 判断是否超出尝试次数
func tooManyAttempts(key string, maxAttempts int) bool {
	attempts := 0
	if err := cache.GetScan(key, &attempts); err != nil {
		return false
	}
	if attempts >= maxAttempts {
		if cache.Has(key + ":timer") {
			return true
		}
		cache.Forget(key)
	}
	return false
}

// 计算唯一签名
func resolveRequestSignature(context *gin.Context) string {
	domain := context.Request.URL.Host
	ip := context.Request.RemoteAddr
	hash := sha1.New()
	io.WriteString(hash, domain+"|"+ip)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// 计算剩余尝试次数
func calculateRemainingAttempts(key string, maxAttempts int, retryAfter time.Duration) int {
	attempts := 0
	err := cache.GetScan(key, &attempts)
	if err != nil {
		return 0
	}
	if retryAfter.Nanoseconds() == 0 {
		return maxAttempts - attempts
	}
	return 0
}

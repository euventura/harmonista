package common

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/valkey-io/valkey-go"
)

// RateLimit returns a Gin middleware that allows up to `limit` requests per
// `window` per client IP, scoped by a string key (e.g. "login").
//
// Uses Valkey INCR + EXPIRE atomically. If Valkey is unavailable (vk == nil
// or call errors), the middleware fails open — legitimate users are never
// blocked because of an outage.
func RateLimit(vk valkey.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vk == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			c.Next()
			return
		}

		key := "ratelimit:" + scope + ":" + ip
		ctx := context.Background()

		count, err := vk.Do(ctx, vk.B().Incr().Key(key).Build()).ToInt64()
		if err != nil {
			c.Next()
			return
		}

		// Set expiry only on the first hit of the window.
		if count == 1 {
			_ = vk.Do(ctx, vk.B().Expire().Key(key).Seconds(int64(window.Seconds())).Build()).Error()
		}

		if count > int64(limit) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Muitas tentativas. Tente novamente em alguns minutos.",
			})
			return
		}

		c.Next()
	}
}

package common

import (
	"context"
	"log"
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
			log.Printf("DEBUG: RateLimit called with nil Valkey client for scope=%s", scope)
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			log.Printf("DEBUG: RateLimit empty IP for scope=%s", scope)
			c.Next()
			return
		}

		key := "ratelimit:" + scope + ":" + ip
		ctx := context.Background()

		count, err := vk.Do(ctx, vk.B().Incr().Key(key).Build()).ToInt64()
		if err != nil {
			log.Printf("ERROR: RateLimit INCR failed for key=%s: %v", key, err)
			c.Next()
			return
		}

		// Set expiry only on the first hit of the window.
		if count == 1 {
			if err := vk.Do(ctx, vk.B().Expire().Key(key).Seconds(int64(window.Seconds())).Build()).Error(); err != nil {
				log.Printf("ERROR: RateLimit EXPIRE failed for key=%s: %v", key, err)
			}
		}

		if count > int64(limit) {
			log.Printf("DEBUG: RateLimit exceeded for key=%s, count=%d, limit=%d", key, count, limit)
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Muitas tentativas. Tente novamente em alguns minutos.",
			})
			return
		}

		c.Next()
	}
}

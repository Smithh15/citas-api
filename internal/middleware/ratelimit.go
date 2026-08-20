package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit aplica un límite de peticiones por IP usando un token bucket en
// memoria. Pensado para endpoints públicos sensibles (login, registro) en una
// sola instancia; si el servicio llegara a escalar horizontalmente, esto
// tendría que moverse a Redis para compartir el estado entre instancias.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)

	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		v, ok := visitors[ip]
		if !ok {
			v = &visitor{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			visitors[ip] = v
		}
		v.lastSeen = time.Now()
		allow := v.limiter.Allow()
		mu.Unlock()

		if !allow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, try again later"})
			return
		}

		c.Next()
	}
}

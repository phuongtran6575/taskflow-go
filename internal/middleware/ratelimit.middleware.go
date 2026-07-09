package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"TaskFlow-Go/internal/shared/appresponse"
)

type RateLimitBy string

const (
	RateLimitByIP   RateLimitBy = "ip"
	RateLimitByUser RateLimitBy = "user"
)

type RateLimitConfig struct {
	Enabled                   bool
	MaxReqs                   int
	Window                    time.Duration
	By                        RateLimitBy
	BlockOnConsecutiveFailures bool
	BlockDuration              time.Duration
	MaxConsecutiveFailures     int
}

var DefaultRateLimitConfig = RateLimitConfig{
	Enabled:                   true,
	MaxReqs:                   10,
	Window:                    time.Minute,
	By:                        RateLimitByIP,
	BlockOnConsecutiveFailures: false,
	BlockDuration:              15 * time.Minute,
	MaxConsecutiveFailures:     10,
}

func (mw *Middleware) RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	if cfg.Window == 0 {
		cfg.Window = time.Minute
	}
	if cfg.MaxReqs <= 0 {
		cfg.MaxReqs = 10
	}
	if cfg.BlockDuration == 0 {
		cfg.BlockDuration = 15 * time.Minute
	}
	if cfg.MaxConsecutiveFailures <= 0 {
		cfg.MaxConsecutiveFailures = 10
	}

	return func(c *gin.Context) {
		key := mw.resolveKey(c, cfg.By)
		if key == "" {
			c.Next()
			return
		}

		blockKey := "rl:block:" + key
		if _, err := mw.cache.Get(blockKey); err == nil {
			appresponse.Fail(c, http.StatusTooManyRequests, "RATE_LIMIT_BLOCKED", "Too many failed attempts. Try again later.")
			c.Abort()
			return
		}

		counterKey := "rl:counter:" + key + ":" + c.FullPath()
		raw, err := mw.cache.Get(counterKey)
		var count int
		if err == nil {
			count, _ = strconv.Atoi(string(raw))
		}

		count++
		if count > cfg.MaxReqs {
			appresponse.Fail(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please slow down.")
			c.Abort()
			return
		}

		mw.cache.Set(counterKey, []byte(strconv.Itoa(count)), cfg.Window)

		if cfg.BlockOnConsecutiveFailures {
			c.Set("_rl_consecutive_key", counterKey)
			c.Set("_rl_block_key", blockKey)
			c.Set("_rl_block_cfg", cfg)
		}

		c.Next()
	}
}

func (mw *Middleware) resolveKey(c *gin.Context, by RateLimitBy) string {
	switch by {
	case RateLimitByIP:
		ip := c.ClientIP()
		if ip == "" {
			ip = c.Request.RemoteAddr
		}
		return strings.Split(ip, ":")[0]
	case RateLimitByUser:
		uid := c.GetString("user_id")
		if uid == "" {
			uid = c.ClientIP()
		}
		return uid
	default:
		return c.ClientIP()
	}
}

func (mw *Middleware) TrackFailedAttempt(c *gin.Context) {
	key, exists := c.Get("_rl_consecutive_key")
	if !exists {
		return
	}
	blockKey, _ := c.Get("_rl_block_key")
	cfg, _ := c.Get("_rl_block_cfg")
	rlCfg, ok := cfg.(RateLimitConfig)
	if !ok {
		return
	}

	counterKey := key.(string)
	raw, err := mw.cache.Get(counterKey)
	if err != nil {
		return
	}
	count, _ := strconv.Atoi(string(raw))

	if count >= rlCfg.MaxConsecutiveFailures {
		mw.cache.Set(blockKey.(string), []byte("1"), rlCfg.BlockDuration)
		mw.cache.Delete(counterKey)
	}
}

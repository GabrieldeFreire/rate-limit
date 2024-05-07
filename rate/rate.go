package rate

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GabrieldeFreire/rate-limit/cache"
	"github.com/GabrieldeFreire/rate-limit/log"
	"github.com/go-redis/redis/v8"
)

var (
	logger        = log.GetLogger()
	rdb           *redis.Client
	limiterConfig *RequestsConfig
	ctx           = context.Background()
)

type RequestsConfig struct {
	maxRequests       uint64
	windowMillisecond int64
	blockMillisecond  int64
}

type (
	getKeyFromRequest func(*http.Request) string
)

type RateLimiter struct {
	cache.CacheStrategy
	RequestsConfig
	keyName string
	GetKey  getKeyFromRequest
	Count   uint64
}

func NewIpRateLimiter(cacheStrategy cache.CacheStrategy) *RateLimiter {
	maxRequests, err := strconv.Atoi(os.Getenv("MAX_REQUESTS_IP"))
	if err != nil {
		maxRequests, err = strconv.Atoi(os.Getenv("MAX_REQUESTS"))
		if err != nil {
			logger.Error(("MAX_REQUESTS_IP or MAX_REQUESTS must be set and must be an integer"))
			os.Exit(1)
		}
	}
	windowSeconds, err := strconv.Atoi(os.Getenv("WINDOW_SECONDS_IP"))
	if err != nil {
		windowSeconds, err = strconv.Atoi(os.Getenv("WINDOW_SECONDS"))
		if err != nil {
			logger.Error("WINDOW_SECONDS_IP or WINDOW_SECONDS must be set and must be an integer")
			os.Exit(1)
		}
	}
	blockSeconds, err := strconv.Atoi(os.Getenv("BLOCK_SECONDS_IP"))
	if err != nil {
		blockSeconds, err = strconv.Atoi(os.Getenv("BLOCK_SECONDS"))
		if err != nil {
			logger.Error("BLOCK_SECONDS_IP or BLOCK_SECONDS must be set and must be an integer")
			os.Exit(1)
		}
	}
	logger.Info(
		"Rate limit IP setup",
		"maxRequests", maxRequests,
		"windowSeconds", windowSeconds,
		"blockSeconds", blockSeconds,
	)
	windowMillisecond := windowSeconds * 1000
	blockMillisecond := blockSeconds * 1000
	requestsConfig := RequestsConfig{
		maxRequests:       uint64(maxRequests),
		windowMillisecond: int64(windowMillisecond),
		blockMillisecond:  int64(blockMillisecond),
	}

	getKey := func(r *http.Request) string {
		key := "IP_" + strings.Split(r.RemoteAddr, ":")[0]
		return key
	}

	return &RateLimiter{
		CacheStrategy:  cacheStrategy,
		RequestsConfig: requestsConfig,
		keyName:        "ip",
		GetKey:         getKey,
		Count:          0,
	}
}

func NewTokenRateLimitter(cacheStrategy cache.CacheStrategy) *RateLimiter {
	maxRequests, err := strconv.Atoi(os.Getenv("MAX_REQUESTS_TOKEN"))
	if err != nil {
		maxRequests, err = strconv.Atoi(os.Getenv("MAX_REQUESTS"))
		if err != nil {
			logger.Error(("MAX_REQUESTS_TOKEN or MAX_REQUESTS must be set and must be an integer"))
			os.Exit(1)
		}
	}
	windowSeconds, err := strconv.Atoi(os.Getenv("WINDOW_SECONDS_TOKEN"))
	if err != nil {
		windowSeconds, err = strconv.Atoi(os.Getenv("WINDOW_SECONDS"))
		if err != nil {
			logger.Error("WINDOW_SECONDS_TOKEN or WINDOW_SECONDS must be set and must be an integer")
			os.Exit(1)
		}
	}
	blockSeconds, err := strconv.Atoi(os.Getenv("BLOCK_SECONDS_TOKEN"))
	if err != nil {
		blockSeconds, err = strconv.Atoi(os.Getenv("BLOCK_SECONDS"))
		if err != nil {
			logger.Error("BLOCK_SECONDS_TOKEN or BLOCK_SECONDS must be set and must be an integer")
			os.Exit(1)
		}
	}
	logger.Info(
		"Rate limit TOKEN setup",
		"maxRequests", maxRequests,
		"windowSeconds", windowSeconds,
		"blockSeconds", blockSeconds,
	)
	windowMillisecond := windowSeconds * 1000
	blockMillisecond := blockSeconds * 1000
	requestsConfig := RequestsConfig{
		maxRequests:       uint64(maxRequests),
		windowMillisecond: int64(windowMillisecond),
		blockMillisecond:  int64(blockMillisecond),
	}

	return &RateLimiter{
		CacheStrategy:  cacheStrategy,
		RequestsConfig: requestsConfig,
		keyName:        "token",
		GetKey: func(r *http.Request) string {
			apiKey := r.Header.Get("API_KEY")
			if apiKey == "" {
				return ""
			}
			key := "TOKEN_" + apiKey
			return key
		},
		Count: 0,
	}
}

func (rl *RateLimiter) IsBlocked(key string, timeNow time.Time) bool {
	err := rl.Expire(key, timeNow.UnixMilli()-rl.blockMillisecond)
	if err != nil {
		return true
	}

	count, err := rl.Get(key, timeNow.UnixMilli()-rl.blockMillisecond)
	if err != nil || count > 0 {
		return true
	}

	return false
}

func (rl *RateLimiter) AllowRequest(key string) bool {
	timeNow := time.Now()
	blockKey := key + "_block"

	if isKeyBlocl := rl.IsBlocked(blockKey, timeNow); isKeyBlocl {
		return false
	}

	err := rl.Expire(key, timeNow.UnixMilli()-rl.windowMillisecond)
	if err != nil {
		logger.Error("Redis ZREMRANGEBYSCORE error", err)
		return false
	}

	rl.Count, err = rl.Get(key, timeNow.UnixMilli()-rl.windowMillisecond)
	if err != nil || rl.Count >= rl.maxRequests {
		rl.Add(blockKey, timeNow)
		return false
	}

	err = rl.Add(key, timeNow)
	if err != nil {
		logger.Error("Redis ZAdd error", err)
		return false
	}

	rl.Count++

	logger.Debug(
		"RedisStrategy", "key", rl.keyName, "countOnWindow", rl.Count,
	)
	return true
}

func CreateLimitRateMiddleware(cacheStrategy cache.CacheStrategy) func(next http.HandlerFunc) http.HandlerFunc {
	tokenRateLimit := NewTokenRateLimitter(cacheStrategy)
	ipRateLimit := NewIpRateLimiter(cacheStrategy)

	limitRateMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var limiter *RateLimiter
			identifier := tokenRateLimit.GetKey(r)
			if identifier != "" {
				limiter = tokenRateLimit
			} else {
				limiter = ipRateLimit
				identifier = ipRateLimit.GetKey(r)
			}
			allowRequest := limiter.AllowRequest(identifier)
			count := strconv.FormatUint(limiter.Count, 10)
			w.Header().Set("Indentity", identifier)
			w.Header().Set("Request-Count", count)
			if !allowRequest {
				http.Error(w, "You have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		}
	}

	return limitRateMiddleware
}

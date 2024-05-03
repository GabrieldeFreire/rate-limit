package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheStrategy interface {
	Add(key string, timeNow time.Time) error
	Get(key string, timeEpoxLimit int64) (uint64, error)
	Expire(key string, timeEpoxLimit int64) error
}

type RedisStrategy struct {
	*redis.Client
	Ctx context.Context
}

func NewRedisStrategy(redisOptions *redis.Options) *RedisStrategy {
	return &RedisStrategy{
		Client: redis.NewClient(redisOptions),
		Ctx:    context.Background(),
	}
}

func (r *RedisStrategy) Add(key string, timeNow time.Time) error {
	score := timeNow.UnixMilli()
	_, err := r.ZAdd(r.Ctx, key, &redis.Z{
		Score:  float64(score),
		Member: float64(score),
	}).Result()
	if err != nil {
		return errors.Join(err, fmt.Errorf("redis ZAdd error"))
	}
	return nil
}

func (r *RedisStrategy) Get(key string, timeEpoxLimit int64) (uint64, error) {
	fromString := strconv.FormatInt(timeEpoxLimit, 10)

	count, err := r.ZCount(r.Ctx, key, fromString, "+inf").Result()
	if err != nil {
		return 0, errors.Join(err, fmt.Errorf("failed to count items key %v", key))
	}

	return uint64(count), nil
}

func (r *RedisStrategy) Expire(key string, timeEpoxLimit int64) error {
	expireString := strconv.FormatInt(timeEpoxLimit, 10)

	removeByScore := r.ZRemRangeByScore(r.Ctx, key, "-inf", expireString)
	if removeByScore.Err() != nil {
		return errors.Join(removeByScore.Err(), fmt.Errorf("redis ZREMRANGEBYSCORE error"))
	}
	return nil
}

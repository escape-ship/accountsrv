package redis

import (
	"github.com/redis/go-redis/v9" // Redis 클라이언트 라이브러리
)

type RedisClient struct {
	RedisClient *redis.Client
}

// NewClient는 Redis 클라이언트를 초기화하고 반환한다.
func NewClient() *RedisClient {
	rc := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // Redis 서버 주소
		Password: "",           // 비밀번호 (기본값 없음)
		DB:       0,            // 사용할 Redis DB 번호
	})
	return &RedisClient{
		RedisClient: rc}
}

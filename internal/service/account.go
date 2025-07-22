package service

import (
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/pkg/postgres"
	pb "github.com/escape-ship/protos/gen"
)

type AccountService struct {
	pb.AccountServer
	pg          postgres.DBEngine
	RedisClient *redis.RedisClient
}

func NewAccountService(pg postgres.DBEngine, redisClient *redis.RedisClient) *AccountService {
	return &AccountService{
		pg:          pg,
		RedisClient: redisClient,
	}
}

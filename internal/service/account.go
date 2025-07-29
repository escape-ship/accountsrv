package service

import (
	"github.com/escape-ship/accountsrv/config"
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/pkg/postgres"
	pb "github.com/escape-ship/protos/gen"
)

type AccountService struct {
	pb.AccountServiceServer
	pg          postgres.DBEngine
	RedisClient *redis.RedisClient
	config      *config.Config
}

func NewAccountService(pg postgres.DBEngine, redisClient *redis.RedisClient, cfg *config.Config) *AccountService {
	return &AccountService{
		pg:          pg,
		RedisClient: redisClient,
		config:      cfg,
	}
}

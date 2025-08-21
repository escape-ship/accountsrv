package service

import (
	"log/slog"

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
	logger      *slog.Logger
}

func NewAccountService(pg postgres.DBEngine, redisClient *redis.RedisClient, cfg *config.Config) *AccountService {
	logger := slog.Default().With("service", "account")
	return &AccountService{
		pg:          pg,
		RedisClient: redisClient,
		config:      cfg,
		logger:      logger,
	}
}

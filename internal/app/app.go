package app

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	"github.com/escape-ship/accountsrv/internal/service"
)

func init() {
	// 환경 변수 로드
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: No .env file found")
	}
}

type App struct {
	AccountGRPCServer *service.Server
	Queris            *postgresql.Queries
	Redis             *redis.RedisClient
}

func New(accountGrpc *service.Server, db *postgresql.Queries, redis *redis.RedisClient) *App {
	return &App{
		AccountGRPCServer: accountGrpc,
		Queris:            db,
		Redis:             redis,
	}
}

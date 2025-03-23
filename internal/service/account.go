package service

import (
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/internal/infra/sqlc/mysql"
	pb "github.com/escape-ship/accountsrv/proto/gen"
)

type Server struct {
	pb.AccountServer
	Queris      *mysql.Queries
	RedisClient *redis.RedisClient
}

func New(query *mysql.Queries, redisClient *redis.RedisClient) *Server {
	return &Server{
		Queris:      query,
		RedisClient: redisClient,
	}
}

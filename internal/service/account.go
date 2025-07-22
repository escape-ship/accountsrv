package service

import (
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
)

type Server struct {
	pb.AccountServer
	Queris      *postgresql.Queries
	RedisClient *redis.RedisClient
}

func New(query *postgresql.Queries, redisClient *redis.RedisClient) *Server {
	return &Server{
		Queris:      query,
		RedisClient: redisClient,
	}
}

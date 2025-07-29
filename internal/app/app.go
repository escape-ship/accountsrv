package app

import (
	"log"
	"net"

	"github.com/escape-ship/accountsrv/config"
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/internal/service"
	"github.com/escape-ship/accountsrv/pkg/postgres"
	pb "github.com/escape-ship/protos/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	pg             postgres.DBEngine
	AccountService *service.AccountService
	Listener       net.Listener
}

func New(pg postgres.DBEngine, listener net.Listener, redisClient *redis.RedisClient, cfg *config.Config) *App {
	return &App{
		pg:             pg,
		Listener:       listener,
		AccountService: service.NewAccountService(pg, redisClient, cfg),
	}
}

// App 실행: gRPC 서버와 Kafka consumer를 모두 실행
func (a *App) Run() {
	grpcServer := grpc.NewServer()
	// gRPC 서비스 등록
	pb.RegisterAccountServiceServer(grpcServer, a.AccountService)

	reflection.Register(grpcServer)

	log.Println("gRPC server listening on :8082")
	if err := grpcServer.Serve(a.Listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

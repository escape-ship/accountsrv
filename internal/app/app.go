package app

import (
	"log"

	"github.com/escape-ship/accountsrv/internal/app/domain"
	"github.com/joho/godotenv"
)

func init() {
	// 환경 변수 로드
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: No .env file found")
	}
}

type App struct {
	AccountGRPCServer *domain.Server
}

func New() *App {
	return &App{
		AccountGRPCServer: domain.New(),
	}
}

package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/escape-ship/accountsrv/config"
	"github.com/escape-ship/accountsrv/internal/app"
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/pkg/postgres"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx 드라이버 등록
)

func main() {
	// JSON 형태의 구조화된 로깅 설정 (Docker logs에서 잘 보임)
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting AccountService server", slog.String("port", "8081"))

	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		logger.Error("Failed to start TCP listener", slog.String("error", err.Error()))
		return
	}

	logger.Info("Loading configuration")
	cfg, err := config.New("config.yaml")
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	logger.Info("Connecting to Redis")
	redisClient := redis.NewClient()
	logger.Info("Redis connection established")

	logger.Info("Connecting to database")
	db, err := postgres.New(makeDSN(cfg.Database))
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Database connection established")

	logger.Info("Initializing application")
	application := app.New(db, lis, redisClient, cfg)
	logger.Info("Starting gRPC server")
	application.Run()
}

// config.Database 값 사용
func makeDSN(db config.Database) postgres.DBConnString {
	return postgres.DBConnString(
		fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s",
			db.User, db.Password,
			db.Host, db.Port,
			db.DataBaseName, db.SSLMode, db.SchemaName,
		),
	)
}

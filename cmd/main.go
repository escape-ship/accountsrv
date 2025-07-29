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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		logger.Error(err.Error())
		return
	}
	cfg, err := config.New("config.yaml")
	if err != nil {
		logger.Error("App: config load error", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient()

	db, err := postgres.New(makeDSN(cfg.Database))
	if err != nil {
		logger.Error("App: database connection error", "error", err)
		os.Exit(1)
	}

	application := app.New(db, lis, redisClient, cfg)
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

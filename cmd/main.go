package main

import (
	"database/sql"
	"fmt"
	"net"

	"github.com/escape-ship/accountsrv/internal/app"
	"github.com/escape-ship/accountsrv/internal/infra/redis"
	"github.com/escape-ship/accountsrv/internal/infra/sqlc/mysql"
	"github.com/escape-ship/accountsrv/internal/service"
	pb "github.com/escape-ship/accountsrv/proto/gen"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		return
	}
	// // 환경변수 읽어오기
	// app.LoadEnv()

	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s?parseTime=true",
		"testuser", "testpassword", "0.0.0.0", "3306", "escape")

	fmt.Println("Connecting to DB:", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// m, err := migrate.New("file://db/migrations", dsn)
	// if err != nil {
	// 	log.Fatal("Migration init failed:", err)
	// }
	// if err := m.Up(); err != nil && err != migrate.ErrNoChange {
	// 	log.Fatal("Migration failed:", err)
	// }
	// fmt.Println("Database migrated successfully!")

	// account srv 초기화
	queries := mysql.New(db)
	redisClient := redis.NewClient()
	accountGRPCServer := service.New(queries, redisClient)

	newSrv := app.New(accountGRPCServer, queries, redisClient)
	s := grpc.NewServer()

	pb.RegisterAccountServer(s, newSrv.AccountGRPCServer)

	reflection.Register(s)

	fmt.Println("Serving accountsrv on http://0.0.0.0:9090")

	if err := s.Serve(lis); err != nil {
		return
	}
}

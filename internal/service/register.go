package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AccountService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {

	db := s.pg.GetDB()
	querier := postgresql.New(db)

	tx, err := db.Begin()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	qtx := querier.WithTx(tx)
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// 이메일 중복 체크
	_, err = qtx.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "email already registered")
	}
	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
	}

	// 비밀번호 해시 생성
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	// 사용자 삽입
	userid, err := qtx.InsertUser(ctx, postgresql.InsertUserParams{
		Email:        req.Email,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	return &pb.RegisterResponse{
		Message: fmt.Sprintf("Registration successful, user ID: %d", userid),
	}, nil
}

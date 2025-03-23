package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/mysql"
	pb "github.com/escape-ship/accountsrv/proto/gen"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// 이메일 중복 체크
	_, err := s.Queris.GetUserByEmail(ctx, req.Email)
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
	if err := s.Queris.InsertUser(ctx, mysql.InsertUserParams{
		Email:        req.Email,
		PasswordHash: string(passwordHash),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	// 삽입된 ID 가져오기
	userID, err := s.Queris.GetLastInsertID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user ID: %v", err)
	}

	return &pb.RegisterResponse{
		Message: fmt.Sprintf("Registration successful, user ID: %d", userID),
	}, nil
}

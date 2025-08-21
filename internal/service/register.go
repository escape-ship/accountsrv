package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AccountService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	logger := s.logger.With("method", "Register", "email", req.Email)
	logger.Info("Starting user registration")

	db := s.pg.GetDB()
	querier := postgresql.New(db)

	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	qtx := querier.WithTx(tx)
	defer func() {
		if err != nil {
			logger.Warn("Rolling back transaction due to error")
			tx.Rollback()
		} else {
			logger.Debug("Committing transaction")
			tx.Commit()
		}
	}()

	// 이메일 중복 체크
	logger.Debug("Checking email duplication")
	_, err = qtx.GetUserByEmail(ctx, req.Email)
	if err == nil {
		logger.Warn("Email already registered", slog.String("email", req.Email))
		return nil, status.Errorf(codes.AlreadyExists, "email already registered")
	}
	if err != sql.ErrNoRows {
		logger.Error("Failed to check email duplication", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
	}
	logger.Debug("Email is available")

	// 비밀번호 해시 생성
	logger.Debug("Generating password hash")
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash password", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	// 사용자 삽입
	userID := uuid.New()
	logger.Info("Creating new user", slog.String("user_id", userID.String()))
	returnedUserID, err := qtx.InsertUser(ctx, postgresql.InsertUserParams{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		logger.Error("Failed to register user",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	logger.Info("User registration successful",
		slog.String("user_id", returnedUserID.String()),
		slog.String("email", req.Email))

	return &pb.RegisterResponse{
		Message: fmt.Sprintf("Registration successful, user ID: %s", returnedUserID.String()),
	}, nil
}

package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AccountService) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {

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

	user, err := qtx.GetUserByEmail(ctx, in.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	// 2. 비밀번호 검증
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}
	// 3. 액세스 토큰 생성
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}
	// Redis에 저장
	if err := s.RedisClient.RedisClient.Set(ctx, fmt.Sprintf("access_token:%d", user.ID), accessToken, 15*time.Minute).Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store access token: %v", err)
	}
	// 4. 리프레시 토큰 생성
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token: %v", err)
	}
	// DB에 저장
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := qtx.InsertRefreshToken(ctx, postgresql.InsertRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}
	// 5. 응답 반환
	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AccountService) generateAccessToken(userID int64) (string, error) {
	fmt.Printf("AccountSrv - JWT Secret length: %d\n", len(s.config.Auth.JWTSecret))
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Auth.JWTSecret))
}

func (s *AccountService) generateRefreshToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(14 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Auth.JWTSecret))
}

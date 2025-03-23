package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/mysql"
	pb "github.com/escape-ship/accountsrv/proto/gen"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var jwtSecret = []byte("jwt secret key")

func (s *Server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.Queris.GetUserByEmail(ctx, in.Email)
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
	accessToken, err := generateAccessToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}
	// Redis에 저장
	if err := s.RedisClient.RedisClient.Set(ctx, fmt.Sprintf("access_token:%d", user.ID), accessToken, 15*time.Minute).Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store access token: %v", err)
	}
	// 4. 리프레시 토큰 생성
	refreshToken, err := generateRefreshToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token: %v", err)
	}
	// DB에 저장
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := s.Queris.InsertRefreshToken(ctx, mysql.InsertRefreshTokenParams{
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

func generateAccessToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func generateRefreshToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(14 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AccountService) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	logger := s.logger.With("method", "Login", "email", in.Email)
	logger.Info("Starting user login")

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

	logger.Debug("Looking up user by email")
	user, err := qtx.GetUserByEmail(ctx, in.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("User not found", slog.String("email", in.Email))
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		logger.Error("Failed to get user", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	logger.Debug("User found", slog.String("user_id", user.ID.String()))

	// 2. 비밀번호 검증
	logger.Debug("Verifying password")
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		logger.Warn("Invalid password attempt", slog.String("user_id", user.ID.String()))
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}
	logger.Debug("Password verified successfully")

	// 3. 액세스 토큰 생성
	logger.Debug("Generating access token")
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		logger.Error("Failed to generate access token",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	// Redis에 저장
	logger.Debug("Storing access token in Redis")
	if err := s.RedisClient.RedisClient.Set(ctx, fmt.Sprintf("access_token:%s", user.ID.String()), accessToken, 15*time.Minute).Err(); err != nil {
		logger.Error("Failed to store access token in Redis",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to store access token: %v", err)
	}

	// 4. 리프레시 토큰 생성
	logger.Debug("Generating refresh token")
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		logger.Error("Failed to generate refresh token",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token: %v", err)
	}

	// DB에 저장
	logger.Debug("Storing refresh token in database")
	refreshTokenID := uuid.New()
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := qtx.InsertRefreshToken(ctx, postgresql.InsertRefreshTokenParams{
		ID:        refreshTokenID,
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	}); err != nil {
		logger.Error("Failed to store refresh token",
			slog.String("user_id", user.ID.String()),
			slog.String("refresh_token_id", refreshTokenID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	logger.Info("User login successful", slog.String("user_id", user.ID.String()))

	// 5. 응답 반환
	return &pb.LoginResponse{
		UserId:       user.ID.String(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AccountService) generateAccessToken(userID uuid.UUID) (string, error) {
	logger := s.logger.With("method", "generateAccessToken", "user_id", userID.String())
	logger.Debug("Generating access token")

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		logger.Error("Failed to sign access token", slog.String("error", err.Error()))
		return "", err
	}

	logger.Debug("Access token generated successfully")
	return signedToken, nil
}

func (s *AccountService) generateRefreshToken(userID uuid.UUID) (string, error) {
	logger := s.logger.With("method", "generateRefreshToken", "user_id", userID.String())
	logger.Debug("Generating refresh token")

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(14 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		logger.Error("Failed to sign refresh token", slog.String("error", err.Error()))
		return "", err
	}

	logger.Debug("Refresh token generated successfully")
	return signedToken, nil
}

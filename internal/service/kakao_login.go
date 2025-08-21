package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	kakaoAuthURL  = "https://kauth.kakao.com/oauth/authorize"
	kakaoTokenURL = "https://kauth.kakao.com/oauth/token"
	kakaoUserURL  = "https://kapi.kakao.com/v2/user/me?property_keys=[\"kakao_account.nickname\",\"kakao_account.email\"]"
)

type response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type kakaoUserInfo struct {
	ID           int64          `json:"id"`
	ConnectedAt  string         `json:"connected_at"`
	Properties   UserProperties `json:"properties"`
	KakaoAccount KakaoAccount   `json:"kakao_account"`
}

// UserProperties 사용자 프로필 정보 구조체
type UserProperties struct {
	Nickname       string `json:"nickname"`
	ProfileImage   string `json:"profile_image"`
	ThumbnailImage string `json:"thumbnail_image"`
}

// KakaoAccount 카카오 계정 정보 구조체
type KakaoAccount struct {
	Email                 string `json:"email"`
	ProfileNeedsAgreement bool   `json:"profile_needs_agreement"`
	HasEmail              bool   `json:"has_email"`
}

// 카카오 로그인 URL 반환
func (*AccountService) GetKakaoLoginURL(ctx context.Context, in *pb.GetKakaoLoginURLRequest) (*pb.GetKakaoLoginURLResponse, error) {
	clientID := os.Getenv("KAKAO_CLIENT_ID")
	redirectURI := os.Getenv("KAKAO_REDIRECT_URI")

	url := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code",
		kakaoAuthURL, clientID, redirectURI)

	return &pb.GetKakaoLoginURLResponse{LoginUrl: url}, nil
}

// 카카오 토큰 요청
func (s *AccountService) getKakaoToken(code string) (*response, error) {
	logger := s.logger.With("method", "getKakaoToken")
	logger.Debug("Requesting Kakao token")

	clientID := os.Getenv("KAKAO_CLIENT_ID")
	clientSecret := os.Getenv("KAKAO_CLIENT_SECRET")
	redirectURI := os.Getenv("KAKAO_REDIRECT_URI")

	reqBody := fmt.Sprintf("grant_type=authorization_code&client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		clientID, clientSecret, redirectURI, code)

	resp, err := http.Post(kakaoTokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(reqBody))
	if err != nil {
		logger.Error("Failed to make HTTP request to Kakao token endpoint", slog.String("error", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 본문을 읽어오기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 디코딩을 위한 구조체로 변환
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		logger.Error("Failed to unmarshal Kakao token response",
			slog.String("error", err.Error()),
			slog.String("response_body", string(body)))
		return nil, err
	}

	accessToken, ok := res["access_token"].(string)
	if !ok {
		logger.Error("No access_token found in Kakao response", slog.String("response_body", string(body)))
		return nil, fmt.Errorf("no access_token found")
	}
	refreshToken, ok := res["refresh_token"].(string)
	if !ok {
		logger.Error("No refresh_token found in Kakao response", slog.String("response_body", string(body)))
		return nil, fmt.Errorf("no refresh_token found")
	}

	logger.Debug("Kakao token response parsed successfully")
	result := &response{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return result, nil
}

func (s *AccountService) getKakaoUserInfo(accessToken string) (*kakaoUserInfo, error) {
	logger := s.logger.With("method", "getKakaoUserInfo")
	logger.Debug("Requesting Kakao user info")

	req, err := http.NewRequest("GET", kakaoUserURL, nil)
	if err != nil {
		logger.Error("Failed to create HTTP request", slog.String("error", err.Error()))
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to make HTTP request to Kakao user info endpoint", slog.String("error", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 상태 코드와 본문 출력
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", slog.String("error", err.Error()))
		return nil, err
	}

	logger.Debug("Kakao user info response received", slog.String("response_body", string(body)))

	var result kakaoUserInfo
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("Failed to unmarshal Kakao user info response",
			slog.String("error", err.Error()),
			slog.String("response_body", string(body)))
		return nil, err
	}

	logger.Debug("Kakao user info parsed successfully")
	return &result, nil
}

// 콜백 엔드포인트
func (s *AccountService) GetKakaoCallBack(ctx context.Context, in *pb.GetKakaoCallBackRequest) (*pb.GetKakaoCallBackResponse, error) {
	logger := s.logger.With("method", "GetKakaoCallBack", "code", in.Code)
	logger.Info("Starting Kakao login callback")

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

	code := in.Code
	logger.Debug("Processing Kakao authorization code")

	// 1. 액세스 토큰 요청
	logger.Debug("Requesting Kakao access token")
	token, err := s.getKakaoToken(code)
	if err != nil {
		logger.Error("Failed to get Kakao access token", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to get access token: %v", err)
	}
	logger.Debug("Kakao access token obtained successfully")

	// 2. 사용자 정보 요청
	logger.Debug("Requesting Kakao user info")
	userInfo, err := s.getKakaoUserInfo(token.AccessToken)
	if err != nil {
		logger.Error("Failed to get Kakao user info", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to get user info: %v", err)
	}
	logger.Debug("Kakao user info obtained", slog.String("email", userInfo.KakaoAccount.Email))

	logger.Debug("Checking if user exists in database")
	existingUser, err := qtx.GetUserByEmail(ctx, userInfo.KakaoAccount.Email)
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Failed to check if user exists",
			slog.String("email", userInfo.KakaoAccount.Email),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to check if user exists: %v", err)
	}

	// 2. 사용자가 존재하지 않으면 새로 추가
	var userid uuid.UUID
	if err == sql.ErrNoRows {
		logger.Info("Creating new user from Kakao login", slog.String("email", userInfo.KakaoAccount.Email))
		// 사용자 삽입
		userID := uuid.New()
		userid, err = qtx.InsertUser(ctx, postgresql.InsertUserParams{
			ID:           userID,
			Email:        userInfo.KakaoAccount.Email,
			PasswordHash: "", // 카카오 로그인에서는 패스워드가 없으므로 빈 값으로 처리
		})
		if err != nil {
			logger.Error("Failed to register Kakao user",
				slog.String("email", userInfo.KakaoAccount.Email),
				slog.String("user_id", userID.String()),
				slog.String("error", err.Error()))
			return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
		}
		logger.Info("New user created successfully", slog.String("user_id", userid.String()))
	} else {
		// 사용자가 이미 존재하면 user_id를 가져옴
		userid = existingUser.ID
		logger.Info("Existing user found", slog.String("user_id", userid.String()))
	}

	// Redis에 저장
	logger.Debug("Storing Kakao access token in Redis")
	kakoRedisKey := fmt.Sprintf("kakao_access_token:%s", userid.String())
	if err := s.RedisClient.RedisClient.Set(ctx, kakoRedisKey, token.AccessToken, 15*time.Minute).Err(); err != nil {
		logger.Error("Failed to store Kakao access token in Redis",
			slog.String("user_id", userid.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to store access token: %v", err)
	}

	// DB에 저장
	logger.Debug("Storing refresh token in database")
	refreshTokenID := uuid.New()
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := qtx.InsertRefreshToken(ctx, postgresql.InsertRefreshTokenParams{
		ID:        refreshTokenID,
		UserID:    userid,
		Token:     token.RefreshToken,
		ExpiresAt: expiresAt,
	}); err != nil {
		logger.Error("Failed to store refresh token",
			slog.String("user_id", userid.String()),
			slog.String("refresh_token_id", refreshTokenID.String()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	logger.Info("Kakao login completed successfully",
		slog.String("user_id", userid.String()),
		slog.String("email", userInfo.KakaoAccount.Email))

	return &pb.GetKakaoCallBackResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		UserInfoJson: string(userInfo.KakaoAccount.Email),
	}, nil
}

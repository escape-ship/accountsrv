package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/escape-ship/accountsrv/internal/infra/sqlc/postgresql"
	pb "github.com/escape-ship/protos/gen"
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
func getKakaoToken(code string) (*response, error) {
	clientID := os.Getenv("KAKAO_CLIENT_ID")
	clientSecret := os.Getenv("KAKAO_CLIENT_SECRET")
	redirectURI := os.Getenv("KAKAO_REDIRECT_URI")

	reqBody := fmt.Sprintf("grant_type=authorization_code&client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		clientID, clientSecret, redirectURI, code)

	resp, err := http.Post(kakaoTokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 본문을 읽어오기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 디코딩을 위한 구조체로 변환
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		fmt.Printf("Kakao token response error: %v\n", err)
		return nil, err
	}
	for key, value := range res {
		fmt.Printf("%s: %v\n", key, value)
	}

	accessToken, ok := res["access_token"].(string)
	if !ok {
		fmt.Println("no access_token found")
		return nil, fmt.Errorf("no access_token found")
	}
	refreshToken, ok := res["refresh_token"].(string)
	if !ok {
		fmt.Println("no refresh_token found")
		return nil, fmt.Errorf("no refresh_token found")
	}
	result := &response{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return result, nil
}
func getKakaoUserInfo(accessToken string) (*kakaoUserInfo, error) {
	req, err := http.NewRequest("GET", kakaoUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 상태 코드와 본문 출력
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Kakao user info response body: %s\n", body)

	var result kakaoUserInfo
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Kakao token response error: %v\n", err)
		return nil, err
	}

	return &result, nil
}

// 콜백 엔드포인트
func (s *AccountService) GetKakaoCallBack(ctx context.Context, in *pb.GetKakaoCallBackRequest) (*pb.GetKakaoCallBackResponse, error) {

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

	in.GetCode()
	code := in.Code

	// 1. 액세스 토큰 요청
	token, err := getKakaoToken(code)
	if err != nil {
		fmt.Printf("Error getting access token: %v\n", err)
		return nil, err
	}

	// 2. 사용자 정보 요청
	userInfo, err := getKakaoUserInfo(token.AccessToken)
	if err != nil {
		fmt.Printf("Error getting user info: %v\n", err)
		return nil, err
	}

	existingUser, err := qtx.GetUserByEmail(ctx, userInfo.KakaoAccount.Email)
	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to check if user exists: %v", err)
	}
	// 2. 사용자가 존재하지 않으면 새로 추가
	var userid int64
	if err == sql.ErrNoRows {
		// 사용자 삽입
		userid, err = qtx.InsertUser(ctx, postgresql.InsertUserParams{
			Email:        userInfo.KakaoAccount.Email,
			PasswordHash: "", // 카카오 로그인에서는 패스워드가 없으므로 빈 값으로 처리
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
		}
	} else {
		// 사용자가 이미 존재하면 user_id를 가져옴
		userid = existingUser.ID
	}

	// Redis에 저장
	kakoRedisKey := fmt.Sprintf("kakao_access_token:%d", userid)
	if err := s.RedisClient.RedisClient.Set(ctx, kakoRedisKey, token.AccessToken, 15*time.Minute).Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store access token: %v", err)
	}
	// DB에 저장
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := qtx.InsertRefreshToken(ctx, postgresql.InsertRefreshTokenParams{
		UserID:    userid,
		Token:     token.RefreshToken,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	return &pb.GetKakaoCallBackResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		UserInfoJson: string(userInfo.KakaoAccount.Email),
	}, nil
}

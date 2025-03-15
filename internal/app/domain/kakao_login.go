package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	pb "github.com/escape-ship/accountsrv/proto/gen"
)

const (
	kakaoAuthURL  = "https://kauth.kakao.com/oauth/authorize"
	kakaoTokenURL = "https://kauth.kakao.com/oauth/token"
	kakaoUserURL  = "https://kapi.kakao.com/v2/user/me"
)

// 카카오 로그인 URL 반환
func (*Server) GetKakaoLoginURL(ctx context.Context, in *pb.KakaoLoginRequest) (*pb.KakaoLoginResponse, error) {
	clientID := os.Getenv("KAKAO_CLIENT_ID")
	redirectURI := os.Getenv("KAKAO_REDIRECT_URI")

	url := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code",
		kakaoAuthURL, clientID, redirectURI)

	return &pb.KakaoLoginResponse{LoginURL: url}, nil
}

// 카카오 토큰 요청
func GetKakaoToken(code string) (string, error) {
	clientID := os.Getenv("KAKAO_CLIENT_ID")
	clientSecret := os.Getenv("KAKAO_CLIENT_SECRET")
	redirectURI := os.Getenv("KAKAO_REDIRECT_URI")

	reqBody := fmt.Sprintf("grant_type=authorization_code&client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		clientID, clientSecret, redirectURI, code)

	resp, err := http.Post(kakaoTokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 응답 본문을 읽어오기
	fmt.Printf("Kakao token response status: %s\n", resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	fmt.Printf("Kakao token response body: %s\n", body)

	// 디코딩을 위한 구조체로 변환
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Kakao token response error: %v\n", err)
		return "", err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		fmt.Println("no access_token found")
		return "", fmt.Errorf("no access_token found")
	}

	return accessToken, nil
}
func getKakaoUserInfo(accessToken string) (map[string]interface{}, error) {
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
	fmt.Printf("Kakao user info response status: %s\n", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Kakao user info response body: %s\n", body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Kakao token response error: %v\n", err)
		return nil, err
	}

	return result, nil
}

// 콜백 엔드포인트
func (*Server) GetKakaoCallBack(ctx context.Context, in *pb.KakaoCallBackRequest) (*pb.KakaoCallBackResponse, error) {
	fmt.Println("Received code:", in.Code)
	in.GetCode()
	code := in.Code

	// 1. 액세스 토큰 요청
	accessToken, err := GetKakaoToken(code)
	if err != nil {
		fmt.Printf("Error getting access token: %v\n", err)
		return nil, err
	}

	// 2. 사용자 정보 요청
	userInfo, err := getKakaoUserInfo(accessToken)
	if err != nil {
		fmt.Printf("Error getting user info: %v\n", err)
		return nil, err
	}

	userInfoJson, _ := json.Marshal(userInfo)

	return &pb.KakaoCallBackResponse{
		AccessToken:  accessToken,
		UserInfoJson: string(userInfoJson),
	}, nil
}

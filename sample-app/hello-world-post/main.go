package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yuki5155/go-aws/lambda/middlewares"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")

	// ErrInvalidRequestBody Invalid request body
	ErrInvalidRequestBody = errors.New("Invalid request body")
)

// LoginRequest リクエストボディの構造体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse レスポンスボディの構造体
type LoginResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received POST request: %+v", request)

	// POSTメソッド以外は拒否
	if request.HTTPMethod != "POST" {
		return events.APIGatewayProxyResponse{
			StatusCode: 405,
			Body:       "Method Not Allowed",
		}, nil
	}

	// リクエストボディの解析
	var loginReq LoginRequest
	if err := json.Unmarshal([]byte(request.Body), &loginReq); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request format",
		}, ErrInvalidRequestBody
	}

	// ユーザー名とパスワードの検証（実際の認証ロジックはここに実装）
	if loginReq.Username == "" || loginReq.Password == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Username and password are required",
		}, nil
	}

	// 認証成功を想定（実際にはDB検証などが必要）
	log.Printf("User %s successfully authenticated", loginReq.Username)

	// 環境変数の取得
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = ".mydevportal.com" // デフォルト値
	}
	fmt.Println("cookieDomain", cookieDomain)

	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowOrigin == "" {
		allowOrigin = "https://mydevportal.com" // デフォルト値
	}
	fmt.Println("allowOrigin", allowOrigin)

	// セッションID生成（実際には安全な方法で生成する必要あり）
	sessionID := "session-" + loginReq.Username + "-" + fmt.Sprint(os.Getpid())

	// Cookie文字列の構築
	cookieStr := "session=" + sessionID +
		"; HttpOnly" +
		"; Secure" +
		"; SameSite=Lax" +
		"; Domain=" + cookieDomain +
		"; Path=/" +
		"; Max-Age=86400" // 24時間有効

	// レスポンス作成
	response := LoginResponse{
		Status:  "success",
		Message: "Login successful",
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Origin":      allowOrigin,
			"Set-Cookie":                       cookieStr,
		},
	}, nil
}

func main() {
	fmt.Println("Starting Lambda POST Handler")
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
	)
	lambda.Start(handlerWithMiddleware)
}

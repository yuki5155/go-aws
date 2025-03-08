package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yuki5155/go-aws/cognito"
	"github.com/yuki5155/go-aws/lambda/middlewares"
)

// CallbackRequest represents the expected request with an authorization code
type CallbackRequest struct {
	Code string `json:"code"`
}

// CallbackResponse represents the response structure
type CallbackResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received callback request: %+v", request)

	// POSTメソッド以外は拒否
	if request.HTTPMethod != "POST" {
		return events.APIGatewayProxyResponse{
			StatusCode: 405,
			Body:       "Method Not Allowed",
		}, nil
	}

	// Base64エンコードされたボディをデコードする処理
	var requestBody []byte
	var err error
	if request.IsBase64Encoded {
		requestBody, err = base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			log.Printf("Error decoding base64 request body: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Invalid request format",
			}, fmt.Errorf("invalid base64 encoded body: %w", err)
		}
	} else {
		requestBody = []byte(request.Body)
	}

	// リクエストボディの解析
	var callbackReq CallbackRequest
	if err := json.Unmarshal(requestBody, &callbackReq); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request format",
		}, fmt.Errorf("invalid request body: %w", err)
	}

	// 認証コードの検証
	if callbackReq.Code == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Authorization code is required",
		}, nil
	}

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

	// コールバックURLの取得と設定
	callbackURL := os.Getenv("COGNITO_CALLBACK_URL")
	if callbackURL == "" {
		log.Printf("COGNITO_CALLBACK_URL not set")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Server configuration error",
		}, fmt.Errorf("COGNITO_CALLBACK_URL not set")
	}

	// Cognito設定の取得
	cognitoConfig := cognito.CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       os.Getenv("AWS_REGION"),
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
		UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
	}

	// 認証コードからトークン取得
	tokens, err := cognito.GetTokens(callbackReq.Code, callbackURL, cognitoConfig)
	if err != nil {
		log.Printf("Error getting tokens: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Failed to authenticate",
		}, err
	}

	// アクセストークンをCookieに設定
	cookieStr := "accessToken=" + tokens.AccessToken +
		"; HttpOnly" +
		"; Secure" +
		"; SameSite=Lax" +
		"; Domain=" + cookieDomain +
		"; Path=/" +
		"; Max-Age=3600" // 1時間有効

	// IDトークンをCookieに設定
	idTokenCookieStr := "idToken=" + tokens.IdToken +
		"; HttpOnly" +
		"; Secure" +
		"; SameSite=Lax" +
		"; Domain=" + cookieDomain +
		"; Path=/" +
		"; Max-Age=3600" // 1時間有効

	// レスポンス作成
	response := CallbackResponse{
		Status:  "success",
		Message: "Authentication successful",
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, err
	}

	// レスポンスヘッダーにCookieを含めて返す
	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Origin":      allowOrigin,
		},
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {cookieStr, idTokenCookieStr},
		},
	}, nil
}

func main() {
	fmt.Println("Starting Lambda Callback Handler")
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
	)
	lambda.Start(handlerWithMiddleware)
}

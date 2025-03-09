package service

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/yuki5155/go-aws/cognito"
	"github.com/yuki5155/go-aws/lambda"
	"github.com/yuki5155/go-aws/sample-app/callback/config"
	"github.com/yuki5155/go-aws/sample-app/callback/models"
)

// Service handles the business logic of the callback handler
type Service struct {
	Config *config.Config
}

// NewService creates a new service
func NewService(cfg *config.Config) *Service {
	return &Service{
		Config: cfg,
	}
}

// ProcessCallback processes a callback request
func (s *Service) ProcessCallback(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Check HTTP method
	if request.HTTPMethod != "POST" {
		err := lambda.NewMethodNotAllowedError()
		return err.ToAPIGatewayResponse(), nil
	}

	// Decode request body if needed
	requestBody, err := s.decodeRequestBody(request)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		lambdaErr, ok := err.(*lambda.LambdaError)
		if ok {
			return lambdaErr.ToAPIGatewayResponse(), err
		}
		// If not our custom error, convert to internal server error
		internalErr := lambda.NewInternalServerError("Internal server error", err)
		return internalErr.ToAPIGatewayResponse(), err
	}

	// Parse request
	callbackReq, err := s.parseRequest(requestBody)
	if err != nil {
		log.Printf("Error parsing request: %v", err)
		lambdaErr, ok := err.(*lambda.LambdaError)
		if ok {
			return lambdaErr.ToAPIGatewayResponse(), err
		}
		// If not our custom error, convert to internal server error
		internalErr := lambda.NewInternalServerError("Internal server error", err)
		return internalErr.ToAPIGatewayResponse(), err
	}

	// Validate request
	if callbackReq.Code == "" {
		err := lambda.NewInvalidRequestError("Authorization code is required", nil)
		return err.ToAPIGatewayResponse(), nil
	}

	// Get tokens from Cognito
	tokens, err := s.getTokens(callbackReq.Code)
	if err != nil {
		log.Printf("Error getting tokens: %v", err)
		lambdaErr, ok := err.(*lambda.LambdaError)
		if ok {
			return lambdaErr.ToAPIGatewayResponse(), err
		}
		// If it's not our custom error, wrap it
		httpErr := lambda.NewInternalServerError("Failed to process authentication", err)
		return httpErr.ToAPIGatewayResponse(), err
	}

	// Create response
	return s.createSuccessResponse(tokens)
}

// decodeRequestBody decodes the request body
func (s *Service) decodeRequestBody(request events.APIGatewayProxyRequest) ([]byte, error) {
	if request.IsBase64Encoded {
		requestBody, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return nil, lambda.NewInvalidRequestError("Invalid base64 encoded body", err)
		}
		return requestBody, nil
	}
	return []byte(request.Body), nil
}

// parseRequest parses the request body
func (s *Service) parseRequest(requestBody []byte) (*models.CallbackRequest, error) {
	var callbackReq models.CallbackRequest
	if err := json.Unmarshal(requestBody, &callbackReq); err != nil {
		return nil, lambda.NewInvalidRequestError("Invalid request format", err)
	}
	return &callbackReq, nil
}

// getTokens gets tokens from Cognito
func (s *Service) getTokens(code string) (*cognito.TokenResponse, error) {
	tokens, err := cognito.GetTokens(code, s.Config.CallbackURL, s.Config.CognitoConfig)
	if err != nil {
		// Handle specific cognito errors and convert them to LambdaErrors
		// You can add more specific error handling here if needed
		return nil, lambda.NewUnauthorizedError("Failed to authenticate", err)
	}
	return tokens, nil
}

// createSuccessResponse creates a success response
func (s *Service) createSuccessResponse(tokens *cognito.TokenResponse) (events.APIGatewayProxyResponse, error) {
	// Create cookies
	accessTokenCookie := &models.Cookie{
		Name:     "accessToken",
		Value:    tokens.AccessToken,
		Domain:   s.Config.CookieDomain,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		Secure:   true,
		HttpOnly: true,
		SameSite: "Lax",
	}

	idTokenCookie := &models.Cookie{
		Name:     "idToken",
		Value:    tokens.IdToken,
		Domain:   s.Config.CookieDomain,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		Secure:   true,
		HttpOnly: true,
		SameSite: "Lax",
	}

	// Create cookie strings using the ToCookieString method
	cookieStr := accessTokenCookie.ToCookieString()
	idTokenCookieStr := idTokenCookie.ToCookieString()

	// Create response body
	response := models.CallbackResponse{
		Status:  "success",
		Message: "Authentication successful",
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		httpErr := lambda.NewInternalServerError("Failed to create response", err)
		return httpErr.ToAPIGatewayResponse(), err
	}

	// Create API Gateway response
	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Origin":      s.Config.AllowOrigin,
		},
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {cookieStr, idTokenCookieStr},
		},
	}, nil
}

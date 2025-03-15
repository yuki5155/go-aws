package middlewares

import (
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/yuki5155/go-aws/cognito"
)

// ContextKeys for user identifiers
const (
	// SubjectContextKey is the key for the subject claim
	SubjectContextKey = "sub"
	// EmailContextKey is the key for the email claim
	EmailContextKey = "email"
)

// CognitoTokenMiddlewareConfig contains the configuration for the token middleware
type CognitoTokenMiddlewareConfig struct {
	CognitoConfig cognito.CognitoConfig
}

// CognitoTokenMiddleware returns a middleware that validates the ID token from cookies
// and sets subject and email onto the Lambda context
func CognitoTokenMiddleware(config CognitoTokenMiddlewareConfig) Middleware {
	return func(next LambdaHandler) LambdaHandler {
		return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// Extract ID token from cookies
			idToken := extractTokenFromCookies(req.Headers["cookie"], "idToken")
			if idToken == "" {
				log.Println("No ID token found in cookies")
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message": "Authentication required"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// Validate the ID token
			isValid, claims, err := cognito.ValidateIDToken(idToken, config.CognitoConfig)
			if err != nil {
				log.Printf("Error validating ID token: %v", err)
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message": "Invalid authentication token"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			if !isValid {
				log.Println("ID token validation failed")
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message": "Invalid authentication token"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// Extract sub and email from claims
			sub, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)

			if sub == "" {
				log.Println("No subject claim found in token")
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message": "Invalid token: missing subject"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// Store user information in the request context via Authorizer map
			// This is a common pattern in API Gateway for passing context
			if req.RequestContext.Authorizer == nil {
				req.RequestContext.Authorizer = make(map[string]interface{})
			}
			req.RequestContext.Authorizer[SubjectContextKey] = sub
			req.RequestContext.Authorizer[EmailContextKey] = email

			log.Printf("Authenticated user: %s (%s)", sub, email)

			// Call the next handler with the updated request
			return next(req)
		}
	}
}

// GetSubjectFromContext extracts the subject from the API Gateway request context
func GetSubjectFromContext(req events.APIGatewayProxyRequest) string {
	if req.RequestContext.Authorizer == nil {
		return ""
	}

	if sub, ok := req.RequestContext.Authorizer[SubjectContextKey].(string); ok {
		return sub
	}

	return ""
}

// GetEmailFromContext extracts the email from the API Gateway request context
func GetEmailFromContext(req events.APIGatewayProxyRequest) string {
	if req.RequestContext.Authorizer == nil {
		return ""
	}

	if email, ok := req.RequestContext.Authorizer[EmailContextKey].(string); ok {
		return email
	}

	return ""
}

// extractTokenFromCookies extracts a specific token from the cookies string
func extractTokenFromCookies(cookiesStr string, tokenName string) string {
	if cookiesStr == "" {
		return ""
	}

	cookies := strings.Split(cookiesStr, ";")
	for _, cookie := range cookies {
		parts := strings.SplitN(strings.TrimSpace(cookie), "=", 2)
		if len(parts) == 2 && parts[0] == tokenName {
			return parts[1]
		}
	}

	return ""
}

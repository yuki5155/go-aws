package middlewares

import (
	"fmt"
	"log"

	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/yuki5155/go-aws/cognito"
	"github.com/yuki5155/go-aws/lambda/utils"
)

// CognitoDummyUser represents user data that would come from Cognito
type CognitoUser struct {
	UserID    string
	Username  string
	Email     string
	Groups    []string
	TokenData map[string]interface{}
}

// CognitoAuthMiddleware returns a Middleware that adds Cognito authentication data to the request
func CognitoAuthMiddleware() Middleware {
	log.Println("Creating Cognito authentication middleware")

	return func(next LambdaHandler) LambdaHandler {
		log.Println("Setting up Cognito auth handler wrapper")

		return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			log.Printf("Processing request with Cognito auth: %s", req.RequestContext.RequestID)

			// Extract ID token from cookies
			idToken := utils.GetCookieByName(req, "id_token")
			if idToken == "" {
				log.Println("No ID token found in request cookies")
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message":"Unauthorized: No valid token provided"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// Create Cognito configuration
			cognitoConfig := cognito.CognitoConfig{
				Domain:       os.Getenv("COGNITO_DOMAIN"),
				Region:       os.Getenv("AWS_REGION"),
				ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
				ClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
				UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
			}

			// Validate the ID token
			isValid, claims, err := cognito.ValidateIDToken(idToken, cognitoConfig)
			if err != nil {
				log.Printf("Token validation error: %v", err)
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message":"Unauthorized: Invalid token"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			if !isValid {
				log.Println("Token is not valid")
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"message":"Unauthorized: Token validation failed"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}
			fmt.Println(claims)

			// Extract user info from claims
			user := CognitoUser{}

			// Extract standard claims
			if sub, ok := claims["sub"].(string); ok {
				user.UserID = sub
			}
			if email, ok := claims["email"].(string); ok {
				user.Email = email
			}
			if username, ok := claims["cognito:username"].(string); ok {
				user.Username = username
			}
			// Extract groups if available
			if cognitoGroups, ok := claims["cognito:groups"].([]interface{}); ok {
				for _, group := range cognitoGroups {
					if groupStr, ok := group.(string); ok {
						user.Groups = append(user.Groups, groupStr)
					}
				}
			}

			// Add user info to request headers for passing to the next handler
			if req.Headers == nil {
				req.Headers = make(map[string]string)
			}

			req.Headers["X-Cognito-User-ID"] = user.UserID
			req.Headers["X-Cognito-Username"] = user.Username
			req.Headers["X-Cognito-Email"] = user.Email

			// Process the request with the next handler
			resp, err := next(req)

			// Log authentication result
			if err != nil {
				log.Printf("Error processing authenticated request: %v", err)
			} else {
				log.Printf("Authenticated request completed - Status: %d", resp.StatusCode)
			}

			return resp, err
		}
	}
}

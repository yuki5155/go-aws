package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yuki5155/go-aws/lambda/middlewares"
)

// User represents a user entity
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received get-user request: %+v", request)

	// In a real application, you would fetch the user from a database
	// For this example, we'll return a mock user
	userID := request.PathParameters["id"]
	if userID == "" {
		userID = "default-user-id"
	}

	// If using the Cognito middleware, we can get the user data from the request headers
	cognitoUserID := request.Headers["X-Cognito-User-ID"]
	cognitoUsername := request.Headers["X-Cognito-Username"]
	cognitoEmail := request.Headers["X-Cognito-Email"]

	// If we have Cognito user data, use it instead of the mock data
	if cognitoUserID != "" {
		user := User{
			ID:       cognitoUserID,
			Username: cognitoUsername,
			Email:    cognitoEmail,
		}

		log.Printf("Using Cognito user data: %+v", user)

		// Convert the user to JSON
		userJSON, err := json.Marshal(user)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Error converting user to JSON: %v", err),
			}, nil
		}

		allowOrigin := os.Getenv("ALLOW_ORIGIN")
		if allowOrigin == "" {
			allowOrigin = "https://mydevportal.com" // Default value
		}
		fmt.Println("allowOrigin", allowOrigin)

		return events.APIGatewayProxyResponse{
			Body:       string(userJSON),
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":                     "application/json",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Allow-Origin":      allowOrigin,
			},
		}, nil
	}

	// Fallback to mock user data if no Cognito data
	user := User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Convert the user to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error converting user to JSON: %v", err),
		}, nil
	}

	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowOrigin == "" {
		allowOrigin = "https://mydevportal.com" // Default value
	}
	fmt.Println("allowOrigin", allowOrigin)

	return events.APIGatewayProxyResponse{
		Body:       string(userJSON),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Origin":      allowOrigin,
		},
	}, nil
}

func main() {
	fmt.Println("Starting Get User Lambda")
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
		middlewares.CognitoAuthMiddleware(),
	)
	lambda.Start(handlerWithMiddleware)
}

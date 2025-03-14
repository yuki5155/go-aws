package main

import (
	"log"
	"os"

	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yuki5155/go-aws/lambda/middlewares"
)

// UserModel represents the user data structure
type UserModel struct {
	UserID   string `json:"userid"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received get-user request: %+v", request)

	// In a real application, you would retrieve user data from a database
	// This is a mock implementation for demonstration purposes
	user := UserModel{
		UserID:   "user123",
		Email:    "example@example.com",
		Username: "exampleuser",
	}

	// Convert user model to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling user data: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowOrigin == "" {
		allowOrigin = "https://mydevportal.com" // Default value
	}

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
	log.Println("Starting Get User Lambda")
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
	)
	lambda.Start(handlerWithMiddleware)
}

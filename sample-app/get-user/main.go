package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// CallbackResponse represents the data we'll send to the callback API
type CallbackResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	UserID  string `json:"userId"`
}

// sendCallbackResponse sends a test response to the callback API
func sendCallbackResponse(user User) error {
	callbackURL := "https://apiv2localhost.mydevportal.com/api/callback"

	// Create the callback response payload
	callbackData := CallbackResponse{
		Status:  "success",
		Message: "User data retrieved successfully",
		UserID:  user.ID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		return fmt.Errorf("error marshaling callback data: %v", err)
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating callback request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending callback request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Callback response status: %s", resp.Status)
	return nil
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

	var user User

	// If we have Cognito user data, use it instead of the mock data
	if cognitoUserID != "" {
		user = User{
			ID:       cognitoUserID,
			Username: cognitoUsername,
			Email:    cognitoEmail,
		}
		log.Printf("Using Cognito user data: %+v", user)
	} else {
		// Fallback to mock user data if no Cognito data
		user = User{
			ID:       userID,
			Username: "testuser",
			Email:    "test@example.com",
		}
	}

	// Send the test response to the callback API
	err := sendCallbackResponse(user)
	if err != nil {
		log.Printf("Error sending callback response: %v", err)
		// We'll continue even if the callback fails
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

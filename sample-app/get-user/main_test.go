package main

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	t.Run("Basic test with default user ID", func(t *testing.T) {
		// Create a request with no path parameters
		request := events.APIGatewayProxyRequest{}

		// Call the handler
		response, err := handler(request)
		if err != nil {
			t.Fatalf("Error calling handler: %v", err)
		}

		// Check status code
		if response.StatusCode != 200 {
			t.Fatalf("Expected status code 200, got %d", response.StatusCode)
		}

		// Unmarshal the response body into a User struct
		var user User
		if err := json.Unmarshal([]byte(response.Body), &user); err != nil {
			t.Fatalf("Error unmarshaling response body: %v", err)
		}

		// Verify the user has the default ID
		if user.ID != "default-user-id" {
			t.Fatalf("Expected user ID 'default-user-id', got '%s'", user.ID)
		}
	})

	t.Run("Test with specific user ID", func(t *testing.T) {
		// Create a request with a specific path parameter
		request := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"id": "test-user-123",
			},
		}

		// Call the handler
		response, err := handler(request)
		if err != nil {
			t.Fatalf("Error calling handler: %v", err)
		}

		// Check status code
		if response.StatusCode != 200 {
			t.Fatalf("Expected status code 200, got %d", response.StatusCode)
		}

		// Unmarshal the response body into a User struct
		var user User
		if err := json.Unmarshal([]byte(response.Body), &user); err != nil {
			t.Fatalf("Error unmarshaling response body: %v", err)
		}

		// Verify the user has the requested ID
		if user.ID != "test-user-123" {
			t.Fatalf("Expected user ID 'test-user-123', got '%s'", user.ID)
		}
	})
}

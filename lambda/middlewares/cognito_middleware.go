package middlewares

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
)

// CognitoDummyUser represents user data that would come from Cognito
type CognitoDummyUser struct {
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
			// For now, we're just adding dummy Cognito user data
			log.Printf("Adding dummy Cognito user data to request: %s", req.RequestContext.RequestID)

			// In a real implementation, we would:
			// 1. Extract the JWT token from the Authorization header
			// 2. Validate the token with Cognito
			// 3. Extract user info from the token
			// 4. Add the user info to the request context

			// For now, we'll just add some dummy data to the request context via headers
			// In a real implementation, you'd use context.WithValue, but for Lambda we need to work with the request object

			// We'll store our dummy user data in the request headers for demonstration
			// In a real implementation, you'd want to modify the actual context or create a custom request object
			if req.Headers == nil {
				req.Headers = make(map[string]string)
			}

			// Add dummy user data to headers
			// This is just for demonstration - in a real implementation, you'd use proper context management
			req.Headers["X-Cognito-User-ID"] = "dummy-user-123"
			req.Headers["X-Cognito-Username"] = "dummy-user"
			req.Headers["X-Cognito-Email"] = "dummy@example.com"
			req.Headers["X-Cognito-Groups"] = "admin,users"

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

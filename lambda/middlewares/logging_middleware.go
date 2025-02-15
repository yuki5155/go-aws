package middlewares

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
)

// LoggingMiddleware returns a Middleware that logs requests and responses
func LoggingMiddleware() Middleware {
	log.Println("Creating logging middleware")

	return func(next LambdaHandler) LambdaHandler {
		log.Println("Setting up handler wrapper")

		return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// リクエスト前のログ
			log.Printf("Processing request - ID: %s, Method: %s, Path: %s",
				req.RequestContext.RequestID,
				req.HTTPMethod,
				req.Path,
			)

			resp, err := next(req)

			// レスポンス後のログ
			if err != nil {
				log.Printf("Error processing request: %v", err)
			} else {
				log.Printf("Request completed - Status: %d", resp.StatusCode)
			}

			return resp, err
		}
	}
}

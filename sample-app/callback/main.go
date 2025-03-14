package main

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	lambdaErrors "github.com/yuki5155/go-aws/lambda"
	"github.com/yuki5155/go-aws/lambda/middlewares"
	"github.com/yuki5155/go-aws/sample-app/callback/config"
	"github.com/yuki5155/go-aws/sample-app/callback/service"
)

// handler is the Lambda handler function
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		httpErr := lambdaErrors.NewInternalServerError("Failed to load configuration", err)
		return httpErr.ToAPIGatewayResponse(), err
	}

	// Create service
	svc := service.NewService(cfg)

	// Process request
	return svc.ProcessCallback(request)
}

func main() {
	// Add middleware
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
	)

	// Start the Lambda
	lambda.Start(handlerWithMiddleware)
}

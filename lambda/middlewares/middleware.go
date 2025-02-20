package middlewares

import (
	"github.com/aws/aws-lambda-go/events"
)

// LambdaHandler is a type alias for the standard Lambda handler function
type LambdaHandler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Middleware is a function that wraps a LambdaHandler and returns a new LambdaHandler
type Middleware func(LambdaHandler) LambdaHandler

// Chain applies multiple middleware to a handler in the order they are provided
func Chain(handler LambdaHandler, middlewares ...Middleware) LambdaHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

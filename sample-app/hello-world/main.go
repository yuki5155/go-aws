package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yuki5155/go-aws/lambda/middlewares"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received login request: %+v", request)

	return events.APIGatewayProxyResponse{
		Body:       string("ssss"),
		StatusCode: 200,
	}, nil
}

func main() {
	fmt.Println("Starting Lambda")
	handlerWithMiddleware := middlewares.Chain(
		handler,
		middlewares.LoggingMiddleware(),
	)
	lambda.Start(handlerWithMiddleware)
}

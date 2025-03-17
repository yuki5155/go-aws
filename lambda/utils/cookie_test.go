package utils

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestGetCookieByName(t *testing.T) {
	tests := []struct {
		name          string
		headers       map[string]string
		cookieName    string
		expectedValue string
	}{
		{
			name: "Simple cookie exists",
			headers: map[string]string{
				"Cookie": "testCookie=testValue",
			},
			cookieName:    "testCookie",
			expectedValue: "testValue",
		},
		{
			name: "Cookie does not exist",
			headers: map[string]string{
				"Cookie": "anotherCookie=anotherValue",
			},
			cookieName:    "testCookie",
			expectedValue: "",
		},
		{
			name:          "No cookies in request",
			headers:       map[string]string{},
			cookieName:    "testCookie",
			expectedValue: "",
		},
		{
			name: "Multiple cookies",
			headers: map[string]string{
				"Cookie": "firstCookie=firstValue; testCookie=testValue; lastCookie=lastValue",
			},
			cookieName:    "testCookie",
			expectedValue: "testValue",
		},
		{
			name: "Cookie with lowercase header",
			headers: map[string]string{
				"cookie": "testCookie=testValue",
			},
			cookieName:    "testCookie",
			expectedValue: "testValue",
		},
		{
			name: "Cookie with spaces",
			headers: map[string]string{
				"Cookie": " testCookie = testValue ; otherCookie=otherValue",
			},
			cookieName:    "testCookie",
			expectedValue: "testValue",
		},
		{
			name: "Cookie with empty value",
			headers: map[string]string{
				"Cookie": "testCookie=",
			},
			cookieName:    "testCookie",
			expectedValue: "",
		},
		{
			name: "Cookie with special characters in value",
			headers: map[string]string{
				"Cookie": "testCookie=test@Value#123",
			},
			cookieName:    "testCookie",
			expectedValue: "test@Value#123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := events.APIGatewayProxyRequest{
				Headers: tt.headers,
			}

			result := GetCookieByName(req, tt.cookieName)
			assert.Equal(t, tt.expectedValue, result, "Cookie value should match expected value")
		})
	}
}

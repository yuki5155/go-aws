package utils

import (
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// GetCookieByName extracts a specific cookie from an API Gateway request by name
// Returns the cookie value if found, empty string otherwise
func GetCookieByName(req events.APIGatewayProxyRequest, name string) string {
	// Check if any cookies are present in the request
	cookieHeader, exists := req.Headers["Cookie"]
	if !exists {
		// Also check for lowercase header name, as header names might be case-insensitive
		cookieHeader, exists = req.Headers["cookie"]
		if !exists {
			return ""
		}
	}

	// Split the cookie string into individual cookies
	cookies := strings.Split(cookieHeader, ";")

	// Iterate through cookies to find the requested one
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)

		// Split the cookie into name and value parts
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			cookieName := strings.TrimSpace(parts[0])
			cookieValue := strings.TrimSpace(parts[1])

			// Check if this is the cookie we're looking for
			if cookieName == name {
				return cookieValue
			}
		}
	}

	// Cookie not found
	return ""
}

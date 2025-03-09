package config

import (
	"fmt"
	"os"

	"github.com/yuki5155/go-aws/cognito"
)

// Config holds all the configuration for the Lambda function
type Config struct {
	CookieDomain  string
	AllowOrigin   string
	CallbackURL   string
	CognitoConfig cognito.CognitoConfig
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Cookie domain configuration
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = ".mydevportal.com" // Default value
	}

	// CORS configuration
	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowOrigin == "" {
		allowOrigin = "https://mydevportal.com" // Default value
	}

	// Callback URL is required
	callbackURL := os.Getenv("COGNITO_CALLBACK_URL")
	if callbackURL == "" {
		return nil, fmt.Errorf("missing required environment variable: COGNITO_CALLBACK_URL")
	}

	// Cognito configuration
	cognitoConfig := cognito.CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       os.Getenv("AWS_REGION"),
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
		UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
	}

	// Validate required Cognito configuration
	if cognitoConfig.Domain == "" {
		return nil, fmt.Errorf("missing required environment variable: COGNITO_DOMAIN")
	}
	if cognitoConfig.Region == "" {
		return nil, fmt.Errorf("missing required environment variable: AWS_REGION")
	}
	if cognitoConfig.ClientID == "" {
		return nil, fmt.Errorf("missing required environment variable: COGNITO_CLIENT_ID")
	}

	return &Config{
		CookieDomain:  cookieDomain,
		AllowOrigin:   allowOrigin,
		CallbackURL:   callbackURL,
		CognitoConfig: cognitoConfig,
	}, nil
}

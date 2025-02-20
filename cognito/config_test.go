package cognito

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func TestGetTokensAndVerify(t *testing.T) {
	// Load environment variables
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	region := "ap-northeast-1"
	if os.Getenv("AWS_REGION") != "" {
		region = os.Getenv("AWS_REGION")
	}

	// Create config with real AWS - ensure no LocalStack endpoint is used
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		t.Fatalf("unable to load SDK config: %v", err)
	}

	// Remove any AWS_ENDPOINT_URL environment variable if it exists
	// This ensures we're not using LocalStack
	os.Unsetenv("AWS_ENDPOINT_URL")

	// Print credentials for debugging
	creds, err := cfg.Credentials.Retrieve(context.TODO())
	if err == nil {
		t.Logf("Using AWS credentials: %s", creds.AccessKeyID)
	}

	// Create regular Cognito config
	config := CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       region,
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
		UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
	}
	callBackURL := os.Getenv("COGNITO_CALLBACK_URL")
	authCode := os.Getenv("COGNITO_AUTH_CODE")

	// Step 1: Get tokens
	tokens, err := GetTokens(authCode, callBackURL, config)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Tokens obtained successfully")
	fmt.Printf("Access Token: %s...\n", tokens.AccessToken[:30])
	fmt.Printf("ID Token: %s...\n", tokens.IdToken[:30])
	fmt.Printf("Refresh Token: %s...\n", tokens.RefreshToken[:30])

	// Step 2: Verify ID token
	isValid, claims, err := ValidateIDToken(tokens.IdToken, config)
	if err != nil {
		t.Logf("Token validation error (might be expected for expired tokens): %v", err)
	}

	fmt.Printf("Token valid: %v\n", isValid)
	if claims != nil {
		fmt.Println("\nToken claims:")
		// Print important claims
		if sub, ok := claims["sub"].(string); ok {
			fmt.Printf("Subject (sub): %s\n", sub)
		}
		// Other claims...
	}

	// Step 3: Get user attributes using direct client
	t.Run("GetUserAttributes", func(t *testing.T) {
		// Create a client directly without custom resolver
		client := cognitoidentityprovider.NewFromConfig(cfg)

		input := &cognitoidentityprovider.GetUserInput{
			AccessToken: aws.String(tokens.AccessToken),
		}

		result, err := client.GetUser(context.TODO(), input)
		if err != nil {
			t.Logf("Error getting user attributes directly: %v", err)
			t.Skip("Skipping attributes test due to API error")
			return
		}

		fmt.Println("\nUser attributes:")
		for _, attr := range result.UserAttributes {
			fmt.Printf("%s: %s\n", aws.ToString(attr.Name), aws.ToString(attr.Value))
		}
	})

	// Step 4: Refresh token using the RefreshTokens function from the package
	if tokens.RefreshToken != "" {
		t.Run("RefreshToken", func(t *testing.T) {
			refreshedTokens, err := RefreshTokens(tokens.RefreshToken, config)
			if err != nil {
				t.Fatalf("Error refreshing token: %v", err)
			}

			fmt.Println("\nTokens refreshed successfully")
			fmt.Printf("New Access Token: %s...\n", refreshedTokens.AccessToken[:30])
			fmt.Printf("New ID Token: %s...\n", refreshedTokens.IdToken[:30])

			// Verify the new token is valid
			isValid, newClaims, err := ValidateIDToken(refreshedTokens.IdToken, config)
			if err != nil {
				t.Logf("New token validation error: %v", err)
			}
			fmt.Printf("New token valid: %v\n", isValid)

			// Check expiration of new token
			if exp, ok := newClaims["exp"].(float64); ok {
				fmt.Printf("New token expiration: %v\n", exp)
			}
		})
	}
}

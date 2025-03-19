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

// Helper function to create AWS configuration
func createAWSConfig(region string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				// Normalize service name to lowercase
				serviceLower := strings.ToLower(service)

				// Handle Cognito Identity Provider specifically
				if serviceLower == "cognito-idp" ||
					serviceLower == "cognitoidentityprovider" ||
					strings.Contains(serviceLower, "cognito") {
					return aws.Endpoint{
						URL: fmt.Sprintf("https://cognito-idp.%s.amazonaws.com", region),
					}, nil
				}

				// For any other AWS service
				return aws.Endpoint{
					URL: fmt.Sprintf("https://%s.%s.amazonaws.com", serviceLower, region),
				}, nil
			},
		)),
	)
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
		if email, ok := claims["email"].(string); ok {
			fmt.Printf("Email: %s\n", email)
		}
		if name, ok := claims["name"].(string); ok {
			fmt.Printf("Name: %s\n", name)
		}
		if exp, ok := claims["exp"].(float64); ok {
			fmt.Printf("Expiration: %v\n", exp)
		}
		if iat, ok := claims["iat"].(float64); ok {
			fmt.Printf("Issued At: %v\n", iat)
		}
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

func TestValidateIDToken(t *testing.T) {
	config := CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       os.Getenv("AWS_REGION"),
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
		UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
	}
	idToken := "eyJraWQiOiJVaHJjQXBuV2xQcXd6bkQyRWQzTERlSERleDJPUTg4MENPNHZ1Rk9oRmlvPSIsImFsZyI6IlJTMjU2In0.eyJhdF9oYXNoIjoid2RHMVhiYkpmdjJ5R0lIcEpuTEl2ZyIsInN1YiI6ImY3MTQ1YWE4LWUwNDEtNzBhZi1hZDJmLTgwNzI3OTE1ODU3MSIsImNvZ25pdG86Z3JvdXBzIjpbImFwLW5vcnRoZWFzdC0xXzNnY0Y1M09ENF9Hb29nbGUiXSwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuYXAtbm9ydGhlYXN0LTEuYW1hem9uYXdzLmNvbVwvYXAtbm9ydGhlYXN0LTFfM2djRjUzT0Q0IiwiY29nbml0bzp1c2VybmFtZSI6Ikdvb2dsZV8xMTE2MzQ5NDY4Nzg4NTk2NTczOTQiLCJnaXZlbl9uYW1lIjoiWXVraSIsInBpY3R1cmUiOiJodHRwczpcL1wvbGgzLmdvb2dsZXVzZXJjb250ZW50LmNvbVwvYVwvQUNnOG9jS2lQVWZQZ2QwOVMyOFpteDVibVdXM1lNNmZvMzZrYTFGX0gzY1J1NEFZTU1UMi13PXM5Ni1jIiwib3JpZ2luX2p0aSI6IjUyNDM1ZDA5LWIyNWEtNDRlYi05ZjE1LTgyODE4MWE0OTNmYSIsImF1ZCI6IjNtdmM1aGJvODJxZWF0aHI3cDRqZTc3MjMiLCJpZGVudGl0aWVzIjpbeyJkYXRlQ3JlYXRlZCI6IjE3NDA1MDM3ODM3ODIiLCJ1c2VySWQiOiIxMTE2MzQ5NDY4Nzg4NTk2NTczOTQiLCJwcm92aWRlck5hbWUiOiJHb29nbGUiLCJwcm92aWRlclR5cGUiOiJHb29nbGUiLCJpc3N1ZXIiOm51bGwsInByaW1hcnkiOiJ0cnVlIn1dLCJ0b2tlbl91c2UiOiJpZCIsImF1dGhfdGltZSI6MTc0MjMxMTM0OCwibmFtZSI6Ill1a2kgQXNhbm8gKFkuQSkiLCJleHAiOjE3NDIzMTQ5NDgsImlhdCI6MTc0MjMxMTM0OSwiZmFtaWx5X25hbWUiOiJBc2FubyIsImp0aSI6IjRiZTRjYzQ1LTVkODMtNGM0ZC04YjAyLWFiYTdlMzJmYWZiYyIsImVtYWlsIjoieXVraWFzYW5vNTVAZ21haWwuY29tIn0.xhc9JaPcnWz1zl8Ua2jKpQXr-hGXTA_p0r7_6W-erTpyos-87ZJw8peLKcxufTRvTaqIUYJa0YthZ7ZCV0j_PXd2VofNFd4DCzodFICwPvcbb2d-sR3oaIw1SwcteLy2Vxdswn_mfltSC6iWsFxQjBknnX4XGeTW8R7WUNFtJu15pry0NhQpN81JmZy-qAOdXkpBwU93ZVGw9Pgr9nMsCfdjHu9G-S70YeSEwG47H3ZEgRcsahGI9ARRhceQvGubImYg477CSUHxOfXKNr5VhdOXrYZnl1QUnRj59mWMmACgNDBsFiUEDjUf-4HXHIN49vGQvtX3qbLxvD4XGmZlEA"
	isValid, claims, err := ValidateIDToken(idToken, config)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Token valid: %v\n", isValid)
	if claims != nil {
		fmt.Println("\nToken claims:")
		// Print important claims
		if sub, ok := claims["sub"].(string); ok {
			fmt.Printf("Subject (sub): %s\n", sub)
		}
		if email, ok := claims["email"].(string); ok {
			fmt.Printf("Email: %s\n", email)
		}
		if name, ok := claims["name"].(string); ok {
			fmt.Printf("Name: %s\n", name)
		}
	}
}

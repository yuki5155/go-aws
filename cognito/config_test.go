package cognito

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
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

	config := CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       "ap-northeast-1",
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: "",                                // Optional, only if needed
		UserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"), // Optional
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
		if iss, ok := claims["iss"].(string); ok {
			fmt.Printf("Issuer (iss): %s\n", iss)
		}
		if aud, ok := claims["aud"].(string); ok {
			fmt.Printf("Audience (aud): %s\n", aud)
		}
		if exp, ok := claims["exp"].(float64); ok {
			fmt.Printf("Expiration (exp): %v\n", exp)
		}
		if tokenUse, ok := claims["token_use"].(string); ok {
			fmt.Printf("Token use: %s\n", tokenUse)
		}

		// Check if this is a Google federated login
		if idp, ok := claims["identities"].(string); ok && idp != "" {
			fmt.Printf("Identity Provider: %s\n", idp)
		}
	}

	// Step 3: Get user attributes using the access token
	attributes, err := GetUserAttributes(tokens.AccessToken, config)
	if err != nil {
		t.Logf("Error getting user attributes: %v", err)
	} else {
		fmt.Println("\nUser attributes:")
		for key, value := range attributes {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// Only run refresh token test if token is invalid (likely expired)
	// if !isValid && tokens.RefreshToken != "" {
	// 	t.Run("RefreshExpiredToken", func(t *testing.T) {
	// 		TestRefreshToken(t, tokens.RefreshToken, config)
	// 	})
	// }
}

// func TestRefreshToken(t *testing.T, refreshToken string, config CognitoConfig) {
// 	// If refresh token is not provided as parameter, try to get it from env
// 	if refreshToken == "" {
// 		if err := loadEnv("../.env"); err != nil {
// 			t.Fatal(err)
// 		}
// 		refreshToken = os.Getenv("COGNITO_REFRESH_TOKEN")
// 		if refreshToken == "" {
// 			t.Skip("Skipping refresh token test - no refresh token available")
// 		}

// 		config = CognitoConfig{
// 			Domain:       os.Getenv("COGNITO_DOMAIN"),
// 			Region:       "ap-northeast-1",
// 			ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
// 			ClientSecret: "", // Optional, only if needed
// 		}
// 	}

// 	fmt.Println("\nAttempting to refresh tokens...")
// 	newTokens, err := RefreshTokens(refreshToken, config)
// 	if err != nil {
// 		t.Fatalf("Failed to refresh tokens: %v", err)
// 	}

// 	fmt.Println("Tokens refreshed successfully")
// 	fmt.Printf("New Access Token: %s...\n", newTokens.AccessToken[:30])
// 	fmt.Printf("New ID Token: %s...\n", newTokens.IdToken[:30])

// 	// Verify the new ID token
// 	isValid, claims, err := ValidateIDToken(newTokens.IdToken, config)
// 	if err != nil {
// 		t.Logf("New token validation error: %v", err)
// 	}

// 	fmt.Printf("New token valid: %v\n", isValid)
// 	if exp, ok := claims["exp"].(float64); ok {
// 		fmt.Printf("New token expiration: %v\n", exp)
// 	}
// }

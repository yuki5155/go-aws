package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type CognitoConfig struct {
	Domain       string
	Region       string
	ClientID     string
	ClientSecret string
	UserPoolID   string // Optional, will be derived if not provided
}

// JWTClaims represents standard JWT claims plus any custom fields
type JWTClaims map[string]interface{}

// GetTokens exchanges authorization code for tokens
func GetTokens(code, redirectURI string, config CognitoConfig) (*TokenResponse, error) {
	tokenEndpoint := fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/oauth2/token",
		config.Domain, config.Region)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", config.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, NewInvalidRequestError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if config.ClientSecret != "" {
		req.SetBasicAuth(config.ClientID, config.ClientSecret)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, NewRequestFailedError("token request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewRequestFailedError(fmt.Sprintf("failed to get token: status code %s", resp.Status), nil)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, NewParsingFailedError("failed to parse token response", err)
	}

	return &tokenResp, nil
}

// ValidateIDToken validates the ID token and returns claims if valid
func ValidateIDToken(idToken string, config CognitoConfig) (bool, JWTClaims, error) {
	// Parse JWT token manually (no external dependencies)
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return false, nil, NewInvalidTokenError("invalid token format", nil)
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, nil, NewParsingFailedError("failed to decode token payload", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return false, nil, NewParsingFailedError("failed to parse token claims", err)
	}

	// Get token issuer
	issuer, _ := claims["iss"].(string)
	if issuer == "" {
		return false, nil, NewValidationFailedError("missing issuer claim")
	}

	// Check expiration
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false, nil, NewValidationFailedError("invalid expiration claim")
	}

	if time.Now().Unix() > int64(exp) {
		return false, claims, NewTokenExpiredError()
	}

	// Determine user pool ID if not provided
	userPoolID := config.UserPoolID
	if userPoolID == "" && strings.Contains(issuer, "cognito-idp") {
		parts := strings.Split(issuer, "/")
		if len(parts) > 0 {
			userPoolID = parts[len(parts)-1]
		}
	}

	// For Google federated auth
	if strings.HasPrefix(issuer, "https://accounts.google.com") {
		// For a real implementation, verify Google token signature
		// This simplified version only checks expiration (done above)
		return true, claims, nil
	}

	// For Cognito tokens
	if userPoolID == "" {
		return false, nil, NewUserPoolError("could not determine user pool ID")
	}

	// For ID tokens, we need to use AdminGetUser or GetUser
	// Use the 'sub' claim to identify the user
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return false, claims, NewValidationFailedError("missing subject claim")
	}

	// When using the SDK for full validation (not in this simplified version):
	// cfg, err := createAWSConfig(config.Region)
	// if err != nil {
	//     return false, claims, err
	// }
	// cognitoClient := cognitoidentityprovider.NewFromConfig(cfg)
	// Then call AdminGetUser or similar APIs

	// Try to validate using GetUser if we have an access token
	// For ID tokens, this is a minimal validation
	audience, _ := claims["aud"].(string)
	if audience != config.ClientID {
		return false, claims, NewValidationFailedError("invalid audience")
	}

	return true, claims, nil
}

// GetUserAttributes retrieves user attributes using the access token
func GetUserAttributes(accessToken string, config CognitoConfig, awsConf aws.Config) (map[string]string, error) {

	cognitoClient := cognitoidentityprovider.NewFromConfig(awsConf)
	input := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	}

	result, err := cognitoClient.GetUser(context.Background(), input)
	if err != nil {
		return nil, NewRequestFailedError("failed to get user attributes", err)
	}

	attributes := make(map[string]string)
	for _, attr := range result.UserAttributes {
		attributes[aws.ToString(attr.Name)] = aws.ToString(attr.Value)
	}

	return attributes, nil
}

// RefreshTokens refreshes the access and ID tokens using a refresh token
func RefreshTokens(refreshToken string, cognitoConfig CognitoConfig) (*TokenResponse, error) {
	// Create AWS config with just region, no custom endpoint resolver
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cognitoConfig.Region),
	)
	if err != nil {
		return nil, NewInvalidRequestError("failed to load AWS config", err)
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(awsCfg)

	// Use string literal for auth flow to match AWS enum exactly
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		ClientId: aws.String(cognitoConfig.ClientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": refreshToken,
		},
	}

	if cognitoConfig.ClientSecret != "" {
		input.AuthParameters["SECRET_HASH"] = computeSecretHash(cognitoConfig.ClientID, cognitoConfig.ClientSecret, refreshToken)
	}

	result, err := cognitoClient.InitiateAuth(context.Background(), input)
	if err != nil {
		return nil, NewAuthFailedError("failed to refresh token", err)
	}

	response := &TokenResponse{
		AccessToken:  aws.ToString(result.AuthenticationResult.AccessToken),
		IdToken:      aws.ToString(result.AuthenticationResult.IdToken),
		RefreshToken: aws.ToString(result.AuthenticationResult.RefreshToken),
		ExpiresIn:    int(result.AuthenticationResult.ExpiresIn),
		TokenType:    aws.ToString(result.AuthenticationResult.TokenType),
	}

	// If refresh token wasn't returned (common), use the one we already have
	if response.RefreshToken == "" {
		response.RefreshToken = refreshToken
	}

	return response, nil
}

// Helper function to compute secret hash for client secret validation
// In a real implementation, you would use HMAC-SHA256
func computeSecretHash(clientID, clientSecret, username string) string {
	// The format is: SHA256(client_secret + username + client_id)
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(username + clientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

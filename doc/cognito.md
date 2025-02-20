## Code Examples

### Configuration

Create a configuration object with your Cognito settings:

```go
cognitoConfig := cognito.CognitoConfig{
    Domain:       "your-domain-prefix",
    Region:       "us-east-1",
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret", // Optional
    UserPoolID:   "your-user-pool-id",  // Optional, will be derived if not provided
}
```

### Get Tokens with Authorization Code

After a user authenticates via the Cognito hosted UI, exchange the authorization code for tokens:

```go
// The code received from Cognito redirect after user login
authCode := "auth-code-from-redirect"
// Your configured callback URL (must match what's in Cognito)
redirectURI := "https://your-app.com/callback"

tokens, err := cognito.GetTokens(authCode, redirectURI, cognitoConfig)
if err != nil {
    log.Fatalf("Failed to get tokens: %v", err)
}

// Use the tokens
fmt.Printf("Access Token: %s\n", tokens.AccessToken)
fmt.Printf("ID Token: %s\n", tokens.IdToken)
fmt.Printf("Refresh Token: %s\n", tokens.RefreshToken)
```

### Validate ID Token

Verify that an ID token is valid and extract its claims:

```go
isValid, claims, err := cognito.ValidateIDToken(tokens.IdToken, cognitoConfig)
if err != nil {
    log.Printf("Token validation error: %v", err)
}

if isValid {
    // Access token claims
    if sub, ok := claims["sub"].(string); ok {
        fmt.Printf("User ID: %s\n", sub)
    }
    if email, ok := claims["email"].(string); ok {
        fmt.Printf("Email: %s\n", email)
    }
}
```

### Get User Attributes

Retrieve detailed user information using an access token:

```go
// Create AWS configuration first
awsCfg, err := config.LoadDefaultConfig(context.Background(),
    config.WithRegion(cognitoConfig.Region),
)
if err != nil {
    log.Fatalf("Failed to load AWS configuration: %v", err)
}

attributes, err := cognito.GetUserAttributes(tokens.AccessToken, cognitoConfig, awsCfg)
if err != nil {
    log.Fatalf("Failed to get user attributes: %v", err)
}

// Access user attributes
for name, value := range attributes {
    fmt.Printf("%s: %s\n", name, value)
}
```

### Refresh Tokens

When tokens expire, use the refresh token to get new ones:

```go
refreshedTokens, err := cognito.RefreshTokens(tokens.RefreshToken, cognitoConfig)
if err != nil {
    log.Fatalf("Failed to refresh tokens: %v", err)
}

fmt.Printf("New Access Token: %s\n", refreshedTokens.AccessToken)
fmt.Printf("New ID Token: %s\n", refreshedTokens.IdToken)
```

## Advanced Usage

### Working with JWT Claims

The package provides a `JWTClaims` type for working with token claims:

```go
// Access specific claims
if groups, ok := claims["cognito:groups"].([]interface{}); ok {
    fmt.Println("User groups:")
    for _, group := range groups {
        fmt.Printf("- %s\n", group)
    }
}

// Check custom claims
if customValue, ok := claims["custom:attribute"].(string); ok {
    fmt.Printf("Custom attribute: %s\n", customValue)
}
```

### Handling Federated Authentication

The package supports tokens from federated identity providers like Google:

```go
// For Google-issued tokens, the package will check the issuer claim
isValid, claims, err := cognito.ValidateIDToken(googleIssuedToken, cognitoConfig)
```

## Error Handling

The package returns detailed error messages. Common errors include:

- Invalid or expired tokens
- Network issues when communicating with AWS
- Incorrect client credentials
- Invalid authorization codes

Always check the error returned by each function:

```go
tokens, err := cognito.GetTokens(code, redirectURI, cognitoConfig)
if err != nil {
    // Check for specific error conditions
    if strings.Contains(err.Error(), "expired") {
        // Handle expired authorization code
    } else if strings.Contains(err.Error(), "invalid_grant") {
        // Handle invalid authorization code
    } else {
        // Handle other errors
    }
}
```

## Security Considerations

- Store client secrets securely
- Always use HTTPS for redirects
- Keep refresh tokens secure
- Consider token expiration when designing your authentication flow
- Validate tokens before trusting their contents

## Complete Example

Here's a complete example of an authentication flow:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/your-username/cognito"
)

func main() {
    // 1. Set up configuration
    cognitoConfig := cognito.CognitoConfig{
        Domain:       "your-domain",
        Region:       "us-east-1",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
    }
    
    // 2. Exchange authorization code for tokens
    tokens, err := cognito.GetTokens("auth-code", "https://your-app/callback", cognitoConfig)
    if err != nil {
        log.Fatalf("Failed to get tokens: %v", err)
    }
    
    // 3. Validate ID token
    isValid, claims, err := cognito.ValidateIDToken(tokens.IdToken, cognitoConfig)
    if err != nil {
        log.Printf("Warning: %v", err)
    }
    
    if isValid {
        fmt.Println("Token is valid!")
    }
    
    // 4. Get user attributes
    awsCfg, _ := config.LoadDefaultConfig(context.Background(), 
        config.WithRegion(cognitoConfig.Region),
    )
    
    attributes, err := cognito.GetUserAttributes(tokens.AccessToken, cognitoConfig, awsCfg)
    if err == nil {
        fmt.Println("User attributes:", attributes)
    }
    
    // 5. Later, refresh the tokens when they expire
    newTokens, err := cognito.RefreshTokens(tokens.RefreshToken, cognitoConfig)
    if err != nil {
        log.Fatalf("Failed to refresh tokens: %v", err)
    }
    
    fmt.Println("Successfully refreshed tokens!")
}
```
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
func TestGetTokens(t *testing.T) {

	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	config := CognitoConfig{
		Domain:       os.Getenv("COGNITO_DOMAIN"),
		Region:       "ap-northeast-1",
		ClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		ClientSecret: "", // Optional, only if needed
	}
	callBackURL := os.Getenv("COGNITO_CALLBACK_URL")
	authCode := os.Getenv("COGNITO_AUTH_CODE")
	tokens, err := GetTokens(authCode, callBackURL, config)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tokens)
}

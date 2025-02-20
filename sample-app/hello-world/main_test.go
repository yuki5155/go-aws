package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedError  bool // エラーの有無のみをチェック
		expectedStatus int
	}{
		{
			name:           "Unable to get IP",
			serverHandler:  nil,
			expectedError:  true, // エラーが発生することのみを期待
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Non 200 Response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError:  true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Successful Request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "127.0.0.1")
			},
			expectedError:  false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストサーバーのセットアップ
			var ts *httptest.Server
			if tt.serverHandler != nil {
				ts = httptest.NewServer(tt.serverHandler)
				defer ts.Close()
				DefaultHTTPGetAddress = ts.URL
			} else {
				DefaultHTTPGetAddress = "http://127.0.0.1:12345"
			}

			// ハンドラーの実行
			response, err := handler(events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
			})

			// エラーチェック
			if tt.expectedError {
				if err == nil {
					t.Error("expected an error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if response.StatusCode != tt.expectedStatus {
					t.Errorf("expected status code %d, got %d", tt.expectedStatus, response.StatusCode)
				}
				if response.Body == "" {
					t.Error("expected non-empty body")
				}
			}
		})
	}
}

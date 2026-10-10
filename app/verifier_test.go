package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifierParsing(t *testing.T) {
	tests := []struct {
		name        string
		apiResponse string
		wantErr     bool
		wantMatch   bool
		wantOutdoors bool
	}{
		{
			name: "valid json",
			apiResponse: `{"choices": [{"message": {"content": "{\"match\": true, \"outdoors\": true, \"reason\": \"good\"}"}}]}`,
			wantErr: false,
			wantMatch: true,
			wantOutdoors: true,
		},
		{
			name: "markdown json block",
			apiResponse: "{\"choices\": [{\"message\": {\"content\": \"```json\\n{\\\"match\\\": false, \\\"outdoors\\\": true, \\\"reason\\\": \\\"bad\\\"}\\n```\"}}]}",
			wantErr: false,
			wantMatch: false,
			wantOutdoors: true,
		},
		{
			name: "thought tags stripped",
			apiResponse: `{"choices": [{"message": {"content": "<thought>thinking...</thought>{\"match\": true, \"outdoors\": false, \"reason\": \"ok\"}"}}]}`,
			wantErr: false,
			wantMatch: true,
			wantOutdoors: false,
		},
		{
			name: "unclosed thought tag",
			apiResponse: `{"choices": [{"message": {"content": "<thought>thinking...{\"match\": true, \"outdoors\": false, \"reason\": \"ok\"}"}}]}`,
			wantErr: true,
		},
		{
			name: "malformed json",
			apiResponse: `{"choices": [{"message": {"content": "{bad json}"}}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.apiResponse))
			}))
			defer server.Close()

			verifier := NewRealVerifier(server.URL, "dummy", "model", 1)
			verdict, err := verifier.Verify(context.Background(), []byte("fake"), "image/jpeg", "task")
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if verdict.Match != tt.wantMatch || verdict.Outdoors != tt.wantOutdoors {
				t.Errorf("expected %v %v, got %v %v", tt.wantMatch, tt.wantOutdoors, verdict.Match, verdict.Outdoors)
			}
		})
	}
}

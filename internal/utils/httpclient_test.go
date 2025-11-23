package utils

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLLMHttpClient(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"Content-Type":  "application/json",
	}

	client := NewLLMHttpClient(headers)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		for k, v := range headers {
			assert.Equal(t, v, r.Header.Get(k), "expected header %s to be %s", k, v)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Make a request
	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err, "failed to create request")

	resp, err := client.Do(req)
	assert.NoError(t, err, "request failed")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status code %d", http.StatusOK)
}

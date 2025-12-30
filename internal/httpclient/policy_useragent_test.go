package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewUserAgentPolicy(t *testing.T) {
	policy := NewUserAgentPolicy("")
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.userAgent != "azhexgate-client/1.0" {
		t.Errorf("Expected default userAgent 'azhexgate-client/1.0', got '%s'", policy.userAgent)
	}
}

func TestNewUserAgentPolicyCustom(t *testing.T) {
	policy := NewUserAgentPolicy("custom-agent/2.0")
	if policy.userAgent != "custom-agent/2.0" {
		t.Errorf("Expected userAgent 'custom-agent/2.0', got '%s'", policy.userAgent)
	}
}

func TestUserAgentPolicyDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header is set
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "test-agent/1.0" {
			t.Errorf("Expected User-Agent 'test-agent/1.0', got '%s'", userAgent)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewUserAgentPolicy("test-agent/1.0")
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	}

	resp, err := policy.Do(req, next)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestUserAgentPolicyDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "azhexgate-client/1.0" {
			t.Errorf("Expected default User-Agent 'azhexgate-client/1.0', got '%s'", userAgent)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewUserAgentPolicy("")
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	}

	resp, err := policy.Do(req, next)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
}

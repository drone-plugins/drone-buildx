package artifactory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testSubjectToken = "subject-token-that-must-stay-secret"
	testAccessToken  = "access-token-that-must-stay-secret"
)

func TestExchangeSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		projectKey string
		wantFields map[string]string
	}{
		{
			name:       "with project key",
			projectKey: "project",
			wantFields: map[string]string{
				"grant_type":         tokenExchangeGrantType,
				"subject_token_type": idTokenType,
				"subject_token":      testSubjectToken,
				"provider_name":      "provider",
				"project_key":        "project",
			},
		},
		{
			name: "without project key",
			wantFields: map[string]string{
				"grant_type":         tokenExchangeGrantType,
				"subject_token_type": idTokenType,
				"subject_token":      testSubjectToken,
				"provider_name":      "provider",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewTLSServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost {
						t.Errorf("method = %s, want POST", r.Method)
					}
					if got := r.Header.Get("Content-Type"); got != "application/json" {
						t.Errorf("Content-Type = %q, want application/json", got)
					}
					var got map[string]string
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Errorf("decode request: %v", err)
					}
					if len(got) != len(tt.wantFields) {
						t.Errorf(
							"request field count = %d, want %d: %#v",
							len(got),
							len(tt.wantFields),
							got,
						)
					}
					for key, want := range tt.wantFields {
						if got[key] != want {
							t.Errorf("request[%q] = %q, want %q", key, got[key], want)
						}
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(
						`{"username":"oidc-user","access_token":"` +
							testAccessToken +
							`","expires_in":3600}`,
					))
				},
			))
			defer server.Close()

			got, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
				context.Background(),
				server.URL,
				testSubjectToken,
				"provider",
				tt.projectKey,
			)
			if err != nil {
				t.Fatalf("Exchange() error = %v", err)
			}
			want := Credential{
				Username:    "oidc-user",
				AccessToken: testAccessToken,
				ExpiresIn:   3600,
			}
			if got != want {
				t.Fatalf("Exchange() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestExchangeRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	tests := []string{
		"http://jfrog.example.com/access/api/v1/oidc/token",
		"https://user:secret@jfrog.example.com/access/api/v1/oidc/token",
		"https://jfrog.example.com/access/api/v1/oidc/token?secret=value",
		"https://jfrog.example.com/access/api/v1/oidc/token#fragment",
		"https://jfrog.example.com:/access/api/v1/oidc/token",
		" https://jfrog.example.com/access/api/v1/oidc/token",
	}
	for _, endpoint := range tests {
		endpoint := endpoint
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()
			_, err := (ExchangeClient{HTTPClient: http.DefaultClient}).Exchange(
				context.Background(),
				endpoint,
				testSubjectToken,
				"provider",
				"",
			)
			assertExchangeErrorRedacted(
				t,
				err,
				testSubjectToken,
				testAccessToken,
				"secret=value",
			)
		})
	}
}

func TestExchangeHTTPStatusErrorsAreRedacted(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
	} {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			responseSecret := "complete-response-body-must-stay-secret"
			server := httptest.NewTLSServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(status)
					_, _ = w.Write([]byte(responseSecret))
				},
			))
			defer server.Close()

			_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
				context.Background(),
				server.URL,
				testSubjectToken,
				"provider",
				"",
			)
			assertExchangeErrorRedacted(
				t,
				err,
				testSubjectToken,
				testAccessToken,
				responseSecret,
			)
			if !strings.Contains(err.Error(), http.StatusText(status)) ||
				!strings.Contains(err.Error(), "status") {
				t.Fatalf("error %q does not identify status category", err)
			}
		})
	}
}

func TestExchangeRejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"username":`},
		{name: "empty body", body: ""},
		{
			name: "missing username",
			body: `{"access_token":"token","expires_in":1}`,
		},
		{
			name: "empty username",
			body: `{"username":" ","access_token":"token","expires_in":1}`,
		},
		{
			name: "missing access token",
			body: `{"username":"user","expires_in":1}`,
		},
		{
			name: "empty access token",
			body: `{"username":"user","access_token":" ","expires_in":1}`,
		},
		{
			name: "missing expires in",
			body: `{"username":"user","access_token":"token"}`,
		},
		{
			name: "zero expires in",
			body: `{"username":"user","access_token":"token","expires_in":0}`,
		},
		{
			name: "negative expires in",
			body: `{"username":"user","access_token":"token","expires_in":-1}`,
		},
		{
			name: "malformed expires in",
			body: `{"username":"user","access_token":"token","expires_in":"3600"}`,
		},
		{
			name: "trailing JSON",
			body: `{"username":"user","access_token":"token","expires_in":1} {}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewTLSServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(tt.body))
				},
			))
			defer server.Close()

			_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
				context.Background(),
				server.URL,
				testSubjectToken,
				"provider",
				"",
			)
			assertExchangeErrorRedacted(
				t,
				err,
				testSubjectToken,
				testAccessToken,
				tt.body,
			)
		})
	}
}

func TestExchangeRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	body := `{"username":"user","access_token":"` +
		testAccessToken +
		`","expires_in":1,"padding":"` +
		strings.Repeat("x", maxExchangeResponseBytes) +
		`"}`
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		},
	))
	defer server.Close()

	_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
		context.Background(),
		server.URL,
		testSubjectToken,
		"provider",
		"",
	)
	assertExchangeErrorRedacted(
		t,
		err,
		testSubjectToken,
		testAccessToken,
		body,
	)
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error = %q, want oversized response category", err)
	}
}

func TestExchangeHonorsClientTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {
			time.Sleep(100 * time.Millisecond)
		},
	))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 10 * time.Millisecond
	_, err := (ExchangeClient{HTTPClient: httpClient}).Exchange(
		context.Background(),
		server.URL,
		testSubjectToken,
		"provider",
		"",
	)
	assertExchangeErrorRedacted(t, err, testSubjectToken, testAccessToken)
	if !errors.Is(err, context.DeadlineExceeded) &&
		!strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestExchangeHonorsCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {
			time.Sleep(100 * time.Millisecond)
		},
	))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
		ctx,
		server.URL,
		testSubjectToken,
		"provider",
		"",
	)
	assertExchangeErrorRedacted(t, err, testSubjectToken, testAccessToken)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func TestExchangeAllowsSameOriginHTTPSRedirect(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(
					w,
					r,
					server.URL+"/final",
					http.StatusTemporaryRedirect,
				)
				return
			}
			_, _ = w.Write([]byte(
				`{"username":"user","access_token":"token","expires_in":3600}`,
			))
		},
	))
	defer server.Close()

	_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
		context.Background(),
		server.URL+"/redirect",
		testSubjectToken,
		"provider",
		"",
	)
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
}

func TestExchangeRejectsUnsafeRedirects(t *testing.T) {
	t.Parallel()

	crossOriginTarget := httptest.NewTLSServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {
			t.Error("cross-origin redirect target was called")
		},
	))
	defer crossOriginTarget.Close()

	tests := []struct {
		name     string
		location func(sourceURL string) string
	}{
		{
			name: "cross origin",
			location: func(string) string {
				return crossOriginTarget.URL
			},
		},
		{
			name: "HTTP",
			location: func(sourceURL string) string {
				return strings.Replace(sourceURL, "https://", "http://", 1)
			},
		},
		{
			name: "credentials",
			location: func(sourceURL string) string {
				return strings.Replace(sourceURL, "https://", "https://user:secret@", 1)
			},
		},
		{
			name: "query",
			location: func(sourceURL string) string {
				return sourceURL + "?secret=value"
			},
		},
		{
			name: "fragment",
			location: func(sourceURL string) string {
				return sourceURL + "#secret"
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewTLSServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(
						w,
						r,
						tt.location(server.URL+"/final"),
						http.StatusTemporaryRedirect,
					)
				},
			))
			defer server.Close()

			client := server.Client()
			if tt.name == "cross origin" {
				client.Transport = trustBothServersTransport(
					t,
					server,
					crossOriginTarget,
				)
			}
			_, err := (ExchangeClient{HTTPClient: client}).Exchange(
				context.Background(),
				server.URL,
				testSubjectToken,
				"provider",
				"",
			)
			assertExchangeErrorRedacted(
				t,
				err,
				testSubjectToken,
				testAccessToken,
				"secret",
			)
			if !strings.Contains(err.Error(), "redirect") {
				t.Fatalf("error = %v, want redirect category", err)
			}
		})
	}
}

func TestExchangeRejectsExcessiveRedirects(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			index, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
			http.Redirect(
				w,
				r,
				fmt.Sprintf("%s/%d", server.URL, index+1),
				http.StatusTemporaryRedirect,
			)
		},
	))
	defer server.Close()

	_, err := (ExchangeClient{HTTPClient: server.Client()}).Exchange(
		context.Background(),
		server.URL+"/0",
		testSubjectToken,
		"provider",
		"",
	)
	assertExchangeErrorRedacted(t, err, testSubjectToken, testAccessToken)
	if !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("error = %v, want redirect category", err)
	}
	if got := requests.Load(); got > maxRedirects {
		t.Fatalf("request count = %d, want at most %d", got, maxRedirects)
	}
}

func trustBothServersTransport(
	t *testing.T,
	servers ...*httptest.Server,
) http.RoundTripper {
	t.Helper()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig =
		servers[0].Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	for _, server := range servers[1:] {
		transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())
	}
	return transport
}

func assertExchangeErrorRedacted(
	t *testing.T,
	err error,
	secrets ...string,
) {
	t.Helper()
	if err == nil {
		t.Fatal("Exchange() error = nil, want error")
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(err.Error(), secret) {
			t.Fatalf("error exposes secret %q: %v", secret, err)
		}
	}
}

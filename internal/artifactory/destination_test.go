package artifactory

import (
	"strings"
	"testing"
)

func TestResolveTokenEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		server  string
		want    string
		wantErr bool
	}{
		{
			name:   "host only",
			server: "https://jfrog.example.com",
			want:   "https://jfrog.example.com/access/api/v1/oidc/token",
		},
		{
			name:   "root slash",
			server: "https://jfrog.example.com/",
			want:   "https://jfrog.example.com/access/api/v1/oidc/token",
		},
		{
			name:   "artifactory",
			server: "https://jfrog.example.com/artifactory",
			want:   "https://jfrog.example.com/access/api/v1/oidc/token",
		},
		{
			name:   "artifactory slash",
			server: "https://jfrog.example.com/artifactory/",
			want:   "https://jfrog.example.com/access/api/v1/oidc/token",
		},
		{
			name:   "explicit port",
			server: "https://jfrog.example.com:8443/artifactory",
			want:   "https://jfrog.example.com:8443/access/api/v1/oidc/token",
		},
		{name: "malformed", server: "://bad", wantErr: true},
		{name: "http", server: "http://jfrog.example.com", wantErr: true},
		{
			name:    "userinfo",
			server:  "https://user:secret@jfrog.example.com",
			wantErr: true,
		},
		{
			name:    "query",
			server:  "https://jfrog.example.com?token=secret",
			wantErr: true,
		},
		{
			name:    "fragment",
			server:  "https://jfrog.example.com#secret",
			wantErr: true,
		},
		{
			name:    "unexpected path",
			server:  "https://jfrog.example.com/ui",
			wantErr: true,
		},
		{
			name:    "encoded path",
			server:  "https://jfrog.example.com/%61rtifactory",
			wantErr: true,
		},
		{
			name:    "missing host",
			server:  "https:///artifactory",
			wantErr: true,
		},
		{
			name:    "empty port",
			server:  "https://jfrog.example.com:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveTokenEndpoint(tt.server)
			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"ResolveTokenEndpoint(%q) returned %q, want error",
						tt.server,
						got,
					)
				}
				if strings.Contains(err.Error(), "secret") {
					t.Fatalf("error exposes secret-bearing URL: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf(
					"ResolveTokenEndpoint(%q) error = %v",
					tt.server,
					err,
				)
			}
			if got != tt.want {
				t.Fatalf(
					"ResolveTokenEndpoint(%q) = %q, want %q",
					tt.server,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestValidateDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		server   string
		registry string
		repo     string
		want     Destination
		wantErr  bool
	}{
		{
			name:     "matching default ports",
			server:   "https://JFROG.example.com/artifactory",
			registry: "jfrog.example.com:443",
			repo:     "jfrog.example.com/team/image",
			want: Destination{
				Registry:   "jfrog.example.com",
				Repository: "jfrog.example.com/team/image",
			},
		},
		{
			name:     "matching explicit port",
			server:   "https://jfrog.example.com:8443",
			registry: "jfrog.example.com:8443",
			repo:     "jfrog.example.com:8443/team/image",
			want: Destination{
				Registry:   "jfrog.example.com:8443",
				Repository: "jfrog.example.com:8443/team/image",
			},
		},
		{
			name:     "preserves explicit default port repository authority",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "jfrog.example.com:443/team/image",
			want: Destination{
				Registry:   "jfrog.example.com:443",
				Repository: "jfrog.example.com:443/team/image",
			},
		},
		{
			name:     "cross origin registry",
			server:   "https://jfrog.example.com",
			registry: "evil.example.com",
			repo:     "jfrog.example.com/team/image",
			wantErr:  true,
		},
		{
			name:     "cross origin repository",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "evil.example.com/team/image",
			wantErr:  true,
		},
		{
			name:     "port mismatch",
			server:   "https://jfrog.example.com:8443",
			registry: "jfrog.example.com",
			repo:     "jfrog.example.com:8443/team/image",
			wantErr:  true,
		},
		{
			name:     "relative repository",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "team/image",
			wantErr:  true,
		},
		{
			name:     "single-label authority is interpreted as Docker Hub namespace",
			server:   "https://jfrog",
			registry: "jfrog",
			repo:     "jfrog/team/image",
			wantErr:  true,
		},
		{
			name:     "missing repository path",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "jfrog.example.com",
			wantErr:  true,
		},
		{
			name:     "tag forbidden",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "jfrog.example.com/team/image:latest",
			wantErr:  true,
		},
		{
			name:     "digest forbidden",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo: "jfrog.example.com/team/image@sha256:" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantErr: true,
		},
		{
			name:     "registry userinfo forbidden",
			server:   "https://jfrog.example.com",
			registry: "user:secret@jfrog.example.com",
			repo:     "jfrog.example.com/team/image",
			wantErr:  true,
		},
		{
			name:     "registry query forbidden",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com?token=secret",
			repo:     "jfrog.example.com/team/image",
			wantErr:  true,
		},
		{
			name:     "repository URL forbidden",
			server:   "https://jfrog.example.com",
			registry: "jfrog.example.com",
			repo:     "https://jfrog.example.com/team/image",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ValidateDestination(tt.server, tt.registry, tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"ValidateDestination(%q, %q, %q) returned %+v, want error",
						tt.server,
						tt.registry,
						tt.repo,
						got,
					)
				}
				if strings.Contains(err.Error(), "secret") {
					t.Fatalf("error exposes secret-bearing input: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateDestination() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ValidateDestination() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRepositoryMatchesServerOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		repository string
		wantMatch  bool
		wantErr    bool
	}{
		{
			name:       "same origin",
			repository: "jfrog.example.com/team/cache",
			wantMatch:  true,
		},
		{
			name:       "same origin explicit default port",
			repository: "jfrog.example.com:443/team/cache",
			wantMatch:  true,
		},
		{
			name:       "same origin tagged cache",
			repository: "jfrog.example.com/team/cache:buildcache",
			wantMatch:  true,
		},
		{
			name:       "same origin digest cache",
			repository: "jfrog.example.com/team/cache@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantMatch:  true,
		},
		{
			name:       "cross origin",
			repository: "cache.example.com/team/cache:buildcache",
		},
		{
			name:       "malformed",
			repository: "https://cache.example.com/team/cache",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RepositoryMatchesServerOrigin(
				"https://jfrog.example.com/artifactory",
				tt.repository,
			)
			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"RepositoryMatchesServerOrigin() = %v, want error",
						got,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("RepositoryMatchesServerOrigin() error = %v", err)
			}
			if got != tt.wantMatch {
				t.Fatalf(
					"RepositoryMatchesServerOrigin() = %v, want %v",
					got,
					tt.wantMatch,
				)
			}
		})
	}
}

func TestRegistryMatchesServerOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		registry  string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "same bare registry",
			registry:  "jfrog.example.com",
			wantMatch: true,
		},
		{
			name:      "same registry URL with path",
			registry:  "https://jfrog.example.com/v2/",
			wantMatch: true,
		},
		{
			name:     "different registry",
			registry: "https://index.docker.io/v1/",
		},
		{
			name:     "insecure registry URL",
			registry: "http://jfrog.example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RegistryMatchesServerOrigin(
				"https://jfrog.example.com/artifactory",
				tt.registry,
			)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RegistryMatchesServerOrigin() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("RegistryMatchesServerOrigin() error = %v", err)
			}
			if got != tt.wantMatch {
				t.Fatalf(
					"RegistryMatchesServerOrigin() = %v, want %v",
					got,
					tt.wantMatch,
				)
			}
		})
	}
}

func TestParseBaseImageRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		registry   string
		wantOrigin string
		wantErr    bool
	}{
		{
			name:       "bare host",
			registry:   "base.example.com",
			wantOrigin: "base.example.com\x00443",
		},
		{
			name:       "Docker Hub URL",
			registry:   "https://index.docker.io/v1/",
			wantOrigin: "index.docker.io\x00443",
		},
		{
			name:       "explicit port URL",
			registry:   "https://base.example.com:8443/v2/",
			wantOrigin: "base.example.com\x008443",
		},
		{
			name:     "HTTP URL",
			registry: "http://base.example.com",
			wantErr:  true,
		},
		{
			name:     "userinfo",
			registry: "https://user:secret@base.example.com",
			wantErr:  true,
		},
		{
			name:     "query",
			registry: "https://base.example.com?token=secret",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, origin, err := parseBaseImageRegistry(tt.registry)
			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"parseBaseImageRegistry(%q) returned %+v, want error",
						tt.registry,
						parsed,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBaseImageRegistry() error = %v", err)
			}
			if origin != tt.wantOrigin {
				t.Fatalf("origin = %q, want %q", origin, tt.wantOrigin)
			}
		})
	}
}

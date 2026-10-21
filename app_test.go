package docker

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli"

	"github.com/drone-plugins/drone-buildx/internal/artifactory"
)

func newOIDCContext(t *testing.T, args ...string) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("url", "", "")
	set.String("artifactory-oidc-provider-name", "", "")
	set.String("artifactory-oidc-project-key", "", "")
	if err := set.Parse(args); err != nil {
		t.Fatal(err)
	}
	return cli.NewContext(nil, set, nil)
}

func secureDockerConfig(t *testing.T) string {
	t.Helper()
	dockerConfig := t.TempDir()
	if err := os.Chmod(dockerConfig, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_CONFIG", dockerConfig)
	return dockerConfig
}

func stubOIDCExchange(
	t *testing.T,
	exchange func(endpoint, idToken, providerName, projectKey string) (artifactory.Credential, error),
) {
	t.Helper()
	original := exchangeArtifactoryOIDCCredential
	t.Cleanup(func() { exchangeArtifactoryOIDCCredential = original })
	exchangeArtifactoryOIDCCredential = func(
		_ context.Context,
		_ *http.Client,
		endpoint string,
		idToken string,
		providerName string,
		projectKey string,
	) (artifactory.Credential, error) {
		return exchange(endpoint, idToken, providerName, projectKey)
	}
}

func validCredential() artifactory.Credential {
	return artifactory.Credential{
		Username:    "oidc-user",
		AccessToken: "jfrog-access-token",
		ExpiresIn:   3600,
	}
}

func TestConfigureArtifactoryOIDCPreparesAuthentication(t *testing.T) {
	t.Setenv("ARTIFACTORY_OIDC_TOKEN", "harness-token")
	secureDockerConfig(t)

	exchangeCalls := 0
	stubOIDCExchange(t, func(endpoint, idToken, providerName, projectKey string) (artifactory.Credential, error) {
		exchangeCalls++
		if endpoint != "https://jfrog.example.com/access/api/v1/oidc/token" {
			t.Fatalf("endpoint = %q", endpoint)
		}
		if idToken != "harness-token" {
			t.Fatalf("ID token = %q", idToken)
		}
		if providerName != "harness" || projectKey != "project" {
			t.Fatalf("provider/project = %q/%q", providerName, projectKey)
		}
		return validCredential(), nil
	})

	plugin := Plugin{
		Login: Login{Registry: "jfrog.example.com"},
		Build: Build{Repo: "jfrog.example.com/team/image"},
	}
	c := newOIDCContext(t,
		"--url", "https://jfrog.example.com/artifactory",
		"--artifactory-oidc-provider-name", "harness",
		"--artifactory-oidc-project-key", "project",
	)
	if err := configureArtifactoryOIDC(context.Background(), c, &plugin); err != nil {
		t.Fatalf("configureArtifactoryOIDC() error = %v", err)
	}
	if exchangeCalls != 1 {
		t.Fatalf("exchange calls = %d, want 1", exchangeCalls)
	}
	if plugin.artifactoryOIDC == nil {
		t.Fatal("prepared Artifactory OIDC authentication is nil")
	}
	if len(plugin.artifactoryOIDC.Registries) != 1 ||
		plugin.artifactoryOIDC.Registries[0] != "jfrog.example.com" {
		t.Fatalf("registries = %#v", plugin.artifactoryOIDC.Registries)
	}
	if plugin.artifactoryOIDC.Username != "oidc-user" ||
		plugin.artifactoryOIDC.AccessToken != "jfrog-access-token" {
		t.Fatal("prepared credentials do not match exchange response")
	}
}

func TestConfigureArtifactoryOIDCRejectsPartialInputsBeforeExchange(t *testing.T) {
	exchangeCalled := false
	stubOIDCExchange(t, func(string, string, string, string) (artifactory.Credential, error) {
		exchangeCalled = true
		return artifactory.Credential{}, errors.New("unexpected exchange")
	})

	plugin := Plugin{
		Login: Login{Registry: "jfrog.example.com"},
		Build: Build{Repo: "jfrog.example.com/team/image"},
	}
	c := newOIDCContext(t, "--artifactory-oidc-provider-name", "harness")
	if err := configureArtifactoryOIDC(context.Background(), c, &plugin); err == nil {
		t.Fatal("configureArtifactoryOIDC() error = nil, want partial OIDC error")
	}
	if exchangeCalled {
		t.Fatal("OIDC exchange occurred for partial inputs")
	}
	if plugin.artifactoryOIDC != nil {
		t.Fatal("OIDC authentication was prepared from partial inputs")
	}
}

func TestConfigureArtifactoryOIDCDoesNotFallbackWhenExchangeFails(t *testing.T) {
	t.Setenv("ARTIFACTORY_OIDC_TOKEN", "harness-token")
	secureDockerConfig(t)
	stubOIDCExchange(t, func(string, string, string, string) (artifactory.Credential, error) {
		return artifactory.Credential{}, errors.New("exchange failed")
	})

	plugin := Plugin{
		Login: Login{Registry: "jfrog.example.com"},
		Build: Build{Repo: "jfrog.example.com/team/image"},
	}
	c := newOIDCContext(t,
		"--url", "https://jfrog.example.com",
		"--artifactory-oidc-provider-name", "harness",
	)
	if err := configureArtifactoryOIDC(context.Background(), c, &plugin); err == nil {
		t.Fatal("configureArtifactoryOIDC() error = nil, want exchange failure")
	}
	if plugin.artifactoryOIDC != nil {
		t.Fatal("OIDC authentication was prepared after the exchange failed")
	}
}

func TestConfigureArtifactoryOIDCScopesToSameOriginRegistryCaches(t *testing.T) {
	t.Setenv("ARTIFACTORY_OIDC_TOKEN", "harness-token")
	secureDockerConfig(t)
	stubOIDCExchange(t, func(string, string, string, string) (artifactory.Credential, error) {
		return validCredential(), nil
	})

	plugin := Plugin{
		Login: Login{Registry: "jfrog.example.com"},
		Build: Build{
			Repo:      "jfrog.example.com/team/image",
			CacheFrom: []string{"type=registry,ref=jfrog.example.com:443/team/cache:buildcache"},
			CacheTo:   []string{"type=registry,ref=cache.example.com/team/cache:buildcache"},
		},
	}
	c := newOIDCContext(t,
		"--url", "https://jfrog.example.com",
		"--artifactory-oidc-provider-name", "harness",
	)
	if err := configureArtifactoryOIDC(context.Background(), c, &plugin); err != nil {
		t.Fatalf("configureArtifactoryOIDC() error = %v", err)
	}
	want := []string{"jfrog.example.com", "jfrog.example.com:443"}
	got := plugin.artifactoryOIDC.Registries
	if len(got) != len(want) {
		t.Fatalf("registries = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("registries = %#v, want %#v", got, want)
		}
	}
}

func TestConfigureArtifactoryOIDCRejectsInsecureDockerConfigBeforeExchange(t *testing.T) {
	t.Setenv("ARTIFACTORY_OIDC_TOKEN", "harness-token")
	dockerConfig := t.TempDir()
	if err := os.Chmod(dockerConfig, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_CONFIG", dockerConfig)

	exchangeCalled := false
	stubOIDCExchange(t, func(string, string, string, string) (artifactory.Credential, error) {
		exchangeCalled = true
		return artifactory.Credential{}, nil
	})

	plugin := Plugin{
		Login: Login{Registry: "jfrog.example.com"},
		Build: Build{Repo: "jfrog.example.com/team/image"},
	}
	c := newOIDCContext(t,
		"--url", "https://jfrog.example.com",
		"--artifactory-oidc-provider-name", "harness",
	)
	if err := configureArtifactoryOIDC(context.Background(), c, &plugin); err == nil {
		t.Fatal("configureArtifactoryOIDC() error = nil, want Docker config permission error")
	}
	if exchangeCalled {
		t.Fatal("OIDC exchange occurred with an insecure Docker config directory")
	}
}

func TestConfigureArtifactoryOIDCSkipsNonOIDCInvocation(t *testing.T) {
	exchangeCalled := false
	stubOIDCExchange(t, func(string, string, string, string) (artifactory.Credential, error) {
		exchangeCalled = true
		return artifactory.Credential{}, nil
	})

	plugin := Plugin{
		Login: Login{Registry: "registry.example.com"},
		Build: Build{Repo: "team/image"},
	}
	if err := configureArtifactoryOIDC(context.Background(), newOIDCContext(t), &plugin); err != nil {
		t.Fatalf("configureArtifactoryOIDC() error = %v", err)
	}
	if exchangeCalled {
		t.Fatal("OIDC exchange occurred for a non-OIDC invocation")
	}
	if plugin.artifactoryOIDC != nil {
		t.Fatal("OIDC authentication was prepared for a non-OIDC invocation")
	}
}

func TestRemoveDockerConfigFileRemovesCredentials(t *testing.T) {
	dockerConfig := secureDockerConfig(t)
	configFile := filepath.Join(dockerConfig, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"auths":{}}`), 0600); err != nil {
		t.Fatal(err)
	}

	removeDockerConfigFile()

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Fatalf("config.json still exists, stat error = %v", err)
	}
}

func TestRemoveDockerConfigFileToleratesMissingFile(t *testing.T) {
	secureDockerConfig(t)

	removeDockerConfigFile()
}

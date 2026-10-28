package artifactory

import (
	"strings"
	"testing"
)

func TestHasOIDCInputs(t *testing.T) {
	t.Parallel()

	if HasOIDCInputs(Inputs{}) {
		t.Fatal("HasOIDCInputs() = true for empty inputs")
	}
	tests := []struct {
		name   string
		mutate func(*Inputs)
	}{
		{name: "server URL", mutate: func(in *Inputs) { in.ServerURL = "value" }},
		{name: "ID token", mutate: func(in *Inputs) { in.IDToken = "value" }},
		{name: "provider", mutate: func(in *Inputs) { in.ProviderName = "value" }},
		{name: "project key", mutate: func(in *Inputs) { in.ProjectKey = "value" }},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var inputs Inputs
			tt.mutate(&inputs)
			if !HasOIDCInputs(inputs) {
				t.Fatal("HasOIDCInputs() = false, want true")
			}
		})
	}
}

func TestValidateInputsRejectsPartialOIDCConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Inputs)
	}{
		{name: "missing URL", mutate: func(in *Inputs) { in.ServerURL = "" }},
		{name: "missing ID token", mutate: func(in *Inputs) { in.IDToken = "" }},
		{name: "missing provider", mutate: func(in *Inputs) { in.ProviderName = "" }},
		{name: "missing registry", mutate: func(in *Inputs) { in.Registry = "" }},
		{name: "missing repository", mutate: func(in *Inputs) { in.Repository = "" }},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inputs := validInputs()
			tt.mutate(&inputs)
			if err := ValidateInputs(inputs); err == nil {
				t.Fatal("ValidateInputs() error = nil, want error")
			}
		})
	}
}

func TestValidateInputsRejectsConflictingOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Inputs)
	}{
		{
			name: "Docker config",
			mutate: func(in *Inputs) {
				in.DockerConfigOverride = `{"auths":{}}`
			},
		},
		{
			name:   "destination username",
			mutate: func(in *Inputs) { in.Username = "user" },
		},
		{
			name:   "destination password",
			mutate: func(in *Inputs) { in.Password = "password" },
		},
		{
			name: "same-origin base image credentials",
			mutate: func(in *Inputs) {
				in.BaseImageRegistry = "https://JFROG.example.com:443/artifactory"
				in.BaseImageUsername = "base-user"
				in.BaseImagePassword = "base-password"
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inputs := validInputs()
			tt.mutate(&inputs)
			err := ValidateInputs(inputs)
			if err == nil {
				t.Fatal("ValidateInputs() error = nil, want conflict error")
			}
			if strings.Contains(err.Error(), inputs.IDToken) {
				t.Fatalf("error exposes ID token: %v", err)
			}
		})
	}
}

func TestValidateInputsAllowsOptionalProjectKey(t *testing.T) {
	t.Parallel()

	withProject := validInputs()
	withProject.ProjectKey = "project"
	if err := ValidateInputs(withProject); err != nil {
		t.Fatalf("ValidateInputs() with project error = %v", err)
	}

	withoutProject := validInputs()
	if err := ValidateInputs(withoutProject); err != nil {
		t.Fatalf("ValidateInputs() without project error = %v", err)
	}
}

func TestValidateInputsAllowsSameOriginCacheRepositories(t *testing.T) {
	t.Parallel()

	inputs := validInputs()
	inputs.CacheRepositories = []string{
		"jfrog.example.com/team/cache-from",
		"jfrog.example.com:443/team/cache-to",
	}
	if err := ValidateInputs(inputs); err != nil {
		t.Fatalf("ValidateInputs() error = %v", err)
	}
}

func TestValidateInputsAllowsCrossOriginCacheRepository(t *testing.T) {
	t.Parallel()

	inputs := validInputs()
	inputs.CacheRepositories = []string{"other.example.com/team/cache"}
	if err := ValidateInputs(inputs); err != nil {
		t.Fatalf("ValidateInputs() error = %v", err)
	}
}

func TestValidateInputsAllowsDifferentHostBaseImageCredentials(t *testing.T) {
	t.Parallel()

	inputs := validInputs()
	inputs.BaseImageRegistry = "https://index.docker.io/v1/"
	inputs.BaseImageUsername = "base-user"
	inputs.BaseImagePassword = "base-password"
	if err := ValidateInputs(inputs); err != nil {
		t.Fatalf("ValidateInputs() error = %v", err)
	}
}

func TestValidateInputsRejectsInvalidBaseImageRegistry(t *testing.T) {
	t.Parallel()

	inputs := validInputs()
	inputs.BaseImageRegistry = "http://base.example.com"
	inputs.BaseImageUsername = "base-user"
	inputs.BaseImagePassword = "base-password"
	err := ValidateInputs(inputs)
	if err == nil || !strings.Contains(err.Error(), "invalid base image registry") {
		t.Fatalf("ValidateInputs() error = %v, want invalid base registry", err)
	}
}

func validInputs() Inputs {
	return Inputs{
		ServerURL:    "https://JFROG.example.com/artifactory",
		IDToken:      "harness-id-token",
		ProviderName: "provider",
		Registry:     "jfrog.example.com:443",
		Repository:   "jfrog.example.com/team/image",
	}
}

package artifactory

import (
	"fmt"
	"strings"
)

// Inputs contains the OIDC-specific settings and standard Buildx options that
// can conflict with OIDC-managed destination credentials.
type Inputs struct {
	ServerURL    string
	IDToken      string
	ProviderName string
	ProjectKey   string
	Registry     string
	Repository   string

	DockerConfigOverride string
	Username             string
	Password             string

	BaseImageRegistry string
	BaseImageUsername string
	BaseImagePassword string

	CacheRepositories []string
}

// HasOIDCInputs reports whether any Artifactory OIDC-specific input was set.
// Callers use this to fail partial configurations closed instead of falling
// back to another authentication mode.
func HasOIDCInputs(inputs Inputs) bool {
	return inputs.ServerURL != "" ||
		inputs.IDToken != "" ||
		inputs.ProviderName != "" ||
		inputs.ProjectKey != ""
}

// ValidateInputs rejects options that conflict with OIDC-managed destination
// credentials and validates the required OIDC and destination values.
func ValidateInputs(inputs Inputs) error {
	if strings.TrimSpace(inputs.ServerURL) == "" {
		return fmt.Errorf("JFrog server URL is required")
	}
	if strings.TrimSpace(inputs.IDToken) == "" {
		return fmt.Errorf("OIDC ID token is required")
	}
	if strings.TrimSpace(inputs.ProviderName) == "" {
		return fmt.Errorf("OIDC provider name is required")
	}
	if strings.TrimSpace(inputs.Registry) == "" {
		return fmt.Errorf("registry is required")
	}
	if strings.TrimSpace(inputs.Repository) == "" {
		return fmt.Errorf("repository is required")
	}
	if inputs.DockerConfigOverride != "" || inputs.Username != "" ||
		inputs.Password != "" {
		return fmt.Errorf("option is incompatible with JFrog OIDC credentials")
	}

	destination, err := ValidateDestination(
		inputs.ServerURL,
		inputs.Registry,
		inputs.Repository,
	)
	if err != nil {
		return err
	}
	for _, cacheRepository := range inputs.CacheRepositories {
		if _, err := RepositoryMatchesServerOrigin(
			inputs.ServerURL,
			cacheRepository,
		); err != nil {
			return fmt.Errorf("invalid cache repository: %w", err)
		}
	}

	if inputs.BaseImageRegistry == "" {
		return nil
	}
	_, baseOrigin, err := parseBaseImageRegistry(inputs.BaseImageRegistry)
	if err != nil {
		return fmt.Errorf("invalid base image registry")
	}
	_, destinationOrigin, err := parseRegistry(destination.Registry)
	if err != nil {
		return fmt.Errorf("invalid registry")
	}
	if baseOrigin == destinationOrigin &&
		(inputs.BaseImageUsername != "" || inputs.BaseImagePassword != "") {
		return fmt.Errorf(
			"base image credentials conflict with JFrog OIDC credentials",
		)
	}
	return nil
}

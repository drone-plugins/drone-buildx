package artifactory

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const tokenEndpointPath = "/access/api/v1/oidc/token"

var repositoryPathPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*)*$`,
)
var tagPattern = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)
var digestPattern = regexp.MustCompile(
	`^[A-Za-z][A-Za-z0-9]*(?:[+._-][A-Za-z0-9]+)*:[0-9A-Fa-f]{32,}$`,
)

// Destination is a validated registry and untagged repository pair.
type Destination struct {
	Registry   string
	Repository string
}

// ResolveTokenEndpoint converts an approved JFrog connector URL into its OIDC
// token exchange endpoint.
func ResolveTokenEndpoint(serverURL string) (string, error) {
	parsed, _, err := parseServerURL(serverURL)
	if err != nil {
		return "", err
	}
	parsed.Path = tokenEndpointPath
	parsed.RawPath = ""
	return parsed.String(), nil
}

// ValidateDestination verifies that the configured registry and repository
// resolve to the same HTTPS origin as the JFrog connector.
func ValidateDestination(serverURL, registry, repository string) (Destination, error) {
	_, serverOrigin, err := parseServerURL(serverURL)
	if err != nil {
		return Destination{}, err
	}

	_, registryOrigin, err := parseRegistry(registry)
	if err != nil {
		return Destination{}, err
	}
	if registryOrigin != serverOrigin {
		return Destination{}, fmt.Errorf("registry origin does not match JFrog server")
	}

	repositoryAuthority, repositoryOrigin, err := parseRepository(repository)
	if err != nil {
		return Destination{}, err
	}
	if repositoryOrigin != serverOrigin {
		return Destination{}, fmt.Errorf("repository origin does not match JFrog server")
	}

	return Destination{
		Registry:   repositoryAuthority,
		Repository: repository,
	}, nil
}

// RepositoryMatchesServerOrigin validates a registry cache image reference and
// reports whether its registry is the selected JFrog origin. Callers may
// preserve cross-origin Buildx caches while withholding JFrog credentials.
func RepositoryMatchesServerOrigin(
	serverURL string,
	repository string,
) (bool, error) {
	_, serverOrigin, err := parseServerURL(serverURL)
	if err != nil {
		return false, err
	}
	_, repositoryOrigin, err := parseRepositoryReference(repository)
	if err != nil {
		return false, err
	}
	return repositoryOrigin == serverOrigin, nil
}

// RegistryMatchesServerOrigin validates a base-image registry and reports
// whether it resolves to the selected JFrog origin.
func RegistryMatchesServerOrigin(serverURL, registry string) (bool, error) {
	_, serverOrigin, err := parseServerURL(serverURL)
	if err != nil {
		return false, err
	}
	_, registryOrigin, err := parseBaseImageRegistry(registry)
	if err != nil {
		return false, err
	}
	return registryOrigin == serverOrigin, nil
}

func parseRepository(repository string) (string, string, error) {
	if strings.TrimSpace(repository) != repository ||
		strings.Contains(repository, "://") ||
		strings.ContainsAny(repository, "@?#") {
		return "", "", fmt.Errorf("invalid repository")
	}
	repositoryParts := strings.SplitN(repository, "/", 2)
	if len(repositoryParts) != 2 ||
		!repositoryPathPattern.MatchString(repositoryParts[1]) {
		return "", "", fmt.Errorf("invalid repository")
	}
	if !isQualifiedRegistryAuthority(repositoryParts[0]) {
		return "", "", fmt.Errorf("repository must be fully qualified")
	}
	_, repositoryOrigin, err := parseRegistry(repositoryParts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid repository registry")
	}
	return repositoryParts[0], repositoryOrigin, nil
}

func parseRepositoryReference(reference string) (string, string, error) {
	if strings.TrimSpace(reference) != reference ||
		strings.Contains(reference, "://") ||
		strings.ContainsAny(reference, "?#") {
		return "", "", fmt.Errorf("invalid repository")
	}
	parts := strings.SplitN(reference, "/", 2)
	if len(parts) != 2 || !isQualifiedRegistryAuthority(parts[0]) {
		return "", "", fmt.Errorf("repository must be fully qualified")
	}

	repositoryPath := parts[1]
	if strings.Count(repositoryPath, "@") > 1 {
		return "", "", fmt.Errorf("invalid repository")
	}
	if path, digest, found := strings.Cut(repositoryPath, "@"); found {
		if !digestPattern.MatchString(digest) {
			return "", "", fmt.Errorf("invalid repository digest")
		}
		repositoryPath = path
	}
	if tagSeparator := strings.LastIndex(repositoryPath, ":"); tagSeparator >= 0 {
		if !tagPattern.MatchString(repositoryPath[tagSeparator+1:]) {
			return "", "", fmt.Errorf("invalid repository tag")
		}
		repositoryPath = repositoryPath[:tagSeparator]
	}
	if !repositoryPathPattern.MatchString(repositoryPath) {
		return "", "", fmt.Errorf("invalid repository")
	}

	_, repositoryOrigin, err := parseRegistry(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid repository registry")
	}
	return parts[0], repositoryOrigin, nil
}

func isQualifiedRegistryAuthority(authority string) bool {
	return strings.EqualFold(authority, "localhost") ||
		strings.Contains(authority, ".") ||
		strings.Contains(authority, ":")
}

func parseServerURL(raw string) (*url.URL, string, error) {
	if strings.TrimSpace(raw) != raw {
		return nil, "", fmt.Errorf("invalid JFrog server URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" ||
		strings.HasSuffix(parsed.Host, ":") || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		parsed.Opaque != "" {
		return nil, "", fmt.Errorf("invalid JFrog server URL")
	}
	if parsed.Path != "" && parsed.Path != "/" &&
		parsed.Path != "/artifactory" && parsed.Path != "/artifactory/" {
		return nil, "", fmt.Errorf("unsupported JFrog server URL path")
	}
	if parsed.RawPath != "" && parsed.EscapedPath() != parsed.Path {
		return nil, "", fmt.Errorf("invalid JFrog server URL path")
	}
	origin, err := normalizedHTTPSOrigin(parsed)
	if err != nil {
		return nil, "", fmt.Errorf("invalid JFrog server URL")
	}
	return parsed, origin, nil
}

func parseRegistry(registry string) (*url.URL, string, error) {
	if registry == "" || strings.TrimSpace(registry) != registry ||
		strings.ContainsAny(registry, "/@?#") || strings.Contains(registry, "://") {
		return nil, "", fmt.Errorf("invalid registry")
	}
	parsed, err := url.Parse("https://" + registry)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil ||
		strings.HasSuffix(parsed.Host, ":") || parsed.Path != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, "", fmt.Errorf("invalid registry")
	}
	origin, err := normalizedHTTPSOrigin(parsed)
	if err != nil {
		return nil, "", fmt.Errorf("invalid registry")
	}
	return parsed, origin, nil
}

func parseBaseImageRegistry(registry string) (*url.URL, string, error) {
	if registry == "" || strings.TrimSpace(registry) != registry {
		return nil, "", fmt.Errorf("invalid base image registry")
	}
	if !strings.Contains(registry, "://") {
		parsed, origin, err := parseRegistry(registry)
		if err != nil {
			return nil, "", fmt.Errorf("invalid base image registry")
		}
		return parsed, origin, nil
	}

	parsed, err := url.Parse(registry)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") ||
		parsed.Hostname() == "" || strings.HasSuffix(parsed.Host, ":") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery ||
		parsed.Fragment != "" || parsed.Opaque != "" {
		return nil, "", fmt.Errorf("invalid base image registry")
	}
	origin, err := normalizedHTTPSOrigin(parsed)
	if err != nil {
		return nil, "", fmt.Errorf("invalid base image registry")
	}
	return parsed, origin, nil
}

func normalizedHTTPSOrigin(parsed *url.URL) (string, error) {
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return "", fmt.Errorf("HTTPS origin required")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return "", fmt.Errorf("invalid host")
	}
	port := parsed.Port()
	if port == "" {
		port = "443"
	} else {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return "", fmt.Errorf("invalid port")
		}
		port = strconv.FormatUint(number, 10)
	}
	return host + "\x00" + port, nil
}

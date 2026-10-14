package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	defaultDriver         = "docker"
	dockerContainerDriver = "docker-container"
	remoteDriver          = "remote"

	envBuildkitInheritAuth            = "PLUGIN_BUILDKIT_INHERIT_AUTH"
	envHarnessCacheAwsInherit         = "HARNESS_CACHE_AWS_INHERIT_FROM_DELEGATE"
	envAwsRoleArn                     = "AWS_ROLE_ARN"
	envAwsWebIdentityTokenFile        = "AWS_WEB_IDENTITY_TOKEN_FILE"
	envAwsAccessKeyId                 = "AWS_ACCESS_KEY_ID"
	envAwsContainerCredsRelativeUri   = "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"
	envAwsContainerCredsFullUri       = "AWS_CONTAINER_CREDENTIALS_FULL_URI"
)

func cmdSetupBuildx(builder Builder, driverOpts []string, inheritAuth bool) *exec.Cmd {
	args := []string{"buildx", "create", "--use", "--driver", builder.Driver}
	if builder.Name != "" {
		args = append(args, "--name", builder.Name)
	}
	if builder.DaemonConfig != "" {
		args = append(args, "--buildkitd-config", builder.DaemonConfig)
	}
	for _, opt := range driverOpts {
		args = append(args, "--driver-opt", opt)
	}
	args = appendProxyDriverOpts(args)
	args = appendHarnessCADriverOpts(args)

	if builder.RemoteConn != "" && builder.Driver == remoteDriver {
		args = append(args, builder.RemoteConn)
	}
	// Collect buildkitd flags
	var buildkitdFlags []string
	if builder.BuildkitTLSHandshakeTimeout != "" {
		buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--tls-handshake-timeout=%s", builder.BuildkitTLSHandshakeTimeout))
	}
	if builder.BuildkitResponseHeaderTimeout != "" {
		buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--response-header-timeout=%s", builder.BuildkitResponseHeaderTimeout))
	}
	if shouldInheritBuildkitAuth(inheritAuth) {
		args, buildkitdFlags = appendBuildkitInheritAuth(args, buildkitdFlags)
	} else {
		fmt.Println("BuildKit AWS auth: inherit-auth disabled (static keys / OIDC / none)")
	}
	if len(buildkitdFlags) > 0 {
		args = append(args, "--buildkitd-flags", strings.Join(buildkitdFlags, " "))
	}
	return exec.Command(dockerExe, args...)
}

// shouldInheritBuildkitAuth enables forwarding of the CI build pod ambient AWS
// identity into buildkitd. Enabled via PLUGIN_BUILDKIT_INHERIT_AUTH or the
// Cache/Build Intel signal HARNESS_CACHE_AWS_INHERIT_FROM_DELEGATE.
func shouldInheritBuildkitAuth(inheritAuthFlag bool) bool {
	if inheritAuthFlag {
		return true
	}
	if strings.EqualFold(os.Getenv(envBuildkitInheritAuth), "true") {
		return true
	}
	return strings.EqualFold(os.Getenv(envHarnessCacheAwsInherit), "true")
}

// appendBuildkitInheritAuth forwards AWS_* env vars and the web-identity token
// into the buildx docker-container builder so DLC S3 cache uses the same
// build-pod SA identity as Cache Intel / Build Intel.
func appendBuildkitInheritAuth(args, buildkitdFlags []string) ([]string, []string) {
	authMode := detectBuildPodAwsAuthMode()
	fmt.Printf("BuildKit AWS auth: inherit-auth enabled; invoking mode=%s (build-pod ambient identity)\n", authMode)

	forwarded := 0
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envName := parts[0]
				envValue := parts[1]
				args = append(args, "--driver-opt", fmt.Sprintf("env.%s=%s", envName, envValue))
				forwarded++
			}
		}
	}
	fmt.Printf("BuildKit AWS auth: forwarded %d AWS_* env var(s) into buildkitd\n", forwarded)

	if tokenFilePath := os.Getenv(envAwsWebIdentityTokenFile); tokenFilePath != "" {
		if _, err := os.Stat(tokenFilePath); err == nil {
			if content, err := ioutil.ReadFile(tokenFilePath); err == nil {
				const maxSafeSize = 4 * 1024
				if len(content) > maxSafeSize {
					fmt.Fprintf(os.Stderr, "Warning: AWS_WEB_IDENTITY_TOKEN_FILE content size (%d bytes) exceeds safe command line argument size limit (%d bytes)\n", len(content), maxSafeSize)
				}
				buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--aws-token-content=%s", string(content)))
				buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--aws-token-path=%s", tokenFilePath))
				fmt.Printf("BuildKit AWS auth: embedding web-identity token into buildkitd path=%s bytes=%d\n", tokenFilePath, len(content))
			} else {
				fmt.Fprintf(os.Stderr, "Warning: failed to read AWS_WEB_IDENTITY_TOKEN_FILE %s: %v\n", tokenFilePath, err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: AWS_WEB_IDENTITY_TOKEN_FILE is set to '%s' but the file does not exist\n", tokenFilePath)
		}
	} else if authMode == "IRSA" {
		fmt.Fprintf(os.Stderr, "Warning: IRSA mode detected but AWS_WEB_IDENTITY_TOKEN_FILE is unset\n")
	}

	return args, buildkitdFlags
}

// detectBuildPodAwsAuthMode describes which ambient AWS identity the build pod
// appears to carry. Used only for operator-facing logs.
func detectBuildPodAwsAuthMode() string {
	if os.Getenv(envAwsRoleArn) != "" && os.Getenv(envAwsWebIdentityTokenFile) != "" {
		return "IRSA"
	}
	if os.Getenv(envAwsContainerCredsRelativeUri) != "" || os.Getenv(envAwsContainerCredsFullUri) != "" {
		return "CONTAINER_CREDENTIALS"
	}
	if os.Getenv(envAwsAccessKeyId) != "" {
		return "STATIC_ENV_KEYS"
	}
	return "DEFAULT_CREDENTIAL_CHAIN"
}

func cmdInspectBuildx(name string) *exec.Cmd {
	args := []string{"buildx", "inspect", "--bootstrap", "--builder", name}
	return exec.Command(dockerExe, args...)
}

func cmdRemoveBuildx(name string) *exec.Cmd {
	args := []string{"buildx", "rm", name}
	return exec.Command(dockerExe, args...)
}

// appendProxyDriverOpts forwards proxy env into the buildx docker-container
// builder. Uses getProxyValue precedence: standard -> uppercase -> HARNESS_.
// no_proxy values with commas are quoted so buildx's CSV parser does not split them.
func appendProxyDriverOpts(args []string) []string {
	httpProxy := getProxyValue("http_proxy")
	httpsProxy := getProxyValue("https_proxy")
	noProxy := getProxyValue("no_proxy")

	if httpProxy == "" && httpsProxy == "" && noProxy == "" {
		return args
	}

	if httpProxy != "" {
		args = append(args, "--driver-opt", fmt.Sprintf("env.http_proxy=%s", httpProxy))
	}
	if httpsProxy != "" {
		args = append(args, "--driver-opt", fmt.Sprintf("env.https_proxy=%s", httpsProxy))
	}
	if noProxy != "" {
		opt := fmt.Sprintf("env.no_proxy=%s", noProxy)
		// buildx splits --driver-opt on commas unless the k=v pair is quoted.
		if strings.Contains(noProxy, ",") {
			opt = `"` + opt + `"`
		}
		args = append(args, "--driver-opt", opt)
	}
	if httpProxy != "" || httpsProxy != "" {
		args = append(args, "--driver-opt", "network=host")
	}
	return args
}

// appendHarnessCADriverOpts forwards HARNESS_CA_PATH (and file content when
// readable) into the BuildKit container. A host path alone is not visible
// inside the builder filesystem, so content is passed via env.HARNESS_CA_CERT
// for the custom BuildKit entrypoint to install into the trust store.
func appendHarnessCADriverOpts(args []string) []string {
	caPath := os.Getenv("HARNESS_CA_PATH")
	if caPath == "" {
		return args
	}

	args = append(args, "--driver-opt", fmt.Sprintf("env.HARNESS_CA_PATH=%s", caPath))
	content, err := os.ReadFile(caPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: HARNESS_CA_PATH is set to '%s' but the file does not exist\n", caPath)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: unable to read HARNESS_CA_PATH '%s': %v\n", caPath, err)
		}
		return args
	}
	args = append(args, "--driver-opt", fmt.Sprintf("env.HARNESS_CA_CERT=%s", string(content)))
	return args
}

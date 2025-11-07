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
	if harnessHttpProxy := os.Getenv("HARNESS_HTTP_PROXY"); harnessHttpProxy != "" {
		args = append(args, "--driver-opt", fmt.Sprintf("env.http_proxy=%s", harnessHttpProxy))

		if harnessHttpsProxy := os.Getenv("HARNESS_HTTPS_PROXY"); harnessHttpsProxy != "" {
			args = append(args, "--driver-opt", fmt.Sprintf("env.https_proxy=%s", harnessHttpsProxy))
		}

		args = append(args, "--driver-opt", "network=host")
	}

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
	if inheritAuth {
		fmt.Println("Inheriting auth")
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, "AWS_") {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envName := parts[0]
					envValue := parts[1]
					args = append(args, "--driver-opt", fmt.Sprintf("env.%s=%s", envName, envValue))
				}
			}
		}

		if tokenFilePath := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"); tokenFilePath != "" {
			if _, err := os.Stat(tokenFilePath); err == nil {
				if content, err := ioutil.ReadFile(tokenFilePath); err == nil {
					const maxSafeSize = 4 * 1024
					if len(content) > maxSafeSize {
						fmt.Fprintf(os.Stderr, "Warning: AWS_WEB_IDENTITY_TOKEN_FILE content size (%d bytes) exceeds safe command line argument size limit (%d bytes)\n", len(content), maxSafeSize)
					}
					buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--aws-token-content=%s", string(content)))
					buildkitdFlags = append(buildkitdFlags, fmt.Sprintf("--aws-token-path=%s", tokenFilePath))
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: AWS_WEB_IDENTITY_TOKEN_FILE is set to '%s' but the file does not exist\n", tokenFilePath)
			}
		}
	}
	if len(buildkitdFlags) > 0 {
		args = append(args, "--buildkitd-flags", strings.Join(buildkitdFlags, " "))
	}
	return exec.Command(dockerExe, args...)
}

func cmdInspectBuildx(name string) *exec.Cmd {
	args := []string{"buildx", "inspect", "--bootstrap", "--builder", name}
	return exec.Command(dockerExe, args...)
}

func cmdRemoveBuildx(name string) *exec.Cmd {
	args := []string{"buildx", "rm", name}
	return exec.Command(dockerExe, args...)
}

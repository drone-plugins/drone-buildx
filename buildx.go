package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	defaultDriver         = "docker"
	dockerContainerDriver = "docker-container"
	remoteDriver          = "remote"
)

func cmdSetupBuildx(builder Builder, driverOpts []string) *exec.Cmd {
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

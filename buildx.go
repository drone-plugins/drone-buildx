package docker

import (
	"fmt"
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
	fmt.Println("ANURAG builder setup args", args)
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

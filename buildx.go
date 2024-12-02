package docker

import (
	"os/exec"
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

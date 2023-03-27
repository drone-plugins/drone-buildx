package docker

import (
	"os/exec"
)

const (
	defaultDriver         = "docker"
	dockerContainerDriver = "docker-container"
	remoteDriver          = "remote"
)

func commandCreateBuildxBuilder(builder Builder) *exec.Cmd {
	args := []string{"buildx", "create", "--use", "--driver", builder.Driver}
	if builder.Name != "" {
		args = append(args, "--name", builder.Name)
	}
	if builder.DriverOpts != "" {
		args = append(args, "--driver-opt", builder.DriverOpts)
	}
	if builder.RemoteConn != "" && builder.Driver == remoteDriver {
		args = append(args, builder.RemoteConn)
	}
	return exec.Command(dockerExe, args...)
}

func commandInspectBuildxBuilder(name string) *exec.Cmd {
	args := []string{"buildx", "inspect", "--bootstrap", "--builder", name}
	return exec.Command(dockerExe, args...)
}

func commandRemoveBuildxBuilder(name string) *exec.Cmd {
	args := []string{"buildx", "rm", name}
	return exec.Command(dockerExe, args...)
}

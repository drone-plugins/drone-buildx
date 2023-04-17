package docker

import (
	"os/exec"
	"reflect"
	"testing"
)

func TestCommandBuildx(t *testing.T) {
	tcs := []struct {
		name    string
		build   Build
		builder Builder
		dryrun  bool
		want    *exec.Cmd
	}{
		{
			name: "secret from env var",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=FOO_SECRET_ENV_VAR",
				},
				Repo: "plugins/drone-docker",
				Tags: []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
				"--secret id=foo_secret,env=FOO_SECRET_ENV_VAR",
			),
		},
		{
			name: "secret from file",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretFiles: []string{
					"foo_secret=/path/to/foo_secret",
				},
				Repo: "plugins/drone-docker",
				Tags: []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
				"--secret id=foo_secret,src=/path/to/foo_secret",
			),
		},
		{
			name: "multiple mixed secrets",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=FOO_SECRET_ENV_VAR",
					"bar_secret=BAR_SECRET_ENV_VAR",
				},
				SecretFiles: []string{
					"foo_secret=/path/to/foo_secret",
					"bar_secret=/path/to/bar_secret",
				},
				Repo: "plugins/drone-docker",
				Tags: []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
				"--secret id=foo_secret,env=FOO_SECRET_ENV_VAR",
				"--secret id=bar_secret,env=BAR_SECRET_ENV_VAR",
				"--secret id=foo_secret,src=/path/to/foo_secret",
				"--secret id=bar_secret,src=/path/to/bar_secret",
			),
		},
		{
			name: "invalid mixed secrets",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=",
					"=FOO_SECRET_ENV_VAR",
					"",
				},
				SecretFiles: []string{
					"foo_secret=",
					"=/path/to/bar_secret",
					"",
				},
				Repo: "plugins/drone-docker",
				Tags: []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
			),
		},
		{
			name: "platform argument",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				Platform:   "test/platform",
				Repo:       "plugins/drone-docker",
				Tags:       []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
				"--platform",
				"test/platform",
			),
		},
		{
			name: "ssh agent",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SSHKeyPath: "id_rsa=/root/.ssh/id_rsa",
				Repo:       "plugins/drone-docker",
				Tags:       []string{"latest"},
			},
			want: exec.Command(
				dockerExe,
				"buildx",
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				"plugins/drone-docker:latest",
				"--push",
				".",
				"--ssh id_rsa=/root/.ssh/id_rsa",
			),
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cmd := commandBuildx(tc.build, tc.builder, tc.dryrun)

			if !reflect.DeepEqual(cmd.String(), tc.want.String()) {
				t.Errorf("Got cmd %v, want %v", cmd, tc.want)
			}
		})
	}
}

package docker

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestCommandBuildx(t *testing.T) {
	tcs := []struct {
		name     string
		build    Build
		builder  Builder
		dryrun   bool
		metadata string
		want     *exec.Cmd
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
		{
			name: "metadata file",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				Repo:       "plugins/drone-docker",
				Tags:       []string{"latest"},
			},
			metadata: "/tmp/metadata.json",
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
				"--metadata-file /tmp/metadata.json",
			),
		},
		// Add this new test case
		{
			name: "decoded secret from env var",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=FOO_SECRET_ENV_VAR",
				},
				EncodedSecretEnvs: []string{
					"encoded_secret=RU5DT0RFRF9TRUNSRVRfRU5WX1ZBUg==", // Base64 for "ENCODED_SECRET_ENV_VAR"
				},
				DecodeEnvSecret: true,
				Repo:            "plugins/drone-docker",
				Tags:            []string{"latest"},
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
				"--secret", "id=foo_secret,env=FOO_SECRET_ENV_VAR",
				"--secret", "id=encoded_secret,env=ENCODED_SECRET_ENV_VAR",
			),
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cmd := commandBuildx(tc.build, tc.builder, tc.dryrun, tc.metadata)

			if !reflect.DeepEqual(cmd.String(), tc.want.String()) {
				t.Errorf("Got cmd %v, want %v", cmd, tc.want)
			}
		})
	}
}

func TestGetDigest(t *testing.T) {

	tcs := []struct {
		name   string
		path   string
		digest string
	}{
		{
			name:   "single platform",
			path:   "testdata/metadata.json",
			digest: "sha256:02fd68a300e3a863791744802ea3eeaac43a36e98888cdb9ffb22da8006f7eee",
		},
		{
			name:   "cross platform",
			path:   "testdata/metadata_cross_platform.json",
			digest: "sha256:0cff645119742b04807c4a3953925a579e21654baaeaf20f33c05554a6decbce",
		},
	}

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("unable to get working dir")
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			digest, err := getDigest(dir + "/" + tc.path)
			if err != nil {
				t.Errorf("unable to get digest with error %s", err)
			}
			if digest != tc.digest {
				t.Errorf("Got digest %s, want %s", digest, tc.digest)
			}
		})
	}
}

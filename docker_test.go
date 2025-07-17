package docker

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
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
		{
			name: "buildx options with semicolon delimiter",
			build: Build{
				Name:                   "plugins/drone-docker:latest",
				Dockerfile:             "Dockerfile",
				Context:                ".",
				Repo:                   "plugins/drone-docker",
				Tags:                   []string{"latest"},
				BuildxOptionsSemicolon: "--platform=linux/amd64,linux/arm64;--provenance=false;--output=type=tar,dest=image.tar",
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
				"--platform=linux/amd64,linux/arm64",
				"--provenance=false",
				"--output=type=tar,dest=image.tar",
				".",
			),
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := commandBuildx(tc.build, tc.builder, tc.dryrun, tc.metadata, "")
			if !reflect.DeepEqual(cmd.String(), tc.want.String()) {
				t.Errorf("Got cmd %v, want %v", cmd, tc.want)
			}
		})
	}
}

func TestSanitizeCacheCommand(t *testing.T) {
	tests := []struct {
		name              string
		build             Build
		expectedCacheFrom []string
		expectedCacheTo   []string
	}{
		{
			name: "Replace AWS placeholders in CacheFrom and CacheTo",
			build: Build{
				CacheFrom:                    []string{"type=s3,access_key_id=harness_placeholder_aws_creds,secret_access_key=harness_placeholder_aws_creds"},
				CacheTo:                      []string{"type=s3,access_key_id=harness_placeholder_aws_creds,secret_access_key=harness_placeholder_aws_creds"},
				HarnessSelfHostedS3AccessKey: "actual_access_key",
				HarnessSelfHostedS3SecretKey: "actual_secret_key",
			},
			expectedCacheFrom: []string{"type=s3,access_key_id=actual_access_key,secret_access_key=actual_secret_key"},
			expectedCacheTo:   []string{"type=s3,access_key_id=actual_access_key,secret_access_key=actual_secret_key"},
		},
		{
			name: "Replace GCP placeholder in CacheFrom",
			build: Build{
				CacheFrom:                   []string{"type=gcs,gcp_json_key=harness_placeholder_gcp_creds"},
				CacheTo:                     []string{},
				HarnessSelfHostedGcpJsonKey: "actual_gcp_key",
			},
			expectedCacheFrom: []string{"type=gcs,gcp_json_key=YWN0dWFsX2djcF9rZXk="},
			expectedCacheTo:   []string{},
		},
		{
			name: "Remove GCP placeholder when key is empty",
			build: Build{
				CacheFrom:                   []string{"type=gcs,gcp_json_key=harness_placeholder_gcp_creds"},
				CacheTo:                     []string{"type=gcs,bucket=test,gcp_json_key=harness_placeholder_gcp_creds,prefix=dlc"},
				HarnessSelfHostedGcpJsonKey: "",
			},
			expectedCacheFrom: []string{"type=gcs"},
			expectedCacheTo:   []string{"type=gcs,bucket=test,prefix=dlc"},
		},
		{
			name: "Multiple placeholders in CacheFrom",
			build: Build{
				CacheFrom:                    []string{"type=gcs,gcp_json_key=harness_placeholder_gcp_creds,access_key_id=harness_placeholder_aws_creds,secret_access_key=harness_placeholder_aws_creds"},
				CacheTo:                      []string{},
				HarnessSelfHostedS3AccessKey: "actual_access_key",
				HarnessSelfHostedS3SecretKey: "actual_secret_key",
				HarnessSelfHostedGcpJsonKey:  "actual_gcp_key",
			},
			expectedCacheFrom: []string{"type=gcs,gcp_json_key=YWN0dWFsX2djcF9rZXk=,access_key_id=actual_access_key,secret_access_key=actual_secret_key"},
			expectedCacheTo:   []string{},
		},
		{
			name: "No placeholders in CacheFrom and CacheTo",
			build: Build{
				CacheFrom: []string{"type=s3,bucket=test"},
				CacheTo:   []string{"type=gcs,bucket=test"},
			},
			expectedCacheFrom: []string{"type=s3,bucket=test"},
			expectedCacheTo:   []string{"type=gcs,bucket=test"},
		},
		{
			name: "Empty CacheFrom and CacheTo",
			build: Build{
				CacheFrom: []string{},
				CacheTo:   []string{},
			},
			expectedCacheFrom: []string{},
			expectedCacheTo:   []string{},
		},
		{
			name: "Add use_path_style=true if not present and Pathstyle is true",
			build: Build{
				CacheFrom: []string{"type=s3,bucket=my-bucket"},
				CacheTo:   []string{},
				PathStyle: true,
			},
			expectedCacheFrom: []string{"type=s3,bucket=my-bucket,use_path_style=true"},
			expectedCacheTo:   []string{},
		},
		{
			name: "Leave use_path_style=false untouched when Pathstyle is true",
			build: Build{
				CacheFrom: []string{"type=s3,bucket=my-bucket,use_path_style=false"},
				CacheTo:   []string{"type=s3,use_path_style=false"},
				PathStyle: true,
			},
			expectedCacheFrom: []string{"type=s3,bucket=my-bucket,use_path_style=false"},
			expectedCacheTo:   []string{"type=s3,use_path_style=false"},
		},
		{
			name: "Leave use_path_style=true untouched when already correct",
			build: Build{
				CacheFrom: []string{"type=s3,use_path_style=true"},
				CacheTo:   []string{},
				PathStyle: true,
			},
			expectedCacheFrom: []string{"type=s3,use_path_style=true"},
			expectedCacheTo:   []string{},
		},
		{
			name: "Don't modify if not type=s3 even when Pathstyle is true",
			build: Build{
				CacheFrom: []string{"type=gcs,bucket=test"},
				CacheTo:   []string{"type=local"},
				PathStyle: true,
			},
			expectedCacheFrom: []string{"type=gcs,bucket=test"},
			expectedCacheTo:   []string{"type=local"},
		},
		{
			name: "Don't add use_path_style when Pathstyle is false",
			build: Build{
				CacheFrom: []string{"type=s3,bucket=my-bucket"},
				CacheTo:   []string{},
				PathStyle: false,
			},
			expectedCacheFrom: []string{"type=s3,bucket=my-bucket"},
			expectedCacheTo:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizeCacheCommand(&tt.build)
			if !reflect.DeepEqual(tt.build.CacheFrom, tt.expectedCacheFrom) {
				t.Errorf("CacheFrom = %v, want %v", tt.build.CacheFrom, tt.expectedCacheFrom)
			}
			if !reflect.DeepEqual(tt.build.CacheTo, tt.expectedCacheTo) {
				t.Errorf("CacheTo = %v, want %v", tt.build.CacheTo, tt.expectedCacheTo)
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

func TestCommandLoadTar(t *testing.T) {
	tests := []struct {
		name    string
		tarPath string
		want    *exec.Cmd
	}{
		{
			name:    "simple path",
			tarPath: "/path/to/image.tar",
			want:    exec.Command(dockerExe, "load", "-i", "/path/to/image.tar"),
		},
		{
			name:    "path with spaces",
			tarPath: "/path with spaces/image.tar",
			want:    exec.Command(dockerExe, "load", "-i", "/path with spaces/image.tar"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commandLoadTar(tt.tarPath)
			if !reflect.DeepEqual(cmd.String(), tt.want.String()) {
				t.Errorf("commandLoadTar() = %v, want %v", cmd, tt.want)
			}
		})
	}
}

func TestCommandSaveTar(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		tarPath string
		want    *exec.Cmd
	}{
		{
			name:    "simple inputs",
			tag:     "myimage:latest",
			tarPath: "/path/to/output.tar",
			want:    exec.Command(dockerExe, "save", "-o", "/path/to/output.tar", "myimage:latest"),
		},
		{
			name:    "complex registry",
			tag:     "registry.example.com/project/image:v1.2.3",
			tarPath: "/output/dir/image.tar",
			want:    exec.Command(dockerExe, "save", "-o", "/output/dir/image.tar", "registry.example.com/project/image:v1.2.3"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commandSaveTar(tt.tag, tt.tarPath)
			if !reflect.DeepEqual(cmd.String(), tt.want.String()) {
				t.Errorf("commandSaveTar() = %v, want %v", cmd, tt.want)
			}
		})
	}
}

func TestSourceImageParsing(t *testing.T) {
	tests := []struct {
		name          string
		sourceImage   string
		expectedImage string
		expectedTags  []string
	}{
		{
			name:          "simple image with tag",
			sourceImage:   "image:tag",
			expectedImage: "image",
			expectedTags:  []string{"tag"},
		},
		{
			name:          "registry with port and tag",
			sourceImage:   "registry:5000/namespace/image:tag",
			expectedImage: "registry:5000/namespace/image",
			expectedTags:  []string{"tag"},
		},
		{
			name:          "registry with port no tag",
			sourceImage:   "registry:5000/namespace/image",
			expectedImage: "registry:5000/namespace/image",
			expectedTags:  []string{"latest"},
		},
		{
			name:          "image with empty tag",
			sourceImage:   "image:",
			expectedImage: "image",
			expectedTags:  []string{"latest"},
		},
		{
			name:          "no tag",
			sourceImage:   "image",
			expectedImage: "image",
			expectedTags:  []string{"latest"},
		},
		{
			name:          "complex registry path with tag",
			sourceImage:   "docker.op.com:4444/test/op:v.123",
			expectedImage: "docker.op.com:4444/test/op",
			expectedTags:  []string{"v.123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the Plugin struct
			p := Plugin{
				SourceImage: tt.sourceImage,
				Build: Build{
					Tags: []string{"target-tag-1", "target-tag-2"},
				},
			}

			// Extract the variables that would be set in pushOnly method
			sourceImageName := p.SourceImage
			var sourceTags []string

			// Replicating the image parsing logic
			lastColonIndex := strings.LastIndex(sourceImageName, ":")
			if lastColonIndex > 0 && lastColonIndex < len(sourceImageName) {
				// Check if there's a slash after the last colon (indicating it's a port, not a tag)
				if strings.LastIndex(sourceImageName, "/") > lastColonIndex {
					// The last colon is part of the registry:port, not a tag separator
					sourceTags = []string{"latest"}
				} else {
					// The last colon separates the tag
					tag := sourceImageName[lastColonIndex+1:]
					sourceImageName = sourceImageName[:lastColonIndex]

					if tag == "" {
						tag = "latest"
					}
					sourceTags = []string{tag}
				}
			} else {
				// Default to "latest" if no tag specified
				sourceTags = []string{"latest"}
			}

			// Verify results
			if sourceImageName != tt.expectedImage {
				t.Errorf("Image name = %v, want %v", sourceImageName, tt.expectedImage)
			}

			if !reflect.DeepEqual(sourceTags, tt.expectedTags) {
				t.Errorf("Tags = %v, want %v", sourceTags, tt.expectedTags)
			}
		})
	}
}

package docker

import (
	"os"

	"github.com/drone-plugins/drone-plugin-lib/drone"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	version = "unknown"
)

func Run() {
	// Load env-file if it exists first
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	app := cli.NewApp()
	app.Name = "docker plugin"
	app.Usage = "docker plugin"
	app.Action = run
	app.Version = version
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "dry-run",
			Usage:  "dry run disables docker push",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.StringFlag{
			Name:   "remote.url",
			Usage:  "git remote url",
			EnvVar: "DRONE_REMOTE_URL",
		},
		cli.StringFlag{
			Name:   "commit.sha",
			Usage:  "git commit sha",
			EnvVar: "DRONE_COMMIT_SHA",
			Value:  "00000000",
		},
		cli.StringFlag{
			Name:   "commit.ref",
			Usage:  "git commit ref",
			EnvVar: "DRONE_COMMIT_REF",
		},
		cli.StringFlag{
			Name:   "daemon.mirror",
			Usage:  "docker daemon registry mirror",
			EnvVar: "PLUGIN_MIRROR,DOCKER_PLUGIN_MIRROR",
		},
		cli.StringFlag{
			Name:   "daemon.storage-driver",
			Usage:  "docker daemon storage driver",
			EnvVar: "PLUGIN_STORAGE_DRIVER",
		},
		cli.StringFlag{
			Name:   "daemon.storage-path",
			Usage:  "docker daemon storage path",
			Value:  "/var/lib/docker",
			EnvVar: "PLUGIN_STORAGE_PATH",
		},
		cli.StringFlag{
			Name:   "daemon.bip",
			Usage:  "docker daemon bride ip address",
			EnvVar: "PLUGIN_BIP",
		},
		cli.StringFlag{
			Name:   "daemon.mtu",
			Usage:  "docker daemon custom mtu setting",
			EnvVar: "PLUGIN_MTU",
		},
		cli.StringSliceFlag{
			Name:   "daemon.dns",
			Usage:  "docker daemon dns server",
			EnvVar: "PLUGIN_CUSTOM_DNS",
		},
		cli.StringSliceFlag{
			Name:   "daemon.dns-search",
			Usage:  "docker daemon dns search domains",
			EnvVar: "PLUGIN_CUSTOM_DNS_SEARCH",
		},
		cli.BoolFlag{
			Name:   "daemon.insecure",
			Usage:  "docker daemon allows insecure registries",
			EnvVar: "PLUGIN_INSECURE",
		},
		cli.BoolFlag{
			Name:   "daemon.ipv6",
			Usage:  "docker daemon IPv6 networking",
			EnvVar: "PLUGIN_IPV6",
		},
		cli.BoolFlag{
			Name:   "daemon.debug",
			Usage:  "docker daemon executes in debug mode",
			EnvVar: "PLUGIN_DEBUG,DOCKER_LAUNCH_DEBUG",
		},
		cli.BoolFlag{
			Name:   "daemon.off",
			Usage:  "don't start the docker daemon",
			EnvVar: "PLUGIN_DAEMON_OFF",
		},
		cli.StringFlag{
			Name:   "artifact.registry",
			Usage:  "artifact registry",
			Value:  "https://index.docker.io/v1/",
			EnvVar: "ARTIFACT_REGISTRY,PLUGIN_REGISTRY,DOCKER_REGISTRY",
		},
		cli.StringFlag{
			Name:   "dockerfile",
			Usage:  "build dockerfile",
			Value:  "Dockerfile",
			EnvVar: "PLUGIN_DOCKERFILE",
		},
		cli.StringFlag{
			Name:   "context",
			Usage:  "build context",
			Value:  ".",
			EnvVar: "PLUGIN_CONTEXT",
		},
		cli.StringSliceFlag{
			Name:     "tags",
			Usage:    "build tags",
			Value:    &cli.StringSlice{"latest"},
			EnvVar:   "PLUGIN_TAG,PLUGIN_TAGS",
			FilePath: ".tags",
		},
		cli.BoolFlag{
			Name:   "tags.auto",
			Usage:  "default build tags",
			EnvVar: "PLUGIN_DEFAULT_TAGS,PLUGIN_AUTO_TAG",
		},
		cli.StringFlag{
			Name:   "tags.suffix",
			Usage:  "default build tags with suffix",
			EnvVar: "PLUGIN_DEFAULT_SUFFIX,PLUGIN_AUTO_TAG_SUFFIX",
		},
		cli.StringSliceFlag{
			Name:   "args",
			Usage:  "build args",
			EnvVar: "PLUGIN_BUILD_ARGS",
		},
		cli.StringSliceFlag{
			Name:   "args-from-env",
			Usage:  "build args",
			EnvVar: "PLUGIN_BUILD_ARGS_FROM_ENV",
		},
		cli.GenericFlag{
			Name:   "args-new",
			Usage:  "build args new",
			EnvVar: "PLUGIN_BUILD_ARGS_NEW",
			Value:  new(CustomStringSliceFlag),
		},
		cli.BoolFlag{
			Name:   "plugin-multiple-build-agrs",
			Usage:  "plugin multiple build agrs",
			EnvVar: "PLUGIN_MULTIPLE_BUILD_ARGS",
		},
		cli.BoolFlag{
			Name:   "quiet",
			Usage:  "quiet docker build",
			EnvVar: "PLUGIN_QUIET",
		},
		cli.StringFlag{
			Name:   "target",
			Usage:  "build target",
			EnvVar: "PLUGIN_TARGET",
		},
		cli.GenericFlag{
			Name:   "cache-from",
			Usage:  "cache import location",
			EnvVar: "PLUGIN_CACHE_FROM",
			Value:  new(CustomStringSliceFlag),
		},
		cli.GenericFlag{
			Name:   "cache-to",
			Usage:  "cache export location",
			EnvVar: "PLUGIN_CACHE_TO",
			Value:  new(CustomStringSliceFlag),
		},
		cli.BoolFlag{
			Name:   "squash",
			Usage:  "squash the layers at build time",
			EnvVar: "PLUGIN_SQUASH",
		},
		cli.BoolTFlag{
			Name:   "pull-image",
			Usage:  "force pull base image at build time",
			EnvVar: "PLUGIN_PULL_IMAGE",
		},
		cli.BoolFlag{
			Name:   "compress",
			Usage:  "compress the build context using gzip",
			EnvVar: "PLUGIN_COMPRESS",
		},
		cli.StringFlag{
			Name:   "repo",
			Usage:  "docker repository",
			EnvVar: "PLUGIN_REPO",
		},
		cli.StringSliceFlag{
			Name:   "custom-labels",
			Usage:  "additional k=v labels",
			EnvVar: "PLUGIN_CUSTOM_LABELS",
		},
		cli.StringSliceFlag{
			Name:   "label-schema",
			Usage:  "label-schema labels",
			EnvVar: "PLUGIN_LABEL_SCHEMA",
		},
		cli.BoolTFlag{
			Name:   "auto-label",
			Usage:  "auto-label true|false",
			EnvVar: "PLUGIN_AUTO_LABEL",
		},
		cli.StringFlag{
			Name:   "link",
			Usage:  "link https://example.com/org/repo-name",
			EnvVar: "PLUGIN_REPO_LINK,DRONE_REPO_LINK",
		},
		cli.StringFlag{
			Name:   "docker.registry",
			Usage:  "docker registry",
			Value:  "https://index.docker.io/v1/",
			EnvVar: "PLUGIN_REGISTRY,DOCKER_REGISTRY",
		},
		cli.StringFlag{
			Name:   "docker.username",
			Usage:  "docker username",
			EnvVar: "PLUGIN_USERNAME,DOCKER_USERNAME",
		},
		cli.StringFlag{
			Name:   "docker.password",
			Usage:  "docker password",
			EnvVar: "PLUGIN_PASSWORD,DOCKER_PASSWORD",
		},
		cli.StringFlag{
			Name:   "docker.baseimageusername",
			Usage:  "Docker username for base image registry",
			EnvVar: "PLUGIN_DOCKER_USERNAME,PLUGIN_BASE_IMAGE_USERNAME",
		},
		cli.StringFlag{
			Name:   "docker.baseimagepassword",
			Usage:  "Docker password for base image registry",
			EnvVar: "PLUGIN_DOCKER_PASSWORD,PLUGIN_BASE_IMAGE_PASSWORD",
		},
		cli.StringFlag{
			Name:   "docker.baseimageregistry",
			Usage:  "Docker registry for base image registry",
			EnvVar: "PLUGIN_DOCKER_REGISTRY,PLUGIN_BASE_IMAGE_REGISTRY",
		},
		cli.StringFlag{
			Name:   "docker.email",
			Usage:  "docker email",
			EnvVar: "PLUGIN_EMAIL,DOCKER_EMAIL",
		},
		cli.StringFlag{
			Name:   "docker.config",
			Usage:  "docker json dockerconfig content",
			EnvVar: "PLUGIN_CONFIG,DOCKER_PLUGIN_CONFIG",
		},
		cli.BoolTFlag{
			Name:   "docker.purge",
			Usage:  "docker should cleanup images",
			EnvVar: "PLUGIN_PURGE",
		},
		cli.StringFlag{
			Name:   "repo.branch",
			Usage:  "repository default branch",
			EnvVar: "DRONE_REPO_BRANCH",
		},
		cli.BoolFlag{
			Name:   "no-cache",
			Usage:  "do not use cached intermediate containers",
			EnvVar: "PLUGIN_NO_CACHE",
		},
		cli.StringSliceFlag{
			Name:   "add-host",
			Usage:  "additional host:IP mapping",
			EnvVar: "PLUGIN_ADD_HOST",
		},
		cli.StringFlag{
			Name:   "secret",
			Usage:  "secret key value pair eg id=MYSECRET",
			EnvVar: "PLUGIN_SECRET",
		},
		cli.StringSliceFlag{
			Name:   "secrets-from-env",
			Usage:  "secret key value pair eg secret_name=secret",
			EnvVar: "PLUGIN_SECRETS_FROM_ENV",
		},
		cli.StringSliceFlag{
			Name:   "secrets-from-file",
			Usage:  "secret key value pairs eg secret_name=/path/to/secret",
			EnvVar: "PLUGIN_SECRETS_FROM_FILE",
		},
		cli.StringFlag{
			Name:   "drone-card-path",
			Usage:  "card path location to write to",
			EnvVar: "DRONE_CARD_PATH",
		},
		cli.StringFlag{
			Name:   "platform",
			Usage:  "platform value to pass to docker",
			EnvVar: "PLUGIN_PLATFORM",
		},
		cli.StringFlag{
			Name:   "ssh-agent-key",
			Usage:  "ssh agent key to use",
			EnvVar: "PLUGIN_SSH_AGENT_KEY",
		},
		cli.StringFlag{
			Name:   "builder-name",
			EnvVar: "PLUGIN_BUILDER_NAME",
		},
		cli.StringFlag{
			Name:   "builder-daemon-config",
			Usage:  "Path to config file for Buildkit daemon",
			EnvVar: "PLUGIN_BUILDER_CONFIG",
		},
		cli.StringFlag{
			Name:   "builder-driver",
			EnvVar: "PLUGIN_BUILDER_DRIVER",
		},
		cli.GenericFlag{
			Name:   "builder-driver-opts",
			Usage:  "buildx builder driver opts",
			EnvVar: "PLUGIN_BUILDER_DRIVER_OPTS",
			Value:  new(CustomStringSliceFlag),
		},
		cli.GenericFlag{
			Name:   "builder-driver-opts-new",
			Usage:  "buildx builder driver opts new",
			EnvVar: "PLUGIN_BUILDER_DRIVER_OPTS_NEW",
			Value:  new(CustomStringSliceFlag),
		},
		cli.StringFlag{
			Name:   "builder-remote-conn",
			EnvVar: "PLUGIN_BUILDER_REMOTE_CONN",
		},
		cli.BoolFlag{
			Name:   "buildx-load",
			EnvVar: "PLUGIN_BUILDX_LOAD",
		},
		cli.StringFlag{
			Name:   "metadata-file",
			Usage:  "Location of metadata file that will be generated by the plugin. This file will include information of docker images that are uploaded by the plugin which will be used to create the artifact file.",
			EnvVar: "PLUGIN_METADATA_FILE",
		},
		cli.StringFlag{
			Name:   "artifact-file",
			Usage:  "Artifact file location that will be generated by the plugin. This file will include information of docker images that are uploaded by the plugin.",
			EnvVar: "PLUGIN_ARTIFACT_FILE",
		},
		cli.StringFlag{
			Name:   "cache-metrics-file",
			Usage:  "Location of cache metrics file that will be generated by the plugin.",
			EnvVar: "PLUGIN_CACHE_METRICS_FILE",
		},
		cli.StringFlag{
			Name:   "registry-type",
			Usage:  "registry type",
			EnvVar: "PLUGIN_REGISTRY_TYPE",
		},
		cli.StringFlag{
			Name:   "access-token",
			Usage:  "access token",
			EnvVar: "ACCESS_TOKEN",
		},
		cli.BoolTFlag{
			Name:   "use-loaded-buildkit",
			Usage:  "Use preloaded buildkit image. Default is true.",
			EnvVar: "PLUGIN_USE_LOADED_BUILDKIT",
		},
		cli.StringFlag{
			Name:   "buildkit-assets-dir",
			Usage:  "directory where buildkit assets are stored",
			Value:  "buildkit",
			EnvVar: "PLUGIN_BUILDKIT_ASSETS_DIR",
		},
		cli.StringFlag{
			Name:   "buildkit-version",
			Usage:  "Buildkit version to use",
			EnvVar: "PLUGIN_BUILDKIT_VERSION",
		},
		cli.StringFlag{
			Name:   "harness-self-hosted-s3-access-key",
			Usage:  "build target",
			EnvVar: "PLUGIN_HARNESS_SELF_HOSTED_S3_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "harness-self-hosted-s3-secret-key",
			Usage:  "build target",
			EnvVar: "PLUGIN_HARNESS_SELF_HOSTED_S3_SECRET_KEY",
		},
		cli.StringFlag{
			Name:   "harness-self-hosted-gcp-json-key",
			Usage:  "build target",
			EnvVar: "PLUGIN_HARNESS_SELF_HOSTED_GCP_JSON_KEY",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	registryType := drone.Docker
	if c.String("registry-type") != "" {
		registryType = drone.RegistryType(c.String("registry-type"))
	}

	plugin := Plugin{
		Dryrun:  c.Bool("dry-run"),
		Cleanup: c.BoolT("docker.purge"),
		Login: Login{
			Registry:    c.String("docker.registry"),
			Username:    c.String("docker.username"),
			Password:    c.String("docker.password"),
			Email:       c.String("docker.email"),
			Config:      c.String("docker.config"),
			AccessToken: c.String("access-token"),
		},
		CardPath:         c.String("drone-card-path"),
		MetadataFile:     c.String("metadata-file"),
		ArtifactFile:     c.String("artifact-file"),
		CacheMetricsFile: c.String("cache-metrics-file"),
		Build: Build{
			Remote:                       c.String("remote.url"),
			Name:                         c.String("commit.sha"),
			Dockerfile:                   c.String("dockerfile"),
			Context:                      c.String("context"),
			Tags:                         c.StringSlice("tags"),
			Args:                         c.StringSlice("args"),
			ArgsEnv:                      c.StringSlice("args-from-env"),
			ArgsNew:                      c.Generic("args-new").(*CustomStringSliceFlag).GetValue(),
			IsMultipleBuildArgs:          c.Bool("plugin-multiple-build-agrs"),
			Target:                       c.String("target"),
			Squash:                       c.Bool("squash"),
			Pull:                         c.BoolT("pull-image"),
			CacheFrom:                    c.Generic("cache-from").(*CustomStringSliceFlag).GetValue(),
			CacheTo:                      c.Generic("cache-to").(*CustomStringSliceFlag).GetValue(),
			Compress:                     c.Bool("compress"),
			Repo:                         c.String("repo"),
			Labels:                       c.StringSlice("custom-labels"),
			LabelSchema:                  c.StringSlice("label-schema"),
			AutoLabel:                    c.BoolT("auto-label"),
			Link:                         c.String("link"),
			NoCache:                      c.Bool("no-cache"),
			Secret:                       c.String("secret"),
			SecretEnvs:                   c.StringSlice("secrets-from-env"),
			SecretFiles:                  c.StringSlice("secrets-from-file"),
			AddHost:                      c.StringSlice("add-host"),
			Quiet:                        c.Bool("quiet"),
			Platform:                     c.String("platform"),
			SSHAgentKey:                  c.String("ssh-agent-key"),
			BuildxLoad:                   c.Bool("buildx-load"),
			HarnessSelfHostedS3AccessKey: c.String("harness-self-hosted-s3-access-key"),
			HarnessSelfHostedS3SecretKey: c.String("harness-self-hosted-s3-secret-key"),
			HarnessSelfHostedGcpJsonKey:  c.String("harness-self-hosted-gcp-json-key"),
		},
		Daemon: Daemon{
			Registry:         c.String("docker.registry"),
			Mirror:           c.String("daemon.mirror"),
			StorageDriver:    c.String("daemon.storage-driver"),
			StoragePath:      c.String("daemon.storage-path"),
			Insecure:         c.Bool("daemon.insecure"),
			Disabled:         c.Bool("daemon.off"),
			IPv6:             c.Bool("daemon.ipv6"),
			Debug:            c.Bool("daemon.debug"),
			Bip:              c.String("daemon.bip"),
			DNS:              c.StringSlice("daemon.dns"),
			DNSSearch:        c.StringSlice("daemon.dns-search"),
			MTU:              c.String("daemon.mtu"),
			RegistryType:     registryType,
			ArtifactRegistry: c.String("artifact.registry"),
		},
		Builder: Builder{
			Name:              c.String("builder-name"),
			DaemonConfig:      c.String("builder-daemon-config"),
			Driver:            c.String("builder-driver"),
			DriverOpts:        c.Generic("builder-driver-opts").(*CustomStringSliceFlag).GetValue(),
			DriverOptsNew:     c.Generic("builder-driver-opts-new").(*CustomStringSliceFlag).GetValue(),
			RemoteConn:        c.String("builder-remote-conn"),
			UseLoadedBuildkit: c.BoolT("use-loaded-buildkit"),
			AssestsDir:        c.String("buildkit-assets-dir"),
			BuildkitVersion:   c.String("buildkit-version"),
		},
		BaseImageRegistry: c.String("docker.baseimageregistry"),
		BaseImageUsername: c.String("docker.baseimageusername"),
		BaseImagePassword: c.String("docker.baseimagepassword"),
	}

	if c.Bool("tags.auto") {
		if UseDefaultTag( // return true if tag event or default branch
			c.String("commit.ref"),
			c.String("repo.branch"),
		) {
			tag, err := DefaultTagSuffix(
				c.String("commit.ref"),
				c.String("tags.suffix"),
			)
			if err != nil {
				logrus.Printf("cannot build docker image for %s, invalid semantic version", c.String("commit.ref"))
				return err
			}
			plugin.Build.Tags = tag
		} else {
			logrus.Printf("skipping automated docker build for %s", c.String("commit.ref"))
			return nil
		}
	}

	return plugin.Exec()
}

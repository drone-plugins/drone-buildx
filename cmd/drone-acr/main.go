package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	docker "github.com/drone-plugins/drone-buildx"
)

func main() {
	// Load env-file if it exists first
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	var (
		repo     = getenv("PLUGIN_REPO")
		registry = getenv("PLUGIN_REGISTRY")
		username = getenv("SERVICE_PRINCIPAL_CLIENT_ID")
		password = getenv("SERVICE_PRINCIPAL_CLIENT_SECRET")
	)

	// default registry value
	if registry == "" {
		registry = "azurecr.io"
	}

	// must use the fully qualified repo name. If the
	// repo name does not have the registry prefix we
	// should prepend.
	if !strings.HasPrefix(repo, registry) {
		repo = fmt.Sprintf("%s/%s", registry, repo)
	}

	os.Setenv("PLUGIN_REPO", repo)
	os.Setenv("PLUGIN_REGISTRY", registry)
	os.Setenv("DOCKER_USERNAME", username)
	os.Setenv("DOCKER_PASSWORD", password)

	// invoke the base docker buildx plugin
	docker.Run()
}

func getenv(key ...string) (s string) {
	for _, k := range key {
		s = os.Getenv(k)
		if s != "" {
			return
		}
	}
	return
}

package main

import (
	"fmt"
	"log"
	"os"

	docker "github.com/drone-plugins/drone-buildx"
)

func main() {
	// Create an isolated Docker config directory for this invocation so that
	// parallel Docker build steps on the same host VM don't race on the shared
	// /root/.docker/config.json file. Docker CLI respects the DOCKER_CONFIG
	// env var automatically for all docker login / docker buildx build calls.
	isolatedDockerConfig, err := os.MkdirTemp("", "harness-buildx-docker-config-*")
	if err != nil {
		log.Fatal(fmt.Sprintf("error creating isolated docker config dir: %v", err))
	}
	os.Setenv("DOCKER_CONFIG", isolatedDockerConfig)

	docker.Run()
}

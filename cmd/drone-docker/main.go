package main

import (
	"log"
	"os"

	docker "github.com/drone-plugins/drone-buildx"
)

func main() {
	// Create an isolated Docker config directory for this invocation so that
	// parallel Docker build steps on the same host VM don't race on the shared
	// /root/.docker/config.json file. Docker CLI respects the DOCKER_CONFIG
	// env var automatically for all docker login / docker buildx build calls.
	// On failure, fall back to the default Docker config dir so the step can
	// still succeed (e.g. if /tmp is full but the root filesystem is not).
	isolatedDockerConfig, err := os.MkdirTemp("", "harness-buildx-docker-config-*")
	if err != nil {
		log.Printf("warning: could not create isolated docker config dir, falling back to default: %v", err)
	} else {
		os.Setenv("DOCKER_CONFIG", isolatedDockerConfig)
	}

	docker.Run()
}
